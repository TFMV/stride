# Stride

[![Go Reference](https://pkg.go.dev/badge/github.com/TFMV/stride.svg)](https://pkg.go.dev/github.com/TFMV/stride)
[![Go Report Card](https://goreportcard.com/badge/github.com/TFMV/stride)](https://goreportcard.com/report/github.com/TFMV/stride)
[![License](https://img.shields.io/github/license/TFMV/stride)](LICENSE)

Stride is a high-performance, concurrent filesystem traversal library for Go that builds upon the standard `filepath.Walk` functionality while adding concurrency, filtering, and monitoring capabilities.

## Features

- **Concurrent Processing**: Process files in parallel with configurable worker pools
- **Flexible Filtering**: Filter files by size, extension, modification time, and permissions
- **Progress Monitoring**: Real-time statistics during traversal
- **Symlink Handling**: Configurable symlink following behavior
- **Error Handling**: Multiple strategies for handling errors during traversal
- **Memory Limits**: Set soft and hard memory limits for processing
- **Context Support**: Cancel operations with context
- **Structured Logging**: Integrated with zap logger

## Installation

```bash
go get github.com/TFMV/stride
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/TFMV/stride"
)

func main() {
    // Basic usage - similar to filepath.Walk
    err := stride.Walk(".", func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        fmt.Println(path)
        return nil
    })
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    }

    // Concurrent processing with 4 workers
    ctx := context.Background()
    err = stride.WalkLimit(ctx, ".", func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        fmt.Println(path)
        return nil
    }, 4)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    }
}
```

## Advanced Usage

### With Filtering

```go
filter := stride.FilterOptions{
    // Basic filtering
    MinSize:      1024,                // Skip files smaller than 1KB
    MaxSize:      1024 * 1024 * 10,    // Skip files larger than 10MB
    Pattern:      "*.log",             // Match files using glob pattern
    ExcludePattern: []string{"*.tmp"}, // Exclude files matching these patterns
    IncludeTypes: []string{".go", ".md"}, // Only process Go and Markdown files
    ExcludeDir:   []string{"vendor", "node_modules"}, // Skip these directories
    
    // Time-based filtering
    ModifiedAfter:  time.Now().Add(-24 * time.Hour), // Only files modified in the last 24 hours
    ModifiedBefore: time.Now().Add(-1 * time.Hour),  // Only files modified before the last hour
    AccessedAfter:  time.Now().Add(-7 * 24 * time.Hour), // Files accessed in the last week
    CreatedAfter:   time.Now().Add(-30 * 24 * time.Hour), // Files created in the last month
    
    // Permission filtering
    MinPermissions: 0644,              // Files must be at least readable by owner and group
    MaxPermissions: 0755,              // Files must not have more permissions than rwxr-xr-x
    ExactPermissions: 0644,            // Match files with exactly these permissions
    UseExactPermissions: true,         // Enable exact permission matching
    
    // Owner/Group filtering
    OwnerUID:  1000,                   // Filter by owner UID
    OwnerGID:  1000,                   // Filter by group GID
    OwnerName: "username",             // Filter by owner username
    GroupName: "groupname",            // Filter by group name
    
    // File type filtering
    FileTypes: []string{"file", "dir", "symlink"}, // Only include these file types
    
    // Depth control
    MinDepth: 1,                       // Skip the root directory
    MaxDepth: 3,                       // Don't go deeper than 3 levels
    
    // Empty files/directories
    IncludeEmptyFiles: true,           // Only include empty files
    IncludeEmptyDirs: true,            // Only include empty directories
}

err := stride.WalkLimitWithFilter(ctx, ".", func(path string, info os.FileInfo, err error) error {
    // Process files that pass the filter
    return nil
}, 8, filter)
```

### With Progress Reporting

```go
progressFn := func(stats stride.Stats) {
    fmt.Printf("\rProcessed: %d files, %d dirs, %.2f MB, %.2f MB/s",
        stats.FilesProcessed,
        stats.DirsProcessed,
        float64(stats.BytesProcessed) / (1024 * 1024),
        stats.SpeedMBPerSec)
}

err := stride.WalkLimitWithProgress(ctx, ".", func(path string, info os.FileInfo, err error) error {
    // Process files
    return nil
}, 8, progressFn)
```

### With Full Options

```go
opts := stride.WalkOptions{
    ErrorHandling:   stride.ErrorHandlingContinue,
    SymlinkHandling: stride.SymlinkFollow,
    LogLevel:        stride.LogLevelInfo,
    BufferSize:      8,
    Filter: stride.FilterOptions{
        MinSize:      1024,
        MaxSize:      1024 * 1024 * 10,
        IncludeTypes: []string{".go"},
    },
    Progress: func(stats stride.Stats) {
        // Report progress
    },
    MemoryLimit: stride.MemoryLimit{
        SoftLimit: 1024 * 1024 * 100, // 100MB
        HardLimit: 1024 * 1024 * 200, // 200MB
    },
}

err := stride.WalkLimitWithOptions(ctx, ".", func(path string, info os.FileInfo, err error) error {
    // Process files
    return nil
}, opts)
```

## Enhanced API

Stride offers an enhanced API with context-aware callbacks, comprehensive configuration options, and middleware support.

### Context-Aware Processing Function

The new API provides a context-aware processing function signature:

```go
// WalkFunc defines the signature for file processing callbacks.
type WalkFunc func(ctx context.Context, path string, info os.FileInfo) error
```

This allows for cancellation, timeouts, and passing values through context.

### Extended Function with Statistics

For advanced use cases, an extended callback provides access to traversal statistics:

```go
// AdvancedWalkFunc includes statistics for each callback.
type AdvancedWalkFunc func(ctx context.Context, path string, info os.FileInfo, stats Stats) error
```

### Comprehensive Options Structure

All configuration options are encapsulated in a cohesive structure:

```go
type WalkOptions struct {
    Context           context.Context
    Filter            FilterOptions
    Logger            *zap.Logger
    WorkerCount       int
    ErrorHandlingMode ErrorHandlingMode
    ProgressCallback  func(stats Stats)
    MemoryLimits      MemoryLimitOptions
    DryRun            bool
    Middleware        []MiddlewareFunc
}
```

### Error Handling Mode

Error handling is now expressed as an enum with clear semantic values:

```go
type ErrorHandlingMode string

const (
    ContinueOnError ErrorHandlingMode = "continue"
    StopOnError     ErrorHandlingMode = "stop"
    SkipOnError     ErrorHandlingMode = "skip"
)
```

### Middleware Support

The new API supports middleware functions for adding cross-cutting concerns:

```go
type MiddlewareFunc func(next WalkFunc) WalkFunc
```

### Middleware Extensibility

The enhanced API supports middleware patterns for cross-cutting concerns like logging, metrics, and timing:

```go
// Create a logging middleware
func LoggingMiddleware(logger *zap.Logger) stride.MiddlewareFunc {
    return func(next stride.WalkFunc) stride.WalkFunc {
        return func(ctx context.Context, path string, info os.FileInfo) error {
            logger.Debug("Processing file", zap.String("path", path))
            err := next(ctx, path, info)
            if err != nil {
                logger.Error("Error processing file", zap.String("path", path), zap.Error(err))
            }
            return err
        }
    }
}

// Create a timing middleware
func TimingMiddleware(metrics MetricsService) stride.MiddlewareFunc {
    return func(next stride.WalkFunc) stride.WalkFunc {
        return func(ctx context.Context, path string, info os.FileInfo) error {
            start := time.Now()
            err := next(ctx, path, info)
            duration := time.Since(start)
            metrics.RecordDuration(path, duration)
            return err
        }
    }
}

// Apply middlewares in sequence
opts := stride.WalkOptions{
    Context: ctx,
    Middleware: []stride.MiddlewareFunc{
        LoggingMiddleware(logger),     // Will execute first (outer wrapper)
        TimingMiddleware(metrics),     // Will execute second (inner wrapper)
        RateLimitingMiddleware(limiter), // Will execute third (innermost)
    },
}
```

Middleware functions are applied in order, with each function wrapping the next one. This creates a pipeline where:

1. The first middleware is the outermost wrapper and runs first before calling the next function
2. The last middleware is the innermost wrapper and runs last, right before the actual file processing

This pattern allows for clean separation of concerns, making your file processing code focus on its primary responsibility while cross-cutting concerns are handled separately.

### Examples

#### Simple Example

```go
opts := stride.WalkOptions{
    Context:     context.Background(),
    WorkerCount: 4,
}

err := stride.WalkWithOptions(".", func(ctx context.Context, path string, info os.FileInfo) error {
    fmt.Println(path)
    return nil
}, opts)
```

#### Advanced Example with Filtering & Progress

```go
opts := stride.WalkOptions{
    Context:     context.Background(),
    WorkerCount: 8,
    Filter: stride.FilterOptions{
        Pattern: "*.log",
        MinSize: 1024,
    },
    ProgressCallback: func(stats stride.Stats) {
        fmt.Printf("\rProcessed: %d files, %.2f MB at %.2f MB/s",
            stats.FilesProcessed,
            float64(stats.BytesProcessed)/(1024*1024),
            stats.SpeedMBPerSec,
        )
    },
}

err := stride.WalkWithOptions(".", func(ctx context.Context, path string, info os.FileInfo) error {
    // Custom processing logic
    processFile(path)
    return nil
}, opts)
```

#### Using Middleware

```go
// Define a logging middleware
func LoggingMiddleware(logger *zap.Logger) stride.MiddlewareFunc {
    return func(next stride.WalkFunc) stride.WalkFunc {
        return func(ctx context.Context, path string, info os.FileInfo) error {
            logger.Debug("Processing file", zap.String("path", path))
            err := next(ctx, path, info)
            if err != nil {
                logger.Error("Error processing file", zap.String("path", path), zap.Error(err))
            }
            return err
        }
    }
}

// Use the middleware
opts := stride.WalkOptions{
    Context: context.Background(),
    Middleware: []stride.MiddlewareFunc{
        LoggingMiddleware(zapLogger),
    },
}

err := stride.WalkWithOptions(".", func(ctx context.Context, path string, info os.FileInfo) error {
    return processFile(path)
}, opts)
```

### Symlink Handling

Stride provides robust symlink handling capabilities with three modes:

- **SymlinkIgnore**: Ignores symbolic links during traversal (default)
- **SymlinkFollow**: Follows symbolic links, including during directory traversal
- **SymlinkReport**: Reports symbolic links but doesn't follow them

When following symlinks, Stride includes cycle detection to prevent infinite loops:

```go
opts := stride.WalkOptions{
    SymlinkHandling: stride.SymlinkFollow, // Follow symbolic links
    // Other options...
}

err := stride.WalkLimitWithOptions(ctx, root, walkFn, opts)
```

From the command line:

```bash
stride --follow-symlinks /path/to/directory
```

## Command Line Tool

Stride includes a powerful CLI tool for quick and efficient filesystem traversal.

```bash
# Install the command-line tool
go install github.com/TFMV/stride@latest

# Basic usage
stride /path/to/directory

# With options
stride --workers=8 --pattern="*.go" --exclude-dir="vendor,node_modules" --progress /path/to/directory

# With permission filtering
stride --min-permissions=0644 --max-permissions=0755 /path/to/directory
stride --exact-permissions=0644 /path/to/directory
```

### Available Options

```bash
# Basic filtering
--pattern string           File pattern to match
--exclude-pattern string   Patterns to exclude files (comma-separated)
--exclude-dir string       Directories to exclude (comma-separated)
--min-size string          Minimum file size to process
--max-size string          Maximum file size to process
--file-types string        File types to include (comma-separated: file,dir,symlink,pipe,socket,device,char)

# Time-based filtering
--modified-after string    Include files modified after (format: YYYY-MM-DD)
--modified-before string   Include files modified before (format: YYYY-MM-DD)
--accessed-after string    Include files accessed after (format: YYYY-MM-DD)
--accessed-before string   Include files accessed before (format: YYYY-MM-DD)
--created-after string     Include files created after (format: YYYY-MM-DD)
--created-before string    Include files created before (format: YYYY-MM-DD)

# Permission filtering
--min-permissions string   Minimum file permissions (octal, e.g. 0644)
--max-permissions string   Maximum file permissions (octal, e.g. 0755)
--exact-permissions string Exact file permissions to match (octal, e.g. 0644)

# Owner/Group filtering
--owner string             Filter by owner username
--group string             Filter by group name
--owner-uid int            Filter by owner UID
--owner-gid int            Filter by group GID

# Depth control
--min-depth int            Minimum directory depth to process
--max-depth int            Maximum directory depth to process

# Empty files/directories
--empty-files              Include only empty files
--empty-dirs               Include only empty directories

# Processing options
--error-mode string        Error handling mode (continue|stop|skip) [default: continue]
--follow-symlinks          Follow symbolic links [default: false]
--format string            Output format (text|json) [default: text]
--progress                 Show progress updates
--silent                   Disable all output except errors
-v, --verbose              Enable verbose logging
-w, --workers string       Number of concurrent workers [default: 4]
```

### Examples

```bash
# Find all Go files modified in the last week
stride --file-types=file --pattern="*.go" --modified-after=2023-05-01 /path/to/directory

# Find all empty directories
stride --file-types=dir --empty-dirs /path/to/directory

# Find all executable files owned by current user
stride --min-permissions=0755 --owner=$(whoami) /path/to/directory

# Find large files (>100MB) not accessed in the last month
stride --min-size=104857600 --accessed-before=$(date -d "30 days ago" +%Y-%m-%d) /path/to/directory

# Find all files at depth 2-3 with specific permissions
stride --min-depth=2 --max-depth=3 --exact-permissions=0644 /path/to/directory
```

## Architecture

```mermaid
flowchart LR
    %% Styling definitions
    classDef clientNode fill:#e1f5fe,stroke:#01579b,stroke-width:2px,color:#01579b,font-weight:bold
    classDef coreNode fill:#fff9c4,stroke:#f57f17,stroke-width:2px,color:#f57f17,font-weight:bold
    classDef optionsNode fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px,color:#6a1b9a,font-weight:bold
    classDef dataNode fill:#ffebee,stroke:#b71c1c,stroke-width:2px,color:#b71c1c,font-weight:bold
    classDef controlNode fill:#e0f7fa,stroke:#006064,stroke-width:2px,color:#006064,font-weight:bold

    %% Client and API cluster
    Client[🧑‍💻 Client Code]:::clientNode
    API[📋 Walk API]:::coreNode
    API2["Walk(), WalkLimit(), WalkLimitWithOptions()"]:::coreNode
    Client -->|Invokes| API
    API --> API2

    %% Configuration Options cluster
    subgraph Options [Configuration Options]
        Filter[🔍 Filtering\nSize, Type, Pattern]:::optionsNode
        Error[⚠️ Error Handling\nContinue, Stop, Skip]:::optionsNode
        Progress[📊 Progress Reporting\nReal-time Stats]:::optionsNode
        Symlink[🔗 Symlink Handling\nFollow, Ignore]:::optionsNode
        Memory[📏 Memory Limits\nSoft/Hard]:::optionsNode
    end

    %% Core Components cluster
    subgraph Core [Core Components]
        WP[👥 Worker Pool\nConcurrent Processing]:::coreNode
        FS[💾 File System\nDirectory Traversal]:::coreNode
        Files[📄 Files]:::dataNode
        Stats[📈 Statistics]:::dataNode
    end

    %% API relationships: it configures options and creates the worker pool.
    API -->|Configures| Filter
    API -->|Configures| Error
    API -->|Configures| Progress
    API -->|Configures| Symlink
    API -->|Configures| Memory
    API -->|Creates| WP

    %% Core relationships
    WP -->|Traverses| FS
    WP -->|Processes| Files
    WP -->|Updates| Stats
    Stats -->|Reports via| Progress

    %% Option-to-Core links
    Filter -- Determines --> Files
    Error -- Affects --> WP
    Symlink -- Configures --> FS
    Memory -- Constrains --> WP

    %% External control for cancellation
    Context[⏱️ Context\nCancellation]:::controlNode
    Context -- Controls --> WP
```

## Performance

Stride has been benchmarked with various configurations to measure its performance characteristics. Here are some key results:

### Comparison with Standard Library

Our benchmarks show interesting performance characteristics when comparing Stride with the standard library's `filepath.Walk`:

| Implementation | Time (ns/op) | Memory (B/op) | Allocations (allocs/op) |
|----------------|--------------|---------------|-------------------------|
| filepath.Walk  | 3,782,222    | 700,032       | 5,511                   |
| stride.Walk    | 6,239,953    | 1,218,906     | 6,918                   |
| stride.WalkLimit (4 workers) | 6,238,284 | 1,221,242 | 6,906          |
| stride.WalkLimit (16 workers) | 6,057,815 | 1,222,156 | 6,909         |
| stride.WalkLimit (32 workers) | 5,975,106 | 1,223,357 | 6,911         |

While the standard library's `filepath.Walk` is faster for simple traversal with no processing, Stride's value becomes apparent when:

1. **Processing files concurrently**: When actual work is performed on each file, Stride's concurrent processing provides significant speedups
2. **Filtering is needed**: Stride's built-in filtering avoids unnecessary processing
3. **Progress reporting is required**: Stride provides real-time statistics
4. **Complex error handling is needed**: Stride offers multiple error handling strategies

For CPU-bound file processing tasks, the concurrent nature of Stride can provide up to N times speedup (where N is the number of CPU cores) compared to sequential processing.

```mermaid
%%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#f4f4f4', 'primaryTextColor': '#333', 'primaryBorderColor': '#ddd', 'lineColor': '#666', 'secondaryColor': '#fafafa', 'tertiaryColor': '#fff'}}}%%
pie showData
    title Memory Usage Comparison (B/op)
    "filepath.Walk" : 700032
    "stride.Walk" : 1218906
    "stride.WalkLimit (4)" : 1221242
    "stride.WalkLimit (16)" : 1222156
    "stride.WalkLimit (32)" : 1223357
```

### Worker Count Impact

Increasing the number of workers can improve performance, but the optimal number depends on your specific workload and system:

```mermaid
%%{init: {'theme': 'neutral'}}%%
xychart-beta
    title "Worker Count vs Performance"
    x-axis [1, 2, 4, 8, 16, 32, 64]
    y-axis "Time (ns/op)" 5000000 --> 7000000
    bar [6630635, 6191419, 6238284, 6438669, 6057815, 5975106, 6891416]
    line [6630635, 6191419, 6238284, 6438669, 6057815, 5975106, 6891416]
```

### Filtering Impact

Filtering adds minimal overhead while significantly reducing the number of files processed:

```mermaid
%%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#e8f5e9', 'secondaryColor': '#c8e6c9', 'tertiaryColor': '#f1f8e9'}}}%%
xychart-beta
    title "Filtering Impact on Performance"
    x-axis ["No Filter", "Extension Only", "Size Only", "Combined"]
    y-axis "Time (ns/op)" 8400000 --> 8900000
    bar [8583018, 8510879, 8795316, 8798272]
```

| Filter Type    | Time (ns/op) | Memory (B/op) | Allocations (allocs/op) |
|----------------|--------------|---------------|-------------------------|
| No Filter      | 8,583,018    | 1,660,929     | 9,566                   |
| Extension Only | 8,510,879    | 1,660,889     | 9,566                   |
| Size Only      | 8,795,316    | 1,660,925     | 9,566                   |
| Combined       | 8,798,272    | 1,660,948     | 9,566                   |

### Progress Reporting Overhead

Progress reporting adds a small overhead:

```mermaid
%%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#e3f2fd', 'secondaryColor': '#bbdefb', 'tertiaryColor': '#e1f5fe'}}}%%
xychart-beta
    title "Progress Reporting Impact"
    x-axis ["Without Progress", "With Progress"]
    y-axis "Time (ns/op)" 8000000 --> 10000000
    bar [8482935, 9557600]
```

| Progress | Time (ns/op) | Memory (B/op) | Allocations (allocs/op) |
|----------|--------------|---------------|-------------------------|
| Without  | 8,482,935    | 2,062,705     | 9,610                   |
| With     | 9,557,600    | 2,262,628     | 12,586                  |

### Core Functions Performance

The core utility functions are highly optimized:

```mermaid
%%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#fff3e0', 'secondaryColor': '#ffe0b2', 'tertiaryColor': '#fff8e1'}}}%%
xychart-beta
    title "Core Function Performance (ns/op)"
    x-axis ["filePassesFilter", "shouldSkipDir"]
    y-axis "Time (ns/op)" 0 --> 12
    bar [8.98, 10.04]
```

| Function        | Time (ns/op) | Memory (B/op) | Allocations (allocs/op) |
|-----------------|--------------|---------------|-------------------------|
| filePassesFilter| 8.980        | 0             | 0                       |
| shouldSkipDir   | 10.04        | 0             | 0                       |

### Realistic Workload Performance

When performing actual file processing (reading files and computing SHA-256 hashes), Stride shows significant performance advantages:

```mermaid
%%{init: {'theme': 'forest', 'themeVariables': { 'primaryColor': '#e8f5e9', 'secondaryColor': '#c8e6c9', 'tertiaryColor': '#f1f8e9', 'primaryTextColor': '#1b5e20', 'fontSize': '16px'}}}%%
xychart-beta
    title "Realistic Workload Performance (lower is better)"
    x-axis ["filepath.Walk", "stride.Walk", "stride.WalkLimit (4)", "stride.WalkLimit (16)", "stride.WalkLimit (32)"]
    y-axis "Time (ns/op)" 0 --> 25000000
    bar [19446695, 10734420, 12858610, 10858304, 11165940]
```

| Implementation | Time (ns/op) | Memory (B/op) | Allocations (allocs/op) |
|----------------|--------------|---------------|-------------------------|
| filepath.Walk  | 19,446,695   | 1,010,387     | 10,515                  |
| stride.Walk    | 10,734,420   | 1,532,700     | 11,939                  |
| stride.WalkLimit (1 worker) | 23,479,985 | 1,525,112 | 11,923        |
| stride.WalkLimit (4 workers) | 12,858,610 | 1,525,640 | 11,924       |
| stride.WalkLimit (16 workers) | 10,858,304 | 1,526,309 | 11,925      |
| stride.WalkLimit (32 workers) | 11,165,940 | 1,527,260 | 11,926      |

As shown above, with realistic file processing workloads:

- Stride's basic `Walk` function is ~45% faster than the standard library
- Stride's concurrent implementation with 16 workers is ~44% faster than the standard library
- The optimal worker count depends on the specific workload and system

## Testing

Stride includes comprehensive test coverage:

- Unit tests for all core functions
- Integration tests for end-to-end functionality
- Edge case tests for error handling, symlinks, etc.
- Benchmarks for performance measurement

Run the tests with:

```bash
go test ./...
```

Run benchmarks with:

```bash
go test -bench=. -benchmem ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
