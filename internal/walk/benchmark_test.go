package stride

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupLargeTestDir creates a larger test directory structure for benchmarking
func setupLargeTestDir(b *testing.B) string {
	// Create a temporary directory
	tempDir := b.TempDir()

	// Create a deeper directory structure
	for i := 0; i < 5; i++ {
		dirPath := filepath.Join(tempDir, fmt.Sprintf("dir%d", i))
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}

		// Create subdirectories
		for j := 0; j < 5; j++ {
			subdirPath := filepath.Join(dirPath, fmt.Sprintf("subdir%d", j))
			if err := os.MkdirAll(subdirPath, 0755); err != nil {
				b.Fatalf("Failed to create subdirectory: %v", err)
			}

			// Create files in subdirectories
			for k := 0; k < 10; k++ {
				// Create different file types
				extensions := []string{".txt", ".go", ".md", ".json", ".yaml"}
				for _, ext := range extensions {
					filePath := filepath.Join(subdirPath, fmt.Sprintf("file%d%s", k, ext))
					// Create files with different sizes
					size := (k + 1) * 1024 // 1KB to 10KB
					data := make([]byte, size)
					if err := os.WriteFile(filePath, data, 0644); err != nil {
						b.Fatalf("Failed to create file: %v", err)
					}
				}
			}
		}
	}

	// Create a symlink
	if err := os.Symlink(filepath.Join(tempDir, "dir0"), filepath.Join(tempDir, "symlink")); err != nil {
		b.Logf("Failed to create symlink (might be expected on some platforms): %v", err)
	}

	return tempDir
}

// BenchmarkLargeDirectoryWalk benchmarks walking a large directory structure
func BenchmarkLargeDirectoryWalk(b *testing.B) {
	tempDir := setupLargeTestDir(b)

	// Add standard library benchmark
	b.Run("filepath.Walk", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				return nil
			})
		}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			return nil
		})
	}
}

// BenchmarkLargeDirectoryWalkLimit benchmarks walking a large directory with different worker counts
func BenchmarkLargeDirectoryWalkLimit(b *testing.B) {
	tempDir := setupLargeTestDir(b)
	ctx := context.Background()

	workerCounts := []int{1, 2, 4, 8, 16, 32, 64}
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers-%d", workers), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = WalkLimit(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
					return nil
				}, workers)
			}
		})
	}
}

// BenchmarkLargeDirectoryWithFiltering benchmarks filtering in a large directory
func BenchmarkLargeDirectoryWithFiltering(b *testing.B) {
	tempDir := setupLargeTestDir(b)
	ctx := context.Background()

	filterTypes := []struct {
		name   string
		filter FilterOptions
	}{
		{
			name:   "NoFilter",
			filter: FilterOptions{},
		},
		{
			name: "ExtensionFilter",
			filter: FilterOptions{
				IncludeTypes: []string{".txt"},
			},
		},
		{
			name: "SizeFilter",
			filter: FilterOptions{
				MinSize: 5 * 1024,  // 5KB
				MaxSize: 10 * 1024, // 10KB
			},
		},
		{
			name: "CombinedFilter",
			filter: FilterOptions{
				MinSize:      5 * 1024,
				MaxSize:      10 * 1024,
				IncludeTypes: []string{".txt", ".go"},
			},
		},
	}

	for _, ft := range filterTypes {
		b.Run(ft.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = WalkLimitWithFilter(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
					return nil
				}, 8, ft.filter)
			}
		})
	}
}

// BenchmarkSymlinkHandling benchmarks different symlink handling strategies
func BenchmarkSymlinkHandling(b *testing.B) {
	tempDir := setupLargeTestDir(b)
	ctx := context.Background()

	symHandling := []struct {
		name     string
		handling SymlinkHandling
	}{
		{
			name:     "IgnoreSymlinks",
			handling: SymlinkIgnore,
		},
		{
			name:     "FollowSymlinks",
			handling: SymlinkFollow,
		},
	}

	for _, sh := range symHandling {
		b.Run(sh.name, func(b *testing.B) {
			opts := WalkOptions{
				SymlinkHandling: sh.handling,
				BufferSize:      8,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = WalkLimitWithOptions(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
					return nil
				}, opts)
			}
		})
	}
}

