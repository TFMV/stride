/*
Package walk provides high-performance, concurrent filesystem traversal capabilities.

This package offers a powerful alternative to the standard library's filepath.Walk
with additional features like concurrency, filtering, progress monitoring, and
middleware support.

# Basic Usage

The simplest way to use the package is similar to filepath.Walk:

	err := walk.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	})

# Concurrent Processing

For better performance, use concurrent processing with a worker pool:

	ctx := context.Background()
	err := walk.WalkLimit(ctx, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	}, 4) // 4 concurrent workers

# Filtering

Apply filters to process only specific files:

	filter := walk.FilterOptions{
		MinSize:      1024,                // Skip files smaller than 1KB
		MaxSize:      1024 * 1024 * 10,    // Skip files larger than 10MB
		Pattern:      "*.log",             // Match files using glob pattern
		ExcludePattern: []string{"*.tmp"}, // Exclude files matching these patterns
		IncludeTypes: []string{".go", ".md"}, // Only process Go and Markdown files
	}

	err := walk.WalkLimitWithFilter(ctx, ".", func(path string, info os.FileInfo, err error) error {
		// Process files that pass the filter
		return nil
	}, 8, filter)

# Progress Monitoring

Monitor progress during traversal:

	progressFn := func(stats walk.Stats) {
		fmt.Printf("\rProcessed: %d files, %d dirs, %.2f MB, %.2f MB/s",
			stats.FilesProcessed,
			stats.DirsProcessed,
			float64(stats.BytesProcessed) / (1024 * 1024),
			stats.SpeedMBPerSec)
	}

	err := walk.WalkLimitWithProgress(ctx, ".", func(path string, info os.FileInfo, err error) error {
		// Process files
		return nil
	}, 8, progressFn)

# Enhanced API with Context and Middleware

Use the enhanced API with context-aware callbacks and middleware:

	// Create options
	opts := walk.NewWalkOptions()
	opts.WorkerCount = 8
	opts.Filter.Pattern = "*.log"
	opts.Filter.MinSize = 1024

	// Add middleware
	opts.Middleware = []walk.MiddlewareFunc{
		walk.LoggingMiddleware(logger),
		walk.TimingMiddleware(10 * time.Millisecond),
	}

	// Use context-aware callback
	err := walk.WalkWithOptions(".", func(ctx context.Context, path string, info os.FileInfo) error {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Process file
			return nil
		}
	}, opts)

# Advanced Statistics

Access real-time statistics during processing:

	err := walk.WalkWithAdvancedOptions(".", func(ctx context.Context, path string, info os.FileInfo, stats walk.Stats) error {
		// Access stats during processing
		if stats.FilesProcessed % 100 == 0 {
			fmt.Printf("\rProcessed %d files (%.2f MB)",
				stats.FilesProcessed,
				float64(stats.BytesProcessed)/(1024*1024))
		}
		return nil
	}, opts)

For more details and examples, see the package documentation and examples directory.
*/
package walk
