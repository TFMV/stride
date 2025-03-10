package stride

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AnalyzeResult represents the results of filesystem analysis
type AnalyzeResult struct {
	Duplicates      map[string][]string       // Map of content hash to file paths
	CodeStats       map[string]LanguageStats  // Map of language to stats
	StorageReport   StorageReport             // Storage usage information
	SecurityIssues  []SecurityIssue           // List of security issues found
	ContentPatterns map[string]ContentPattern // Map of pattern name to pattern info
	Advanced        *AdvancedAnalysis         // Results from advanced analysis
}

// LanguageStats holds statistics for a programming language
type LanguageStats struct {
	Files      int      // Number of files
	Lines      int      // Total lines of code
	Blanks     int      // Blank lines
	Comments   int      // Comment lines
	Size       int64    // Total size in bytes
	Extensions []string // File extensions
}

// StorageReport contains information about storage usage
type StorageReport struct {
	TotalSize    int64                // Total size in bytes
	FileCount    int                  // Total number of files
	DirCount     int                  // Total number of directories
	TypeStats    map[string]TypeStats // Statistics by file type
	LargestFiles []FileInfo           // List of largest files
	OldestFiles  []FileInfo           // List of oldest files
	NewestFiles  []FileInfo           // List of newest files
}

// TypeStats holds statistics for a file type
type TypeStats struct {
	Count int   // Number of files
	Size  int64 // Total size in bytes
}

// FileInfo holds information about a file
type FileInfo struct {
	Path      string // File path
	Size      int64  // File size in bytes
	Modified  string // Last modified time
	Extension string // File extension
}

// SecurityIssue represents a security concern found during analysis
type SecurityIssue struct {
	Path        string // File path
	Description string // Description of the issue
	Severity    string // High, Medium, Low
}

// ContentPattern holds information about content patterns
type ContentPattern struct {
	Count    int      // Number of occurrences
	Files    []string // Files containing the pattern
	Examples []string // Example matches
}

// Analyzer provides filesystem analysis functionality
type Analyzer struct {
	outputFormat  string
	outputFile    string
	maxDepth      int
	minSize       int64
	maxSize       int64
	includeHidden bool
	languages     []string

	// Feature flags
	detectDuplicates bool
	analyzeCode      bool
	doStorage        bool
	doSecurity       bool
	doPatterns       bool

	// Advanced analysis flags
	detectNearDups bool
	analyzeDeps    bool
}

// NewAnalyzer creates a new Analyzer instance
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		outputFormat: "text",
		maxDepth:     0, // unlimited
		languages:    []string{},
	}
}

// SetOutputFormat sets the output format
func (a *Analyzer) SetOutputFormat(format string) {
	a.outputFormat = format
}

// SetOutputFile sets the output file path
func (a *Analyzer) SetOutputFile(path string) {
	a.outputFile = path
}

// SetMaxDepth sets the maximum directory depth to analyze
func (a *Analyzer) SetMaxDepth(depth int) {
	a.maxDepth = depth
}

// SetSizeRange sets the file size range to analyze
func (a *Analyzer) SetSizeRange(min, max string) {
	a.minSize = parseSize(min)
	a.maxSize = parseSize(max)
}

// SetIncludeHidden sets whether to include hidden files
func (a *Analyzer) SetIncludeHidden(include bool) {
	a.includeHidden = include
}

// SetLanguages sets the programming languages to analyze
func (a *Analyzer) SetLanguages(langs []string) {
	a.languages = langs
}

// EnableDuplicateDetection enables duplicate file detection
func (a *Analyzer) EnableDuplicateDetection() {
	a.detectDuplicates = true
}

// EnableCodeStats enables code statistics analysis
func (a *Analyzer) EnableCodeStats() {
	a.analyzeCode = true
}

// EnableStorageReport enables storage usage analysis
func (a *Analyzer) EnableStorageReport() {
	a.doStorage = true
}

// EnableSecurityScan enables security scanning
func (a *Analyzer) EnableSecurityScan() {
	a.doSecurity = true
}

