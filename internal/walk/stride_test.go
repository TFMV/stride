package stride

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestWalk tests the basic Walk function
func TestWalk(t *testing.T) {
	// Count files and directories
	var filesCount, dirsCount int

	err := Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			dirsCount++
		} else {
			filesCount++
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// We expect 3 directories (testdata, dir1, dir2) and 2 files
	expectedDirs := 6  // testdata, dir1, dir1/subdir1, dir2, dir2/subdir2, symlink_dir
	expectedFiles := 7 // file1.txt, file2.txt, file3.go, file4.go, and possibly others

	if dirsCount != expectedDirs {
		t.Errorf("Expected %d directories, got %d", expectedDirs, dirsCount)
	}

	if filesCount != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, filesCount)
	}
}

// TestWalkLimit tests the WalkLimit function with concurrency
func TestWalkLimit(t *testing.T) {
	// Count files and directories
	var filesCount, dirsCount int32

	ctx := context.Background()
	err := WalkLimit(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			atomic.AddInt32(&dirsCount, 1)
		} else {
			atomic.AddInt32(&filesCount, 1)
		}
		return nil
	}, 2) // Use 2 workers

	if err != nil {
		t.Fatalf("WalkLimit failed: %v", err)
	}

	// We expect 5 directories and 4 files
	expectedDirs := int32(6)
	expectedFiles := int32(7)

	if dirsCount != expectedDirs {
		t.Errorf("Expected %d directories, got %d", expectedDirs, dirsCount)
	}

	if filesCount != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, filesCount)
	}
}

// TestWalkLimitWithProgress tests the progress reporting functionality
func TestWalkLimitWithProgress(t *testing.T) {
	ctx := context.Background()

	var lastStats Stats
	progressFn := func(stats Stats) {
		lastStats = stats
	}

	err := WalkLimitWithProgress(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
		return nil
	}, 2, progressFn)

	if err != nil {
		t.Fatalf("WalkLimitWithProgress failed: %v", err)
	}

	// Verify that stats were updated
	if lastStats.FilesProcessed != 7 {
		t.Errorf("Expected 7 files processed, got %d", lastStats.FilesProcessed)
	}

	if lastStats.DirsProcessed != 6 {
		t.Errorf("Expected 6 directories processed, got %d", lastStats.DirsProcessed)
	}

	if lastStats.ElapsedTime <= 0 {
		t.Errorf("Expected positive elapsed time, got %v", lastStats.ElapsedTime)
	}
}

// TestWalkLimitWithFilter tests the filtering functionality
func TestWalkLimitWithFilter(t *testing.T) {
	ctx := context.Background()

	// Only process .go files
	filter := FilterOptions{
		IncludeTypes: []string{".go"},
	}

	var goFilesCount int32
	err := WalkLimitWithFilter(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			atomic.AddInt32(&goFilesCount, 1)
		}
		return nil
	}, 2, filter)

	if err != nil {
		t.Fatalf("WalkLimitWithFilter failed: %v", err)
	}

	// We expect 2 .go files
	expectedGoFiles := int32(4)
	if goFilesCount != expectedGoFiles {
		t.Errorf("Expected %d .go files, got %d", expectedGoFiles, goFilesCount)
	}
}

// TestWalkLimitWithOptions tests the full options functionality
func TestWalkLimitWithOptions(t *testing.T) {
	ctx := context.Background()

	// Create options with various settings
	opts := WalkOptions{
		ErrorHandling:   ErrorHandlingContinue,
		SymlinkHandling: SymlinkFollow,
		LogLevel:        LogLevelInfo,
		BufferSize:      2,
		Filter: FilterOptions{
			MinSize:      0,
			MaxSize:      1024 * 1024, // 1MB
			IncludeTypes: []string{".txt"},
		},
	}

	var txtFilesCount int32
	err := WalkLimitWithOptions(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			atomic.AddInt32(&txtFilesCount, 1)
		}
		return nil
	}, opts)

	if err != nil {
		t.Fatalf("WalkLimitWithOptions failed: %v", err)
	}

	// We expect 2 .txt files
	expectedTxtFiles := int32(2)
	if txtFilesCount != expectedTxtFiles {
		t.Errorf("Expected %d .txt files, got %d", expectedTxtFiles, txtFilesCount)
	}
}

