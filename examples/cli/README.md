# Stride CLI Command Reference

This document provides a comprehensive reference for the Stride command-line interface (CLI), which offers powerful filesystem traversal and search capabilities.

## Global Options

These options apply to all Stride commands:

```bash
# Display help information
stride --help

# Display version information
stride --version

# Set verbosity level (0-3)
stride --verbose=2

# Silence all output except errors
stride --silent

# Set number of concurrent workers
stride --workers=8

# Set output format (text, json, csv)
stride --format=json

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

# Watch for file changes
stride find /path/to/search --watch
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

# Files modified after specific date
stride find /path/to/search --modified-after="2023-01-01"

# Files modified before specific date
stride find /path/to/search --modified-before="2023-12-31"

# Files accessed after specific date
stride find /path/to/search --accessed-after="2023-01-01"

# Files created after specific date
stride find /path/to/search --created-after="2023-01-01"
```

### Size-Based Filtering

```bash
# Files larger than 1MB
stride find /path/to/search --larger-than=1MB

# Files smaller than 10KB
stride find /path/to/search --smaller-than=10KB

# Files exactly 4KB in size
stride find /path/to/search --size=4KB
```

### Permission Filtering

```bash
# Files with minimum permissions
stride find /path/to/search --min-permissions=0644

# Files with maximum permissions
stride find /path/to/search --max-permissions=0755

# Files with exact permissions
stride find /path/to/search --permissions=0644

# Files owned by specific user
stride find /path/to/search --owner=username

# Files owned by specific group
stride find /path/to/search --group=groupname

# Files with specific UID
stride find /path/to/search --uid=1000

# Files with specific GID
stride find /path/to/search --gid=1000
```

### Traversal Options

```bash
# Maximum directory depth
stride find /path/to/search --max-depth=3

# Minimum directory depth
stride find /path/to/search --min-depth=1

# Follow symbolic links
stride find /path/to/search --follow-symlinks

# Include hidden files
stride find /path/to/search --include-hidden

# Include only empty files
stride find /path/to/search --empty-files

# Include only empty directories
stride find /path/to/search --empty-dirs
```

### Output and Action Options

```bash
# Execute command for each match
stride find /path/to/search --exec="echo Processing: {}"

# Format output using template
stride find /path/to/search --format="{base} ({size} bytes)"

# Output as JSON
stride find /path/to/search --output=json

# Output as CSV
stride find /path/to/search --output=csv

# Count matches only
stride find /path/to/search --count

# Show progress
stride find /path/to/search --progress
```

### Watch Options

```bash
# Watch for file changes
stride find /path/to/search --watch

# Watch for specific events
stride find /path/to/search --watch --events=create,modify,delete

# Watch with timeout
stride find /path/to/search --watch --timeout=1h
```

## Hash Command

The `hash` command computes file hashes:

```bash
# Compute MD5 hashes
stride hash /path/to/directory --md5

# Compute SHA1 hashes
stride hash /path/to/directory --sha1

# Compute SHA256 hashes
stride hash /path/to/directory --sha256

# Compute multiple hash types
stride hash /path/to/directory --md5 --sha256

# Hash only specific files
stride hash /path/to/directory --pattern="*.go"

# Output as CSV
stride hash /path/to/directory --output=csv

# With size filtering
stride hash /path/to/directory --min-size=1KB --max-size=10MB
```

## Stat Command

The `stat` command provides filesystem statistics:

```bash
# Basic statistics
stride stat /path/to/directory

# Detailed statistics
stride stat /path/to/directory --detailed

# Statistics by file type
stride stat /path/to/directory --by-type

# Statistics by file size
stride stat /path/to/directory --by-size

# Statistics by modification time
stride stat /path/to/directory --by-time

# Statistics with filtering
stride stat /path/to/directory --pattern="*.go"

# Output as JSON
stride stat /path/to/directory --output=json
```

## Duplicate Command

The `duplicate` command finds duplicate files:

```bash
# Find duplicates by content
stride duplicate /path/to/directory

# Find duplicates by name
stride duplicate /path/to/directory --by-name

# Find duplicates by size
stride duplicate /path/to/directory --by-size

# Find duplicates with minimum size
stride duplicate /path/to/directory --min-size=1MB

# Output as JSON
stride duplicate /path/to/directory --output=json

# Execute command on duplicates
stride duplicate /path/to/directory --exec="echo Duplicate: {}"
```

## Template Placeholders

The following placeholders can be used in `--format` and `--exec` options:

```
{}        - Full path to the file
{base}    - Base name of the file
{dir}     - Directory containing the file
{size}    - Size in bytes
{time}    - Modification time
{atime}   - Access time
{ctime}   - Creation time
{mode}    - File permissions
{owner}   - File owner
{group}   - File group
{type}    - File type (file, dir, symlink)
{ext}     - File extension
{version} - Version identifier (if available)
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

# Report all errors
stride --error-mode=report
```

## Examples

### Find and Process Large Log Files

```bash
stride find /var/log --name="*.log" --larger-than=10MB --exec="gzip {}"
```

### Find Recent Source Code Changes

```bash
stride find ~/projects --name="*.go" --newer-than=24h --format="{base} modified at {time}"
```

### Find Duplicate Images

```bash
stride duplicate ~/Pictures --pattern="*.jpg" --min-size=1MB --output=json
```

### Generate File Statistics Report

```bash
stride stat ~/Documents --detailed --output=csv > report.csv
```

### Watch for New Files

```bash
stride find ~/Downloads --watch --exec="notify-send 'New file: {base}'"
```
