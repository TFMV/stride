// Package walk provides high-performance, concurrent filesystem traversal capabilities.
//
// This package offers a powerful alternative to the standard library's filepath.Walk
// with additional features like concurrency, filtering, progress monitoring, and
// middleware support.
package walk

import (
	"context"
	"os"
	"time"

	internal "github.com/TFMV/stride/internal/walk"
	"go.uber.org/zap"
)

// Re-export all the types and constants from the internal package
type (
	// Stats holds traversal statistics that are updated atomically during the walk.
	Stats = internal.Stats

	// FilterOptions defines criteria for including/excluding files and directories.
	FilterOptions = internal.FilterOptions

	// WalkOptions provides comprehensive configuration for the walk operation.
	WalkOptions = internal.WalkOptions

	// WalkFunc defines the signature for file processing callbacks.
	WalkFunc = internal.WalkFunc

	// AdvancedWalkFunc includes statistics for each callback.
	AdvancedWalkFunc = internal.AdvancedWalkFunc

	// ErrorHandlingMode defines how errors are handled during traversal.
	ErrorHandlingMode = internal.ErrorHandlingMode

	// MemoryLimitOptions sets memory usage boundaries for the traversal.
	MemoryLimitOptions = internal.MemoryLimitOptions

	// MiddlewareFunc defines a middleware function for extensibility.
	MiddlewareFunc = internal.MiddlewareFunc

	// ErrorHandling defines how errors are handled during traversal.
	ErrorHandling = internal.ErrorHandling

	// SymlinkHandling defines how symbolic links are processed.
	SymlinkHandling = internal.SymlinkHandling

	// LogLevel defines the verbosity of logging.
	LogLevel = internal.LogLevel

	// MemoryLimit sets memory usage boundaries for the traversal.
	MemoryLimit = internal.MemoryLimit

	// ProgressFn is called periodically with traversal statistics.
	ProgressFn = internal.ProgressFn

	// Re-export watch types and functions
	WatchEvent   = internal.WatchEvent
	WatchOptions = internal.WatchOptions
	WatchMessage = internal.WatchMessage
	WatchResult  = internal.WatchResult
	WatchHandler = internal.WatchHandler
)

// Re-export all the constants
const (
	// Error handling modes
	ContinueOnError = internal.ContinueOnError
	StopOnError     = internal.StopOnError
	SkipOnError     = internal.SkipOnError

	// Symlink handling modes
	SymlinkFollow = internal.SymlinkFollow
	SymlinkIgnore = internal.SymlinkIgnore
	SymlinkReport = internal.SymlinkReport

	// Log levels
	LogLevelError = internal.LogLevelError
	LogLevelWarn  = internal.LogLevelWarn
	LogLevelInfo  = internal.LogLevelInfo
	LogLevelDebug = internal.LogLevelDebug

	// Error handling modes (string-based)
	ErrorHandlingContinue = internal.ErrorHandlingContinue
	ErrorHandlingStop     = internal.ErrorHandlingStop
	ErrorHandlingSkip     = internal.ErrorHandlingSkip

	// Watch event constants
	EventCreate = internal.EventCreate
	EventModify = internal.EventModify
	EventDelete = internal.EventDelete
	EventRename = internal.EventRename
	EventChmod  = internal.EventChmod
)

// Walk traverses the file tree rooted at root, calling walkFn for each file or directory.
// It's similar to filepath.Walk but with better error handling.
func Walk(root string, walkFn func(path string, info os.FileInfo, err error) error) error {
	return internal.Walk(root, walkFn)
}

// WalkLimit traverses the file tree with a limited number of concurrent workers.
func WalkLimit(ctx context.Context, root string, walkFn func(path string, info os.FileInfo, err error) error, limit int) error {
	return internal.WalkLimit(ctx, root, walkFn, limit)
}

