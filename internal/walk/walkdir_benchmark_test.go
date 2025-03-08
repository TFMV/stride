package stride

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkWalkDirComparison(b *testing.B) {
	// Create a temporary directory for testing
	tmpDir := b.TempDir()

	// Create a directory structure for testing
	createTestDirectoryStructure(b, tmpDir, 5, 10)

	b.ResetTimer()

	b.Run("filepath.WalkDir", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			count := 0
			err := filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				count++
				return nil
			})
			if err != nil {
				b.Fatalf("Error walking directory: %v", err)
			}
			if count == 0 {
				b.Fatal("No files found")
			}
		}
	})
}

// createTestDirectoryStructure creates a test directory structure with the specified depth and files per directory
func createTestDirectoryStructure(b *testing.B, root string, depth, filesPerDir int) {
	if depth <= 0 {
		return
	}

	// Create files in the current directory
	for i := 0; i < filesPerDir; i++ {
		filename := filepath.Join(root, "file"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create subdirectories
	for i := 0; i < 3; i++ {
		subdir := filepath.Join(root, "dir"+string(rune('a'+i)))
		if err := os.Mkdir(subdir, 0755); err != nil {
			b.Fatalf("Failed to create test directory: %v", err)
		}
		createTestDirectoryStructure(b, subdir, depth-1, filesPerDir)
	}
}
