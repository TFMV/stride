// Package stride provides concurrent filesystem traversal with filtering and progress reporting.
package stride

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/karrick/godirwalk"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DefaultConcurrentWalks defines the default number of concurrent workers
// when no specific limit is provided.
const DefaultConcurrentWalks int = 100

// --------------------------------------------------------------------------
// Core types for progress monitoring
// --------------------------------------------------------------------------

// ProgressFn is called periodically with traversal statistics.
// Implementations must be thread-safe as this may be called concurrently.
type ProgressFn func(stats Stats)

// Stats holds traversal statistics that are updated atomically during the walk.
type Stats struct {
	FilesProcessed int64         // Number of files processed
	DirsProcessed  int64         // Number of directories processed
	EmptyDirs      int64         // Number of empty directories
	BytesProcessed int64         // Total bytes processed
	ErrorCount     int64         // Number of errors encountered
	ElapsedTime    time.Duration // Total time elapsed
	AvgFileSize    int64         // Average file size in bytes
	SpeedMBPerSec  float64       // Processing speed in MB/s
}

// updateDerivedStats calculates derived statistics like averages and speeds.
func (s *Stats) updateDerivedStats() {
	filesProcessed := atomic.LoadInt64(&s.FilesProcessed)
	bytesProcessed := atomic.LoadInt64(&s.BytesProcessed)

	if filesProcessed > 0 {
		s.AvgFileSize = bytesProcessed / filesProcessed
	}

	elapsedSec := s.ElapsedTime.Seconds()
	if elapsedSec > 0 && bytesProcessed > 0 {
		megabytes := float64(bytesProcessed) / (1024.0 * 1024.0)
		s.SpeedMBPerSec = megabytes / elapsedSec
	} else {
		s.SpeedMBPerSec = 0
	}
}

// --------------------------------------------------------------------------
// Configuration types
// --------------------------------------------------------------------------

// ErrorHandling defines how errors are handled during traversal.
type ErrorHandling int

const (
	ErrorHandlingContinue ErrorHandling = iota // Continue on errors
	ErrorHandlingStop                          // Stop on first error
	ErrorHandlingSkip                          // Skip problematic files/dirs
)

// SymlinkHandling defines how symbolic links are processed.
type SymlinkHandling int

const (
	SymlinkFollow SymlinkHandling = iota // Follow symbolic links
	SymlinkIgnore                        // Ignore symbolic links
	SymlinkReport                        // Report links but don't follow
)

// MemoryLimit sets memory usage boundaries for the traversal.  Not implemented in this example.
type MemoryLimit struct {
	SoftLimit int64 // Pause processing when reached
	HardLimit int64 // Stop processing when reached
}

// LogLevel defines the verbosity of logging.
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// WalkOptions provides comprehensive configuration for the walk operation.
type WalkOptions struct {
	// Core options
	Context           context.Context   // Context for cancellation and deadlines
	ErrorHandling     ErrorHandling     // Legacy error handling mode
	ErrorHandlingMode ErrorHandlingMode // String-based error handling mode
	Filter            FilterOptions     // File filtering options

	// Progress monitoring
	Progress         ProgressFn        // Legacy progress function
	ProgressCallback func(stats Stats) // Enhanced progress callback

	// Logging and debug
	Logger   *zap.Logger // Structured logger
	LogLevel LogLevel    // Logging verbosity level
	DryRun   bool        // Simulate operations without executing

	// Performance tuning
	BufferSize  int // Size of internal buffers
	NumWorkers  int // Legacy worker count
	WorkerCount int // Enhanced worker count

	// Special handling
	SymlinkHandling SymlinkHandling    // How to handle symbolic links
	MemoryLimit     MemoryLimit        // Legacy memory limits
	MemoryLimits    MemoryLimitOptions // Enhanced memory limits

	// Extensibility
	Middleware []MiddlewareFunc // Middleware functions for customization
}

