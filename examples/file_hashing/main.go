// Package main demonstrates using Stride to compute file hashes efficiently.
package main

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	stride "github.com/TFMV/stride/walk"
	"go.uber.org/zap"
)

// HashResult stores the result of a file hash operation
type HashResult struct {
	Path     string
	Size     int64
	MD5      string
	SHA1     string
	SHA256   string
	Duration time.Duration
	Error    error
}

// HashOptions defines which hash algorithms to use
type HashOptions struct {
	MD5    bool
	SHA1   bool
	SHA256 bool
}

// Global variables to store results and provide synchronization
var (
	results      []HashResult
	resultsMutex sync.Mutex
	startTime    time.Time
	totalBytes   int64
	hashOpts     HashOptions
)

func main() {
	// Parse command line flags
	rootDir := flag.String("dir", ".", "Directory to scan")
	workerCount := flag.Int("workers", 4, "Number of concurrent workers")
	minSize := flag.Int64("min-size", 0, "Minimum file size in bytes")
	maxSize := flag.Int64("max-size", 1024*1024*100, "Maximum file size in bytes (default 100MB)")
	pattern := flag.String("pattern", "", "File pattern to match (e.g. *.go)")
	useMD5 := flag.Bool("md5", true, "Compute MD5 hash")
	useSHA1 := flag.Bool("sha1", false, "Compute SHA1 hash")
	useSHA256 := flag.Bool("sha256", true, "Compute SHA256 hash")
	outputFormat := flag.String("format", "text", "Output format (text, csv)")
	flag.Parse()

	// Override rootDir if provided as positional argument
	if flag.NArg() > 0 {
		*rootDir = flag.Arg(0)
	}

	// Set hash options
	hashOpts = HashOptions{
		MD5:    *useMD5,
		SHA1:   *useSHA1,
		SHA256: *useSHA256,
	}

	// Validate that at least one hash algorithm is selected
	if !hashOpts.MD5 && !hashOpts.SHA1 && !hashOpts.SHA256 {
		fmt.Println("Error: At least one hash algorithm must be selected")
		flag.Usage()
		os.Exit(1)
	}

	// Create logger
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, _ := config.Build()
	defer logger.Sync()

	// Get absolute path for nicer output
	absPath, _ := filepath.Abs(*rootDir)
	fmt.Printf("Computing hashes for files in: %s\n", absPath)
	fmt.Printf("Using algorithms: %s\n", selectedAlgorithms(hashOpts))

	// Initialize results slice
	results = make([]HashResult, 0, 100)
	startTime = time.Now()

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create options for the traversal
	opts := stride.WalkOptions{
		Context:     ctx,
		WorkerCount: *workerCount,
		Filter: stride.FilterOptions{
			MinSize:   *minSize,
			MaxSize:   *maxSize,
			Pattern:   *pattern,
			FileTypes: []string{"file"}, // Only process regular files
		},
		ProgressCallback: func(stats stride.Stats) {
			fmt.Printf("\rProcessed: %d files, %.2f MB at %.2f MB/s",
				stats.FilesProcessed,
				float64(stats.BytesProcessed)/(1024*1024),
				stats.SpeedMBPerSec,
			)
		},
		Logger: logger,
	}

	// Our processing function
	walkFn := func(ctx context.Context, path string, info os.FileInfo) error {
		// Skip directories (should be filtered out already, but just in case)
		if info.IsDir() {
			return nil
		}

		// Compute hashes
		result, err := computeFileHashes(path, info, hashOpts)
		if err != nil {
			return err
		}

		// Store result
		resultsMutex.Lock()
		results = append(results, result)
		resultsMutex.Unlock()

		return nil
	}

	// Start the traversal
	err := stride.WalkWithOptions(*rootDir, walkFn, opts)

	// Print final newline after progress updates
	fmt.Println()

	if err != nil {
		fmt.Printf("Error during traversal: %v\n", err)
		os.Exit(1)
	}

	// Sort results by path for consistent output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	// Output results
	switch strings.ToLower(*outputFormat) {
	case "csv":
		outputCSV(results, hashOpts)
	default:
		outputText(results, hashOpts)
	}

	// Print summary
	duration := time.Since(startTime)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Directory: %s\n", absPath)
	fmt.Printf("  Files processed: %d\n", len(results))
	fmt.Printf("  Total size: %.2f MB\n", float64(totalBytes)/(1024*1024))
	fmt.Printf("  Duration: %v\n", duration)

	if duration > 0 {
		mbPerSec := float64(totalBytes) / (1024 * 1024 * duration.Seconds())
		fmt.Printf("  Processing speed: %.2f MB/s\n", mbPerSec)
	}
}

