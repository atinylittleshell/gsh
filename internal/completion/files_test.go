package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileCompletions(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "completion_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

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
	defer os.Remove(testFileInHome)

	tests := []struct {
		name        string
		prefix      string
		currentDir  string
		expected    []string
		shouldMatch bool                                 // true for exact match, false for contains
		verify      func(t *testing.T, results []string) // optional additional verification
	}{
		{
			name:        "empty prefix lists all files",
			prefix:      "",
			currentDir:  tmpDir,
			expected:    []string{"file1.txt", "file2.txt", "folder1/", "folder2/"},
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
			expected:    []string{"folder1/", "folder2/"},
			shouldMatch: true,
		},
		{
			name:        "absolute path prefix",
			prefix:      filepath.Join(tmpDir, "folder1") + "/",
			currentDir:  "/some/other/dir",
			expected:    []string{filepath.Join(tmpDir, "folder1/inside.txt")},
			shouldMatch: true,
			verify: func(t *testing.T, results []string) {
				// All results should be absolute paths
				for _, result := range results {
					assert.True(t, filepath.IsAbs(result), "Expected absolute path, got: %s", result)
				}
			},
		},
		{
			name:        "relative path in subdirectory",
			prefix:      "folder1/i",
			currentDir:  tmpDir,
			expected:    []string{"folder1/inside.txt"},
			shouldMatch: true,
			verify: func(t *testing.T, results []string) {
				// All results should be relative paths
				for _, result := range results {
					assert.False(t, filepath.IsAbs(result), "Expected relative path, got: %s", result)
				}
			},
		},
		{
			name:        "home directory prefix",
			prefix:      "~/",
			currentDir:  "/some/other/dir",
			expected:    []string{},
			shouldMatch: false,
			verify: func(t *testing.T, results []string) {
				// All results should start with "~/"
				assert.Greater(t, len(results), 0, "Expected some results")
				for _, result := range results {
					assert.True(t, strings.HasPrefix(result, "~/"), "Expected path starting with ~/, got: %s", result)
					assert.False(t, strings.Contains(result, homeDir), "Path should not contain actual home directory")
				}
			},
		},
		{
			name:        "home directory with partial filename",
			prefix:      "~/gsh_test",
			currentDir:  "/some/other/dir",
			expected:    []string{"~/gsh_test_file.txt"},
			shouldMatch: true,
			verify: func(t *testing.T, results []string) {
				// All results should start with "~/"
				for _, result := range results {
					assert.True(t, strings.HasPrefix(result, "~/"), "Expected path starting with ~/, got: %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := getFileCompletions(tt.prefix, tt.currentDir)
			if tt.verify != nil {
				tt.verify(t, results)
			}
			if tt.shouldMatch {
				assert.ElementsMatch(t, tt.expected, results)
			} else {
				for _, exp := range tt.expected {
					found := false
					for _, res := range results {
						if filepath.Base(res) == exp {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected to find %s in results, but got %s", exp, results)
				}
			}
		})
	}
}
