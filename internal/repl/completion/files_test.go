package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDirectory creates a test directory structure for completion tests.
// Structure:
//
//	tmpDir/
//	  file1.txt
//	  file2.txt
//	  .hidden
//	  folder1/
//	    inside.txt
//	    deep/
//	      nested.txt
//	  folder2/
//	    other.txt
func setupTestDirectory(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "completion_test")
	require.NoError(t, err)

	structure := []string{
		"file1.txt",
		"file2.txt",
		".hidden",
		"folder1/inside.txt",
		"folder1/deep/nested.txt",
		"folder2/other.txt",
	}

	for _, f := range structure {
		path := filepath.Join(tmpDir, f)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
	}

	return tmpDir
}

// TestFileCompletions_Systematic tests file completions systematically across
// different path prefix types, depths, and trailing content combinations.
func TestFileCompletions_Systematic(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	// Test matrix: prefix type × depth × trailing content
	//
	// Prefix types:
	//   - "" (empty/implicit current dir)
	//   - "./" (explicit current dir)
	//   - "../" (parent dir) - tested separately due to setup complexity
	//   - "~/" (home dir) - tested separately
	//   - absolute path - tested separately
	//
	// Depth:
	//   - root level (e.g., "./file")
	//   - 1-level deep (e.g., "./folder1/inside")
	//   - 2-level deep (e.g., "./folder1/deep/nested")
	//
	// Trailing content:
	//   - "/" (list directory contents)
	//   - partial name (filter by prefix)
	//   - no match (empty results)

	tests := []struct {
		name     string
		prefix   string
		expected []string
	}{
		// === Empty prefix (implicit current dir) ===
		{
			name:     "empty prefix lists root contents",
			prefix:   "",
			expected: []string{".hidden", "file1.txt", "file2.txt", "folder1/", "folder2/"},
		},
		{
			name:     "implicit: partial file name",
			prefix:   "file",
			expected: []string{"file1.txt", "file2.txt"},
		},
		{
			name:     "implicit: partial directory name",
			prefix:   "folder",
			expected: []string{"folder1/", "folder2/"},
		},
		{
			name:     "implicit: exact file name",
			prefix:   "file1.txt",
			expected: []string{"file1.txt"},
		},
		{
			name:     "implicit: 1-level deep with trailing slash",
			prefix:   "folder1/",
			expected: []string{"folder1/deep/", "folder1/inside.txt"},
		},
		{
			name:     "implicit: 1-level deep with partial name",
			prefix:   "folder1/i",
			expected: []string{"folder1/inside.txt"},
		},
		{
			name:     "implicit: 2-level deep with trailing slash",
			prefix:   "folder1/deep/",
			expected: []string{"folder1/deep/nested.txt"},
		},
		{
			name:     "implicit: 2-level deep with partial name",
			prefix:   "folder1/deep/n",
			expected: []string{"folder1/deep/nested.txt"},
		},
		{
			name:     "implicit: no match",
			prefix:   "nonexistent",
			expected: []string{},
		},

		// === Explicit "./" prefix ===
		{
			name:     "dot-slash: root listing",
			prefix:   "./",
			expected: []string{"./.hidden", "./file1.txt", "./file2.txt", "./folder1/", "./folder2/"},
		},
		{
			name:     "dot-slash: partial file name",
			prefix:   "./file",
			expected: []string{"./file1.txt", "./file2.txt"},
		},
		{
			name:     "dot-slash: partial directory name",
			prefix:   "./folder",
			expected: []string{"./folder1/", "./folder2/"},
		},
		{
			name:     "dot-slash: exact file name",
			prefix:   "./file1.txt",
			expected: []string{"./file1.txt"},
		},
		{
			name:     "dot-slash: 1-level deep with trailing slash",
			prefix:   "./folder1/",
			expected: []string{"./folder1/deep/", "./folder1/inside.txt"},
		},
		{
			name:     "dot-slash: 1-level deep with partial name",
			prefix:   "./folder1/i",
			expected: []string{"./folder1/inside.txt"},
		},
		{
			name:     "dot-slash: 1-level deep partial directory name",
			prefix:   "./folder1/d",
			expected: []string{"./folder1/deep/"},
		},
		{
			name:     "dot-slash: 2-level deep with trailing slash",
			prefix:   "./folder1/deep/",
			expected: []string{"./folder1/deep/nested.txt"},
		},
		{
			name:     "dot-slash: 2-level deep with partial name",
			prefix:   "./folder1/deep/n",
			expected: []string{"./folder1/deep/nested.txt"},
		},
		{
			name:     "dot-slash: hidden file",
			prefix:   "./.h",
			expected: []string{"./.hidden"},
		},
		{
			name:     "dot-slash: no match",
			prefix:   "./nonexistent",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetFileCompletions(tt.prefix, tmpDir)
			assert.ElementsMatch(t, tt.expected, results)

			// Additional verification: all results should maintain prefix format
			if strings.HasPrefix(tt.prefix, "./") {
				for _, r := range results {
					assert.True(t, strings.HasPrefix(r, "./"),
						"Result %q should preserve './' prefix for input %q", r, tt.prefix)
				}
			}
		})
	}
}

