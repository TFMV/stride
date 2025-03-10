# Stride

[![Go Reference](https://pkg.go.dev/badge/github.com/TFMV/stride.svg)](https://pkg.go.dev/github.com/TFMV/stride)
[![Go Report Card](https://goreportcard.com/badge/github.com/TFMV/stride)](https://goreportcard.com/report/github.com/TFMV/stride)
[![License](https://img.shields.io/github/license/TFMV/stride)](LICENSE)

Stride is a high-performance, concurrent filesystem traversal and search library for Go. It extends the standard `filepath.Walk` with enhanced concurrency, filtering, and monitoring features while providing a Linux-like `find`-like API and CLI.

## Features

| Feature | Description |
|---------|-------------|
| Concurrent Processing | Traverse directories in parallel with configurable worker pools |
| Advanced File Searching | Search files by name, path, size, modification time, metadata, and tags |
| Regular Expressions & Wildcards | Support for flexible pattern matching |
| Progress Monitoring | Real-time statistics during traversal |
| Symlink Handling | Configurable behavior for following symbolic links |
| Memory Constraints | Define soft and hard memory limits to prevent excessive resource usage |
| Context Support | Gracefully cancel operations using Go's context.Context |
| Custom Execution | Run shell commands for matching files (like find -exec) |
| Filesystem Watching | Monitor directories for changes and react to events in real-time |
| Advanced Analysis | Comprehensive filesystem analysis including duplicates, dependencies, and security scanning |

## Installation

```bash
go get github.com/TFMV/stride
```

## Quick Start

For usage examples, see the [examples directory](examples/).

### Walk

```go
// Basic usage - similar to filepath.Walk
err := stride.Walk(".", func(path string, info os.FileInfo, err error) error {
    if err != nil {
        return err
    }
    fmt.Println(path)
    return nil
})
```

### Find

```go
opts := walk.FindOptions{
    NamePattern: "*.go", // Find all Go files
}

// Find files and process them
err := walk.Find(ctx, rootDir, opts, func(ctx context.Context, result walk.FindResult) error {
    if result.Error != nil {
        return result.Error
    }
    fmt.Printf("Found Go file: %s\n", result.Message.Path)
    return nil
})
```

### Watch

```go
opts := walk.WatchOptions{
    Recursive: true,
    Events:    []walk.WatchEvent{walk.EventCreate, walk.EventModify},
    Pattern:   "*.go",
}

// Watch for changes and process them
err := walk.Watch(ctx, watchDir, opts, func(ctx context.Context, result walk.WatchResult) error {
    if result.Error != nil {
        return result.Error
    }
    fmt.Printf("Event: %s, File: %s\n", result.Message.Event, result.Message.Path)
    return nil
})
```

### Analyze

```go
// Create an analyzer with desired features
analyzer := walk.NewAnalyzer()
analyzer.EnableDuplicateDetection()
analyzer.EnableCodeStats()
analyzer.EnableStorageReport()
analyzer.EnableSecurityScan()
analyzer.EnableDependencyAnalysis()

// Configure analyzer options
analyzer.SetLanguages([]string{"go", "js", "py"})
analyzer.SetMaxDepth(5)
analyzer.SetSizeRange("1KB", "1GB")
analyzer.SetOutputFormat("json")

// Run analysis
result, err := analyzer.Analyze("/path/to/analyze")
if err != nil {
    log.Fatal(err)
}

// Process results
fmt.Printf("Total files: %d\n", result.StorageReport.FileCount)
fmt.Printf("Total size: %d bytes\n", result.StorageReport.TotalSize)

// Check for duplicates
for hash, paths := range result.Duplicates {
    if len(paths) > 1 {
        fmt.Printf("Found duplicate files:\n")
        for _, path := range paths {
            fmt.Printf("  %s\n", path)
        }
    }
}

// Check for security issues
for _, issue := range result.SecurityIssues {
    fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Path, issue.Description)
}

// Check for near-duplicates
if result.Advanced != nil {
    for _, group := range result.Advanced.NearDuplicates {
        fmt.Printf("\nSimilarity: %.0f%%\n", group.Similarity*100)
        fmt.Printf("Files:\n")
        for _, file := range group.Files {
            fmt.Printf("  %s\n", file)
        }
        fmt.Printf("Suggested action: %s\n", group.Resolution)
    }
}

// Check for dependency issues
if result.Advanced != nil && result.Advanced.Dependencies != nil {
    if len(result.Advanced.Dependencies.Orphans) > 0 {
        fmt.Printf("\nOrphan files:\n")
        for _, file := range result.Advanced.Dependencies.Orphans {
            fmt.Printf("  %s\n", file)
        }
    }
}
```

## Key Components

### Walk API

The library provides several functions for traversing the filesystem:

- `Walk()` - Basic traversal similar to `filepath.Walk`
- `WalkLimit()` - Concurrent traversal with a specified number of workers
- `WalkLimitWithFilter()` - Concurrent traversal with filtering options
- `WalkLimitWithProgress()` - Concurrent traversal with progress reporting
- `WalkLimitWithOptions()` - Concurrent traversal with comprehensive options

### Find API

The library includes find capabilities:

- `Find()` - Search for files with pattern matching and filtering
- `FindWithExec()` - Execute commands for matched files
- `FindWithFormat()` - Format output for matched files

### Watch API

The library provides filesystem monitoring capabilities:

- `Watch()` - Monitor directories for changes
- `WatchWithExec()` - Execute commands when files change
- `WatchWithFormat()` - Format output for file change events

### Analyze API

The library provides advanced filesystem analysis capabilities:

- `Analyze()` - Perform comprehensive filesystem analysis
- Basic Analysis:
  - Storage usage and file type statistics
  - Code statistics (lines, comments, blanks)
  - Security scanning (permissions, suspicious files)
  - Content pattern detection
- Advanced Analysis:
  - Intelligent duplicate detection (exact and near-duplicates)
  - Code dependency analysis
  - Dead code detection
  - Automated deduplication suggestions

### Command Line Tool

Stride includes a CLI tool for quick filesystem traversal:

A CLI command reference is available in the [examples/cli/](examples/cli/README.md) directory.

```bash
# Install the command-line tool
go install github.com/TFMV/stride@latest

# Basic walk usage
stride /path/to/directory

# Basic find usage
stride find /path/to/search --name="*.go"

# Basic watch usage
stride watch /path/to/watch --recursive --pattern="*.go"

# Basic analyze usage
stride analyze /path/to/analyze

# Advanced analyze usage examples
stride analyze /path/to/analyze --duplicates --near-duplicates  # Find exact and similar duplicates
stride analyze /path/to/analyze --code-stats --languages=go,js  # Analyze code statistics
stride analyze /path/to/analyze --storage-report --output=html  # Generate storage usage report
stride analyze /path/to/analyze --security-scan                 # Perform security analysis
stride analyze /path/to/analyze --dependencies                  # Analyze code dependencies
stride analyze /path/to/analyze --all                          # Run all analysis types
```

Example analyze output:

```
Storage Report:
Total Size: 1.2GB
Files: 1,234
Directories: 56

Code Statistics:
Go: 45 files, 12,345 lines (1,234 comments, 567 blanks)
JavaScript: 23 files, 5,678 lines (456 comments, 234 blanks)

Security Issues:
[High] /path/to/file.sh: File is world-writable
[Medium] /path/to/config.json: Contains API key

Near-Duplicate Files:
Similarity: 95%
Files in group:
  /path/to/original.js
  /path/to/copy.js
Suggested action: Review files for possible consolidation

Dependency Analysis:
Orphan Files (not imported by any other file):
  /path/to/unused.go
  /path/to/dead_code.go

Unused Files (no imports or importers):
  /path/to/old_utils.go
```

## Performance

Stride has been optimized for performance, especially for CPU-bound file processing tasks. The concurrent nature of Stride can provide significant speedups compared to sequential processing.

For detailed benchmarks, see the [benchmarks](BENCHMARK.md).

## Testing

Run the tests with:

```bash
go test ./...
```

Run benchmarks with:

```bash
go test -bench=. -benchmem ./...
```

## License

This project is licensed under the [MIT License](LICENSE).

## Author

Built with :heart: by TFMV.
