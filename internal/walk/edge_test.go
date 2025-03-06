package stride

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEmptyDirectory tests walking an empty directory
func TestEmptyDirectory(t *testing.T) {
	// Create a temporary empty directory
	tempDir := t.TempDir()

	var count int
	err := Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// We expect only 1 entry (the directory itself)
	if count != 1 {
		t.Errorf("Expected 1 entry, got %d", count)
	}
}

// TestNonExistentDirectory tests walking a non-existent directory
func TestNonExistentDirectory(t *testing.T) {
	err := Walk("/path/that/does/not/exist", func(path string, info os.FileInfo, err error) error {
		return nil
	})

	if err == nil {
		t.Errorf("Expected error for non-existent directory, got nil")
	}
}

// TestWalkWithError tests error handling in the walk function
func TestWalkWithError(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Custom error for testing
	customErr := errors.New("custom error")

	err := Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return customErr
		}
		return nil
	})

	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	if !errors.Is(err, customErr) {
		t.Errorf("Expected custom error, got %v", err)
	}
}

// TestSkipDir tests that filepath.SkipDir is honored
func TestSkipDir(t *testing.T) {
	// Create a test directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a file in the subdirectory
	testFile := filepath.Join(subDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var paths []string
	err := Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		paths = append(paths, path)
		if info.IsDir() && path != tempDir {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// We expect only 2 entries (the root dir and the subdir)
	if len(paths) != 2 {
		t.Errorf("Expected 2 entries, got %d: %v", len(paths), paths)
	}

	// The file in the subdirectory should not be visited
	for _, path := range paths {
		if path == testFile {
			t.Errorf("File in skipped directory was visited: %s", testFile)
		}
	}
}

// TestConcurrentModification tests walking a directory that's being modified concurrently
func TestConcurrentModification(t *testing.T) {
	tempDir := t.TempDir()

	// Start a goroutine that creates and deletes files
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		counter := 0
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// Create a new file
				fileName := filepath.Join(tempDir, "concurrent_file.txt")
				_ = os.WriteFile(fileName, []byte("test"), 0644)

				// Delete it after a short delay
				time.Sleep(5 * time.Millisecond)
				_ = os.Remove(fileName)

				counter++
				if counter > 10 {
					// Create a new directory occasionally
					dirName := filepath.Join(tempDir, "concurrent_dir")
					_ = os.MkdirAll(dirName, 0755)

					// Delete it after a short delay
					time.Sleep(5 * time.Millisecond)
					_ = os.RemoveAll(dirName)

					counter = 0
				}
			}
		}
	}()

	// Give the goroutine time to start creating files
	time.Sleep(50 * time.Millisecond)

	// Walk the directory while it's being modified
	ctx := context.Background()
	err := WalkLimit(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
		// If we get an error (e.g., file not found), just continue
		if err != nil {
			return nil
		}
		return nil
	}, 4)

	// Stop the file creation goroutine
	close(done)

	// The walk might succeed or fail depending on timing, but it shouldn't panic
	if err != nil {
		t.Logf("Walk returned error (expected in some cases): %v", err)
	}
}

// TestLongPaths tests walking directories with long paths
func TestLongPaths(t *testing.T) {
	// Skip on platforms where this test might fail due to path length limitations
	if testing.Short() {
		t.Skip("Skipping long path test in short mode")
	}

	tempDir := t.TempDir()

	// Create a deeply nested directory structure
	currentDir := tempDir
	for i := 0; i < 15; i++ { // This should create a reasonably long path
		currentDir = filepath.Join(currentDir, "subdir")
		if err := os.MkdirAll(currentDir, 0755); err != nil {
			t.Fatalf("Failed to create deep directory: %v", err)
		}
	}

	// Create a file at the deepest level
	deepFile := filepath.Join(currentDir, "deep_file.txt")
	if err := os.WriteFile(deepFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create deep file: %v", err)
	}

	var deepestPath string
	err := Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			deepestPath = path
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if deepestPath != deepFile {
		t.Errorf("Expected to find deepest file at %s, got %s", deepFile, deepestPath)
	}
}

// TestHiddenFiles tests that hidden files are processed correctly
func TestHiddenFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create a hidden file (works on Unix-like systems)
	hiddenFile := filepath.Join(tempDir, ".hidden")
	if err := os.WriteFile(hiddenFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	var foundHidden bool
	err := Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if path == hiddenFile {
			foundHidden = true
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if !foundHidden {
		t.Errorf("Hidden file was not found during walk")
	}
}

// TestCancelledContext tests that a cancelled context stops the walk
func TestCancelledContext(t *testing.T) {
	// Create a directory with many files to ensure the walk takes some time
	tempDir := t.TempDir()
	for i := 0; i < 100; i++ {
		filePath := filepath.Join(tempDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The walk should return quickly with a context.Canceled error
	start := time.Now()
	err := WalkLimit(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
		// Simulate slow processing
		time.Sleep(10 * time.Millisecond)
		return nil
	}, 4)

	elapsed := time.Since(start)

	if err == nil || (err != context.Canceled && err.Error() != "context canceled") {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// The walk should return quickly, not process all files
	if elapsed > 500*time.Millisecond {
		t.Errorf("Walk took too long to cancel: %v", elapsed)
	}
}

// TestMemoryLimit tests the memory limit functionality
func TestMemoryLimit(t *testing.T) {
	tempDir := t.TempDir()

	// Create some test files
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tempDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	ctx := context.Background()
	opts := WalkOptions{
		BufferSize: 4,
		MemoryLimit: MemoryLimit{
			SoftLimit: 1024, // 1KB soft limit
			HardLimit: 2048, // 2KB hard limit
		},
	}

	// The walk should complete without errors, even with low memory limits
	err := WalkLimitWithOptions(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
		return nil
	}, opts)

	if err != nil {
		t.Errorf("Expected no error with memory limits, got %v", err)
	}
}

// TestTimeFilter tests the time-based filtering
func TestTimeFilter(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file
	oldFile := filepath.Join(tempDir, "old.txt")
	if err := os.WriteFile(oldFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create old file: %v", err)
	}

	// Set the modification time to the past
	oldTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set file time: %v", err)
	}

	// Create a new file
	newFile := filepath.Join(tempDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Test filtering by ModifiedAfter
	ctx := context.Background()
	filter := FilterOptions{
		ModifiedAfter: time.Now().Add(-1 * time.Hour),
	}

	var newFiles []string
	err := WalkLimitWithFilter(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			newFiles = append(newFiles, path)
		}
		return nil
	}, 4, filter)

	if err != nil {
		t.Fatalf("WalkLimitWithFilter failed: %v", err)
	}

	// We should only find the new file
	if len(newFiles) != 1 || newFiles[0] != newFile {
		t.Errorf("Expected only new file, got: %v", newFiles)
	}

	// Test filtering by ModifiedBefore
	filter = FilterOptions{
		ModifiedBefore: time.Now().Add(-1 * time.Hour),
	}

	var oldFiles []string
	err = WalkLimitWithFilter(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			oldFiles = append(oldFiles, path)
		}
		return nil
	}, 4, filter)

	if err != nil {
		t.Fatalf("WalkLimitWithFilter failed: %v", err)
	}

	// We should only find the old file
	if len(oldFiles) != 1 || oldFiles[0] != oldFile {
		t.Errorf("Expected only old file, got: %v", oldFiles)
	}
}
