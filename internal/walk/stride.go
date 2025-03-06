// Package stride provides concurrent filesystem traversal with filtering and progress reporting.
package stride

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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
	ErrorHandling   ErrorHandling
	Filter          FilterOptions
	Progress        ProgressFn
	Logger          *zap.Logger
	LogLevel        LogLevel // New field for logging verbosity
	BufferSize      int
	SymlinkHandling SymlinkHandling
	MemoryLimit     MemoryLimit // No-op in this implementation, but included for future expansion
	NumWorkers      int         // Explicit worker count.
}

// FilterOptions defines criteria for including/excluding files and directories.
type FilterOptions struct {
	MinSize        int64     // Minimum file size in bytes
	MaxSize        int64     // Maximum file size in bytes
	Pattern        string    // Glob pattern for matching files
	ExcludeDir     []string  // Directory patterns to exclude
	IncludeTypes   []string  // File extensions to include (e.g. ".txt", ".go")
	ModifiedAfter  time.Time // Only include files modified after
	ModifiedBefore time.Time // Only include files modified before
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
	}

	// Use godirwalk.Walk with the options.
	err := godirwalk.Walk(root, options)
	if err != nil && !errors.Is(err, filepath.SkipDir) {
		errLock.Lock()
		walkErrors = append(walkErrors, err)
		errLock.Unlock()
	}

	close(tasks)
	tasksWg.Wait()
	workerWg.Wait()

	if len(walkErrors) > 0 {
		return errors.Join(walkErrors...)
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
	if opts.BufferSize <= 0 {
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
		zap.Int("num_workers", opts.NumWorkers),
		zap.Any("error_handling", opts.ErrorHandling),
		zap.Any("symlink_handling", opts.SymlinkHandling),
	)

	stats := &Stats{}
	startTime := time.Now()

	symlinkLock.Lock()
	visitedSymlinks = sync.Map{} // Clear symlink cache
	symlinkLock.Unlock()
	root = filepath.Clean(root)

	var progressTicker *time.Ticker
	if opts.Progress != nil {
		progressTicker = time.NewTicker(500 * time.Millisecond) // Update progress every 500ms
		go func() {
			for range progressTicker.C {
				if ctx.Err() != nil {
					return
				}
				stats.ElapsedTime = time.Since(startTime)
				stats.updateDerivedStats()
				opts.Progress(*stats)
			}
		}()
	}

	// Use an internal walk function to capture any errors.
	var internalWalkErr error
	var internalWalkErrLock sync.Mutex

	wrappedWalkFn := func(path string, info os.FileInfo, err error) error {
		// Capture any errors from the user's walkFn.
		var userWalkErr error

		if err != nil {
			// Handle the error based on the ErrorHandling option
			switch opts.ErrorHandling {
			case ErrorHandlingContinue, ErrorHandlingSkip:
				// For Continue and Skip, just mark the error.
				internalWalkErrLock.Lock()
				internalWalkErr = errors.Join(internalWalkErr, err)
				internalWalkErrLock.Unlock()
				return nil
			default: // ErrorHandlingStop
				return err
			}
		}
		// Check if info is nil to avoid nil pointer dereference
		if info == nil {
			return nil
		}

		// Resolve symlinks *before* other checks.
		resolvedPath, resolvedInfo, isSymlink, err := resolveSymlink(path, opts.SymlinkHandling) // Use resolved symlink
		if err != nil {
			return err
		}
		if isSymlink && resolvedInfo == nil {
			return nil // Symlink was ignored.
		}
		if isSymlink {
			path = resolvedPath
			info = resolvedInfo
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
		userWalkErr = walkFn(path, info, nil) // Call the users walkFn
		if userWalkErr != nil && opts.ErrorHandling == ErrorHandlingStop {
			return userWalkErr
		}
		return nil // always return nil so we dont stop
	}

	finalErr := WalkLimit(ctx, root, wrappedWalkFn, opts.NumWorkers)

	// Stop progress updates
	if progressTicker != nil {
		progressTicker.Stop()
		// Do a final progress update.
		stats.ElapsedTime = time.Since(startTime)
		stats.updateDerivedStats()
		opts.Progress(*stats)
	}
	// Check and combined captured errors
	internalWalkErrLock.Lock()
	err := errors.Join(finalErr, internalWalkErr)
	internalWalkErrLock.Unlock()
	return err
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

	// Glob pattern matching.  Use info.Name() (base name) for pattern matching, not the full path!
	if filter.Pattern != "" {
		matched, err := filepath.Match(filter.Pattern, info.Name())
		if err != nil || !matched {
			return false
		}
	}

	// Type filtering (extension check).
	if len(filter.IncludeTypes) > 0 {
		ext := filepath.Ext(info.Name())
		var found bool
		for _, typ := range filter.IncludeTypes {
			if ext == typ {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
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
