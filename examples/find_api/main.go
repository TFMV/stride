package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	stride "github.com/TFMV/stride/walk"
)

func main() {
	fmt.Println("=== Find API Examples ===")

	// Get directory to search
	var rootDir string
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	} else {
		// Use current directory if none provided
		var err error
		rootDir, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			os.Exit(1)
		}
	}

	ctx := context.Background()

	// Example 1: Basic file search
	fmt.Println("\n--- Example 1: Basic file search ---")
	basicSearch(ctx, rootDir)

	// Example 2: Advanced filtering
	fmt.Println("\n--- Example 2: Advanced filtering ---")
	advancedFiltering(ctx, rootDir)

	// Example 3: Execute commands on found files
	fmt.Println("\n--- Example 3: Execute commands on found files ---")
	executeCommands(ctx, rootDir)

	// Example 4: Custom output formatting
	fmt.Println("\n--- Example 4: Custom output formatting ---")
	customFormatting(ctx, rootDir)

	// Example 5: Permission handling
	fmt.Println("\n--- Example 5: Permission handling ---")
	permissionHandling(ctx, rootDir)
}

// Basic file search example
func basicSearch(ctx context.Context, rootDir string) {
	// Create basic find options
	opts := stride.FindOptions{
		NamePattern: "*.go", // Find all Go files
	}

	// Find files and process them
	count := 0
	err := stride.Find(ctx, rootDir, opts, func(ctx context.Context, result stride.FindResult) error {
		if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
			return nil // Continue despite errors
		}
		fmt.Printf("Found Go file: %s\n", result.Message.Path)
		count++
		return nil
	})

	if err != nil {
		fmt.Printf("Error in basic search: %v\n", err)
	}
	fmt.Printf("Total Go files found: %d\n", count)
}

// Advanced filtering example
func advancedFiltering(ctx context.Context, rootDir string) {
	// Create advanced find options
	opts := stride.FindOptions{
		// Pattern matching options
		NamePattern:   "*.go",                          // Match by file name (supports wildcards)
		IgnorePattern: "*_test.go",                     // Skip test files
		RegexPattern:  regexp.MustCompile(`main\.go$`), // Match files ending with main.go

		// Time-based filtering (adjust these values as needed for your filesystem)
		OlderThan: 365 * 24 * time.Hour, // Files older than 1 year
		NewerThan: 1 * time.Hour,        // Files newer than 1 hour

		// Size-based filtering
		LargerSize:  100,         // Files larger than 100 bytes
		SmallerSize: 1024 * 1024, // Files smaller than 1MB

		// Traversal options
		MaxDepth:       3,    // Maximum directory depth to traverse
		FollowSymlinks: true, // Follow symbolic links
		IncludeHidden:  true, // Include hidden files
	}

	// Find files and process them
	count := 0
	err := stride.Find(ctx, rootDir, opts, func(ctx context.Context, result stride.FindResult) error {
		if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
			return nil // Continue despite errors
		}
		fmt.Printf("Found file matching advanced criteria: %s (Size: %d bytes, Modified: %s)\n",
			result.Message.Path,
			result.Message.Size,
			result.Message.Time.Format(time.RFC3339))
		count++
		return nil
	})

	if err != nil {
		fmt.Printf("Error in advanced filtering: %v\n", err)
	}
	fmt.Printf("Total files matching advanced criteria: %d\n", count)
}

// Execute commands on found files
func executeCommands(ctx context.Context, rootDir string) {
	// Create find options
	opts := stride.FindOptions{
		NamePattern: "*.go", // Find all Go files
		MaxDepth:    2,      // Limit depth to avoid too many results
	}

	// Execute a command for each found file
	// The command template supports placeholders:
	// {} - Full path to the file
	// {base} - Base name of the file
	// {dir} - Directory containing the file
	// {size} - Size in bytes
	// {time} - Modification time
	cmdTemplate := "echo 'Processing: {base} (Size: {size} bytes)'"

	fmt.Println("Executing command for each Go file:")
	err := stride.FindWithExec(ctx, rootDir, opts, cmdTemplate)
	if err != nil {
		fmt.Printf("Error executing commands: %v\n", err)
	}
}

// Custom output formatting
func customFormatting(ctx context.Context, rootDir string) {
	// Create find options
	opts := stride.FindOptions{
		NamePattern: "*.go", // Find all Go files
		MaxDepth:    2,      // Limit depth to avoid too many results
	}

	// Format the output using a template
	// The format template supports the same placeholders as the command template
	formatTemplate := "{base} ({size} bytes) in {dir}"

	fmt.Println("Custom formatted output for each Go file:")
	err := stride.FindWithFormat(ctx, rootDir, opts, formatTemplate)
	if err != nil {
		fmt.Printf("Error formatting output: %v\n", err)
	}
}

// Permission handling example
func permissionHandling(ctx context.Context, rootDir string) {
	// Create find options
	opts := stride.FindOptions{
		FollowSymlinks: true, // Follow symbolic links
		IncludeHidden:  true, // Include hidden files
		MaxDepth:       5,    // Go deeper to find potential permission issues
	}

	// Track permission errors
	permissionErrors := 0
	totalFiles := 0

	// Find files and handle permission errors
	err := stride.Find(ctx, rootDir, opts, func(ctx context.Context, result stride.FindResult) error {
		if result.Error != nil {
			if os.IsPermission(result.Error) || (result.Error.Error() != "" &&
				(strings.Contains(result.Error.Error(), "permission denied") ||
					strings.Contains(result.Error.Error(), "operation not permitted"))) {
				permissionErrors++
				fmt.Printf("Permission error: %v\n", result.Error)
			} else {
				fmt.Printf("Other error: %v\n", result.Error)
			}
			return nil // Continue despite errors
		}

		totalFiles++
		return nil
	})

	if err != nil {
		fmt.Printf("Error in permission handling example: %v\n", err)
	}

	fmt.Printf("Total files processed: %d\n", totalFiles)
	fmt.Printf("Permission errors encountered: %d\n", permissionErrors)

	// Demonstrate how to handle permission errors with different strategies
	fmt.Println("\nDifferent error handling strategies:")

	// 1. Skip on error
	fmt.Println("1. Skip on error strategy:")
	skipOpts := stride.WalkOptions{
		ErrorHandlingMode: stride.SkipOnError,
		SymlinkHandling:   stride.SymlinkFollow,
	}

	skipCount := 0
	err = stride.WalkWithOptions(rootDir, func(ctx context.Context, path string, info os.FileInfo) error {
		skipCount++
		return nil
	}, skipOpts)

	if err != nil {
		fmt.Printf("  Error with skip strategy: %v\n", err)
	}
	fmt.Printf("  Files processed with skip strategy: %d\n", skipCount)

	// 2. Continue on error
	fmt.Println("2. Continue on error strategy:")
	continueOpts := stride.WalkOptions{
		ErrorHandlingMode: stride.ContinueOnError,
		SymlinkHandling:   stride.SymlinkFollow,
	}

	continueCount := 0
	err = stride.WalkWithOptions(rootDir, func(ctx context.Context, path string, info os.FileInfo) error {
		continueCount++
		return nil
	}, continueOpts)

	if err != nil {
		fmt.Printf("  Error with continue strategy: %v\n", err)
	}
	fmt.Printf("  Files processed with continue strategy: %d\n", continueCount)
}
