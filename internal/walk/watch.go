// Package walk provides functions for walking a directory tree and watching for filesystem changes.
//
// This package is used to watch for changes in a directory tree and execute a command or format the output.
//
// The package uses the fsnotify package to watch for changes in a directory tree.
//
// The package uses the filepath package to walk the directory tree.
package stride

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchEvent represents a filesystem event type
type WatchEvent string

// Watch event types
const (
	EventCreate WatchEvent = "create"
	EventModify WatchEvent = "modify"
	EventDelete WatchEvent = "delete"
	EventRename WatchEvent = "rename"
	EventChmod  WatchEvent = "chmod"
)

// WatchOptions defines options for watching filesystem changes
type WatchOptions struct {
	// Context for cancellation
	Context context.Context

	// Events to watch for (create, modify, delete, rename, chmod)
	// If empty, all events are watched
	Events []WatchEvent

	// Whether to watch subdirectories recursively
	Recursive bool

	// Pattern to match files (e.g., "*.go")
	Pattern string

	// Pattern to ignore files
	IgnorePattern string

	// Whether to include hidden files and directories
	IncludeHidden bool

	// Timeout duration (0 means no timeout)
	Timeout time.Duration
}

// WatchMessage contains information about a filesystem event
type WatchMessage struct {
	Path     string            // Full path to the file
	Name     string            // Base name of the file
	Dir      string            // Directory containing the file
	Size     int64             // Size in bytes (may be 0 for deleted files)
	Time     time.Time         // Modification time
	IsDir    bool              // Whether it's a directory
	Event    WatchEvent        // Event type (create, modify, delete, etc.)
	Metadata map[string]string // Additional metadata
}

// WatchResult represents a watch event result
type WatchResult struct {
	Message WatchMessage
	Error   error
}

// WatchHandler is a function that processes watch events
type WatchHandler func(ctx context.Context, result WatchResult) error

// defaultWatchHandler returns a default handler that prints events
func defaultWatchHandler() WatchHandler {
	return func(ctx context.Context, result WatchResult) error {
		if result.Error != nil {
			return result.Error
		}
		fmt.Printf("%s: %s\n", strings.ToUpper(string(result.Message.Event)), result.Message.Path)
		return nil
	}
}

