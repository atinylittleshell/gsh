package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/stretchr/testify/assert"
)

func TestFileCompletions(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "completion_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	})

	// Create some test files and directories
	files := []string{
		"file1.txt",
		"file2.txt",
		"folder1/",
		"folder2/",
		"folder1/inside.txt",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if filepath.Ext(path) == "" {
			// It's a directory
			err = os.MkdirAll(path, 0755)
		} else {
			// It's a file
			err = os.MkdirAll(filepath.Dir(path), 0755)
			if err == nil {
				err = os.WriteFile(path, []byte("test"), 0644)
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get user's home directory for testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	// Create a test file in home directory
	testFileInHome := filepath.Join(homeDir, "gsh_test_file.txt")
	err = os.WriteFile(testFileInHome, []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(testFileInHome))
	})

	// Helper to normalize path separators
	norm := func(path string) string {
		if strings.HasSuffix(path, "/") {
			// For directories, we expect OS separator at the end
			return filepath.FromSlash(strings.TrimSuffix(path, "/")) + string(os.PathSeparator)
		}
		return filepath.FromSlash(path)
	}

	tests := []struct {
		name        string
		prefix      string
		currentDir  string
		expected    []string
		shouldMatch bool                                 // true for exact match, false for contains
		verify      func(t *testing.T, results []shellinput.CompletionCandidate) // optional additional verification
	}{
		{
			name:        "empty prefix lists all files",
			prefix:      "",
			currentDir:  tmpDir,
			expected:    []string{"file1.txt", "file2.txt", norm("folder1/"), norm("folder2/")},
			shouldMatch: true,
		},
		{
			name:        "prefix matches start of filename",
			prefix:      "file",
			currentDir:  tmpDir,
			expected:    []string{"file1.txt", "file2.txt"},
			shouldMatch: true,
		},
		{
			name:        "prefix matches directories",
			prefix:      "folder",
			currentDir:  tmpDir,
			expected:    []string{norm("folder1/"), norm("folder2/")},
			shouldMatch: true,
		},
		{
			name:        "absolute path prefix",
			prefix:      filepath.Join(tmpDir, "folder1") + string(os.PathSeparator),
			currentDir:  "/some/other/dir",
			expected:    []string{filepath.Join(tmpDir, "folder1", "inside.txt")},
			shouldMatch: true,
			verify: func(t *testing.T, results []shellinput.CompletionCandidate) {
				// All results should be absolute paths
				for _, result := range results {
					assert.True(t, filepath.IsAbs(result.Value), "Expected absolute path, got: %s", result.Value)
				}
			},
		},
		{
			name:        "relative path in subdirectory",
			prefix:      "folder1/i", // Input uses forward slash usually, but let's make it robust
			currentDir:  tmpDir,
			expected:    []string{filepath.Join("folder1", "inside.txt")},
			shouldMatch: true,
			verify: func(t *testing.T, results []shellinput.CompletionCandidate) {
				// All results should be relative paths
				for _, result := range results {
					assert.False(t, filepath.IsAbs(result.Value), "Expected relative path, got: %s", result.Value)
				}
			},
		},
		{
			name:        "home directory prefix",
			prefix:      "~/",
			currentDir:  "/some/other/dir",
			expected:    []string{},
			shouldMatch: false,
			verify: func(t *testing.T, results []shellinput.CompletionCandidate) {
				// All results should start with "~/" or "~\", depending on OS input handling but usually output has ~
				assert.Greater(t, len(results), 0, "Expected some results")
				for _, result := range results {
					// On Windows, completions might use backslash but still start with ~
					// Check if it starts with ~ and path separator
					assert.True(t, strings.HasPrefix(result.Value, "~"+string(os.PathSeparator)), "Expected path starting with ~%s, got: %s", string(os.PathSeparator), result.Value)
					assert.False(t, strings.Contains(result.Value, homeDir), "Path should not contain actual home directory")
				}
			},
		},
		{
			name:        "home directory with partial filename",
			prefix:      "~/gsh_test",
			currentDir:  "/some/other/dir",
			expected:    []string{"~" + string(os.PathSeparator) + "gsh_test_file.txt"},
			shouldMatch: true,
			verify: func(t *testing.T, results []shellinput.CompletionCandidate) {
				// All results should start with "~/"
				for _, result := range results {
					assert.True(t, strings.HasPrefix(result.Value, "~"+string(os.PathSeparator)), "Expected path starting with ~%s, got: %s", string(os.PathSeparator), result.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure prefix uses correct separator for the test env if needed, though input usually uses typed separator.
			// The test cases use hardcoded forward slashes in prefix for some.
			// Let's just run as is, assuming the completion logic handles mixed separators or expects normalized ones.
			// Actually, "folder1/i" might need to be "folder1\i" on windows for strict matching if not normalized.
			// But the completion provider should handle it.

			results := getFileCompletions(tt.prefix, tt.currentDir)
			if tt.verify != nil {
				tt.verify(t, results)
			}
			if tt.shouldMatch {
				// Convert CompletionCandidate to strings for comparison
				resultStrings := make([]string, len(results))
				for i, r := range results {
					// Combine Value and Suffix for comparison with expected
					resultStrings[i] = r.Value + r.Suffix
				}
				assert.ElementsMatch(t, tt.expected, resultStrings)
			} else {
				for _, exp := range tt.expected {
					found := false
					for _, res := range results {
						if filepath.Base(res.Value+res.Suffix) == exp {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected to find %s in results, but got %v", exp, results)
				}
			}
		})
	}
}
