package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	stride "github.com/TFMV/stride/internal/walk"
	"go.uber.org/zap"
)

func main() {
	// Example 1: Basic Walk
	fmt.Println("--- Basic Walk ---")
	err := stride.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fmt.Println(path, info.Size())
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Example 2: Walk with Progress
	fmt.Println("\n--- Walk with Progress ---")
	err = stride.WalkLimitWithProgress(context.Background(), ".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fmt.Println("Processing:", path)
			}

			return nil
		},
		10, // Limit concurrency to 10
		func(stats stride.Stats) {
			fmt.Printf("Files: %d, Dirs: %d, Bytes: %d, Elapsed: %s, Speed: %.2f MB/s\n",
				stats.FilesProcessed, stats.DirsProcessed, stats.BytesProcessed,
				stats.ElapsedTime, stats.SpeedMBPerSec)
		},
	)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Example 3: Walk with Filtering
	fmt.Println("\n--- Walk with Filtering ---")
	filter := stride.FilterOptions{
		MinSize:    1024,                       // Minimum file size of 1KB
		Pattern:    "*.go",                     // Only Go files
		ExcludeDir: []string{"vendor", ".git"}, // Exclude vendor and .git directories
		// Permission filtering is also available:
		// MinPermissions: 0644,                // Minimum permissions (at least readable)
		// MaxPermissions: 0755,                // Maximum permissions (no more than rwxr-xr-x)
		// ExactPermissions: 0644,              // Exact permissions to match
		// UseExactPermissions: true,           // Use exact permission matching
	}
	err = stride.WalkLimitWithFilter(context.Background(), ".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fmt.Println("Filtered:", path, info.Size())
			return nil
		},
		5, // Concurrency limit
		filter,
	)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Example 4: Walk with all options
	fmt.Println("\n--- Walk with All Options ---")
	logger, _ := zap.NewProduction() // Create a logger instance
	options := stride.WalkOptions{
		ErrorHandling:   stride.ErrorHandlingContinue, // Continue on errors
		SymlinkHandling: stride.SymlinkFollow,         // Follow symlinks
		Filter: stride.FilterOptions{
			MaxSize:        1024 * 1024 * 10,           // Maximum file size of 10MB
			IncludeTypes:   []string{".txt", ".md"},    // include only txt and md
			ModifiedBefore: time.Now().Add(-time.Hour), // Modified within last hour
		},
		Progress: func(stats stride.Stats) {
			fmt.Printf("Files: %d, Dirs: %d, Bytes: %d\n",
				stats.FilesProcessed, stats.DirsProcessed, stats.BytesProcessed)
		},
		Logger:     logger,
		LogLevel:   stride.LogLevelDebug, // Set log level
		BufferSize: 20,                   // Set buffer size
		NumWorkers: runtime.NumCPU() * 2, // Use twice the number of CPUs for workers
	}

	err = stride.WalkLimitWithOptions(
		context.Background(),
		".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fmt.Println("Filtered:", path, info.Size())
			}
			return nil
		},
		options,
	)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
