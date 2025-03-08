// Package walk provides high-performance filesystem traversal with advanced filtering
// and monitoring capabilities.
//
// This package contains the implementation of the `stride` command, which is a
// high-performance file walking utility that extends the standard `filepath.Walk`
// functionality with concurrency, filtering, and monitoring capabilities.

// Watch Functionality
//
// The watch package provides functionality for monitoring filesystem changes:
//
//	// Basic usage
//	opts := walk.WatchOptions{
//		Recursive: true,
//	}
//	err := walk.Watch(context.Background(), "/path/to/watch", opts, nil)
//
//	// With event filtering
//	opts := walk.WatchOptions{
//		Events: []walk.WatchEvent{walk.EventCreate, walk.EventModify},
//	}
//	err := walk.Watch(context.Background(), "/path/to/watch", opts, nil)
//
//	// With custom handler
//	err := walk.Watch(context.Background(), "/path/to/watch", opts, func(ctx context.Context, result walk.WatchResult) error {
//		if result.Error != nil {
//			return result.Error
//		}
//		fmt.Printf("Event: %s, File: %s\n", result.Message.Event, result.Message.Path)
//		return nil
//	})
//
//	// Execute command for each event
//	err := walk.WatchWithExec(context.Background(), "/path/to/watch", opts, "echo Event: {event}, File: {}")
//
//	// Format output for each event
//	err := walk.WatchWithFormat(context.Background(), "/path/to/watch", opts, "{event}: {base} at {time}")

package walk