// FilterOptions defines criteria for including/excluding files and directories.
type FilterOptions struct {
	MinSize             int64       // Minimum file size in bytes
	MaxSize             int64       // Maximum file size in bytes
	Pattern             string      // Glob pattern for matching files
	ExcludeDir          []string    // Directory patterns to exclude
	IncludeTypes        []string    // File extensions to include (e.g. ".txt", ".go")
	FileTypes           []string    // File types to include (file, dir, symlink)
	ExcludePattern      []string    // Patterns to exclude files
	ModifiedAfter       time.Time   // Only include files modified after
	ModifiedBefore      time.Time   // Only include files modified before
	AccessedAfter       time.Time   // Include files accessed after this time
	AccessedBefore      time.Time   // Include files accessed before this time
	CreatedAfter        time.Time   // Include files created after this time
	CreatedBefore       time.Time   // Include files created before this time
	MinPermissions      os.FileMode // Minimum file permissions (e.g. 0644)
	MaxPermissions      os.FileMode // Maximum file permissions (e.g. 0755)
	ExactPermissions    os.FileMode // Exact file permissions to match
	UseExactPermissions bool        // Whether to use exact permissions matching
	OwnerUID            int         // Filter by owner UID
	OwnerGID            int         // Filter by group GID
	OwnerName           string      // Filter by owner username
	GroupName           string      // Filter by group name
	MinDepth            int         // Minimum traversal depth
	MaxDepth            int         // Maximum traversal depth
	IncludeEmptyFiles   bool        // Include only empty files
	IncludeEmptyDirs    bool        // Include only empty directories
}

// --------------------------------------------------------------------------
// Primary API functions
// --------------------------------------------------------------------------

// Walk traverses a directory tree using the default concurrency limit.
// It's a convenience wrapper around WalkLimit.
func Walk(root string, walkFn filepath.WalkFunc) error {
	return WalkLimit(context.Background(), root, walkFn, DefaultConcurrentWalks)
}

// WalkLimit traverses a directory tree with a specified concurrency limit.
// It distributes work across a pool of goroutines while respecting context cancellation.
// Directories are processed synchronously so that a SkipDir result prevents descending.
func WalkLimit(ctx context.Context, root string, walkFn filepath.WalkFunc, limit int) error {
	if limit < 1 {
		return errors.New("stride: concurrency limit must be greater than zero")
	}

	logger := createLogger(LogLevelInfo) // Default log level
	defer logger.Sync()

	logger.Debug("starting walk", zap.String("root", root), zap.Int("workers", limit))

	tasks := make(chan walkArgs, limit)
	var tasksWg sync.WaitGroup
	var workerWg sync.WaitGroup

	// Error collection.
	var walkErrors []error
	var errLock sync.Mutex

	// Worker processes tasks (files only).
	worker := func() {
		defer workerWg.Done()
		for task := range tasks {
			if ctx.Err() != nil {
				logger.Debug("worker canceled", zap.String("path", task.path))
				tasksWg.Done()
				continue
			}
			if err := walkFn(task.path, task.info, task.err); err != nil {
				// Do not collect SkipDir errors.
				if !errors.Is(err, filepath.SkipDir) {
					errLock.Lock()
					walkErrors = append(walkErrors, fmt.Errorf("path %q: %w", task.path, err))
					errLock.Unlock()
				}
			}
			tasksWg.Done()
		}
	}

	// Launch worker pool.
	for i := 0; i < limit; i++ {
		workerWg.Add(1)
		go worker()
	}

	// Create godirwalk.Options with the callback function.
	options := &godirwalk.Options{
		Callback: func(path string, info *godirwalk.Dirent) error {
			if ctx.Err() != nil {
				logger.Warn("walk canceled", zap.String("path", path))
				return context.Canceled
			}
			// Convert godirwalk.Dirent to os.FileInfo
			fileInfo, err := os.Lstat(path)
			if err != nil {
				return err
			}

			// For directories, process synchronously so that SkipDir is honored.
			if fileInfo.IsDir() {
				ret := walkFn(path, fileInfo, nil)
				if errors.Is(ret, filepath.SkipDir) {
					return filepath.SkipDir
				}
				if ret != nil {
					errLock.Lock()
					walkErrors = append(walkErrors, fmt.Errorf("path %q: %w", path, ret))
					errLock.Unlock()
				}
			} else {
				// For files, send the task to workers.
				tasksWg.Add(1)
				select {
				case <-ctx.Done():
					tasksWg.Done()
					return context.Canceled
				case tasks <- walkArgs{path: path, info: fileInfo, err: nil}: // Pass fileInfo
				}
			}
			return nil
		},
		// Default to not following symlinks for backward compatibility
		FollowSymbolicLinks: false,
	}

	// Use godirwalk.Walk with the options.
	err := godirwalk.Walk(root, options)
	if err != nil && !errors.Is(err, filepath.SkipDir) {
		errLock.Lock()
		walkErrors = append(walkErrors, err)
		errLock.Unlock()
	}

	close(tasks)
	workerWg.Wait()

	// Collect errors.
	if len(walkErrors) > 0 {
		// If there's only one error and it's context.Canceled, return it directly
		if len(walkErrors) == 1 && (errors.Is(walkErrors[0], context.Canceled) ||
			walkErrors[0].Error() == "context canceled") {
			return context.Canceled
		}

		// Check if all errors are the same custom error
		if len(walkErrors) > 0 {
			firstErr := walkErrors[0]
			allSame := true
			for _, err := range walkErrors[1:] {
				if !errors.Is(err, firstErr) && err.Error() != firstErr.Error() {
					allSame = false
					break
				}
			}
			if allSame {
				return firstErr
			}
		}

		return fmt.Errorf("multiple errors: %v", walkErrors)
	}
	return nil
}