// EnableContentPatternAnalysis enables content pattern analysis
func (a *Analyzer) EnableContentPatternAnalysis() {
	a.doPatterns = true
}

// EnableNearDuplicateDetection enables detection of similar (not just identical) files
func (a *Analyzer) EnableNearDuplicateDetection() {
	a.detectNearDups = true
}

// EnableDependencyAnalysis enables code dependency analysis
func (a *Analyzer) EnableDependencyAnalysis() {
	a.analyzeDeps = true
}

// Analyze performs the filesystem analysis
func (a *Analyzer) Analyze(root string) (*AnalyzeResult, error) {
	result := &AnalyzeResult{
		Duplicates: make(map[string][]string),
		CodeStats:  make(map[string]LanguageStats),
		StorageReport: StorageReport{
			TypeStats: make(map[string]TypeStats),
		},
		SecurityIssues:  []SecurityIssue{},
		ContentPatterns: make(map[string]ContentPattern),
	}

	// For near-duplicate detection, we need to collect all file contents
	var fileContents map[string][]byte
	if a.detectNearDups {
		fileContents = make(map[string][]byte)
	}

	// Walk the filesystem
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check max depth
		if a.maxDepth > 0 {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if strings.Count(relPath, string(os.PathSeparator)) >= a.maxDepth {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip hidden files/directories if not included
		if !a.includeHidden {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") && base != "." {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip directories in file-specific analysis
		if info.IsDir() {
			result.StorageReport.DirCount++
			return nil
		}

		// Check file size constraints
		size := info.Size()
		if (a.minSize > 0 && size < a.minSize) || (a.maxSize > 0 && size > a.maxSize) {
			return nil
		}

		result.StorageReport.FileCount++
		result.StorageReport.TotalSize += size

		// For near-duplicate detection, collect file contents
		if a.detectNearDups {
			content, err := os.ReadFile(path)
			if err == nil {
				fileContents[path] = content
			}
		}

		// Analyze based on enabled features
		if a.detectDuplicates {
			a.analyzeDuplicates(path, result)
		}
		if a.analyzeCode {
			a.analyzeCodeFile(path, info, result)
		}
		if a.doStorage {
			a.analyzeStorage(path, info, result)
		}
		if a.doSecurity {
			a.analyzeSecurity(path, info, result)
		}
		if a.doPatterns {
			a.analyzePatterns(path, result)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Perform advanced analysis if enabled
	if a.detectNearDups || a.analyzeDeps {
		result.Advanced = &AdvancedAnalysis{}

		if a.detectNearDups && len(fileContents) > 0 {
			result.Advanced.NearDuplicates = a.detectNearDuplicates(fileContents)
		}

		if a.analyzeDeps {
			graph, err := a.analyzeDependencies(root)
			if err != nil {
				return nil, fmt.Errorf("dependency analysis failed: %v", err)
			}
			result.Advanced.Dependencies = graph
		}
	}

	return result, nil
}

// Helper function to parse size strings (e.g., "1MB", "500KB")
func parseSize(size string) int64 {
	if size == "" {
		return 0
	}

	size = strings.ToUpper(size)
	multiplier := int64(1)

	if strings.HasSuffix(size, "KB") {
		multiplier = 1024
		size = size[:len(size)-2]
	} else if strings.HasSuffix(size, "MB") {
		multiplier = 1024 * 1024
		size = size[:len(size)-2]
	} else if strings.HasSuffix(size, "GB") {
		multiplier = 1024 * 1024 * 1024
		size = size[:len(size)-2]
	}

	var value int64
	fmt.Sscanf(size, "%d", &value)
	return value * multiplier
}

// SaveToFile saves the analysis results to a file
func (r *AnalyzeResult) SaveToFile(path string) error {
	// Implementation will depend on the output format
	// For now, just write a simple text representation
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(r.String())
	return err
}

// String returns a string representation of the analysis results
func (r *AnalyzeResult) String() string {
	var sb strings.Builder

	// Add storage report
	sb.WriteString("Storage Report:\n")
	sb.WriteString(fmt.Sprintf("Total Size: %d bytes\n", r.StorageReport.TotalSize))
	sb.WriteString(fmt.Sprintf("Files: %d\n", r.StorageReport.FileCount))
	sb.WriteString(fmt.Sprintf("Directories: %d\n", r.StorageReport.DirCount))

	// Add code stats
	if len(r.CodeStats) > 0 {
		sb.WriteString("\nCode Statistics:\n")
		for lang, stats := range r.CodeStats {
			sb.WriteString(fmt.Sprintf("%s: %d files, %d lines\n", lang, stats.Files, stats.Lines))
		}
	}

	// Add security issues
	if len(r.SecurityIssues) > 0 {
		sb.WriteString("\nSecurity Issues:\n")
		for _, issue := range r.SecurityIssues {
			sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", issue.Severity, issue.Path, issue.Description))
		}
	}

	// Add advanced analysis results
	if r.Advanced != nil {
		if len(r.Advanced.NearDuplicates) > 0 {
			sb.WriteString("\nNear-Duplicate Files:\n")
			for _, group := range r.Advanced.NearDuplicates {
				sb.WriteString(fmt.Sprintf("\nSimilarity: %.0f%%\n", group.Similarity*100))
				sb.WriteString(fmt.Sprintf("Files in group:\n"))
				for _, file := range group.Files {
					sb.WriteString(fmt.Sprintf("  %s\n", file))
				}
				sb.WriteString(fmt.Sprintf("Suggested action: %s\n", group.Resolution))
			}
		}

		if r.Advanced.Dependencies != nil {
			sb.WriteString("\nDependency Analysis:\n")
			if len(r.Advanced.Dependencies.Orphans) > 0 {
				sb.WriteString("\nOrphan Files (not imported by any other file):\n")
				for _, file := range r.Advanced.Dependencies.Orphans {
					sb.WriteString(fmt.Sprintf("  %s\n", file))
				}
			}
			if len(r.Advanced.Dependencies.UnusedFiles) > 0 {
				sb.WriteString("\nUnused Files (no imports or importers):\n")
				for _, file := range r.Advanced.Dependencies.UnusedFiles {
					sb.WriteString(fmt.Sprintf("  %s\n", file))
				}
			}
		}
	}

	return sb.String()
}

// analyzeDuplicates detects duplicate files by comparing their content hashes
func (a *Analyzer) analyzeDuplicates(path string, result *AnalyzeResult) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		// Skip files that can't be read
		return
	}

	// Calculate SHA-256 hash of file content
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	// Add file path to the list of files with this hash
	result.Duplicates[hash] = append(result.Duplicates[hash], path)
}

// analyzeCodeFile analyzes a source code file for statistics
func (a *Analyzer) analyzeCodeFile(path string, info os.FileInfo, result *AnalyzeResult) {
	// Get file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return
	}

	// Check if this is a language we're interested in
	lang := getLanguageFromExt(ext)
	if lang == "" || (len(a.languages) > 0 && !contains(a.languages, lang)) {
		return
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Get or create language stats
	stats := result.CodeStats[lang]
	stats.Files++
	stats.Size += info.Size()
	if !contains(stats.Extensions, ext) {
		stats.Extensions = append(stats.Extensions, ext)
	}

	// Count lines
	lines := strings.Split(string(content), "\n")
	stats.Lines += len(lines)

	// Count blank lines and comments
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			stats.Blanks++
			continue
		}

		// Check for comments based on language
		if isComment(trimmed, lang) {
			stats.Comments++
		}
	}

	result.CodeStats[lang] = stats
}

