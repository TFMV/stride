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

## Building Your Own

These examples serve as starting points for building your own applications with Stride. The key components to consider:

1. **Choose the right API**: Use the standard API for simple cases or the enhanced API for more complex scenarios
2. **Configure filtering**: Set up appropriate filters to process only the files you need
3. **Handle errors properly**: Select the error handling mode that fits your use case
4. **Monitor progress**: Use the progress reporting for long-running operations
5. **Add middleware**: For the enhanced API, consider adding middleware for logging, metrics, etc.

For more details, refer to the main [Stride documentation](../README.md).
