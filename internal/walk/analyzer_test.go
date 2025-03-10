package stride

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzer(t *testing.T) {
	// Create a temporary test directory
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"file1.txt":          "Hello, World!",
		"file2.txt":          "Hello, World!", // Duplicate content
		"main.go":            "package main\n\n// Main function\nfunc main() {\n\t// Print hello\n\tprintln(\"hello\")\n}\n",
		"script.py":          "# Python script\n\ndef hello():\n    # Say hello\n    print('hello')\n",
		"config.json":        `{"api_key": "secret123", "password": "pass123"}`,
		"id_rsa":             "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----",
		"executable.exe":     "#!/bin/bash\necho 'hello'",
		"world-writable.txt": "This file is world-writable",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Make one file world-writable
	err := os.Chmod(filepath.Join(tmpDir, "world-writable.txt"), 0666)
	if err != nil {
		t.Fatalf("Failed to set file permissions: %v", err)
	}

	// Create a subdirectory with more files
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFiles := map[string]string{
		"sub1.txt": "Hello from subdirectory",
		"sub2.go":  "package sub\n\n// SubFunction does something\nfunc SubFunction() {}\n",
	}

	for name, content := range subFiles {
		path := filepath.Join(subDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create analyzer and enable all features
	analyzer := NewAnalyzer()
	analyzer.EnableDuplicateDetection()
	analyzer.EnableCodeStats()
	analyzer.EnableStorageReport()
	analyzer.EnableSecurityScan()
	analyzer.EnableContentPatternAnalysis()

	// Run analysis
	result, err := analyzer.Analyze(tmpDir)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	// Test duplicate detection
	t.Run("DuplicateDetection", func(t *testing.T) {
		foundDuplicate := false
		for _, paths := range result.Duplicates {
			if len(paths) > 1 {
				foundDuplicate = true
				// Check if our known duplicates are found
				file1Found := false
				file2Found := false
				for _, path := range paths {
					base := filepath.Base(path)
					if base == "file1.txt" {
						file1Found = true
					}
					if base == "file2.txt" {
						file2Found = true
					}
				}
				if !file1Found || !file2Found {
					t.Errorf("Expected to find file1.txt and file2.txt as duplicates")
				}
				break
			}
		}
		if !foundDuplicate {
			t.Error("No duplicates found when duplicates exist")
		}
	})

	// Test code statistics
	t.Run("CodeStatistics", func(t *testing.T) {
		// Check Go stats
		goStats := result.CodeStats["Go"]
		if goStats.Files != 2 {
			t.Errorf("Expected 2 Go files, got %d", goStats.Files)
		}
		if goStats.Comments < 2 {
			t.Errorf("Expected at least 2 comments in Go files, got %d", goStats.Comments)
		}

		// Check Python stats
		pyStats := result.CodeStats["Python"]
		if pyStats.Files != 1 {
			t.Errorf("Expected 1 Python file, got %d", pyStats.Files)
		}
		if pyStats.Comments < 2 {
			t.Errorf("Expected at least 2 comments in Python files, got %d", pyStats.Comments)
		}
	})

	// Test storage report
	t.Run("StorageReport", func(t *testing.T) {
		if result.StorageReport.FileCount < len(files)+len(subFiles) {
			t.Errorf("Expected at least %d files, got %d", len(files)+len(subFiles), result.StorageReport.FileCount)
		}
		if result.StorageReport.DirCount != 1 {
			t.Errorf("Expected 1 directory, got %d", result.StorageReport.DirCount)
		}
		if result.StorageReport.TotalSize == 0 {
			t.Error("Expected non-zero total size")
		}

		// Check file type statistics
		txtStats := result.StorageReport.TypeStats[".txt"]
		if txtStats.Count != 3 {
			t.Errorf("Expected 3 .txt files, got %d", txtStats.Count)
		}
		goStats := result.StorageReport.TypeStats[".go"]
		if goStats.Count != 2 {
			t.Errorf("Expected 2 .go files, got %d", goStats.Count)
		}
	})

	// Test security scanning
	t.Run("SecurityScan", func(t *testing.T) {
		foundWorldWritable := false
		foundExecutable := false

		for _, issue := range result.SecurityIssues {
			switch {
			case issue.Description == "File is world-writable":
				foundWorldWritable = true
			case issue.Description == "File has suspicious extension":
				foundExecutable = true
			}
		}

		if !foundWorldWritable {
			t.Error("Did not detect world-writable file")
		}
		if !foundExecutable {
			t.Error("Did not detect suspicious executable file")
		}
	})

	// Test pattern analysis
	t.Run("PatternAnalysis", func(t *testing.T) {
		// Check for API key pattern
		apiPattern := result.ContentPatterns["API Key"]
		if len(apiPattern.Files) == 0 {
			t.Error("Did not detect API key pattern")
		}

		// Check for password pattern
		passPattern := result.ContentPatterns["Password"]
		if len(passPattern.Files) == 0 {
			t.Error("Did not detect password pattern")
		}

		// Check for private key pattern
		keyPattern := result.ContentPatterns["Private Key"]
		if len(keyPattern.Files) == 0 {
			t.Error("Did not detect private key pattern")
		}
	})
}

func TestSizeParser(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"", 0},
		{"0", 0},
		{"1KB", 1024},
		{"1mb", 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"500", 500},
		{"2.5KB", 2 * 1024}, // Decimal part is ignored
	}

	for _, test := range tests {
		result := parseSize(test.input)
		if result != test.expected {
			t.Errorf("parseSize(%q) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestAnalyzerOptions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("MaxDepth", func(t *testing.T) {
		// Create nested directories
		depth1 := filepath.Join(tmpDir, "depth1")
		depth2 := filepath.Join(depth1, "depth2")
		os.MkdirAll(depth2, 0755)
		os.WriteFile(filepath.Join(depth2, "deep.txt"), []byte("deep"), 0644)

		analyzer := NewAnalyzer()
		analyzer.SetMaxDepth(1)
		analyzer.EnableStorageReport()

		result, err := analyzer.Analyze(tmpDir)
		if err != nil {
			t.Fatalf("Analysis failed: %v", err)
		}

		if result.StorageReport.FileCount > 1 {
			t.Errorf("Expected max 1 file due to depth limit, got %d", result.StorageReport.FileCount)
		}
	})

	t.Run("SizeRange", func(t *testing.T) {
		analyzer := NewAnalyzer()
		analyzer.SetSizeRange("5B", "15B")
		analyzer.EnableStorageReport()

		result, err := analyzer.Analyze(tmpDir)
		if err != nil {
			t.Fatalf("Analysis failed: %v", err)
		}

		if result.StorageReport.FileCount == 0 {
			t.Error("Expected to find files within size range")
		}
	})

	t.Run("HiddenFiles", func(t *testing.T) {
		// Create a hidden file
		hiddenFile := filepath.Join(tmpDir, ".hidden")
		err := os.WriteFile(hiddenFile, []byte("hidden"), 0644)
		if err != nil {
			t.Fatalf("Failed to create hidden file: %v", err)
		}

		// Test with hidden files excluded
		analyzer := NewAnalyzer()
		analyzer.EnableStorageReport()
		result, err := analyzer.Analyze(tmpDir)
		if err != nil {
			t.Fatalf("Analysis failed: %v", err)
		}

		initialCount := result.StorageReport.FileCount

		// Test with hidden files included
		analyzer = NewAnalyzer()
		analyzer.SetIncludeHidden(true)
		analyzer.EnableStorageReport()
		result, err = analyzer.Analyze(tmpDir)
		if err != nil {
			t.Fatalf("Analysis failed: %v", err)
		}

		if result.StorageReport.FileCount <= initialCount {
			t.Error("Expected to find more files when including hidden files")
		}
	})
}