// WalkLimitWithProgress adds progress monitoring to the walk operation.
func WalkLimitWithProgress(ctx context.Context, root string, walkFn filepath.WalkFunc, limit int, progressFn ProgressFn) error {
	stats := &Stats{}
	startTime := time.Now()

	// Ensure a final progress update even on early return.
	defer func() {
		stats.ElapsedTime = time.Since(startTime)
		stats.updateDerivedStats()
		progressFn(*stats)
	}()

	doneCh := make(chan struct{})
	var tickerWg sync.WaitGroup
	tickerWg.Add(1)
	go func() {
		defer tickerWg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-doneCh:
				return
			case <-ticker.C:
				stats.ElapsedTime = time.Since(startTime)
				stats.updateDerivedStats()
				progressFn(*stats)
			}
		}
	}()

	// Wrap walkFn to update progress statistics.
	wrappedWalkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			atomic.AddInt64(&stats.ErrorCount, 1)
			return err
		}
		if info.IsDir() {
			atomic.AddInt64(&stats.DirsProcessed, 1)
			if !hasFiles(path) {
				atomic.AddInt64(&stats.EmptyDirs, 1)
			}
		} else {
			size := info.Size()
			atomic.AddInt64(&stats.FilesProcessed, 1)
			atomic.AddInt64(&stats.BytesProcessed, size)
		}
		err = walkFn(path, info, nil) // Pass nil for err
		if err != nil {
			atomic.AddInt64(&stats.ErrorCount, 1)
		}
		return err
	}

	err := WalkLimit(ctx, root, wrappedWalkFn, limit)
	close(doneCh)
	tickerWg.Wait()
	return err
}

// Thread-safe maps for caching.
var (
	excludedDirs    sync.Map // Cache of excluded directories
	visitedSymlinks sync.Map // Cache of visited symlinks to detect cycles, keyed by initial symlink path
	symlinkLock     sync.RWMutex
)

// isCyclicSymlink checks if following a symlink would create a cycle.
func isCyclicSymlink(initialPath, realPath string) bool {
	// Check if the initial path or the real path has been seen.
	if _, seen := visitedSymlinks.Load(initialPath); seen {
		return true
	}

	// Check the real path against *all* visited paths (initial or real).
	var isCyclic bool
	visitedSymlinks.Range(func(key, _ interface{}) bool {
		if key.(string) == realPath {
			isCyclic = true
			return false // Stop iterating
		}
		return true // Continue iterating
	})
	if isCyclic {
		return true
	}

	// Mark both initial and resolved paths as visited.
	visitedSymlinks.Store(initialPath, struct{}{})
	visitedSymlinks.Store(realPath, struct{}{})
	return false
}

// resolveSymlink handles symlink resolution and cycle detection.
func resolveSymlink(path string, symlinkHandling SymlinkHandling) (string, os.FileInfo, bool, error) {
	fileInfo, err := os.Lstat(path) // Start with Lstat
	if err != nil {
		return "", nil, false, err
	}

	if fileInfo.Mode()&os.ModeSymlink == 0 {
		return path, fileInfo, false, nil // Not a symlink
	}

	if symlinkHandling == SymlinkIgnore {
		return "", nil, true, nil // Ignore symlinks
	}

	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", nil, true, err // Error evaluating symlink
	}
	realFileInfo, err := os.Stat(realPath)
	if err != nil {
		return "", nil, true, err
	}

	symlinkLock.RLock()
	defer symlinkLock.RUnlock()
	if isCyclicSymlink(path, realPath) {
		return "", nil, true, nil // Cyclic symlink, skip
	}

	return realPath, realFileInfo, true, nil // Resolved, not cyclic
}

