package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	stride "github.com/TFMV/stride/walk"
)

func main() {
	fmt.Println("=== Advanced Filtering Examples ===")

	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Example 1: File type filtering
	fmt.Println("\n--- File Type Filtering ---")
	filter1 := stride.FilterOptions{
		FileTypes: []string{"file"}, // Only regular files
	}
	walkWithFilter(dir, filter1, "Regular files only")

	// Example 2: Owner and group filtering
	fmt.Println("\n--- Owner and Group Filtering ---")
	currentUser := os.Getuid()
	filter2 := stride.FilterOptions{
		OwnerUID: currentUser, // Files owned by current user
	}
	walkWithFilter(dir, filter2, "Files owned by current user")

	// Example 3: Depth filtering
	fmt.Println("\n--- Depth Filtering ---")
	filter3 := stride.FilterOptions{
		MinDepth: 1, // Skip the root directory
		MaxDepth: 2, // Don't go deeper than 2 levels
	}
	walkWithFilter(dir, filter3, "Files at depth 1-2")

	// Example 4: Time-based filtering
	fmt.Println("\n--- Time-based Filtering ---")
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	filter4 := stride.FilterOptions{
		ModifiedAfter: oneDayAgo, // Files modified in the last 24 hours
	}
	walkWithFilter(dir, filter4, "Files modified in the last 24 hours")

	// Example 5: Empty files and directories
	fmt.Println("\n--- Empty Files and Directories ---")
	filter5 := stride.FilterOptions{
		IncludeEmptyFiles: true,
	}
	walkWithFilter(dir, filter5, "Empty files only")

	// Example 6: Combined filters
	fmt.Println("\n--- Combined Filters ---")
	filter6 := stride.FilterOptions{
		FileTypes:     []string{"file"},
		IncludeTypes:  []string{".go", ".md"},
		MinSize:       1024, // At least 1KB
		MaxDepth:      3,
		ModifiedAfter: time.Now().Add(-7 * 24 * time.Hour), // Last week
	}
	walkWithFilter(dir, filter6, "Go/MD files >1KB, modified in the last week, max depth 3")
}

func walkWithFilter(root string, filter stride.FilterOptions, description string) {
	ctx := context.Background()
	count := 0

	fmt.Printf("Filter: %s\n", description)
	err := stride.WalkLimitWithFilter(ctx, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(root, path)
			fmt.Printf("  %s (%d bytes, mode: %s)\n", relPath, info.Size(), info.Mode())
			count++
		}
		return nil
	}, 4, filter)

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}
	fmt.Printf("Total matching files: %d\n", count)
}
