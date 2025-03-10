package stride

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// DuplicateGroup represents a group of similar files
type DuplicateGroup struct {
	Files      []string // List of file paths
	Similarity float64  // Similarity score (0-1)
	Resolution string   // Suggested resolution action
	CommonPath string   // Common parent directory
}

// DependencyInfo represents dependency information for a file
type DependencyInfo struct {
	Path         string
	Imports      []string          // Direct imports
	ImportedBy   []string          // Files that import this file
	IsOrphan     bool              // True if not imported by any other file
	IsUnused     bool              // True if file contains unused exports
	Dependencies map[string]string // Map of symbol to file path
}

// CodebaseGraph represents the dependency structure of a codebase
type CodebaseGraph struct {
	Files       map[string]*DependencyInfo
	Orphans     []string
	UnusedFiles []string
}

// AdvancedAnalysis contains results from advanced analysis features
type AdvancedAnalysis struct {
	NearDuplicates []DuplicateGroup
	Dependencies   *CodebaseGraph
}

// detectNearDuplicates identifies files with similar content
func (a *Analyzer) detectNearDuplicates(files map[string][]byte) []DuplicateGroup {
	groups := make([]DuplicateGroup, 0)
	processed := make(map[string]bool)

	for path1, content1 := range files {
		if processed[path1] {
			continue
		}

		group := DuplicateGroup{
			Files: []string{path1},
		}

		for path2, content2 := range files {
			if path1 == path2 || processed[path2] {
				continue
			}

			similarity := calculateSimilarity(content1, content2)
			if similarity >= 0.8 { // 80% similarity threshold
				group.Files = append(group.Files, path2)
				group.Similarity = similarity
				processed[path2] = true
			}
		}

		if len(group.Files) > 1 {
			group.CommonPath = findCommonPath(group.Files)
			group.Resolution = suggestResolution(group)
			groups = append(groups, group)
		}
		processed[path1] = true
	}

	return groups
}

// calculateSimilarity computes similarity between two files using a rolling hash
func calculateSimilarity(content1, content2 []byte) float64 {
	// Implement a simplified rolling hash comparison
	// For better results, consider using a proper fuzzy hashing library
	const windowSize = 64
	matches := 0
	total := 0

	if len(content1) < windowSize || len(content2) < windowSize {
		return 0
	}

	hashes1 := rollingHash(content1, windowSize)
	hashes2 := rollingHash(content2, windowSize)

	for hash := range hashes1 {
		if hashes2[hash] {
			matches++
		}
		total++
	}

	if total == 0 {
		return 0
	}
	return float64(matches) / float64(total)
}

// rollingHash generates rolling hash values for content
func rollingHash(content []byte, windowSize int) map[uint64]bool {
	hashes := make(map[uint64]bool)
	if len(content) < windowSize {
		return hashes
	}

	// Simple rolling hash implementation
	var hash uint64
	for i := 0; i < len(content)-windowSize; i++ {
		if i == 0 {
			// Initial hash
			for j := 0; j < windowSize; j++ {
				hash = (hash << 1) + uint64(content[j])
			}
		} else {
			// Roll the window
			hash = (hash << 1) + uint64(content[i+windowSize-1]) - uint64(content[i-1])
		}
		hashes[hash] = true
	}

	return hashes
}

// analyzeDependencies analyzes code dependencies in a directory
func (a *Analyzer) analyzeDependencies(root string) (*CodebaseGraph, error) {
	graph := &CodebaseGraph{
		Files: make(map[string]*DependencyInfo),
	}

	// First pass: collect all Go files and their imports
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			imports, err := parseFileImports(path)
			if err != nil {
				return nil // Skip files with parse errors
			}

			graph.Files[relPath] = &DependencyInfo{
				Path:         relPath,
				Imports:      imports,
				ImportedBy:   make([]string, 0),
				Dependencies: make(map[string]string),
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: build the dependency graph
	for file, info := range graph.Files {
		for _, imp := range info.Imports {
			// Convert import path to relative path
			impFile := findMatchingFile(imp, graph.Files)
			if impFile != "" {
				graph.Files[impFile].ImportedBy = append(graph.Files[impFile].ImportedBy, file)
			}
		}
	}

	// Find orphans and unused files
	for file, info := range graph.Files {
		if len(info.ImportedBy) == 0 && !isEntryPoint(file) {
			graph.Orphans = append(graph.Orphans, file)
			info.IsOrphan = true
		}

		if len(info.Imports) == 0 && len(info.ImportedBy) == 0 {
			graph.UnusedFiles = append(graph.UnusedFiles, file)
			info.IsUnused = true
		}
	}

	return graph, nil
}

// parseFileImports extracts import statements from a Go file
func parseFileImports(path string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	imports := make([]string, 0)
	for _, imp := range node.Imports {
		// Remove quotes from import path
		importPath := strings.Trim(imp.Path.Value, "\"")
		imports = append(imports, importPath)
	}

	return imports, nil
}

// Helper functions

func findCommonPath(files []string) string {
	if len(files) == 0 {
		return ""
	}

	parts := strings.Split(files[0], string(os.PathSeparator))
	for i := 1; i < len(files); i++ {
		otherParts := strings.Split(files[i], string(os.PathSeparator))
		j := 0
		for j < len(parts) && j < len(otherParts) && parts[j] == otherParts[j] {
			j++
		}
		parts = parts[:j]
	}

	return filepath.Join(parts...)
}

func suggestResolution(group DuplicateGroup) string {
	if group.Similarity == 1.0 {
		return fmt.Sprintf("Delete duplicates and create symbolic links to %s", group.Files[0])
	}
	return fmt.Sprintf("Review files for possible consolidation (%.0f%% similar)", group.Similarity*100)
}

func findMatchingFile(importPath string, files map[string]*DependencyInfo) string {
	// Convert import path to possible file paths
	parts := strings.Split(importPath, "/")
	searchPath := parts[len(parts)-1]

	for file := range files {
		if strings.HasSuffix(file, searchPath+".go") {
			return file
		}
	}
	return ""
}

func isEntryPoint(file string) bool {
	return strings.HasSuffix(file, "main.go") || strings.Contains(file, "/cmd/") || strings.Contains(file, "/cli/")
}

// PerformDeduplication executes the suggested deduplication actions
func (a *Analyzer) PerformDeduplication(group DuplicateGroup, dryRun bool) error {
	if len(group.Files) < 2 {
		return fmt.Errorf("not enough files to deduplicate")
	}

	primaryFile := group.Files[0]
	for i := 1; i < len(group.Files); i++ {
		duplicateFile := group.Files[i]

		if dryRun {
			fmt.Printf("Would delete: %s and create symlink to %s\n", duplicateFile, primaryFile)
			continue
		}

		// Create backup
		backupPath := duplicateFile + ".bak"
		err := os.Rename(duplicateFile, backupPath)
		if err != nil {
			return fmt.Errorf("failed to create backup of %s: %v", duplicateFile, err)
		}

		// Create symlink
		err = os.Symlink(primaryFile, duplicateFile)
		if err != nil {
			// Restore from backup on failure
			os.Rename(backupPath, duplicateFile)
			return fmt.Errorf("failed to create symlink from %s to %s: %v", duplicateFile, primaryFile, err)
		}

		// Remove backup
		os.Remove(backupPath)
	}

	return nil
}