// BenchmarkWithProgress benchmarks the performance impact of progress reporting
func BenchmarkWithProgress(b *testing.B) {
	tempDir := setupLargeTestDir(b)
	ctx := context.Background()

	progressScenarios := []struct {
		name     string
		progress bool
	}{
		{
			name:     "WithoutProgress",
			progress: false,
		},
		{
			name:     "WithProgress",
			progress: true,
		},
	}

	for _, ps := range progressScenarios {
		b.Run(ps.name, func(b *testing.B) {
			var opts WalkOptions
			if ps.progress {
				opts = WalkOptions{
					BufferSize: 8,
					Progress: func(stats Stats) {
						// Do nothing in the benchmark
					},
				}
			} else {
				opts = WalkOptions{
					BufferSize: 8,
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = WalkLimitWithOptions(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
					return nil
				}, opts)
			}
		})
	}
}

// BenchmarkRealWorkload simulates a more realistic workload
func BenchmarkRealWorkload(b *testing.B) {
	tempDir := setupLargeTestDir(b)
	ctx := context.Background()

	// Define a more realistic workload that does some processing on each file
	workload := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Simulate some work on each file
			// For a benchmark, we'll just do a simple calculation based on file size
			size := info.Size()
			ext := filepath.Ext(path)

			// Different processing based on file type
			switch ext {
			case ".txt":
				// Simulate text processing (lighter)
				time.Sleep(time.Microsecond * time.Duration(size/1024))
			case ".go":
				// Simulate code analysis (medium)
				time.Sleep(time.Microsecond * time.Duration(size/512))
			case ".json", ".yaml":
				// Simulate parsing (heavier)
				time.Sleep(time.Microsecond * time.Duration(size/256))
			default:
				// Default processing
				time.Sleep(time.Microsecond)
			}
		}

		return nil
	}

	workerCounts := []int{1, 4, 16, 64}
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers-%d", workers), func(b *testing.B) {
			opts := WalkOptions{
				BufferSize: workers,
				Filter: FilterOptions{
					// A realistic filter that might be used
					MinSize:      1024, // Skip tiny files
					IncludeTypes: []string{".txt", ".go", ".json", ".yaml"},
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = WalkLimitWithOptions(ctx, tempDir, workload, opts)
			}
		})
	}
}

// BenchmarkComparisonWithStdLib compares Stride with the standard library's filepath.Walk
func BenchmarkComparisonWithStdLib(b *testing.B) {
	tempDir := setupLargeTestDir(b)
	ctx := context.Background()

	b.Run("filepath.Walk", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				return nil
			})
		}
	})

	b.Run("stride.Walk", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				return nil
			})
		}
	})

	b.Run("stride.WalkLimit-4", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = WalkLimit(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
				return nil
			}, 4)
		}
	})

	b.Run("stride.WalkLimit-16", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = WalkLimit(ctx, tempDir, func(path string, info os.FileInfo, err error) error {
				return nil
			}, 16)
		}
	})
}

// BenchmarkRealisticWorkload benchmarks a more realistic workload with file processing
func BenchmarkRealisticWorkload(b *testing.B) {
	tempDir := setupLargeTestDir(b)
	ctx := context.Background()

	// Define a realistic workload that does some processing on each file
	workload := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Simulate some CPU-bound work on each file
			// For a benchmark, we'll do a simple hash calculation
			data := make([]byte, 1024) // 1KB buffer
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			// Read and process some data
			_, err = f.Read(data)
			if err != nil && err != io.EOF {
				return err
			}

			// Do some CPU-bound work (calculate hash)
			h := sha256.New()
			h.Write(data)
			_ = h.Sum(nil)
		}

		return nil
	}

	b.Run("filepath.Walk", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filepath.Walk(tempDir, workload)
		}
	})

	b.Run("stride.Walk", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Walk(tempDir, workload)
		}
	})

	workerCounts := []int{1, 4, 16, 32}
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("stride.WalkLimit-%d", workers), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = WalkLimit(ctx, tempDir, workload, workers)
			}
		})
	}
}
