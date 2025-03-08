// Package stride provides high-performance filesystem traversal with advanced filtering
package stride

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/unicode/norm"
)

// FindMessage holds information about a file found during traversal
type FindMessage struct {
	Path      string            // Full path to the file
	Name      string            // Base name of the file
	Dir       string            // Directory containing the file
	Size      int64             // Size in bytes
	Time      time.Time         // Modification time
	IsDir     bool              // Whether the entry is a directory
	Metadata  map[string]string // File metadata
	Tags      map[string]string // File tags
	VersionID string            // Version identifier (if applicable)
}

// FindOptions defines the criteria for finding files
type FindOptions struct {
	// Pattern matching options
	NamePattern   string         // Match by file name (supports wildcards)
	PathPattern   string         // Match by path (supports wildcards)
	IgnorePattern string         // Skip paths matching this pattern
	RegexPattern  *regexp.Regexp // Match by regular expression

	// Time-based filtering
	OlderThan time.Duration // Files older than this duration
	NewerThan time.Duration // Files newer than this duration

	// Size-based filtering
	LargerSize  int64 // Files larger than this size (bytes)
	SmallerSize int64 // Files smaller than this size (bytes)

	// Metadata and tag filtering
	MatchMeta map[string]*regexp.Regexp // Metadata key-value patterns to match
	MatchTags map[string]*regexp.Regexp // Tag key-value patterns to match

	// Execution options
	ExecCmd     string // Command to execute for each match
	PrintFormat string // Format string for output

	// Traversal options
	MaxDepth       uint // Maximum directory depth to traverse
	FollowSymlinks bool // Whether to follow symbolic links
	IncludeHidden  bool // Whether to include hidden files
	WithVersions   bool // Whether to include file versions

	// Watch options
	Watch       bool     // Whether to watch for changes
	WatchEvents []string // Events to watch for (create, modify, delete)
}

// FindResult represents a file that matched the find criteria
type FindResult struct {
	Message FindMessage
	Error   error
}

// FindHandler is a function that processes each found file
type FindHandler func(ctx context.Context, result FindResult) error

// defaultFindHandler returns a default handler that prints found files
func defaultFindHandler() FindHandler {
	return func(ctx context.Context, result FindResult) error {
		if result.Error != nil {
			return result.Error
		}
		fmt.Println(result.Message.Path)
		return nil
	}
}

// execHandler returns a handler that executes a command for each found file
func execHandler(cmdTemplate string) FindHandler {
	return func(ctx context.Context, result FindResult) error {
		if result.Error != nil {
			return result.Error
		}

		// Replace placeholders in the command template
		cmd := formatCommand(cmdTemplate, result.Message)

		// Execute the command
		return executeCommand(ctx, cmd, result.Message)
	}
}

// formatHandler returns a handler that formats output according to a template
func formatHandler(formatTemplate string) FindHandler {
	return func(ctx context.Context, result FindResult) error {
		if result.Error != nil {
			return result.Error
		}

		// Format the output according to the template
		formatted := formatCommand(formatTemplate, result.Message)
		fmt.Println(formatted)
		return nil
	}
}

// formatCommand replaces placeholders in a template with values from the message
func formatCommand(template string, msg FindMessage) string {
	str := template

	// Replace basic placeholders
	str = strings.ReplaceAll(str, "{}", msg.Path)
	str = strings.ReplaceAll(str, "{base}", msg.Name)
	str = strings.ReplaceAll(str, "{dir}", msg.Dir)
	str = strings.ReplaceAll(str, "{size}", fmt.Sprintf("%d", msg.Size))
	str = strings.ReplaceAll(str, "{time}", msg.Time.Format(time.RFC3339))

	// Replace quoted versions
	str = strings.ReplaceAll(str, `{""}`, strconv.Quote(msg.Path))
	str = strings.ReplaceAll(str, `{"base"}`, strconv.Quote(msg.Name))
	str = strings.ReplaceAll(str, `{"dir"}`, strconv.Quote(msg.Dir))
	str = strings.ReplaceAll(str, `{"size"}`, strconv.Quote(fmt.Sprintf("%d", msg.Size)))
	str = strings.ReplaceAll(str, `{"time"}`, strconv.Quote(msg.Time.Format(time.RFC3339)))

	// Replace version if available
	if msg.VersionID != "" {
		str = strings.ReplaceAll(str, "{version}", msg.VersionID)
		str = strings.ReplaceAll(str, `{"version"}`, strconv.Quote(msg.VersionID))
	}

	return str
}

