package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/TFMV/stride/walk"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <directory>")
		os.Exit(1)
	}

	rootDir := os.Args[1]
	ctx := context.Background()

	// Example 1: Basic file search
	fmt.Println("\n=== Example 1: Basic file search ===")
	basicSearch(ctx, rootDir)

	// Example 2: Advanced filtering
	fmt.Println("\n=== Example 2: Advanced filtering ===")
	advancedFiltering(ctx, rootDir)

	// Example 3: Execute commands on found files
	fmt.Println("\n=== Example 3: Execute commands on found files ===")
	executeCommands(ctx, rootDir)

	// Example 4: Custom output formatting
	fmt.Println("\n=== Example 4: Custom output formatting ===")
	customFormatting(ctx, rootDir)

	// Example 5: Using metadata and tags
	fmt.Println("\n=== Example 5: Using metadata and tags ===")
	metadataAndTags(ctx, rootDir)
}

// Basic file search example
func basicSearch(ctx context.Context, rootDir string) {
	// Create basic find options
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

	if err != nil {
		fmt.Printf("Error in basic search: %v\n", err)
	}
}

// Advanced filtering example
func advancedFiltering(ctx context.Context, rootDir string) {
	// Create advanced find options
	opts := walk.FindOptions{
		// Pattern matching options
		NamePattern:   "*.go",                          // Match by file name (supports wildcards)
		IgnorePattern: "*_test.go",                     // Skip test files
		RegexPattern:  regexp.MustCompile(`main\.go$`), // Match files ending with main.go

		// Time-based filtering
		OlderThan: 30 * 24 * time.Hour, // Files older than 30 days
		NewerThan: 1 * time.Hour,       // Files newer than 1 hour

		// Size-based filtering
		LargerSize:  1024,        // Files larger than 1KB
		SmallerSize: 1024 * 1024, // Files smaller than 1MB

		// Traversal options
		MaxDepth:       3,    // Maximum directory depth to traverse
		FollowSymlinks: true, // Follow symbolic links
		IncludeHidden:  true, // Include hidden files
	}

	// Find files and process them
	err := walk.Find(ctx, rootDir, opts, func(ctx context.Context, result walk.FindResult) error {
		if result.Error != nil {
			return result.Error
		}
		fmt.Printf("Found file matching advanced criteria: %s (Size: %d bytes, Modified: %s)\n",
			result.Message.Path,
			result.Message.Size,
			result.Message.Time.Format(time.RFC3339))
		return nil
	})

	if err != nil {
		fmt.Printf("Error in advanced filtering: %v\n", err)
	}
}

// Execute commands on found files
func executeCommands(ctx context.Context, rootDir string) {
	// Create find options
	opts := walk.FindOptions{
		NamePattern: "*.go", // Find all Go files
	}

	// Execute a command for each found file
	// The command template supports placeholders:
	// {} - Full path to the file
	// {base} - Base name of the file
	// {dir} - Directory containing the file
	// {size} - Size in bytes
	// {time} - Modification time
	cmdTemplate := "echo 'Processing: {base} (Size: {size} bytes)'"

	err := walk.FindWithExec(ctx, rootDir, opts, cmdTemplate)
	if err != nil {
		fmt.Printf("Error executing commands: %v\n", err)
	}
}

// Custom output formatting
func customFormatting(ctx context.Context, rootDir string) {
	// Create find options
	opts := walk.FindOptions{
		NamePattern: "*.go", // Find all Go files
	}

	// Format the output using a template
	// The format template supports the same placeholders as the command template
	formatTemplate := "{base} ({size} bytes) in {dir}"

	err := walk.FindWithFormat(ctx, rootDir, opts, formatTemplate)
	if err != nil {
		fmt.Printf("Error formatting output: %v\n", err)
	}
}

// Using metadata and tags
func metadataAndTags(ctx context.Context, rootDir string) {
	// Create metadata patterns
	metaPatterns := map[string]string{
		"author":  ".*",     // Match any author
		"version": "1\\..*", // Match version starting with 1.
	}

	// Create tag patterns
	tagPatterns := map[string]string{
		"status":   "active",
		"category": "example",
	}

	// Compile the regex maps
	metaRegex, err := walk.CompileRegexMap(metaPatterns)
	if err != nil {
		fmt.Printf("Error compiling metadata regex: %v\n", err)
		return
	}

	tagRegex, err := walk.CompileRegexMap(tagPatterns)
	if err != nil {
		fmt.Printf("Error compiling tag regex: %v\n", err)
		return
	}

	// Create find options
	opts := walk.FindOptions{
		NamePattern: "*.go",    // Find all Go files
		MatchMeta:   metaRegex, // Match files with specific metadata
		MatchTags:   tagRegex,  // Match files with specific tags
	}

	// Find files and process them
	err = walk.Find(ctx, rootDir, opts, func(ctx context.Context, result walk.FindResult) error {
		if result.Error != nil {
			return result.Error
		}

		fmt.Printf("Found file with matching metadata/tags: %s\n", result.Message.Path)

		// Print metadata
		if len(result.Message.Metadata) > 0 {
			fmt.Println("  Metadata:")
			for k, v := range result.Message.Metadata {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}

		// Print tags
		if len(result.Message.Tags) > 0 {
			fmt.Println("  Tags:")
			for k, v := range result.Message.Tags {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error in metadata/tags search: %v\n", err)
	}
}