// shouldSkipDir checks if a directory should be excluded, using a cached result.
func shouldSkipDir(path, root string, excludes []string) bool {
	// Use the cache.
	if excluded, ok := excludedDirs.Load(path); ok && excluded.(bool) {
		return true
	}

	dir := path
	for dir != root && dir != "." && dir != "/" { // Correct loop condition
		for _, exclude := range excludes {
			if matched, _ := filepath.Match(exclude, filepath.Base(dir)); matched {
				excludedDirs.Store(path, true) // Cache the result
				return true
			}
		}

		// Check if the path ends with a separator
		if dir[len(dir)-1] == os.PathSeparator {
			dir = filepath.Dir(dir[:len(dir)-1]) // Remove the ending separator before getting the parent
		} else {

			dir = filepath.Dir(dir)
		}
	}
	// Also check the root itself
	for _, exclude := range excludes {
		if matched, _ := filepath.Match(exclude, filepath.Base(dir)); matched {
			// Don't cache the root, because it's not a subpath.
			return true
		}
	}
	return false
}

// WalkLimitWithFilter adds file filtering capabilities to the walk operation.
func WalkLimitWithFilter(ctx context.Context, root string, walkFn filepath.WalkFunc, limit int, filter FilterOptions) error {
	root = filepath.Clean(root)
	symlinkLock.Lock()

	visitedSymlinks = sync.Map{} // Reset visited symlinks
	symlinkLock.Unlock()
	filteredWalkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if errors.Is(err, filepath.SkipDir) {
				return err
			}
			return err
		}

		// Resolve symlinks *before* directory checks.
		resolvedPath, resolvedInfo, isSymlink, err := resolveSymlink(path, SymlinkFollow)
		if err != nil {
			return err
		}
		if isSymlink && resolvedInfo == nil {
			return nil // Symlink was ignored or cyclic.
		}
		if isSymlink {
			path = resolvedPath
			info = resolvedInfo
		}

		if info.IsDir() {
			if shouldSkipDir(path, root, filter.ExcludeDir) {
				return filepath.SkipDir
			}
		} else {
			// Check if the parent directory is excluded.
			parent := filepath.Dir(path)
			if shouldSkipDir(parent, root, filter.ExcludeDir) {
				return nil
			}
			// Use the full path when filtering files.
			if !filePassesFilter(path, info, filter, SymlinkFollow) {
				return nil
			}
		}
		// Pass a nil error to the user's walkFn.
		return walkFn(path, info, nil)
	}

	return WalkLimit(ctx, root, filteredWalkFn, limit)
}