// executeCommand executes a command with the given arguments
func executeCommand(ctx context.Context, cmdStr string, msg FindMessage) error {
	// Split the command string into command and arguments
	args := strings.Fields(cmdStr)
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}

	// Create the command
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("command error: %s: %w", stderr.String(), err)
		}
		return err
	}

	// Print output if any
	if stdout.Len() > 0 {
		fmt.Print(stdout.String())
	}

	return nil
}

// nameMatch checks if a file name matches the given pattern
func nameMatch(pattern, path string) bool {
	matched, err := filepath.Match(pattern, filepath.Base(path))
	if err != nil {
		return false
	}
	if !matched {
		// Try matching against each path component
		for _, pathComponent := range strings.Split(path, string(os.PathSeparator)) {
			matched = pathComponent == pattern
			if matched {
				break
			}
		}
	}
	return matched
}

// pathMatch checks if a path matches the given pattern
func pathMatch(pattern, path string) bool {
	// Simple wildcard matching
	patternParts := strings.Split(pattern, "*")
	if len(patternParts) == 1 {
		return pattern == path
	}

	if !strings.HasPrefix(path, patternParts[0]) {
		return false
	}

	path = path[len(patternParts[0]):]
	for i := 1; i < len(patternParts)-1; i++ {
		idx := strings.Index(path, patternParts[i])
		if idx == -1 {
			return false
		}
		path = path[idx+len(patternParts[i]):]
	}

	return strings.HasSuffix(path, patternParts[len(patternParts)-1])
}

// matchFind checks if a file matches the find criteria
func matchFind(opts FindOptions, msg FindMessage) bool {
	match := true

	// Check name pattern
	if match && opts.NamePattern != "" {
		match = nameMatch(opts.NamePattern, msg.Path)
	}

	// Check path pattern
	if match && opts.PathPattern != "" {
		match = pathMatch(opts.PathPattern, msg.Path)
	}

	// Check ignore pattern
	if match && opts.IgnorePattern != "" {
		match = !pathMatch(opts.IgnorePattern, msg.Path)
	}

	// Check regex pattern
	if match && opts.RegexPattern != nil {
		match = opts.RegexPattern.MatchString(msg.Path)
	}

	// Check time constraints
	if match && opts.OlderThan > 0 {
		match = time.Since(msg.Time) > opts.OlderThan
	}

	if match && opts.NewerThan > 0 {
		match = time.Since(msg.Time) < opts.NewerThan
	}

	// Check size constraints
	if match && opts.LargerSize > 0 {
		match = msg.Size > opts.LargerSize
	}

	if match && opts.SmallerSize > 0 {
		match = msg.Size < opts.SmallerSize
	}

	// Check metadata
	if match && len(opts.MatchMeta) > 0 {
		match = matchRegexMap(opts.MatchMeta, msg.Metadata)
	}

	// Check tags
	if match && len(opts.MatchTags) > 0 {
		match = matchRegexMap(opts.MatchTags, msg.Tags)
	}

	return match
}

// matchRegexMap checks if values in a map match the given regex patterns
func matchRegexMap(patterns map[string]*regexp.Regexp, values map[string]string) bool {
	for k, pattern := range patterns {
		if pattern == nil {
			// If pattern is nil, the key should not exist or have empty value
			if val, exists := values[k]; exists && val != "" {
				return false
			}
			continue
		}

		// Check if the key exists and matches the pattern
		val, exists := values[k]
		if !exists || !pattern.MatchString(norm.NFC.String(val)) {
			return false
		}
	}
	return true
}

