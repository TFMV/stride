package stride

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNearDuplicateDetection(t *testing.T) {
	// Create a temporary test directory
	tmpDir := t.TempDir()

	// Create test files with similar content
	files := map[string]string{
		"original.txt":     "This is a test file with some content\nIt has multiple lines\nAnd some unique text",
		"similar1.txt":     "This is a test file with some content\nIt has multiple lines\nBut slightly different",
		"similar2.txt":     "This is a test file with some content\nWith a few changes\nAnd different text",
		"different.txt":    "This is a completely different file\nNo similarity here\nTotally unique content",
		"exact_copy.txt":   "This is a test file with some content\nIt has multiple lines\nAnd some unique text",
		"small_change.txt": "This is a test file with some content\nIt has multiple lines\nAnd some unique text!",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create analyzer and enable near-duplicate detection
	analyzer := NewAnalyzer()
	analyzer.EnableNearDuplicateDetection()

	// Run analysis
	result, err := analyzer.Analyze(tmpDir)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	if result.Advanced == nil {
		t.Fatal("Expected advanced analysis results")
	}

	// Check near-duplicate groups
	if len(result.Advanced.NearDuplicates) == 0 {
		t.Error("No near-duplicates found when they should exist")
	}

	// Find the group containing the original file
	var originalGroup *DuplicateGroup
	for i := range result.Advanced.NearDuplicates {
		group := &result.Advanced.NearDuplicates[i]
		for _, file := range group.Files {
			if filepath.Base(file) == "original.txt" {
				originalGroup = group
				break
			}
		}
		if originalGroup != nil {
			break
		}
	}

	if originalGroup == nil {
		t.Fatal("Could not find group containing original.txt")
	}

	// Check that exact copy is in the same group with 100% similarity
	exactCopyFound := false
	for _, file := range originalGroup.Files {
		if filepath.Base(file) == "exact_copy.txt" {
			exactCopyFound = true
			if originalGroup.Similarity != 1.0 {
				t.Errorf("Expected 100%% similarity for exact copy, got %.0f%%", originalGroup.Similarity*100)
			}
			break
		}
	}

	if !exactCopyFound {
		t.Error("Exact copy was not found in the same group as original")
	}
}

func TestDependencyAnalysis(t *testing.T) {
	// Create a temporary test directory
	tmpDir := t.TempDir()

	// Create test Go files with dependencies
	files := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"./utils"
)

func main() {
	fmt.Println(utils.Helper())
}`,
		"utils/helper.go": `package utils

func Helper() string {
	return "helper"
}`,
		"utils/unused.go": `package utils

func Unused() string {
	return "unused"
}`,
		"orphan.go": `package orphan

func OrphanFunc() {}`,
	}

	// Create the files
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if filepath.Dir(path) != tmpDir {
			err := os.MkdirAll(filepath.Dir(path), 0755)
			if err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
		}
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create analyzer and enable dependency analysis
	analyzer := NewAnalyzer()
	analyzer.EnableDependencyAnalysis()

	// Run analysis
	result, err := analyzer.Analyze(tmpDir)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	if result.Advanced == nil {
		t.Fatal("Expected advanced analysis results")
	}

	if result.Advanced.Dependencies == nil {
		t.Fatal("Expected dependency analysis results")
	}

	// Check for orphan files
	orphanFound := false
	for _, file := range result.Advanced.Dependencies.Orphans {
		if filepath.Base(file) == "orphan.go" {
			orphanFound = true
			break
		}
	}

	if !orphanFound {
		t.Error("Did not detect orphan.go as an orphan file")
	}

	// Check for unused files
	unusedFound := false
	for _, file := range result.Advanced.Dependencies.UnusedFiles {
		if filepath.Base(file) == "unused.go" {
			unusedFound = true
			break
		}
	}

	if !unusedFound {
		t.Error("Did not detect unused.go as an unused file")
	}

	// Check main.go dependencies
	mainInfo := result.Advanced.Dependencies.Files["main.go"]
	if mainInfo == nil {
		t.Fatal("No dependency info for main.go")
	}

	hasUtilsImport := false
	for _, imp := range mainInfo.Imports {
		if imp == "./utils" {
			hasUtilsImport = true
			break
		}
	}

	if !hasUtilsImport {
		t.Error("Did not detect utils import in main.go")
	}
}

func TestSimilarityCalculation(t *testing.T) {
	tests := []struct {
		name     string
		content1 string
		content2 string
		minScore float64
		maxScore float64
	}{
		{
			name:     "identical",
			content1: "This is a test file with some content",
			content2: "This is a test file with some content",
			minScore: 1.0,
			maxScore: 1.0,
		},
		{
			name:     "very similar",
			content1: "This is a test file with some content",
			content2: "This is a test file with some content!",
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name:     "somewhat similar",
			content1: "This is a test file with some content",
			content2: "This is a different file with other content",
			minScore: 0.5,
			maxScore: 0.9,
		},
		{
			name:     "different",
			content1: "This is a test file with some content",
			content2: "Completely different text here",
			minScore: 0.0,
			maxScore: 0.5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			score := calculateSimilarity([]byte(test.content1), []byte(test.content2))
			if score < test.minScore || score > test.maxScore {
				t.Errorf("Expected similarity score between %.2f and %.2f, got %.2f",
					test.minScore, test.maxScore, score)
			}
		})
	}
}