// WalkLimitWithOptions provides the most flexible configuration,
// combining error handling, filtering, progress reporting, and optional custom logger/symlink handling.
func WalkLimitWithOptions(ctx context.Context, root string, walkFn filepath.WalkFunc, opts WalkOptions) error {
	if opts.BufferSize < 1 {
		opts.BufferSize = DefaultConcurrentWalks
	}

	if opts.NumWorkers <= 0 {
		opts.NumWorkers = runtime.NumCPU() // Use number of CPUs by default
	}

	logger := opts.Logger
	if logger == nil {
		logger = createLogger(opts.LogLevel)
		defer logger.Sync()
	}

	logger.Debug("starting walk with options",
		zap.String("root", root),
		zap.Int("buffer_size", opts.BufferSize),
		zap.Any("error_handling", opts.ErrorHandling),
		zap.Any("symlink_handling", opts.SymlinkHandling),
	)

	stats := &Stats{}
	startTime := time.Now()
	visitedSymlinks = sync.Map{} // Clear symlink cache

	// Set up periodic progress updates if progress function is provided
	if opts.Progress != nil {
		// Create a ticker to send progress updates periodically
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		// Create a done channel to signal when to stop the ticker
		doneCh := make(chan struct{})
		defer close(doneCh)

		// Start a goroutine to send progress updates
		go func() {
			for {
				select {
				case <-ticker.C:
					// Update elapsed time and derived stats
					stats.ElapsedTime = time.Since(startTime)
					stats.updateDerivedStats()
					opts.Progress(*stats)
				case <-doneCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Track the root depth for MinDepth/MaxDepth filtering
	rootDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))

	wrappedWalkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if opts.Progress != nil {
				atomic.AddInt64(&stats.ErrorCount, 1)
				stats.ElapsedTime = time.Since(startTime)
				stats.updateDerivedStats()
				opts.Progress(*stats)
			}
			switch opts.ErrorHandling {
			case ErrorHandlingContinue, ErrorHandlingSkip:
				return nil
			default:
				return err
			}
		}

		// Check if info is nil to avoid nil pointer dereference
		if info == nil {
			return nil
		}

		// Calculate current depth relative to root
		pathDepth := strings.Count(filepath.Clean(path), string(os.PathSeparator)) - rootDepth

		// Apply depth filtering
		if opts.Filter.MinDepth > 0 && pathDepth < opts.Filter.MinDepth {
			if info.IsDir() && pathDepth < opts.Filter.MinDepth-1 {
				// Continue traversing but don't process
				return nil
			}
			return nil // Skip this file/dir but don't skip its children
		}

		if opts.Filter.MaxDepth > 0 && pathDepth > opts.Filter.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir // Skip this directory and its children
			}
			return nil // Skip this file
		}

		if info.IsDir() {
			if shouldSkipDir(path, root, opts.Filter.ExcludeDir) {
				return filepath.SkipDir
			}
		} else {
			parent := filepath.Dir(path)
			if shouldSkipDir(parent, root, opts.Filter.ExcludeDir) {
				return nil
			}
			if !filePassesFilter(path, info, opts.Filter, opts.SymlinkHandling) {
				return nil
			}
		}

		if opts.Progress != nil {
			if info.IsDir() {
				atomic.AddInt64(&stats.DirsProcessed, 1)
				if !hasFiles(path) {
					atomic.AddInt64(&stats.EmptyDirs, 1)
				}

			} else {
				atomic.AddInt64(&stats.FilesProcessed, 1)
				atomic.AddInt64(&stats.BytesProcessed, info.Size())
			}
		}
		return walkFn(path, info, nil) // Call the users walkFn
	}

	// Use a custom implementation for WalkLimit that respects symlink handling
	finalErr := walkLimitWithSymlinkHandling(ctx, root, wrappedWalkFn, opts.NumWorkers, opts.SymlinkHandling)

	// Stop progress updates
	if opts.Progress != nil {
		stats.ElapsedTime = time.Since(startTime)
		stats.updateDerivedStats()
		opts.Progress(*stats)
	}
	return finalErr
}

