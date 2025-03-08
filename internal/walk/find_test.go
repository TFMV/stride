package stride

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestFind(t *testing.T) {
	// Create a test directory structure
	tmpDir := t.TempDir()

	// Create some test files
	files := []struct {
		path string
		size int
		time time.Time
	}{
		{filepath.Join(tmpDir, "file1.txt"), 100, time.Now().Add(-48 * time.Hour)},
		{filepath.Join(tmpDir, "file2.txt"), 200, time.Now().Add(-24 * time.Hour)},
		{filepath.Join(tmpDir, "file3.log"), 300, time.Now().Add(-12 * time.Hour)},
		{filepath.Join(tmpDir, "file4.go"), 400, time.Now().Add(-1 * time.Hour)},
		{filepath.Join(tmpDir, "subdir", "file5.txt"), 500, time.Now()},
		{filepath.Join(tmpDir, "subdir", "file6.go"), 600, time.Now()},
		{filepath.Join(tmpDir, ".hidden.txt"), 700, time.Now()},
	}

	// Create the files
	for _, file := range files {
		dir := filepath.Dir(file.path)
		if dir != tmpDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
		}

		err := os.WriteFile(file.path, make([]byte, file.size), 0644)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = os.Chtimes(file.path, file.time, file.time)
		if err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}
	}

	// Test cases
	tests := []struct {
		name     string
		opts     FindOptions
		expected int
	}{
		{
			name:     "Find all files",
			opts:     FindOptions{},
			expected: 6, // Excludes hidden files by default
		},
		{
			name: "Find by name pattern",
			opts: FindOptions{
				NamePattern: "*.txt",
			},
			expected: 3, // file1.txt, file2.txt, subdir/file5.txt
		},
		{
			name: "Find by path pattern",
			opts: FindOptions{
				PathPattern: "*/subdir/*",
			},
			expected: 2, // subdir/file5.txt, subdir/file6.go
		},
		{
			name: "Find by regex pattern",
			opts: FindOptions{
				RegexPattern: regexp.MustCompile(`.*\.go$`),
			},
			expected: 2, // file4.go, subdir/file6.go
		},
		{
			name: "Find by older than",
			opts: FindOptions{
				OlderThan: 36 * time.Hour,
			},
			expected: 1, // file1.txt
		},
		{
			name: "Find by newer than",
			opts: FindOptions{
				NewerThan: 6 * time.Hour,
			},
			expected: 2, // subdir/file5.txt, subdir/file6.go
		},
		{
			name: "Find by larger size",
			opts: FindOptions{
				LargerSize: 350,
			},
			expected: 3, // file4.go, subdir/file5.txt, subdir/file6.go
		},
		{
			name: "Find by smaller size",
			opts: FindOptions{
				SmallerSize: 250,
			},
			expected: 2, // file1.txt, file2.txt
		},
		{
			name: "Find with max depth",
			opts: FindOptions{
				MaxDepth: 0, // Only files in the root directory
			},
			expected: 4, // file1.txt, file2.txt, file3.log, file4.go
		},
		{
			name: "Find with include hidden",
			opts: FindOptions{
				IncludeHidden: true,
			},
			expected: 7, // All files including .hidden.txt
		},
		{
			name: "Find with combined filters",
			opts: FindOptions{
				NamePattern: "*.txt",
				OlderThan:   30 * time.Hour,
			},
			expected: 1, // file1.txt
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var found int

			err := Find(context.Background(), tmpDir, test.opts, func(ctx context.Context, result FindResult) error {
				if result.Error != nil {
					return result.Error
				}
				found++
				return nil
			})

			if err != nil {
				t.Fatalf("Find failed: %v", err)
			}

			if found != test.expected {
				t.Errorf("Expected to find %d files, found %d", test.expected, found)
			}
		})
	}
}

func TestFindWithExec(t *testing.T) {
	// Create a test directory
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a temporary output file
	outputFile := filepath.Join(tmpDir, "output.txt")

	// Test FindWithExec
	opts := FindOptions{
		NamePattern: "*.txt",
	}

	// Use echo to write to the output file
	cmdTemplate := "echo {} > " + outputFile

	err = FindWithExec(context.Background(), tmpDir, opts, cmdTemplate)
	if err != nil {
		t.Fatalf("FindWithExec failed: %v", err)
	}

	// Check if the output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file was not created")
	}

	// Check the content of the output file
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expected := testFile + "\n"
	if string(content) != expected {
		t.Errorf("Expected output file to contain %q, got %q", expected, string(content))
	}
}

func TestFindWithFormat(t *testing.T) {
	// This test is more difficult to verify without capturing stdout
	// So we'll just check that it doesn't error

	// Create a test directory
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test FindWithFormat
	opts := FindOptions{
		NamePattern: "*.txt",
	}

	formatTemplate := "{base} ({size} bytes)"

	err = FindWithFormat(context.Background(), tmpDir, opts, formatTemplate)
	if err != nil {
		t.Fatalf("FindWithFormat failed: %v", err)
	}
}

func TestCompileRegexMap(t *testing.T) {
	patterns := map[string]string{
		"key1": "value.*",
		"key2": "[0-9]+",
		"key3": "",
	}

	regexMap, err := CompileRegexMap(patterns)
	if err != nil {
		t.Fatalf("CompileRegexMap failed: %v", err)
	}

	if len(regexMap) != 3 {
		t.Errorf("Expected 3 regex patterns, got %d", len(regexMap))
	}

	if regexMap["key1"] == nil {
		t.Errorf("Expected key1 to have a regex pattern")
	}

	if regexMap["key2"] == nil {
		t.Errorf("Expected key2 to have a regex pattern")
	}

	if regexMap["key3"] != nil {
		t.Errorf("Expected key3 to have a nil regex pattern")
	}

	// Test matching
	if !regexMap["key1"].MatchString("value123") {
		t.Errorf("Expected key1 pattern to match 'value123'")
	}

	if !regexMap["key2"].MatchString("12345") {
		t.Errorf("Expected key2 pattern to match '12345'")
	}
}

func TestMatchRegexMap(t *testing.T) {
	patterns := map[string]*regexp.Regexp{
		"key1": regexp.MustCompile("value.*"),
		"key2": regexp.MustCompile("[0-9]+"),
		"key3": nil,
	}

	// Test matching values
	values := map[string]string{
		"key1": "value123",
		"key2": "12345",
	}

	if !matchRegexMap(patterns, values) {
		t.Errorf("Expected patterns to match values")
	}

	// Test non-matching values
	values = map[string]string{
		"key1": "invalid",
		"key2": "12345",
	}

	if matchRegexMap(patterns, values) {
		t.Errorf("Expected patterns not to match values")
	}

	// Test nil pattern (key should not exist or be empty)
	values = map[string]string{
		"key1": "value123",
		"key2": "12345",
		"key3": "something",
	}

	if matchRegexMap(patterns, values) {
		t.Errorf("Expected patterns not to match values (key3 should not exist or be empty)")
	}

	values = map[string]string{
		"key1": "value123",
		"key2": "12345",
		"key3": "",
	}

	if !matchRegexMap(patterns, values) {
		t.Errorf("Expected patterns to match values (key3 is empty)")
	}
}