// analyzeSecurity checks for security issues in files and directories
func (a *Analyzer) analyzeSecurity(path string, info os.FileInfo, result *AnalyzeResult) {
	// Check file permissions
	mode := info.Mode()

	// Check for world-writable files
	if mode&0002 != 0 {
		result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
			Path:        path,
			Description: "File is world-writable",
			Severity:    "High",
		})
	}

	// Check for setuid/setgid bits
	if mode&os.ModeSetuid != 0 {
		result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
			Path:        path,
			Description: "File has setuid bit set",
			Severity:    "High",
		})
	}
	if mode&os.ModeSetgid != 0 {
		result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
			Path:        path,
			Description: "File has setgid bit set",
			Severity:    "High",
		})
	}

	// Check for suspicious file extensions
	ext := strings.ToLower(filepath.Ext(path))
	if isSuspiciousExt(ext) {
		result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
			Path:        path,
			Description: "File has suspicious extension",
			Severity:    "Medium",
		})
	}
}

// analyzePatterns looks for specific content patterns in files
func (a *Analyzer) analyzePatterns(path string, result *AnalyzeResult) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Convert to string for pattern matching
	str := string(content)

	// Check for various patterns
	patterns := map[string]string{
		"API Key":     `(?i)(api[_-]?key|apikey)['\"]?\s*[:=]\s*['"]([^'"]+)['"]`,
		"Password":    `(?i)(password|passwd|pwd)['\"]?\s*[:=]\s*['"]([^'"]+)['"]`,
		"Private Key": `-----BEGIN (\w+) PRIVATE KEY-----`,
		"IP Address":  `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`,
		"Email":       `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
	}

	for name, pattern := range patterns {
		matches := findPatternMatches(str, pattern)
		if len(matches) > 0 {
			cp := result.ContentPatterns[name]
			cp.Count += len(matches)
			if !contains(cp.Files, path) {
				cp.Files = append(cp.Files, path)
			}
			// Store up to 3 examples
			for i := 0; i < len(matches) && i < 3; i++ {
				if !contains(cp.Examples, matches[i]) {
					cp.Examples = append(cp.Examples, matches[i])
				}
			}
			result.ContentPatterns[name] = cp
		}
	}
}

// Helper functions

func getLanguageFromExt(ext string) string {
	langMap := map[string]string{
		".go":    "Go",
		".py":    "Python",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".java":  "Java",
		".c":     "C",
		".cpp":   "C++",
		".rs":    "Rust",
		".rb":    "Ruby",
		".php":   "PHP",
		".cs":    "C#",
		".swift": "Swift",
		".kt":    "Kotlin",
	}
	return langMap[ext]
}

func isComment(line, lang string) bool {
	commentPrefixes := map[string][]string{
		"Go":         {`//`, `/*`},
		"Python":     {`#`},
		"JavaScript": {`//`, `/*`},
		"TypeScript": {`//`, `/*`},
		"Java":       {`//`, `/*`},
		"C":          {`//`, `/*`},
		"C++":        {`//`, `/*`},
		"Rust":       {`//`, `/*`},
		"Ruby":       {`#`},
		"PHP":        {`//`, `#`, `/*`},
		"C#":         {`//`, `/*`},
		"Swift":      {`//`, `/*`},
		"Kotlin":     {`//`, `/*`},
	}

	for _, prefix := range commentPrefixes[lang] {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

func isSuspiciousExt(ext string) bool {
	suspicious := map[string]bool{
		".exe":   true,
		".dll":   true,
		".so":    true,
		".dylib": true,
		".sh":    true,
		".bat":   true,
		".cmd":   true,
		".vbs":   true,
		".ps1":   true,
	}
	return suspicious[ext]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func findPatternMatches(content, pattern string) []string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return []string{}
	}

	matches := re.FindAllString(content, -1)
	if matches == nil {
		return []string{}
	}

	// Deduplicate matches
	seen := make(map[string]bool)
	unique := []string{}
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			unique = append(unique, match)
		}
	}

	return unique
}

func (a *Analyzer) analyzeStorage(path string, info os.FileInfo, result *AnalyzeResult) {
	// Update type statistics
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = "(no extension)"
	}
	stats := result.StorageReport.TypeStats[ext]
	stats.Count++
	stats.Size += info.Size()
	result.StorageReport.TypeStats[ext] = stats
}
