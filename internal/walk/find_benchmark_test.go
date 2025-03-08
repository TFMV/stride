package stride

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// setupBenchmarkFiles creates a directory structure for benchmarking
func setupBenchmarkFiles(b *testing.B, fileCount, dirCount int) string {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "stride-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create directories
	dirs := make([]string, dirCount)
	dirs[0] = tmpDir
	for i := 1; i < dirCount; i++ {
		dirPath := filepath.Join(tmpDir, "dir"+string(rune('A'+i-1)))
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		dirs[i] = dirPath
	}

	// Create files
	for i := 0; i < fileCount; i++ {
		// Distribute files across directories
		dirIndex := i % dirCount
		dir := dirs[dirIndex]

		// Create different file types
		var ext string
		switch i % 4 {
		case 0:
			ext = ".txt"
		case 1:
			ext = ".go"
		case 2:
			ext = ".log"
		case 3:
			ext = ".md"
		}

		// Create file with some content
		filePath := filepath.Join(dir, "file"+string(rune('0'+i%10))+ext)
		content := make([]byte, 1024*(i%10+1)) // Files of different sizes
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			b.Fatalf("Failed to create file: %v", err)
		}

		// Set different modification times
		modTime := time.Now().Add(-time.Duration(i%30) * 24 * time.Hour)
		if err := os.Chtimes(filePath, modTime, modTime); err != nil {
			b.Fatalf("Failed to set file time: %v", err)
		}
	}

	return tmpDir
}

// cleanupBenchmarkFiles removes the temporary directory
func cleanupBenchmarkFiles(tmpDir string) {
	os.RemoveAll(tmpDir)
}

// BenchmarkFindBasic measures the performance of basic find operations
func BenchmarkFindBasic(b *testing.B) {
	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int
		err := Find(context.Background(), tmpDir, FindOptions{}, func(ctx context.Context, result FindResult) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}

		// Sanity check
		if count == 0 {
			b.Fatalf("No files found")
		}
	}
}

// BenchmarkFindWithNamePattern measures the performance of find with name pattern
func BenchmarkFindWithNamePattern(b *testing.B) {
	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int
		err := Find(context.Background(), tmpDir, FindOptions{
			NamePattern: "*.txt",
		}, func(ctx context.Context, result FindResult) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}

		// Sanity check
		if count == 0 {
			b.Fatalf("No files found")
		}
	}
}

// BenchmarkFindWithRegex measures the performance of find with regex pattern
func BenchmarkFindWithRegex(b *testing.B) {
	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Compile regex
	regex := regexp.MustCompile(`.*\.go$`)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int
		err := Find(context.Background(), tmpDir, FindOptions{
			RegexPattern: regex,
		}, func(ctx context.Context, result FindResult) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}

		// Sanity check
		if count == 0 {
			b.Fatalf("No files found")
		}
	}
}

// BenchmarkFindWithTimeFilter measures the performance of find with time filter
func BenchmarkFindWithTimeFilter(b *testing.B) {
	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int
		err := Find(context.Background(), tmpDir, FindOptions{
			OlderThan: 15 * 24 * time.Hour,
		}, func(ctx context.Context, result FindResult) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}

		// Sanity check
		if count == 0 {
			b.Fatalf("No files found")
		}
	}
}

// BenchmarkFindWithSizeFilter measures the performance of find with size filter
func BenchmarkFindWithSizeFilter(b *testing.B) {
	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int
		err := Find(context.Background(), tmpDir, FindOptions{
			LargerSize: 5 * 1024, // Files larger than 5KB
		}, func(ctx context.Context, result FindResult) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}

		// Sanity check
		if count == 0 {
			b.Fatalf("No files found")
		}
	}
}

