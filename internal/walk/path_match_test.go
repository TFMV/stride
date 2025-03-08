package stride

import (
	"strings"
	"testing"
)

// oldPathMatch is the original implementation for comparison
func oldPathMatch(pattern, path string) bool {
	// Simple wildcard matching
	patternParts := strings.Split(pattern, "*")
	if len(patternParts) == 1 {
		return pattern == path
	}

	if !strings.HasPrefix(path, patternParts[0]) {
		return false
	}

	path = path[len(patternParts[0]):]
	for i := 1; i < len(patternParts)-1; i++ {
		idx := strings.Index(path, patternParts[i])
		if idx == -1 {
			return false
		}
		path = path[idx+len(patternParts[i]):]
	}

	return strings.HasSuffix(path, patternParts[len(patternParts)-1])
}

func TestPathMatchEquivalence(t *testing.T) {
	testCases := []struct {
		pattern     string
		path        string
		oldExpected bool
		newExpected bool
	}{
		// Basic matching
		{"*.go", "file.go", true, true},
		{"*.go", "path/to/file.go", true, true},
		{"file.*", "file.go", true, true},
		{"file.*", "path/to/file.go", false, true},

		// Directory matching
		{"path/to/*.go", "path/to/file.go", true, true},
		{"path/to/*.go", "other/path/file.go", false, false},

		// Exact matching
		{"file.go", "file.go", true, true},
		{"file.go", "other.go", false, false},

		// Multiple wildcards
		{"*.*", "file.go", true, true},
		{"*.*.go", "file.test.go", true, true},

		// Edge cases
		{"", "", true, true},
		{"*", "anything", true, true},
		{"*", "", true, true},
	}

	for _, tc := range testCases {
		oldResult := oldPathMatch(tc.pattern, tc.path)
		newResult := pathMatch(tc.pattern, tc.path)

		if oldResult != tc.oldExpected {
			t.Errorf("Pattern %q on path %q: old implementation returned %v, expected %v",
				tc.pattern, tc.path, oldResult, tc.oldExpected)
		}

		if newResult != tc.newExpected {
			t.Errorf("Pattern %q on path %q: new implementation returned %v, expected %v",
				tc.pattern, tc.path, newResult, tc.newExpected)
		}

		// Document the differences
		if tc.oldExpected != tc.newExpected {
			t.Logf("Note: Pattern %q on path %q has different expected behavior: old=%v, new=%v",
				tc.pattern, tc.path, tc.oldExpected, tc.newExpected)
		}
	}
}

func BenchmarkPathMatchComparison(b *testing.B) {
	patterns := []string{
		"*.go",
		"file.*",
		"path/to/*.go",
		"file.go",
		"*.*",
		"*.*.go",
		"*",
	}

	paths := []string{
		"file.go",
		"path/to/file.go",
		"other/path/file.go",
		"file.test.go",
		"very/deep/path/to/some/file.go",
		"",
		"anything",
	}

	b.Run("OldImplementation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, pattern := range patterns {
				for _, path := range paths {
					_ = oldPathMatch(pattern, path)
				}
			}
		}
	})

	b.Run("NewImplementation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, pattern := range patterns {
				for _, path := range paths {
					_ = pathMatch(pattern, path)
				}
			}
		}
	})
}