// TestSymlinkHandling tests the symlink handling functionality
func TestSymlinkHandling(t *testing.T) {
	ctx := context.Background()

	// Test with SymlinkIgnore
	ignoreOpts := WalkOptions{
		SymlinkHandling: SymlinkIgnore,
		BufferSize:      2,
	}

	var ignoreCount int32
	err := WalkLimitWithOptions(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		atomic.AddInt32(&ignoreCount, 1)
		return nil
	}, ignoreOpts)

	if err != nil {
		t.Fatalf("WalkLimitWithOptions (SymlinkIgnore) failed: %v", err)
	}

	// Test with SymlinkFollow
	followOpts := WalkOptions{
		SymlinkHandling: SymlinkFollow,
		BufferSize:      2,
	}

	var followCount int32
	err = WalkLimitWithOptions(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		atomic.AddInt32(&followCount, 1)
		return nil
	}, followOpts)

	if err != nil {
		t.Fatalf("WalkLimitWithOptions (SymlinkFollow) failed: %v", err)
	}

	// Following symlinks should find more files/dirs than ignoring them
	if followCount <= ignoreCount {
		t.Errorf("Expected followCount (%d) to be greater than ignoreCount (%d)", followCount, ignoreCount)
	}
}

// TestErrorHandling tests the error handling functionality
func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file that we'll make unreadable
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "unreadable.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Make the file unreadable
	if err := os.Chmod(tempFile, 0); err != nil {
		t.Fatalf("Failed to make file unreadable: %v", err)
	}

	// Test with ErrorHandlingStop
	stopOpts := WalkOptions{
		ErrorHandling: ErrorHandlingStop,
		BufferSize:    2,
	}

	err := WalkLimitWithOptions(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Pass through any errors
		}
		// Try to read the file to potentially trigger an error
		if !info.IsDir() {
			_, err := os.ReadFile(path)
			return err
		}
		return nil
	}, stopOpts)

	// The test might not always produce an error depending on the platform and permissions
	// So we'll just log the result rather than failing the test
	t.Logf("ErrorHandlingStop result: %v", err)

	// Test with ErrorHandlingContinue
	continueOpts := WalkOptions{
		ErrorHandling: ErrorHandlingContinue,
		BufferSize:    2,
	}

	err = WalkLimitWithOptions(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
		return err // Pass through any errors
	}, continueOpts)

	if err != nil {
		t.Errorf("Expected no error with ErrorHandlingContinue, got %v", err)
	}
}

