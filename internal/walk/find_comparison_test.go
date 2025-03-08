package stride

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// BenchmarkCompareFind compares our Find implementation with filepath.Walk
func BenchmarkCompareFind(b *testing.B) {
	// Setup
	fileCount := 1000
	dirCount := 10
	tmpDir := setupBenchmarkFiles(b, fileCount, dirCount)
	defer cleanupBenchmarkFiles(tmpDir)

	// Define test cases
	testCases := []struct {
		name string
		fn   func(b *testing.B, root string)
	}{
		{
			name: "Stride_Find_Basic",
			fn: func(b *testing.B, root string) {
				for i := 0; i < b.N; i++ {
					var count int
					err := Find(context.Background(), root, FindOptions{}, func(ctx context.Context, result FindResult) error {
						count++
						return nil
					})
					if err != nil {
						b.Fatalf("Find failed: %v", err)
					}
				}
			},
		},
		{
			name: "Filepath_Walk_Basic",
			fn: func(b *testing.B, root string) {
				for i := 0; i < b.N; i++ {
					var count int
					err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						count++
						return nil
					})
					if err != nil {
						b.Fatalf("Walk failed: %v", err)
					}
				}
			},
		},
		{
			name: "Stride_Find_WithNamePattern",
			fn: func(b *testing.B, root string) {
				for i := 0; i < b.N; i++ {
					var count int
					err := Find(context.Background(), root, FindOptions{
						NamePattern: "*.txt",
					}, func(ctx context.Context, result FindResult) error {
						count++
						return nil
					})
					if err != nil {
						b.Fatalf("Find failed: %v", err)
					}
				}
			},
		},
		{
			name: "Filepath_Walk_WithNameFiltering",
			fn: func(b *testing.B, root string) {
				for i := 0; i < b.N; i++ {
					var count int
					err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						// Manual name pattern matching
						matched, err := filepath.Match("*.txt", filepath.Base(path))
						if err != nil {
							return err
						}

						if matched {
							count++
						}
						return nil
					})
					if err != nil {
						b.Fatalf("Walk failed: %v", err)
					}
				}
			},
		},
		{
			name: "Stride_Find_WithRegex",
			fn: func(b *testing.B, root string) {
				regex := regexp.MustCompile(`.*\.go$`)

				for i := 0; i < b.N; i++ {
					var count int
					err := Find(context.Background(), root, FindOptions{
						RegexPattern: regex,
					}, func(ctx context.Context, result FindResult) error {
						count++
						return nil
					})
					if err != nil {
						b.Fatalf("Find failed: %v", err)
					}
				}
			},
		},
		{
			name: "Filepath_Walk_WithRegexFiltering",
			fn: func(b *testing.B, root string) {
				regex := regexp.MustCompile(`.*\.go$`)

				for i := 0; i < b.N; i++ {
					var count int
					err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						// Manual regex matching
						if regex.MatchString(path) {
							count++
						}
						return nil
					})
					if err != nil {
						b.Fatalf("Walk failed: %v", err)
					}
				}
			},
		},
		{
			name: "Stride_Find_WithTimeFilter",
			fn: func(b *testing.B, root string) {
				for i := 0; i < b.N; i++ {
					var count int
					err := Find(context.Background(), root, FindOptions{
						OlderThan: 15 * 24 * time.Hour,
					}, func(ctx context.Context, result FindResult) error {
						count++
						return nil
					})
					if err != nil {
						b.Fatalf("Find failed: %v", err)
					}
				}
			},
		},
		{
			name: "Filepath_Walk_WithTimeFiltering",
			fn: func(b *testing.B, root string) {
				cutoffTime := time.Now().Add(-15 * 24 * time.Hour)

				for i := 0; i < b.N; i++ {
					var count int
					err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						// Manual time filtering
						if info.ModTime().Before(cutoffTime) {
							count++
						}
						return nil
					})
					if err != nil {
						b.Fatalf("Walk failed: %v", err)
					}
				}
			},
		},
		{
			name: "Stride_Find_WithSizeFilter",
			fn: func(b *testing.B, root string) {
				for i := 0; i < b.N; i++ {
					var count int
					err := Find(context.Background(), root, FindOptions{
						LargerSize: 5 * 1024, // Files larger than 5KB
					}, func(ctx context.Context, result FindResult) error {
						count++
						return nil
					})
					if err != nil {
						b.Fatalf("Find failed: %v", err)
					}
				}
			},
		},
		{
			name: "Filepath_Walk_WithSizeFiltering",
			fn: func(b *testing.B, root string) {
				minSize := int64(5 * 1024) // 5KB

				for i := 0; i < b.N; i++ {
					var count int
					err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						// Manual size filtering
						if info.Size() > minSize {
							count++
						}
						return nil
					})
					if err != nil {
						b.Fatalf("Walk failed: %v", err)
					}
				}
			},
		},
		{
			name: "Stride_Find_WithCombinedFilters",
			fn: func(b *testing.B, root string) {
				for i := 0; i < b.N; i++ {
					var count int
					err := Find(context.Background(), root, FindOptions{
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
				}
			},
		},
		{
			name: "Filepath_Walk_WithCombinedFiltering",
			fn: func(b *testing.B, root string) {
				minSize := int64(3 * 1024) // 3KB
				cutoffTime := time.Now().Add(-10 * 24 * time.Hour)

				for i := 0; i < b.N; i++ {
					var count int
					err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						// Manual combined filtering
						matched, err := filepath.Match("*.txt", filepath.Base(path))
						if err != nil {
							return err
						}

						if matched && info.Size() > minSize && info.ModTime().Before(cutoffTime) {
							count++
						}
						return nil
					})
					if err != nil {
						b.Fatalf("Walk failed: %v", err)
					}
				}
			},
		},
	}

	// Run benchmarks
	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			tc.fn(b, tmpDir)
		})
	}
}