// TestFileCompletions_AbsolutePaths tests completion with absolute paths.
func TestFileCompletions_AbsolutePaths(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		prefix   string
		expected []string
	}{
		{
			name:     "absolute: root listing",
			prefix:   tmpDir + "/",
			expected: []string{tmpDir + "/.hidden", tmpDir + "/file1.txt", tmpDir + "/file2.txt", tmpDir + "/folder1/", tmpDir + "/folder2/"},
		},
		{
			name:     "absolute: partial name",
			prefix:   tmpDir + "/file",
			expected: []string{tmpDir + "/file1.txt", tmpDir + "/file2.txt"},
		},
		{
			name:     "absolute: 1-level deep",
			prefix:   tmpDir + "/folder1/",
			expected: []string{tmpDir + "/folder1/deep/", tmpDir + "/folder1/inside.txt"},
		},
		{
			name:     "absolute: 1-level deep with partial name",
			prefix:   tmpDir + "/folder1/i",
			expected: []string{tmpDir + "/folder1/inside.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// currentDir shouldn't matter for absolute paths
			results := GetFileCompletions(tt.prefix, "/some/other/dir")
			assert.ElementsMatch(t, tt.expected, results)

			// All results should be absolute paths
			for _, r := range results {
				assert.True(t, filepath.IsAbs(r), "Result %q should be absolute path", r)
			}
		})
	}
}

// TestFileCompletions_HomePath tests completion with ~ (home directory) paths.
func TestFileCompletions_HomePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	// Create a unique test file in home directory
	testFile := filepath.Join(homeDir, "gsh_completion_test_xyz123.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))
	defer os.Remove(testFile)

	tests := []struct {
		name     string
		prefix   string
		expected []string
	}{
		{
			name:     "home: partial unique name",
			prefix:   "~/gsh_completion_test_x",
			expected: []string{"~/gsh_completion_test_xyz123.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetFileCompletions(tt.prefix, "/some/other/dir")
			assert.ElementsMatch(t, tt.expected, results)

			// All results should start with ~/ and NOT contain the actual home path
			for _, r := range results {
				assert.True(t, strings.HasPrefix(r, "~/"),
					"Result %q should start with ~/", r)
				assert.False(t, strings.Contains(r, homeDir),
					"Result %q should not contain actual home path %q", r, homeDir)
			}
		})
	}

	// Test that ~/ lists home directory contents
	t.Run("home: listing", func(t *testing.T) {
		results := GetFileCompletions("~/", "/some/other/dir")
		assert.NotEmpty(t, results, "Should return home directory contents")
		for _, r := range results {
			assert.True(t, strings.HasPrefix(r, "~/"),
				"Result %q should start with ~/", r)
		}
	})
}

// TestFileCompletions_ParentPath tests completion with ../ (parent directory) paths.
func TestFileCompletions_ParentPath(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	// Use folder1 as current directory to test ../
	currentDir := filepath.Join(tmpDir, "folder1")

	tests := []struct {
		name     string
		prefix   string
		expected []string
	}{
		{
			name:     "parent: listing",
			prefix:   "../",
			expected: []string{"../.hidden", "../file1.txt", "../file2.txt", "../folder1/", "../folder2/"},
		},
		{
			name:     "parent: partial name",
			prefix:   "../file",
			expected: []string{"../file1.txt", "../file2.txt"},
		},
		{
			name:     "parent: into sibling directory",
			prefix:   "../folder2/",
			expected: []string{"../folder2/other.txt"},
		},
		{
			name:     "parent: into sibling with partial name",
			prefix:   "../folder2/o",
			expected: []string{"../folder2/other.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetFileCompletions(tt.prefix, currentDir)
			assert.ElementsMatch(t, tt.expected, results)

			// All results should start with ../
			for _, r := range results {
				assert.True(t, strings.HasPrefix(r, "../"),
					"Result %q should start with ../", r)
			}
		})
	}
}

// TestFileCompletions_EdgeCases tests edge cases and error conditions.
func TestFileCompletions_EdgeCases(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	t.Run("nonexistent directory returns empty", func(t *testing.T) {
		results := GetFileCompletions("nonexistent/path/", tmpDir)
		assert.Empty(t, results)
	})

	t.Run("nonexistent absolute path returns empty", func(t *testing.T) {
		results := GetFileCompletions("/nonexistent/path/", tmpDir)
		assert.Empty(t, results)
	})

	t.Run("permission denied returns empty", func(t *testing.T) {
		// Create a directory without read permission
		noReadDir := filepath.Join(tmpDir, "noread")
		require.NoError(t, os.Mkdir(noReadDir, 0000))
		defer os.Chmod(noReadDir, 0755) // Restore for cleanup

		results := GetFileCompletions(noReadDir+"/", tmpDir)
		assert.Empty(t, results)
	})

	t.Run("empty directory returns empty", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		require.NoError(t, os.Mkdir(emptyDir, 0755))

		results := GetFileCompletions("empty/", tmpDir)
		assert.Empty(t, results)
	})

	t.Run("trailing slashes are handled", func(t *testing.T) {
		// Multiple scenarios with trailing slash
		results := GetFileCompletions("./folder1/", tmpDir)
		assert.NotEmpty(t, results)
		for _, r := range results {
			assert.True(t, strings.HasPrefix(r, "./folder1/"))
		}
	})
}

// TestFileCompletions_DirectoryTrailingSlash verifies directories always have trailing slash.
func TestFileCompletions_DirectoryTrailingSlash(t *testing.T) {
	tmpDir := setupTestDirectory(t)
	defer os.RemoveAll(tmpDir)

	results := GetFileCompletions("./", tmpDir)

	for _, r := range results {
		// Check if it's a directory by looking it up
		cleanPath := strings.TrimPrefix(r, "./")
		info, err := os.Stat(filepath.Join(tmpDir, cleanPath))
		if err != nil {
			continue // File might have trailing slash removed, skip
		}

		if info.IsDir() {
			assert.True(t, strings.HasSuffix(r, "/"),
				"Directory %q should have trailing slash", r)
		} else {
			assert.False(t, strings.HasSuffix(r, "/"),
				"File %q should not have trailing slash", r)
		}
	}
}

func TestFileCompletionsNonExistentDirectory(t *testing.T) {
	results := GetFileCompletions("nonexistent/path/", "/tmp")
	assert.Empty(t, results)
}