// trimPathAtMaxDepth trims a path to the specified maximum depth
func trimPathAtMaxDepth(rootPath, path string, maxDepth uint) string {
	if maxDepth == 0 {
		return path
	}

	// Remove the root prefix
	relPath := strings.TrimPrefix(path, rootPath)
	if relPath == path {
		// If the prefix wasn't removed, try with path separator
		if !strings.HasSuffix(rootPath, string(os.PathSeparator)) {
			rootPath += string(os.PathSeparator)
		}
		relPath = strings.TrimPrefix(path, rootPath)
	}

	// Split the path into components
	pathComponents := strings.Split(relPath, string(os.PathSeparator))

	// Trim to max depth
	if len(pathComponents) > int(maxDepth) {
		pathComponents = pathComponents[:maxDepth]
	}

	// Reconstruct the path
	return filepath.Join(rootPath, filepath.Join(pathComponents...))
}

// Find searches for files matching the given criteria and processes them with the handler
func Find(ctx context.Context, root string, opts FindOptions, handler FindHandler) error {
	if handler == nil {
		handler = defaultFindHandler()
	}

	// Set up walk options
	walkOpts := WalkOptions{
		Filter: FilterOptions{
			ExcludeDir: []string{},
		},
		SymlinkHandling: SymlinkIgnore,
		ErrorHandling:   ErrorHandlingContinue,
	}

	// Configure symlink handling
	if opts.FollowSymlinks {
		walkOpts.SymlinkHandling = SymlinkFollow
	}

	// Set up a channel to receive watch events if watching is enabled
	var watchChan chan FindResult
	var watchWg sync.WaitGroup
	if opts.Watch {
		watchChan = make(chan FindResult, 100)
		watchWg.Add(1)

		go func() {
			defer watchWg.Done()
			for {
				select {
				case result, ok := <-watchChan:
					if !ok {
						return
					}
					if matchFind(opts, result.Message) {
						_ = handler(ctx, result)
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// TODO: Implement file system watcher
	}

	// Walk the file system
	err := WalkLimitWithOptions(ctx, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return handler(ctx, FindResult{
				Error: err,
			})
		}

		// Skip hidden files if not included
		if !opts.IncludeHidden && isHidden(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create the message
		msg := FindMessage{
			Path:     path,
			Name:     filepath.Base(path),
			Dir:      filepath.Dir(path),
			Size:     info.Size(),
			Time:     info.ModTime(),
			IsDir:    info.IsDir(),
			Metadata: make(map[string]string),
			Tags:     make(map[string]string),
		}

		// Apply max depth if specified
		if opts.MaxDepth > 0 && info.IsDir() {
			depth := uint(strings.Count(path, string(os.PathSeparator)) - strings.Count(root, string(os.PathSeparator)))
			if depth > opts.MaxDepth {
				return filepath.SkipDir
			}
		}

		// Skip directories if we're only interested in files
		if info.IsDir() {
			return nil
		}

		// Check if the file matches the criteria
		if matchFind(opts, msg) {
			return handler(ctx, FindResult{
				Message: msg,
			})
		}

		return nil
	}, walkOpts)

	// Close the watch channel if watching was enabled
	if opts.Watch {
		close(watchChan)
		watchWg.Wait()
	}

	return err
}

// isHidden checks if a file is hidden
func isHidden(path string) bool {
	name := filepath.Base(path)
	return strings.HasPrefix(name, ".")
}

// FindWithExec searches for files and executes a command for each match
func FindWithExec(ctx context.Context, root string, opts FindOptions, cmdTemplate string) error {
	opts.ExecCmd = cmdTemplate
	return Find(ctx, root, opts, execHandler(cmdTemplate))
}

// FindWithFormat searches for files and formats output according to a template
func FindWithFormat(ctx context.Context, root string, opts FindOptions, formatTemplate string) error {
	opts.PrintFormat = formatTemplate
	return Find(ctx, root, opts, formatHandler(formatTemplate))
}

// CompileRegexMap compiles a map of key-value regex patterns
func CompileRegexMap(patterns map[string]string) (map[string]*regexp.Regexp, error) {
	result := make(map[string]*regexp.Regexp, len(patterns))

	for k, v := range patterns {
		// Empty value means the key should not exist or be empty
		if v == "" {
			result[k] = nil
			continue
		}

		// Compile the regex
		re, err := regexp.Compile(norm.NFC.String(v))
		if err != nil {
			return nil, fmt.Errorf("invalid regex for key %s: %w", k, err)
		}

		result[k] = re
	}

	return result, nil
}