// computeFileHashes calculates the requested hashes for a file
func computeFileHashes(path string, info os.FileInfo, opts HashOptions) (HashResult, error) {
	start := time.Now()

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return HashResult{
			Path:  path,
			Size:  info.Size(),
			Error: err,
		}, err
	}
	defer file.Close()

	// Initialize hash functions
	var md5Hash, sha1Hash, sha256Hash hash.Hash
	var hashWriters []io.Writer

	if opts.MD5 {
		md5Hash = md5.New()
		hashWriters = append(hashWriters, md5Hash)
	}
	if opts.SHA1 {
		sha1Hash = sha1.New()
		hashWriters = append(hashWriters, sha1Hash)
	}
	if opts.SHA256 {
		sha256Hash = sha256.New()
		hashWriters = append(hashWriters, sha256Hash)
	}

	// Create a multi-writer to write to all hash functions at once
	multiWriter := io.MultiWriter(hashWriters...)

	// Copy the file data to the hash functions
	bytesRead, err := io.Copy(multiWriter, file)
	if err != nil {
		return HashResult{
			Path:  path,
			Size:  info.Size(),
			Error: err,
		}, err
	}

	// Update total bytes processed
	resultsMutex.Lock()
	totalBytes += bytesRead
	resultsMutex.Unlock()

	// Create result
	result := HashResult{
		Path:     path,
		Size:     bytesRead,
		Duration: time.Since(start),
	}

	// Get hash values
	if opts.MD5 {
		result.MD5 = hex.EncodeToString(md5Hash.Sum(nil))
	}
	if opts.SHA1 {
		result.SHA1 = hex.EncodeToString(sha1Hash.Sum(nil))
	}
	if opts.SHA256 {
		result.SHA256 = hex.EncodeToString(sha256Hash.Sum(nil))
	}

	return result, nil
}

// outputText prints results in a human-readable format
func outputText(results []HashResult, opts HashOptions) {
	fmt.Println("\nResults:")
	fmt.Println(strings.Repeat("-", 80))

	// Print header
	fmt.Printf("%-40s %-10s", "File", "Size")
	if opts.MD5 {
		fmt.Printf(" %-32s", "MD5")
	}
	if opts.SHA1 {
		fmt.Printf(" %-40s", "SHA1")
	}
	if opts.SHA256 {
		fmt.Printf(" %-64s", "SHA256")
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 80))

	// Print each result
	for _, r := range results {
		// Truncate path if too long
		path := r.Path
		if len(path) > 40 {
			path = "..." + path[len(path)-37:]
		}

		fmt.Printf("%-40s %10d", path, r.Size)
		if opts.MD5 {
			fmt.Printf(" %s", r.MD5)
		}
		if opts.SHA1 {
			fmt.Printf(" %s", r.SHA1)
		}
		if opts.SHA256 {
			fmt.Printf(" %s", r.SHA256)
		}
		fmt.Println()
	}
}

// outputCSV prints results in CSV format
func outputCSV(results []HashResult, opts HashOptions) {
	// Print header
	fmt.Printf("File,Size")
	if opts.MD5 {
		fmt.Printf(",MD5")
	}
	if opts.SHA1 {
		fmt.Printf(",SHA1")
	}
	if opts.SHA256 {
		fmt.Printf(",SHA256")
	}
	fmt.Println()

	// Print each result
	for _, r := range results {
		fmt.Printf("\"%s\",%d", r.Path, r.Size)
		if opts.MD5 {
			fmt.Printf(",%s", r.MD5)
		}
		if opts.SHA1 {
			fmt.Printf(",%s", r.SHA1)
		}
		if opts.SHA256 {
			fmt.Printf(",%s", r.SHA256)
		}
		fmt.Println()
	}
}

// selectedAlgorithms returns a string listing the selected hash algorithms
func selectedAlgorithms(opts HashOptions) string {
	var algorithms []string
	if opts.MD5 {
		algorithms = append(algorithms, "MD5")
	}
	if opts.SHA1 {
		algorithms = append(algorithms, "SHA1")
	}
	if opts.SHA256 {
		algorithms = append(algorithms, "SHA256")
	}
	return strings.Join(algorithms, ", ")
}
