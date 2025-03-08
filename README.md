# Stride

[![Go Reference](https://pkg.go.dev/badge/github.com/TFMV/stride.svg)](https://pkg.go.dev/github.com/TFMV/stride)
[![Go Report Card](https://goreportcard.com/badge/github.com/TFMV/stride)](https://goreportcard.com/report/github.com/TFMV/stride)
[![License](https://img.shields.io/github/license/TFMV/stride)](LICENSE)

Stride is a high-performance, concurrent filesystem traversal and search library for Go. It extends the standard `filepath.Walk` with enhanced concurrency, filtering, and monitoring features while providing a Linux-like `find`-like API and CLI.

## Features

- Concurrent Processing – Traverse directories in parallel with configurable worker pools.
- Advanced File Searching – Search files by name, path, size, modification time, metadata, and tags.
- Regular Expressions & Wildcards – Support for flexible pattern matching.
- Progress Monitoring – Real-time statistics during traversal.
- Symlink Handling – Configurable behavior for following symbolic links.
- Memory Constraints – Define soft and hard memory limits to prevent excessive resource usage.
- Context Support – Gracefully cancel operations using Go's context.Context.
- Custom Execution – Run shell commands for matching files (like find -exec).
- Filesystem Watching – Monitor directories for changes and react to events in real-time.

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