// walkLimitWithSymlinkHandling is a version of WalkLimit that respects the SymlinkHandling option
func walkLimitWithSymlinkHandling(ctx context.Context, root string, walkFn filepath.WalkFunc, limit int, symlinkHandling SymlinkHandling) error {
	if limit < 1 {
		return errors.New("stride: concurrency limit must be greater than zero")
	}

	logger := createLogger(LogLevelInfo) // Default log level
	defer logger.Sync()

	logger.Debug("starting walk with symlink handling",
		zap.String("root", root),
		zap.Int("workers", limit),
		zap.Any("symlink_handling", symlinkHandling))

	tasks := make(chan walkArgs, limit)
	var tasksWg sync.WaitGroup
	var workerWg sync.WaitGroup

	// Error collection.
	var walkErrors []error
	var errLock sync.Mutex

	// Worker processes tasks (files only).
	worker := func() {
		defer workerWg.Done()
		for task := range tasks {
			if ctx.Err() != nil {
				logger.Debug("worker canceled", zap.String("path", task.path))
				tasksWg.Done()
				continue
			}
			if err := walkFn(task.path, task.info, task.err); err != nil {
				// Do not collect SkipDir errors.
				if !errors.Is(err, filepath.SkipDir) {
					errLock.Lock()
					walkErrors = append(walkErrors, fmt.Errorf("path %q: %w", task.path, err))
					errLock.Unlock()
				}
			}
			tasksWg.Done()
		}
	}

	// Launch worker pool.
	for i := 0; i < limit; i++ {
		workerWg.Add(1)
		go worker()
	}

	// Create godirwalk.Options with the callback function.
	options := &godirwalk.Options{
		Callback: func(path string, info *godirwalk.Dirent) error {
			if ctx.Err() != nil {
				logger.Warn("walk canceled", zap.String("path", path))
				return context.Canceled
			}
			// Convert godirwalk.Dirent to os.FileInfo
			fileInfo, err := os.Lstat(path)
			if err != nil {
				return err
			}

			// For directories, process synchronously so that SkipDir is honored.
			if fileInfo.IsDir() {
				ret := walkFn(path, fileInfo, nil)
				if errors.Is(ret, filepath.SkipDir) {
					return filepath.SkipDir
				}
				if ret != nil {
					errLock.Lock()
					walkErrors = append(walkErrors, fmt.Errorf("path %q: %w", path, ret))
					errLock.Unlock()
				}
			} else {
				// For files, send the task to workers.
				tasksWg.Add(1)
				select {
				case <-ctx.Done():
					tasksWg.Done()
					return context.Canceled
				case tasks <- walkArgs{path: path, info: fileInfo, err: nil}: // Pass fileInfo
				}
			}
			return nil
		},
		// Set FollowSymbolicLinks based on the SymlinkHandling option
		FollowSymbolicLinks: symlinkHandling == SymlinkFollow,
	}

	// Use godirwalk.Walk with the options.
	err := godirwalk.Walk(root, options)
	if err != nil && !errors.Is(err, filepath.SkipDir) {
		errLock.Lock()
		walkErrors = append(walkErrors, err)
		errLock.Unlock()
	}

	close(tasks)
	workerWg.Wait()

	// Collect errors.
	if len(walkErrors) > 0 {
		// If there's only one error and it's context.Canceled, return it directly
		if len(walkErrors) == 1 && (errors.Is(walkErrors[0], context.Canceled) ||
			walkErrors[0].Error() == "context canceled") {
			return context.Canceled
		}

		// Check if all errors are the same custom error
		if len(walkErrors) > 0 {
			firstErr := walkErrors[0]
			allSame := true
			for _, err := range walkErrors[1:] {
				if !errors.Is(err, firstErr) && err.Error() != firstErr.Error() {
					allSame = false
					break
				}
			}
			if allSame {
				return firstErr
			}
		}

		return fmt.Errorf("multiple errors: %v", walkErrors)
	}
	return nil
}

// --------------------------------------------------------------------------
// Internal helper types and functions
// --------------------------------------------------------------------------

// walkArgs holds the parameters passed to workers.
type walkArgs struct {
	path string
	info os.FileInfo
	err  error
}

