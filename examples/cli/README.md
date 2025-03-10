# Stride CLI Command Reference

This document provides a reference for the Stride command-line interface (CLI), which offers powerful filesystem traversal and search capabilities.

## Global Options

These options apply to all Stride commands:

```bash
# Display help information
stride --help

# Display version information
stride --version

# Set verbosity level
stride --verbose

# Set number of concurrent workers
stride --workers=8

# Set error handling mode (continue, stop, skip)
stride --error-mode=continue
```

## Basic Usage

The basic command traverses a directory and lists all files:

```bash
# Traverse current directory
stride

# Traverse specified directory
stride /path/to/directory

# Traverse with worker limit
stride /path/to/directory --workers=4
```

## Find Command

The `find` command provides powerful file searching capabilities:

### Basic Search

```bash
# Basic search
stride find /path/to/search --name="*.go"

# Find files modified in the last 24 hours
stride find /path/to/search --newer-than=24h

# Find large files (>10MB)
stride find /path/to/search --larger-than=10MB

# Find files matching a regex pattern
stride find /path/to/search --regex=".*\.log$"

# Execute a command for each matched file
stride find /path/to/search --exec="echo Processing: {}"

# Format output with a template
stride find /path/to/search --format="{base} ({size} bytes)"

# Find with multiple criteria
stride find /path/to/search --name="*.go" --newer-than=7d --max-depth=3

# Find with permission handling
stride find /path/to/search --follow-symlinks --include-hidden
```

### Pattern Matching Options

```bash
# Match by file name (supports wildcards)
stride find /path/to/search --name="*.go"

# Match by path (supports wildcards)
stride find /path/to/search --path="*/src/*.go"

# Skip paths matching this pattern
stride find /path/to/search --ignore="*/vendor/*"

# Match by regular expression
stride find /path/to/search --regex=".*_test\.go$"
```

### Time-Based Filtering

```bash
# Files older than 7 days
stride find /path/to/search --older-than=7d

# Files newer than 1 hour
stride find /path/to/search --newer-than=1h
```

### Size-Based Filtering

```bash
# Files larger than 1MB
stride find /path/to/search --larger-than=1MB

# Files smaller than 10KB
stride find /path/to/search --smaller-than=10KB
```

### Traversal Options

```bash
# Maximum directory depth
stride find /path/to/search --max-depth=3

# Follow symbolic links
stride find /path/to/search --follow-symlinks

# Include hidden files
stride find /path/to/search --include-hidden
```

### Output and Action Options

```bash
# Execute command for each match
stride find /path/to/search --exec="echo Processing: {}"

# Format output using template
stride find /path/to/search --format="{base} ({size} bytes)"
```

### Watch Options

```bash
# Watch for file changes
stride find /path/to/search --watch

# Watch for specific events
stride find /path/to/search --watch --events=create,modify,delete
```

## Watch Command

The `watch` command monitors filesystem changes and can execute actions when files are created, modified, or deleted:

```bash
# Watch current directory
stride watch

# Watch specific directory
stride watch /path/to/watch

# Watch for specific events
stride watch --events=create,modify /path/to/watch

# Watch recursively (including subdirectories)
stride watch --recursive /path/to/watch

# Execute command when events occur
stride watch --exec="echo Changed: {}" /path/to/watch

# Format output with template
stride watch --format="{base} was {event} at {time}" /path/to/watch

# Watch only specific file types
stride watch --pattern="*.go" /path/to/watch

# Ignore specific files
stride watch --ignore="*.tmp" /path/to/watch

# Watch with timeout
stride watch --timeout=1h /path/to/watch

# Include hidden files and directories
stride watch --include-hidden /path/to/watch
```

## Template Placeholders

The following placeholders can be used in `--format` and `--exec` options:

```bash
{}        - Full path to the file
{base}    - Base name of the file
{dir}     - Directory containing the file
{size}    - Size in bytes
{time}    - Modification time
{version} - Version identifier (if available)
{event}   - Event type (created, modified, deleted, renamed, chmod) - only for watch command
```

Quoted versions are also available for shell escaping: `{""}`, `{"base"}`, etc.

## Error Handling

Control how errors are handled during traversal:

```bash
# Continue on errors (default)
stride --error-mode=continue

# Stop on first error
stride --error-mode=stop

# Skip directories with errors
stride --error-mode=skip
```

## Examples

### Find and Process Go Files

```bash
stride find /path/to/project --name="*.go" --exec="gofmt -w {}"
```

### Find Recent Changes

```bash
stride find ~/projects --name="*.go" --newer-than=24h --format="{base} modified at {time}"
```

### Find Large Files

```bash
stride find /home --larger-than=100MB --format="{} ({size} bytes)"
```

### Watch for New Files

```bash
stride watch ~/Downloads --exec="echo New file: {base}"
```

### Watch for Changes in Go Files

```bash
stride watch --recursive --pattern="*.go" --events=create,modify --exec="go test ./..." ~/projects
```

### Watch with Custom Output Format

```bash
stride watch --format="{base} was {event} at {time}" ~/Documents
```