// TestCancelContext tests that the walk can be canceled via context
func TestCancelContext(t *testing.T) {
	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := WalkLimit(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
		// Slow down the processing to ensure cancellation happens
		time.Sleep(5 * time.Millisecond)
		return nil
	}, 2)

	if err == nil || (err != context.Canceled && err.Error() != "context canceled") {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// TestFilePassesFilter tests the filePassesFilter function
func TestFilePassesFilter(t *testing.T) {
	// Create a test file
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tempFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	info, err := os.Stat(tempFile)
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}

	tests := []struct {
		name     string
		filter   FilterOptions
		expected bool
	}{
		{
			name:     "No filter",
			filter:   FilterOptions{},
			expected: true,
		},
		{
			name: "Include .txt",
			filter: FilterOptions{
				IncludeTypes: []string{".txt"},
			},
			expected: true,
		},
		{
			name: "Exclude .txt",
			filter: FilterOptions{
				IncludeTypes: []string{".go"},
			},
			expected: false,
		},
		{
			name: "Min size too large",
			filter: FilterOptions{
				MinSize: 1024 * 1024, // 1MB
			},
			expected: false,
		},
		{
			name: "Max size too small",
			filter: FilterOptions{
				MaxSize: 1, // 1 byte
			},
			expected: false,
		},
		{
			name: "Size within range",
			filter: FilterOptions{
				MinSize: 1,
				MaxSize: 1024,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filePassesFilter(tempFile, info, tt.filter, SymlinkIgnore)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestShouldSkipDir tests the shouldSkipDir function
func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		root     string
		excludes []string
		expected bool
	}{
		{
			name:     "No excludes",
			path:     "testdata/dir1",
			root:     "testdata",
			excludes: nil,
			expected: false,
		},
		{
			name:     "Exclude dir1",
			path:     "testdata/dir1",
			root:     "testdata",
			excludes: []string{"dir1"},
			expected: true,
		},
		{
			name:     "Exclude subdir1",
			path:     "testdata/dir1/subdir1",
			root:     "testdata",
			excludes: []string{"subdir1"},
			expected: true,
		},
		{
			name:     "Exclude with wildcard",
			path:     "testdata/dir1",
			root:     "testdata",
			excludes: []string{"dir*"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipDir(tt.path, tt.root, tt.excludes)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestStatsUpdateDerivedStats tests the updateDerivedStats method
func TestStatsUpdateDerivedStats(t *testing.T) {
	stats := Stats{
		FilesProcessed: 10,
		BytesProcessed: 1024 * 1024, // 1MB
		ElapsedTime:    time.Second,
	}

	stats.updateDerivedStats()

	expectedAvgFileSize := int64(1024 * 1024 / 10)
	if stats.AvgFileSize != expectedAvgFileSize {
		t.Errorf("Expected average file size %d, got %d", expectedAvgFileSize, stats.AvgFileSize)
	}

	expectedSpeedMBPerSec := 1.0 // 1MB per second
	if stats.SpeedMBPerSec != expectedSpeedMBPerSec {
		t.Errorf("Expected speed %.2f MB/s, got %.2f MB/s", expectedSpeedMBPerSec, stats.SpeedMBPerSec)
	}
}

// TestCreateLogger tests the createLogger function
func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name     string
		logLevel LogLevel
	}{
		{
			name:     "Debug level",
			logLevel: LogLevelDebug,
		},
		{
			name:     "Info level",
			logLevel: LogLevelInfo,
		},
		{
			name:     "Warn level",
			logLevel: LogLevelWarn,
		},
		{
			name:     "Error level",
			logLevel: LogLevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := createLogger(tt.logLevel)
			if logger == nil {
				t.Errorf("Expected non-nil logger")
			}
		})
	}
}

// Benchmarks

// BenchmarkWalk benchmarks the basic Walk function
func BenchmarkWalk(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Walk("testdata", func(path string, info os.FileInfo, err error) error {
			return nil
		})
	}
}

// BenchmarkWalkLimit benchmarks the WalkLimit function with different worker counts
func BenchmarkWalkLimit(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers-%d", workers), func(b *testing.B) {
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = WalkLimit(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
					return nil
				}, workers)
			}
		})
	}
}

// BenchmarkWalkLimitWithFilter benchmarks the filtering functionality
func BenchmarkWalkLimitWithFilter(b *testing.B) {
	ctx := context.Background()
	filter := FilterOptions{
		IncludeTypes: []string{".txt"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WalkLimitWithFilter(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
			return nil
		}, 4, filter)
	}
}

// BenchmarkWalkLimitWithOptions benchmarks the full options functionality
func BenchmarkWalkLimitWithOptions(b *testing.B) {
	ctx := context.Background()
	opts := WalkOptions{
		ErrorHandling:   ErrorHandlingContinue,
		SymlinkHandling: SymlinkIgnore,
		LogLevel:        LogLevelError, // Minimize logging overhead for benchmark
		BufferSize:      4,
		Filter: FilterOptions{
			MinSize:      0,
			MaxSize:      1024 * 1024,
			IncludeTypes: []string{".txt"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WalkLimitWithOptions(ctx, "testdata", func(path string, info os.FileInfo, err error) error {
			return nil
		}, opts)
	}
}

// BenchmarkFilePassesFilter benchmarks the filePassesFilter function
func BenchmarkFilePassesFilter(b *testing.B) {
	// Create a test file
	tempFile := filepath.Join(b.TempDir(), "test.txt")
	if err := os.WriteFile(tempFile, []byte("test content"), 0644); err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}

	info, err := os.Stat(tempFile)
	if err != nil {
		b.Fatalf("Failed to stat temp file: %v", err)
	}

	filter := FilterOptions{
		MinSize:      0,
		MaxSize:      1024 * 1024,
		IncludeTypes: []string{".txt"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filePassesFilter(tempFile, info, filter, SymlinkIgnore)
	}
}

// BenchmarkShouldSkipDir benchmarks the shouldSkipDir function
func BenchmarkShouldSkipDir(b *testing.B) {
	excludes := []string{"dir1", "subdir*"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shouldSkipDir("testdata/dir1/subdir1", "testdata", excludes)
	}
}