// filePassesFilter returns true if the file meets the filtering criteria.
// It uses the full file path for symlink cycle detection.
func filePassesFilter(path string, info os.FileInfo, filter FilterOptions, symlinkHandling SymlinkHandling) bool {
	// Size checks.
	if filter.MinSize > 0 && info.Size() < filter.MinSize {
		return false
	}
	if filter.MaxSize > 0 && info.Size() > filter.MaxSize {
		return false
	}

	// Modification time checks.
	if !filter.ModifiedAfter.IsZero() && info.ModTime().Before(filter.ModifiedAfter) {
		return false
	}
	if !filter.ModifiedBefore.IsZero() && info.ModTime().After(filter.ModifiedBefore) {
		return false
	}

	// Access and creation time checks (platform-dependent)
	if runtime.GOOS != "windows" {
		// On Unix-like systems, we can get access and creation times
		var stat syscall.Stat_t
		if err := syscall.Stat(path, &stat); err == nil {
			// Access time check
			if !filter.AccessedAfter.IsZero() || !filter.AccessedBefore.IsZero() {
				// Use a platform-independent approach to get atime
				atime := getAccessTime(path, info)

				if !filter.AccessedAfter.IsZero() && atime.Before(filter.AccessedAfter) {
					return false
				}
				if !filter.AccessedBefore.IsZero() && atime.After(filter.AccessedBefore) {
					return false
				}
			}

			// Creation time check (birthtime) - not available on all platforms
			// This is a best-effort approach
			if !filter.CreatedAfter.IsZero() || !filter.CreatedBefore.IsZero() {
				// Use a platform-independent approach to get creation time
				ctime := getCreationTime(path, info)

				if !filter.CreatedAfter.IsZero() && ctime.Before(filter.CreatedAfter) {
					return false
				}
				if !filter.CreatedBefore.IsZero() && ctime.After(filter.CreatedBefore) {
					return false
				}
			}

			// Owner and group checks
			if filter.OwnerUID > 0 && int(stat.Uid) != filter.OwnerUID {
				return false
			}
			if filter.OwnerGID > 0 && int(stat.Gid) != filter.OwnerGID {
				return false
			}
		}
	}

	// Owner and group name checks
	if filter.OwnerName != "" || filter.GroupName != "" {
		if runtime.GOOS != "windows" {
			var stat syscall.Stat_t
			if err := syscall.Stat(path, &stat); err == nil {
				// Check owner name
				if filter.OwnerName != "" {
					owner, err := user.LookupId(fmt.Sprintf("%d", stat.Uid))
					if err != nil || owner.Username != filter.OwnerName {
						return false
					}
				}

				// Check group name
				if filter.GroupName != "" {
					group, err := user.LookupGroupId(fmt.Sprintf("%d", stat.Gid))
					if err != nil || group.Name != filter.GroupName {
						return false
					}
				}
			}
		}
	}

	// Glob pattern matching. Use info.Name() (base name) for pattern matching, not the full path!
	if filter.Pattern != "" {
		matched, err := filepath.Match(filter.Pattern, info.Name())
		if err != nil || !matched {
			return false
		}
	}

	// Exclude pattern matching
	if len(filter.ExcludePattern) > 0 {
		for _, pattern := range filter.ExcludePattern {
			matched, err := filepath.Match(pattern, info.Name())
			if err == nil && matched {
				return false
			}
		}
	}

	// Type filtering (extension check).
	if len(filter.IncludeTypes) > 0 {
		ext := filepath.Ext(path)
		matched := false
		for _, includeType := range filter.IncludeTypes {
			if includeType == ext {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// File type filtering
	if len(filter.FileTypes) > 0 {
		var found bool
		mode := info.Mode()
		for _, fileType := range filter.FileTypes {
			switch fileType {
			case "file":
				if mode.IsRegular() {
					found = true
				}
			case "dir":
				if mode.IsDir() {
					found = true
				}
			case "symlink":
				if mode&os.ModeSymlink != 0 {
					found = true
				}
			case "pipe":
				if mode&os.ModeNamedPipe != 0 {
					found = true
				}
			case "socket":
				if mode&os.ModeSocket != 0 {
					found = true
				}
			case "device":
				if mode&os.ModeDevice != 0 {
					found = true
				}
			case "char":
				if mode&os.ModeCharDevice != 0 {
					found = true
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Empty file/directory check
	if filter.IncludeEmptyFiles && !info.IsDir() && info.Size() > 0 {
		return false
	}
	if filter.IncludeEmptyDirs && info.IsDir() {
		// Check if directory is empty
		empty, _ := isDirEmpty(path)
		if !empty {
			return false
		}
	}

	// Permission filtering
	mode := info.Mode().Perm() // Get just the permission bits
	if filter.UseExactPermissions && filter.ExactPermissions != 0 {
		// Exact permission matching
		if mode != filter.ExactPermissions {
			return false
		}
	} else {
		// Range-based permission matching
		if filter.MinPermissions != 0 && mode&filter.MinPermissions != filter.MinPermissions {
			return false
		}
		if filter.MaxPermissions != 0 && mode&^filter.MaxPermissions != 0 {
			return false
		}
	}

	return true
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Read just one entry
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// createLogger creates a zap logger with the specified log level.
func createLogger(level LogLevel) *zap.Logger {
	var config zap.Config

	switch level {
	case LogLevelError:
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case LogLevelWarn:
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case LogLevelInfo:
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case LogLevelDebug:
		config = zap.NewDevelopmentConfig() // Use DevelopmentConfig for more detailed debug output
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Optional: colored output
	default:
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, _ := config.Build()
	return logger
}

// hasFiles checks if a directory contains any entries.
func hasFiles(dir string) bool {
	entries, err := os.ReadDir(dir) // Use the faster ReadDir
	return err == nil && len(entries) > 0
}

// getAccessTime returns the access time of a file
func getAccessTime(path string, info os.FileInfo) time.Time {
	// Use a platform-independent approach to get atime
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err == nil {
		return time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
	}
	return time.Time{}
}

// getCreationTime returns the creation time of a file
func getCreationTime(path string, info os.FileInfo) time.Time {
	// Use a platform-independent approach to get creation time
	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err == nil {
		return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}
	return time.Time{}
}

// WalkFunc defines the signature for file processing callbacks.
type WalkFunc func(ctx context.Context, path string, info os.FileInfo) error

// AdvancedWalkFunc includes statistics for each callback.
type AdvancedWalkFunc func(ctx context.Context, path string, info os.FileInfo, stats Stats) error

// ErrorHandlingMode defines how errors are handled during traversal.
type ErrorHandlingMode string

const (
	ContinueOnError ErrorHandlingMode = "continue"
	StopOnError     ErrorHandlingMode = "stop"
	SkipOnError     ErrorHandlingMode = "skip"
)

// MemoryLimitOptions sets memory usage boundaries for the traversal.
type MemoryLimitOptions struct {
	SoftLimit int64
	HardLimit int64
}

// MiddlewareFunc defines a middleware function for extensibility.
type MiddlewareFunc func(next WalkFunc) WalkFunc

// WalkWithOptions traverses the file tree rooted at root, calling the user-provided walkFn
// for each file or directory in the tree, including root, with the enhanced context-aware API.
func WalkWithOptions(root string, walkFn WalkFunc, options WalkOptions) error {
	// Default context if not provided
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Convert the enhanced WalkFunc to the standard filepath.WalkFunc
	adaptedWalkFn := func(path string, info os.FileInfo, err error) error {
		return walkFn(ctx, path, info)
	}

	// Apply middleware if provided
	if len(options.Middleware) > 0 {
		wrappedFn := walkFn
		// Apply middleware in reverse order (so first in list is outermost)
		for i := len(options.Middleware) - 1; i >= 0; i-- {
			wrappedFn = options.Middleware[i](wrappedFn)
		}

		// Update the adapted function with the middleware-wrapped one
		adaptedWalkFn = func(path string, info os.FileInfo, err error) error {
			return wrappedFn(ctx, path, info)
		}
	}

	// Convert ErrorHandlingMode to ErrorHandling if needed
	if options.ErrorHandlingMode != "" && options.ErrorHandling == 0 {
		switch options.ErrorHandlingMode {
		case ContinueOnError:
			options.ErrorHandling = ErrorHandlingContinue
		case StopOnError:
			options.ErrorHandling = ErrorHandlingStop
		case SkipOnError:
			options.ErrorHandling = ErrorHandlingSkip
		}
	}

	// Use the existing implementation but with our adapted walkFn
	return WalkLimitWithOptions(ctx, root, adaptedWalkFn, options)
}

// WalkWithAdvancedOptions traverses the file tree rooted at root, calling the user-provided advanced walkFn
// for each file or directory in the tree, including root, with access to traversal statistics.
func WalkWithAdvancedOptions(root string, walkFn AdvancedWalkFunc, options WalkOptions) error {
	// Default context if not provided
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}

	stats := Stats{}
	startTime := time.Now()

	// Create a mutex to protect access to stats during updates
	var statsMutex sync.Mutex

	// Setup the progress function to update stats
	originalProgress := options.Progress
	options.Progress = func(s Stats) {
		statsMutex.Lock()
		defer statsMutex.Unlock()

		// Update our stats
		stats = s
		stats.ElapsedTime = time.Since(startTime)
		stats.updateDerivedStats()

		// Call the original progress function if set
		if originalProgress != nil {
			originalProgress(stats)
		}

		// Call the enhanced progress callback if set
		if options.ProgressCallback != nil {
			options.ProgressCallback(stats)
		}
	}

	// Create a WalkFunc that provides stats to the advanced walkFn
	wrappedWalkFn := func(ctx context.Context, path string, info os.FileInfo) error {
		// Get a local copy of the current stats
		statsMutex.Lock()
		localStats := stats
		statsMutex.Unlock()

		return walkFn(ctx, path, info, localStats)
	}

	// Use our standard WalkWithOptions with the wrapped function
	return WalkWithOptions(root, wrappedWalkFn, options)
}
