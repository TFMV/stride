# Stride v0.1.0 Release Notes

We're excited to announce the initial release of Stride, a high-performance, concurrent filesystem traversal library for Go.

## Overview

Stride builds upon the standard `filepath.Walk` functionality while adding concurrency, filtering, progress monitoring, and advanced configuration options.

## Key Features

### High-Performance Concurrent Processing

- Process files in parallel with configurable worker pools
- Significant performance improvements for CPU-bound file processing tasks
- Optimized for multi-core systems with configurable worker count

### Flexible Filtering System

- Filter files by size, extension, pattern, and modification time
- Advanced filtering by permissions, owner/group, and file depth
- Exclude specific directories or file patterns
- Time-based filtering (modified, accessed, created)

### Real-Time Progress Monitoring

- Live statistics during traversal with minimal overhead
- Track files processed, directories traversed, and processing speed
- Support for both text and JSON output formats

### Advanced Find Capabilities

- Powerful file search with pattern matching (name, path, regex)
- Time and size-based filtering with human-friendly formats (e.g., "7d", "10MB")
- Metadata and tag filtering with regular expression support
- Command execution for each matched file with template placeholders
- Output formatting with customizable templates
- File watching for real-time processing of changes

### Robust Symlink Handling

- Three configurable modes: Ignore, Follow, and Report
- Cycle detection to prevent infinite loops when following symlinks
- Proper path resolution for symlinked files and directories

### Comprehensive Error Handling

- Multiple strategies: Continue, Stop, or Skip on errors
- Detailed error reporting and propagation
- Context-aware cancellation support

### Advanced Configuration

- Memory limits with soft and hard thresholds
- Context support for cancellation and deadlines
- Structured logging with zap logger integration
- Middleware support for cross-cutting concerns

### Command-Line Utility

- Powerful CLI tool for quick filesystem traversal
- Extensive filtering options matching the library capabilities
- Progress reporting with real-time updates
- Multiple output formats (text, JSON)
- New `find` command with advanced search capabilities

## Performance Highlights

- Up to N times speedup for CPU-bound tasks (where N is the number of CPU cores)
- Optimized core functions with minimal memory allocations
- Efficient filtering with negligible overhead
- Progress reporting with minimal performance impact
- Find functionality processes files at a rate of approximately 1.6 million files per second
- Pattern matching adds only 1-5% overhead to basic traversal
- Core matching functions optimized with minimal allocations
- While the standard library's `filepath.Walk` is faster for simple traversal, Stride's concurrent processing provides significant advantages for CPU-bound operations
- Stride's built-in filtering capabilities add minimal overhead compared to implementing filtering manually with the standard library

## Installation

The library can be installed using Go modules:

`go get github.com/TFMV/stride`

The command-line utility can be installed with:

`go install github.com/TFMV/stride@latest`

## Documentation

Comprehensive documentation is available in the README, including:

- Quick start guide
- Advanced usage examples
- API reference
- Performance benchmarks
- Architecture overview

## Future Directions

In upcoming releases, we plan to focus on:

- Additional performance optimizations
- Enhanced middleware capabilities
- More filtering options
- Improved error handling and reporting
- Extended CLI functionality

---

For more information, visit the [GitHub repository](https://github.com/TFMV/stride).
