package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	stride "github.com/TFMV/stride/walk"
)

func main() {
	fmt.Println("=== Watch Examples ===")

	// Get directory to watch
	var watchDir string
	if len(os.Args) > 1 {
		watchDir = os.Args[1]
	} else {
		var err error
		watchDir, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Create a context that can be cancelled with Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		cancel()
	}()

	// Example 1: Basic watch
	fmt.Println("\n--- Example 1: Basic watch ---")
	basicWatch(ctx, watchDir)

	// Example 2: Watch with event filtering
	fmt.Println("\n--- Example 2: Watch with event filtering ---")
	eventFilteringWatch(ctx, watchDir)

	// Example 3: Watch with pattern matching
	fmt.Println("\n--- Example 3: Watch with pattern matching ---")
	patternMatchingWatch(ctx, watchDir)

	// Example 4: Watch with command execution
	fmt.Println("\n--- Example 4: Watch with command execution ---")
	commandExecutionWatch(ctx, watchDir)

	// Example 5: Watch with custom formatting
	fmt.Println("\n--- Example 5: Watch with custom formatting ---")
	customFormattingWatch(ctx, watchDir)
}

// Basic watch example
func basicWatch(ctx context.Context, watchDir string) {
	// Create a context with timeout to limit the example duration
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create basic watch options
	opts := stride.WatchOptions{
		Recursive: true,
	}

	fmt.Printf("Watching %s for all events (timeout: 10s)...\n", watchDir)
	fmt.Println("Try creating, modifying, or deleting files in the watched directory.")

	// Start watching with default handler
	err := stride.Watch(ctx, watchDir, opts, nil)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		fmt.Printf("Error watching directory: %v\n", err)
	}
}

// Watch with event filtering example
func eventFilteringWatch(ctx context.Context, watchDir string) {
	// Create a context with timeout to limit the example duration
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create watch options with event filtering
	opts := stride.WatchOptions{
		Recursive: true,
		Events:    []stride.WatchEvent{stride.EventCreate, stride.EventModify},
	}

	fmt.Printf("Watching %s for create and modify events only (timeout: 10s)...\n", watchDir)
	fmt.Println("Try creating, modifying, or deleting files in the watched directory.")

	// Start watching with default handler
	err := stride.Watch(ctx, watchDir, opts, nil)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		fmt.Printf("Error watching directory: %v\n", err)
	}
}

// Watch with pattern matching example
func patternMatchingWatch(ctx context.Context, watchDir string) {
	// Create a context with timeout to limit the example duration
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create watch options with pattern matching
	opts := stride.WatchOptions{
		Recursive:     true,
		Pattern:       "*.txt",
		IgnorePattern: "temp*",
	}

	fmt.Printf("Watching %s for events on *.txt files, ignoring temp* files (timeout: 10s)...\n", watchDir)
	fmt.Println("Try creating, modifying, or deleting .txt files in the watched directory.")

	// Start watching with default handler
	err := stride.Watch(ctx, watchDir, opts, nil)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		fmt.Printf("Error watching directory: %v\n", err)
	}
}

// Watch with command execution example
func commandExecutionWatch(ctx context.Context, watchDir string) {
	// Create a context with timeout to limit the example duration
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create watch options
	opts := stride.WatchOptions{
		Recursive: true,
	}

	// Command template with placeholders
	cmdTemplate := "echo 'Event: {event}, File: {base}, Size: {size} bytes'"

	fmt.Printf("Watching %s and executing command for each event (timeout: 10s)...\n", watchDir)
	fmt.Println("Try creating, modifying, or deleting files in the watched directory.")

	// Start watching with command execution
	err := stride.WatchWithExec(ctx, watchDir, opts, cmdTemplate)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		fmt.Printf("Error watching directory: %v\n", err)
	}
}

// Watch with custom formatting example
func customFormattingWatch(ctx context.Context, watchDir string) {
	// Create a context with timeout to limit the example duration
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create watch options
	opts := stride.WatchOptions{
		Recursive: true,
	}

	// Format template with placeholders
	formatTemplate := "[{time}] {event}: {base} in {dir} ({size} bytes)"

	fmt.Printf("Watching %s with custom output format (timeout: 10s)...\n", watchDir)
	fmt.Println("Try creating, modifying, or deleting files in the watched directory.")

	// Start watching with custom formatting
	err := stride.WatchWithFormat(ctx, watchDir, opts, formatTemplate)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		fmt.Printf("Error watching directory: %v\n", err)
	}
}
