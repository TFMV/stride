# Changes in Stride v0.1.0

## New Advanced Find Functionality

We've added powerful find capabilities to Stride, inspired by Unix's `find` command but with modern enhancements. This allows for sophisticated file searching with pattern matching, filtering, and action execution.

### Key Features Added

1. **Pattern Matching**
   - Find files by name with wildcard support (`*.go`)
   - Find files by path patterns (`*/src/*.go`)
   - Find files by regular expressions (`.*_test\.go$`)
   - Ignore paths matching specific patterns

2. **Time-Based Filtering**
   - Find files older than a specified duration (`7d`, `24h`)
   - Find files newer than a specified duration
   - Human-friendly duration formats

3. **Size-Based Filtering**
   - Find files larger than a specified size (`1MB`, `500KB`)
   - Find files smaller than a specified size
   - Human-friendly size formats (KB, MB, GB, TB)

4. **Metadata and Tag Filtering**
   - Find files with specific metadata using regex patterns
   - Find files with specific tags using regex patterns
   - Support for key-value pattern matching

5. **Command Execution**
   - Execute commands for each matched file
   - Template placeholders for file information:
     - `{}`: Full path to the file
     - `{base}`: Base name of the file
     - `{dir}`: Directory containing the file
     - `{size}`: Size in bytes
     - `{time}`: Modification time
     - `{version}`: Version identifier (if available)
   - Quoted versions for shell safety: `{""}`, `{"base"}`, etc.

6. **Output Formatting**
   - Format output using templates
   - Customize the display of file information

7. **File Watching**
   - Watch for file changes and process them in real-time
   - Filter events by type (create, modify, delete)

8. **Command Line Interface**
   - New `find` command with all the above capabilities
   - Intuitive flags and options
   - Human-friendly parameter formats

### Implementation Details

1. **New Types**
   - `FindMessage`: Holds information about a found file
   - `FindOptions`: Defines criteria for finding files
   - `FindResult`: Represents a file that matched the criteria
   - `FindHandler`: Function type for processing found files

2. **New Functions**
   - `Find`: Main function for searching files
   - `FindWithExec`: Executes a command for each matched file
   - `FindWithFormat`: Formats output for each matched file
   - `CompileRegexMap`: Helper for compiling regex patterns

3. **Helper Functions**
   - `nameMatch`: Checks if a file name matches a pattern
   - `pathMatch`: Checks if a path matches a pattern
   - `matchFind`: Checks if a file matches all criteria
   - `matchRegexMap`: Checks if values match regex patterns
   - `trimPathAtMaxDepth`: Trims paths to a maximum depth

4. **CLI Integration**
   - New `find` command in the CLI
   - Comprehensive flags for all find options
   - Human-friendly parameter parsing

### API Design

The find functionality is exposed through a clean, well-documented API:

```go
// Find searches for files matching the given criteria
func Find(ctx context.Context, root string, opts FindOptions, handler FindHandler) error

// FindWithExec searches for files and executes a command for each match
func FindWithExec(ctx context.Context, root string, opts FindOptions, cmdTemplate string) error

// FindWithFormat searches for files and formats output according to a template
func FindWithFormat(ctx context.Context, root string, opts FindOptions, formatTemplate string) error
```

### Testing

Comprehensive tests have been added to ensure the reliability of the new functionality:

- Tests for basic find operations with various filters
- Tests for command execution
- Tests for output formatting
- Tests for regex pattern matching

## Documentation Updates

- Added detailed documentation for the find functionality in the README
- Added examples for common use cases
- Updated CLI documentation to include the new `find` command
- Added information about the find functionality to the release notes

## Future Enhancements

Potential future enhancements for the find functionality:

1. **File System Watcher Implementation**
   - Complete the file system watcher for real-time monitoring
   - Support for various file system events

2. **Advanced Metadata Extraction**
   - Extract and filter by EXIF data for images
   - Extract and filter by document metadata

3. **Content-Based Filtering**
   - Find files containing specific text
   - Find files matching content patterns

4. **Performance Optimizations**
   - Parallel pattern matching
   - Indexed searches for frequently used patterns
