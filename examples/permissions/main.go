package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	stride "github.com/TFMV/stride/internal/walk"
)

func main() {
	fmt.Println("=== Permission Filtering Examples ===")

	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Example 1: Filter files with exact permissions (0644)
	fmt.Println("\n--- Files with exact permissions (0644) ---")
	filter1 := stride.FilterOptions{
		ExactPermissions:    0644,
		UseExactPermissions: true,
	}
	walkWithFilter(dir, filter1)

	// Example 2: Filter files with minimum read permissions for all (0444)
	fmt.Println("\n--- Files with at least read permissions for all (0444) ---")
	filter2 := stride.FilterOptions{
		MinPermissions: 0444, // At least readable by everyone
	}
	walkWithFilter(dir, filter2)

	// Example 3: Filter files with maximum permissions (0755)
	fmt.Println("\n--- Files with maximum permissions of 0755 ---")
	filter3 := stride.FilterOptions{
		MaxPermissions: 0755, // No more permissions than rwxr-xr-x
	}
	walkWithFilter(dir, filter3)

	// Example 4: Combining permission filters with other filters
	fmt.Println("\n--- Go files with read permissions for all ---")
	filter4 := stride.FilterOptions{
		MinPermissions: 0444,
		IncludeTypes:   []string{".go"},
	}
	walkWithFilter(dir, filter4)
}

func walkWithFilter(root string, filter stride.FilterOptions) {
	ctx := context.Background()
	count := 0

	err := stride.WalkLimitWithFilter(ctx, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(root, path)
			fmt.Printf("%s (mode: %s)\n", relPath, info.Mode())
			count++
		}
		return nil
	}, 4, filter)

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
	}
	fmt.Printf("Total matching files: %d\n", count)
}