// Watch monitors a directory for filesystem changes
func Watch(ctx context.Context, root string, opts WatchOptions, handler WatchHandler) error {
	if handler == nil {
		handler = defaultWatchHandler()
	}

	// Create a context if not provided
	if ctx == nil {
		ctx = context.Background()
	}

	// Create a context with timeout if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating watcher: %w", err)
	}
	defer watcher.Close()

	// Add the root directory to the watcher
	if err := watcher.Add(root); err != nil {
		return fmt.Errorf("error watching directory %s: %w", root, err)
	}

	// If recursive, add all subdirectories
	if opts.Recursive {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// Skip hidden directories if not included
				if !opts.IncludeHidden && isHidden(path) {
					return filepath.SkipDir
				}
				if err := watcher.Add(path); err != nil {
					// Log the error but continue
					fmt.Fprintf(os.Stderr, "Error watching directory %s: %v\n", path, err)
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error walking directory tree: %w", err)
		}
	}

	// Create a map of events to watch for
	eventMap := make(map[fsnotify.Op]bool)
	if len(opts.Events) > 0 {
		for _, e := range opts.Events {
			switch e {
			case EventCreate:
				eventMap[fsnotify.Create] = true
			case EventModify:
				eventMap[fsnotify.Write] = true
			case EventDelete:
				eventMap[fsnotify.Remove] = true
			case EventRename:
				eventMap[fsnotify.Rename] = true
			case EventChmod:
				eventMap[fsnotify.Chmod] = true
			}
		}
	} else {
		// Default to all events
		eventMap[fsnotify.Create] = true
		eventMap[fsnotify.Write] = true
		eventMap[fsnotify.Remove] = true
		eventMap[fsnotify.Rename] = true
		eventMap[fsnotify.Chmod] = true
	}

	// Create a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(1)

	// Start watching for events
	go func() {
		defer wg.Done()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Check if we should process this event
				var eventType WatchEvent
				shouldProcess := false

				if event.Has(fsnotify.Create) && eventMap[fsnotify.Create] {
					shouldProcess = true
					eventType = EventCreate
				} else if event.Has(fsnotify.Write) && eventMap[fsnotify.Write] {
					shouldProcess = true
					eventType = EventModify
				} else if event.Has(fsnotify.Remove) && eventMap[fsnotify.Remove] {
					shouldProcess = true
					eventType = EventDelete
				} else if event.Has(fsnotify.Rename) && eventMap[fsnotify.Rename] {
					shouldProcess = true
					eventType = EventRename
				} else if event.Has(fsnotify.Chmod) && eventMap[fsnotify.Chmod] {
					shouldProcess = true
					eventType = EventChmod
				}

				if shouldProcess {
					// Get file info
					var fileInfo os.FileInfo
					var err error
					if !event.Has(fsnotify.Remove) {
						fileInfo, err = os.Stat(event.Name)
						if err != nil {
							// Report the error but continue
							handler(ctx, WatchResult{
								Error: fmt.Errorf("error getting file info for %s: %w", event.Name, err),
							})
							continue
						}

						// If it's a directory and we're in recursive mode, add it to the watcher
						if opts.Recursive && fileInfo.IsDir() && event.Has(fsnotify.Create) {
							if err := watcher.Add(event.Name); err != nil {
								// Report the error but continue
								handler(ctx, WatchResult{
									Error: fmt.Errorf("error watching new directory %s: %w", event.Name, err),
								})
							}
						}
					}

					// Check if the file matches the pattern
					if opts.Pattern != "" {
						matched, err := filepath.Match(opts.Pattern, filepath.Base(event.Name))
						if err != nil {
							// Report the error but continue
							handler(ctx, WatchResult{
								Error: fmt.Errorf("error matching pattern: %w", err),
							})
							continue
						}
						if !matched {
							continue
						}
					}

					// Check if the file should be ignored
					if opts.IgnorePattern != "" {
						matched, err := filepath.Match(opts.IgnorePattern, filepath.Base(event.Name))
						if err != nil {
							// Report the error but continue
							handler(ctx, WatchResult{
								Error: fmt.Errorf("error matching ignore pattern: %w", err),
							})
							continue
						}
						if matched {
							continue
						}
					}

					// Skip hidden files if not included
					if !opts.IncludeHidden && isHidden(event.Name) {
						continue
					}

					// Create a message for the event
					msg := WatchMessage{
						Path:     event.Name,
						Name:     filepath.Base(event.Name),
						Dir:      filepath.Dir(event.Name),
						Time:     time.Now(),
						Event:    eventType,
						Metadata: make(map[string]string),
					}

					if fileInfo != nil {
						msg.Size = fileInfo.Size()
						msg.IsDir = fileInfo.IsDir()
						msg.Time = fileInfo.ModTime()
					}

					// Process the event
					if err := handler(ctx, WatchResult{Message: msg}); err != nil {
						// If the handler returns an error, report it
						handler(ctx, WatchResult{
							Error: fmt.Errorf("error handling event: %w", err),
						})
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// Report watcher errors
				handler(ctx, WatchResult{
					Error: fmt.Errorf("watcher error: %w", err),
				})

			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for the context to be done
	<-ctx.Done()

	// Wait for all goroutines to finish
	wg.Wait()

	return nil
}

// WatchWithExec watches for filesystem changes and executes a command for each event
func WatchWithExec(ctx context.Context, root string, opts WatchOptions, cmdTemplate string) error {
	return Watch(ctx, root, opts, func(ctx context.Context, result WatchResult) error {
		if result.Error != nil {
			return result.Error
		}

		// Replace {event} placeholder with the event type
		cmd := strings.ReplaceAll(cmdTemplate, "{event}", string(result.Message.Event))

		// Format the command using the message
		formattedCmd := formatCommand(cmd, FindMessage{
			Path:     result.Message.Path,
			Name:     result.Message.Name,
			Dir:      result.Message.Dir,
			Size:     result.Message.Size,
			Time:     result.Message.Time,
			IsDir:    result.Message.IsDir,
			Metadata: result.Message.Metadata,
		})

		// Execute the command
		return executeCommand(ctx, formattedCmd, FindMessage{
			Path:     result.Message.Path,
			Name:     result.Message.Name,
			Dir:      result.Message.Dir,
			Size:     result.Message.Size,
			Time:     result.Message.Time,
			IsDir:    result.Message.IsDir,
			Metadata: result.Message.Metadata,
		})
	})
}

// WatchWithFormat watches for filesystem changes and formats output for each event
func WatchWithFormat(ctx context.Context, root string, opts WatchOptions, formatTemplate string) error {
	return Watch(ctx, root, opts, func(ctx context.Context, result WatchResult) error {
		if result.Error != nil {
			return result.Error
		}

		// Replace {event} placeholder with the event type
		format := strings.ReplaceAll(formatTemplate, "{event}", string(result.Message.Event))

		// Format the output using the message
		output := formatCommand(format, FindMessage{
			Path:     result.Message.Path,
			Name:     result.Message.Name,
			Dir:      result.Message.Dir,
			Size:     result.Message.Size,
			Time:     result.Message.Time,
			IsDir:    result.Message.IsDir,
			Metadata: result.Message.Metadata,
		})

		fmt.Println(output)
		return nil
	})
}
