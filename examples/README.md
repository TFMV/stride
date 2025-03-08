# Stride Examples

This directory contains example applications demonstrating various features of the Stride filesystem traversal library. Each example is designed to showcase specific capabilities and usage patterns.

## Available Examples

### Running the Examples

Each example can be run from its directory. If no directory path is provided, the current directory will be used for traversal.

```bash
# Run with default directory (current directory)
go run main.go

# Run with specific directory
go run main.go /path/to/directory
```

### Basic Usage (`basic/`)

Demonstrates the fundamental usage of Stride for filesystem traversal, including:

- Simple file walking similar to `filepath.Walk`
- Concurrent file processing with worker pools
- Basic error handling

```bash
cd basic
go run main.go [directory_path]
```

### Advanced Filters (`advanced_filters/`)

Shows how to use Stride's powerful filtering capabilities:

- Filtering by file size, extension, and patterns
- Time-based filtering (modified, accessed, created)
- Permission and ownership filtering
- Depth control for directory traversal

```bash
cd advanced_filters
go run main.go [directory_path]
```

### Permissions Handling (`permissions/`)

Demonstrates permission-specific features:

- Filtering by permission modes
- Handling permission errors
- Working with owner and group permissions

```bash
cd permissions
go run main.go [directory_path]
```

### Enhanced API (`enhanced_api/`)

Showcases the new context-aware API with middleware support:

- Context-aware callbacks for cancellation
- Real-time statistics during traversal
- Middleware patterns for cross-cutting concerns
- Logging and timing middleware examples

```bash
cd enhanced_api
go run main.go [directory_path]
```

### Find API (`find_api/`)

Demonstrates Stride's powerful file searching capabilities:

- Basic file searching with pattern matching
- Advanced filtering by name, path, size, and time
- Executing commands on found files (similar to `find -exec`)
- Custom output formatting with templates
- Permission error handling with different strategies
- Tracking and reporting permission issues during traversal

```bash
cd find_api
go run main.go [directory_path]
```

### Watch API (`watch/`)

Demonstrates Stride's filesystem monitoring capabilities:

- Real-time monitoring of filesystem changes
- Event filtering (create, modify, delete, rename, chmod)
- Pattern matching for specific file types
- Command execution on file events
- Custom output formatting

```bash
cd watch
go run main.go [directory_path]
```

### File Hashing (`file_hashing/`)

Demonstrates using Stride to efficiently compute file hashes in parallel:

- Supports multiple hash algorithms (MD5, SHA1, SHA256)
- Processes files concurrently for maximum performance
- Provides real-time progress reporting
- Outputs results in text or CSV format
- Includes filtering by file size and pattern

```bash
cd file_hashing
go run main.go [directory_path]

# With options
go run main.go --workers=8 --pattern="*.go" --md5 --sha256 /path/to/directory
go run main.go --format=csv --min-size=1024 /path/to/directory
```

## Building Your Own

These examples serve as starting points for building your own applications with Stride. The key components to consider:

1. **Choose the right API**: Use the standard API for simple cases or the enhanced API for more complex scenarios
2. **Configure filtering**: Set up appropriate filters to process only the files you need
3. **Handle errors properly**: Select the error handling mode that fits your use case
4. **Monitor progress**: Use the progress reporting for long-running operations
5. **Add middleware**: For the enhanced API, consider adding middleware for logging, metrics, etc.

For more details, refer to the main [Stride documentation](../README.md).