// BenchmarkFindWithCombinedFilters measures the performance of find with multiple filters
func BenchmarkFindWithCombinedFilters(b *testing.B) {
	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int
		err := Find(context.Background(), tmpDir, FindOptions{
			NamePattern: "*.txt",
			LargerSize:  3 * 1024,
			OlderThan:   10 * 24 * time.Hour,
		}, func(ctx context.Context, result FindResult) error {
			count++
			return nil
		})
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}

		// We don't check count here as the combined filters might not match any files
	}
}

// BenchmarkFindWithExec measures the performance of FindWithExec
func BenchmarkFindWithExec(b *testing.B) {
	// Setup
	fileCount := 100 // Fewer files for exec benchmark
	dirCount := 5
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Use a no-op command to minimize external factors
		err := FindWithExec(context.Background(), tmpDir, FindOptions{
			NamePattern: "*.txt",
		}, "true")
		if err != nil {
			b.Fatalf("FindWithExec failed: %v", err)
		}
	}
}

// BenchmarkFindWithFormat measures the performance of FindWithFormat
func BenchmarkFindWithFormat(b *testing.B) {
	// This benchmark is tricky because FindWithFormat prints to stdout
	// We'll use a custom handler instead to measure just the formatting overhead

	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Reset timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int
		err := Find(context.Background(), tmpDir, FindOptions{
			NamePattern: "*.txt",
		}, func(ctx context.Context, result FindResult) error {
			// Simulate the formatting overhead
			_ = formatCommand("{base} ({size} bytes)", result.Message)
			count++
			return nil
		})
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}

		// Sanity check
		if count == 0 {
			b.Fatalf("No files found")
		}
	}
}

// BenchmarkPathMatch measures the performance of the pathMatch function
func BenchmarkPathMatch(b *testing.B) {
	paths := []string{
		"/home/user/documents/file.txt",
		"/home/user/downloads/archive.zip",
		"/var/log/system.log",
		"/etc/config/settings.json",
		"/usr/local/bin/executable",
	}

	patterns := []string{
		"*/documents/*",
		"*/log/*",
		"*/bin/*",
		"*.txt",
		"*.log",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Test each path against each pattern
		for _, path := range paths {
			for _, pattern := range patterns {
				_ = pathMatch(pattern, path)
			}
		}
	}
}

// BenchmarkNameMatch measures the performance of the nameMatch function
func BenchmarkNameMatch(b *testing.B) {
	paths := []string{
		"/home/user/documents/file.txt",
		"/home/user/downloads/archive.zip",
		"/var/log/system.log",
		"/etc/config/settings.json",
		"/usr/local/bin/executable",
	}

	patterns := []string{
		"file.txt",
		"archive.zip",
		"system.log",
		"settings.json",
		"executable",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Test each path against each pattern
		for _, path := range paths {
			for _, pattern := range patterns {
				_ = nameMatch(pattern, path)
			}
		}
	}
}

// BenchmarkMatchRegexMap measures the performance of the matchRegexMap function
func BenchmarkMatchRegexMap(b *testing.B) {
	// Setup regex patterns
	patterns := map[string]*regexp.Regexp{
		"key1": regexp.MustCompile("value.*"),
		"key2": regexp.MustCompile("[0-9]+"),
		"key3": nil,
		"key4": regexp.MustCompile("^test.*"),
		"key5": regexp.MustCompile(".*end$"),
	}

	// Setup values to match against
	values := map[string]string{
		"key1": "value123",
		"key2": "12345",
		"key3": "",
		"key4": "test-string",
		"key5": "string-end",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = matchRegexMap(patterns, values)
	}
}

// BenchmarkCompileRegexMap measures the performance of the CompileRegexMap function
func BenchmarkCompileRegexMap(b *testing.B) {
	// Setup patterns to compile
	patterns := map[string]string{
		"key1": "value.*",
		"key2": "[0-9]+",
		"key3": "",
		"key4": "^test.*",
		"key5": ".*end$",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := CompileRegexMap(patterns)
		if err != nil {
			b.Fatalf("CompileRegexMap failed: %v", err)
		}
	}
}
