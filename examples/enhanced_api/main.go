// Package main demonstrates the enhanced API of the stride package.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	stride "github.com/TFMV/stride/walk"
	"go.uber.org/zap"
)

// Simple logging middleware that logs files being processed
func LoggingMiddleware(logger *zap.Logger) stride.MiddlewareFunc {
	return func(next stride.WalkFunc) stride.WalkFunc {
		return func(ctx context.Context, path string, info os.FileInfo) error {
			// Skip logging directories to reduce noise
			if !info.IsDir() {
				logger.Debug("Processing file",
					zap.String("path", path),
					zap.Int64("size", info.Size()),
					zap.Time("modified", info.ModTime()),
				)
			}
			// Call the next handler in the chain
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

// Timing middleware that measures how long processing takes for each file
func TimingMiddleware() stride.MiddlewareFunc {
	return func(next stride.WalkFunc) stride.WalkFunc {
		return func(ctx context.Context, path string, info os.FileInfo) error {
			// Skip timing directories
			if info.IsDir() {
				return next(ctx, path, info)
			}

			start := time.Now()
			// Call the next handler in the chain
			err := next(ctx, path, info)
			duration := time.Since(start)

			// Only log files that take longer than 10ms to process
			if duration > 10*time.Millisecond {
				fmt.Printf("Processing %s took %v\n", path, duration)
			}

			return err
		}
	}
}

func main() {
	// Set up cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		<-signalCh
		fmt.Println("\nReceived interrupt, cancelling operations...")
		cancel()
	}()

	// Create logger
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, _ := config.Build()
	defer logger.Sync()

	// Parse command line arguments
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	// Stats for summary
	var totalFiles, totalDirs int64
	var totalBytes int64

	// Create options for the traversal
	opts := stride.WalkOptions{
		Context:     ctx,
		WorkerCount: 4,
		Filter: stride.FilterOptions{
			MinSize:   0,
			MaxSize:   1024 * 1024 * 10, // Only process files up to 10MB
			MinDepth:  1,                // Skip the root directory itself
			FileTypes: []string{"file"}, // Only process regular files
		},
		ErrorHandlingMode: stride.ContinueOnError,
		ProgressCallback: func(stats stride.Stats) {
			fmt.Printf("\rProcessed: %d files, %d dirs, %.2f MB at %.2f MB/s",
				stats.FilesProcessed,
				stats.DirsProcessed,
				float64(stats.BytesProcessed)/(1024*1024),
				stats.SpeedMBPerSec,
			)
		},
		Logger: logger,
		Middleware: []stride.MiddlewareFunc{
			LoggingMiddleware(logger),
			TimingMiddleware(),
		},
	}

	// Our processing function
	walkFn := func(ctx context.Context, path string, info os.FileInfo) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		// This is where your actual file processing would go
		// For this example, we just count files and sizes

		return nil
	}

	fmt.Printf("Starting traversal of %s with enhanced API...\n", rootDir)
	startTime := time.Now()

	// Start the traversal
	err := stride.WalkWithOptions(rootDir, walkFn, opts)

	// Print final newline after progress updates
	fmt.Println()

	if err != nil {
		if err == context.Canceled {
			fmt.Println("Traversal was cancelled by user")
		} else {
			fmt.Printf("Error during traversal: %v\n", err)
		}
	}

	// Get absolute path for nicer output
	absPath, _ := filepath.Abs(rootDir)

	// Print summary
	duration := time.Since(startTime)
	fmt.Printf("\nTraversal Summary:\n")
	fmt.Printf("Root: %s\n", absPath)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Files processed: %d\n", totalFiles)
	fmt.Printf("Directories processed: %d\n", totalDirs)
	fmt.Printf("Total size: %.2f MB\n", float64(totalBytes)/(1024*1024))

	if duration > 0 {
		mbPerSec := float64(totalBytes) / (1024 * 1024 * duration.Seconds())
		fmt.Printf("Processing speed: %.2f MB/s\n", mbPerSec)
	}
}