// WalkLimitWithProgress traverses the file tree with progress reporting.
func WalkLimitWithProgress(ctx context.Context, root string, walkFn func(path string, info os.FileInfo, err error) error, limit int, progressFn ProgressFn) error {
	return internal.WalkLimitWithProgress(ctx, root, walkFn, limit, progressFn)
}

// WalkLimitWithFilter traverses the file tree with filtering options.
func WalkLimitWithFilter(ctx context.Context, root string, walkFn func(path string, info os.FileInfo, err error) error, limit int, filter FilterOptions) error {
	return internal.WalkLimitWithFilter(ctx, root, walkFn, limit, filter)
}

// WalkLimitWithOptions traverses the file tree with comprehensive options.
func WalkLimitWithOptions(ctx context.Context, root string, walkFn func(path string, info os.FileInfo, err error) error, opts WalkOptions) error {
	return internal.WalkLimitWithOptions(ctx, root, walkFn, opts)
}

// WalkWithOptions traverses the file tree with the enhanced context-aware API.
func WalkWithOptions(root string, walkFn WalkFunc, options WalkOptions) error {
	return internal.WalkWithOptions(root, walkFn, options)
}

// WalkWithAdvancedOptions traverses the file tree with statistics access.
func WalkWithAdvancedOptions(root string, walkFn AdvancedWalkFunc, options WalkOptions) error {
	return internal.WalkWithAdvancedOptions(root, walkFn, options)
}

// NewFilterOptions creates a new FilterOptions with default values.
func NewFilterOptions() FilterOptions {
	return FilterOptions{
		MinSize:          -1,
		MaxSize:          -1,
		MinPermissions:   0,
		MaxPermissions:   0,
		ExactPermissions: 0,
		MinDepth:         0,
		MaxDepth:         -1, // No limit
	}
}

// NewWalkOptions creates a new WalkOptions with default values.
func NewWalkOptions() WalkOptions {
	return WalkOptions{
		Context:       context.Background(),
		ErrorHandling: ErrorHandlingContinue,
		Filter:        NewFilterOptions(),
		LogLevel:      LogLevelInfo,
		BufferSize:    100,
		WorkerCount:   4,
	}
}

// LoggingMiddleware creates a middleware that logs file processing.
func LoggingMiddleware(logger *zap.Logger) MiddlewareFunc {
	return func(next WalkFunc) WalkFunc {
		return func(ctx context.Context, path string, info os.FileInfo) error {
			if !info.IsDir() {
				logger.Debug("Processing file",
					zap.String("path", path),
					zap.Int64("size", info.Size()),
					zap.Time("modified", info.ModTime()),
				)
			}
			err := next(ctx, path, info)
			if err != nil {
				logger.Error("Error processing file",
					zap.String("path", path),
					zap.Error(err),
				)
			}
			return err
		}
	}
}

// TimingMiddleware creates a middleware that measures processing time.
func TimingMiddleware(threshold time.Duration) MiddlewareFunc {
	return func(next WalkFunc) WalkFunc {
		return func(ctx context.Context, path string, info os.FileInfo) error {
			start := time.Now()
			err := next(ctx, path, info)
			duration := time.Since(start)

			if duration > threshold {
				// You could log this or handle it in some other way
			}

			return err
		}
	}
}

// Watch monitors a directory for filesystem changes
func Watch(ctx context.Context, root string, opts WatchOptions, handler WatchHandler) error {
	return internal.Watch(ctx, root, opts, handler)
}

// WatchWithExec watches for filesystem changes and executes a command for each event
func WatchWithExec(ctx context.Context, root string, opts WatchOptions, cmdTemplate string) error {
	return internal.WatchWithExec(ctx, root, opts, cmdTemplate)
}

// WatchWithFormat watches for filesystem changes and formats output for each event
func WatchWithFormat(ctx context.Context, root string, opts WatchOptions, formatTemplate string) error {
	return internal.WatchWithFormat(ctx, root, opts, formatTemplate)
}
