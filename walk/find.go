// Package walk provides high-performance filesystem traversal with advanced filtering
package walk

import (
	"context"
	"regexp"
	"time"

	internal "github.com/TFMV/stride/internal/walk"
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

// convertToInternalFindMessage converts a public FindMessage to an internal one
func convertToInternalFindMessage(msg FindMessage) internal.FindMessage {
	return internal.FindMessage{
		Path:      msg.Path,
		Name:      msg.Name,
		Dir:       msg.Dir,
		Size:      msg.Size,
		Time:      msg.Time,
		IsDir:     msg.IsDir,
		Metadata:  msg.Metadata,
		Tags:      msg.Tags,
		VersionID: msg.VersionID,
	}
}

// convertFromInternalFindMessage converts an internal FindMessage to a public one
func convertFromInternalFindMessage(msg internal.FindMessage) FindMessage {
	return FindMessage{
		Path:      msg.Path,
		Name:      msg.Name,
		Dir:       msg.Dir,
		Size:      msg.Size,
		Time:      msg.Time,
		IsDir:     msg.IsDir,
		Metadata:  msg.Metadata,
		Tags:      msg.Tags,
		VersionID: msg.VersionID,
	}
}

// convertToInternalFindOptions converts public FindOptions to internal ones
func convertToInternalFindOptions(opts FindOptions) internal.FindOptions {
	return internal.FindOptions{
		NamePattern:    opts.NamePattern,
		PathPattern:    opts.PathPattern,
		IgnorePattern:  opts.IgnorePattern,
		RegexPattern:   opts.RegexPattern,
		OlderThan:      opts.OlderThan,
		NewerThan:      opts.NewerThan,
		LargerSize:     opts.LargerSize,
		SmallerSize:    opts.SmallerSize,
		MatchMeta:      opts.MatchMeta,
		MatchTags:      opts.MatchTags,
		ExecCmd:        opts.ExecCmd,
		PrintFormat:    opts.PrintFormat,
		MaxDepth:       opts.MaxDepth,
		FollowSymlinks: opts.FollowSymlinks,
		IncludeHidden:  opts.IncludeHidden,
		WithVersions:   opts.WithVersions,
		Watch:          opts.Watch,
		WatchEvents:    opts.WatchEvents,
	}
}

// convertToInternalFindHandler converts a public FindHandler to an internal one
func convertToInternalFindHandler(handler FindHandler) internal.FindHandler {
	if handler == nil {
		return nil
	}

	return func(ctx context.Context, result internal.FindResult) error {
		return handler(ctx, FindResult{
			Message: convertFromInternalFindMessage(result.Message),
			Error:   result.Error,
		})
	}
}

// Find searches for files matching the given criteria and processes them with the handler
func Find(ctx context.Context, root string, opts FindOptions, handler FindHandler) error {
	internalOpts := convertToInternalFindOptions(opts)
	internalHandler := convertToInternalFindHandler(handler)

	return internal.Find(ctx, root, internalOpts, internalHandler)
}

// FindWithExec searches for files and executes a command for each match
func FindWithExec(ctx context.Context, root string, opts FindOptions, cmdTemplate string) error {
	internalOpts := convertToInternalFindOptions(opts)
	return internal.FindWithExec(ctx, root, internalOpts, cmdTemplate)
}

// FindWithFormat searches for files and formats output according to a template
func FindWithFormat(ctx context.Context, root string, opts FindOptions, formatTemplate string) error {
	internalOpts := convertToInternalFindOptions(opts)
	return internal.FindWithFormat(ctx, root, internalOpts, formatTemplate)
}

// CompileRegexMap compiles a map of key-value regex patterns
func CompileRegexMap(patterns map[string]string) (map[string]*regexp.Regexp, error) {
	return internal.CompileRegexMap(patterns)
}

// NewFindOptions creates a new FindOptions with default values
func NewFindOptions() FindOptions {
	return FindOptions{
		FollowSymlinks: false,
		IncludeHidden:  false,
		WithVersions:   false,
		Watch:          false,
		WatchEvents:    []string{"create", "modify"},
	}
}
