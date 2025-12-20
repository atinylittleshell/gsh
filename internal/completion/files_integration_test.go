package completion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to check if completions contain a specific value (combining Value and Suffix)
func containsCompletion(completions []shellinput.CompletionCandidate, expected string) bool {
	for _, c := range completions {
		if c.Value+c.Suffix == expected {
			return true
		}
	}
	return false
}

func TestGetFileCompletions_Integration(t *testing.T) {
	// Create a temporary directory with test files and directories
	tmpDir := t.TempDir()

	// Create files
	files := []string{
		"file1.txt",
		"file2.log",
		".hidden",
		"spaced name.txt",
	}

	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// Create directories with files
	dirs := map[string][]string{
		"documents": {"doc1.pdf", "doc2.txt"},
		"projects":  {"main.go"},
	}

	for dir, dirFiles := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)

		for _, file := range dirFiles {
			filePath := filepath.Join(dirPath, file)
			err := os.WriteFile(filePath, []byte("content"), 0644)
			require.NoError(t, err)
		}
	}

	// Create nested directory
	nestedPath := filepath.Join(tmpDir, "projects", "project1")
	err := os.MkdirAll(nestedPath, 0755)
	require.NoError(t, err)

	// Change to temp directory for relative path tests
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(tmpDir)

	// Normalize helper
	norm := func(p string) string {
		return filepath.FromSlash(p)
	}

	tests := []struct {
		name             string
		prefix           string
		currentDir       string
		expectedMin      int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "empty prefix lists all files",
			prefix:        "",
			currentDir:    tmpDir,
			expectedMin:   6,
			shouldContain: []string{"file1.txt", "file2.log", norm("documents/"), norm("projects/"), "spaced name.txt"},
		},
		{
			name:             "file prefix matching",
			prefix:           "file",
			currentDir:       tmpDir,
			expectedMin:      2,
			shouldContain:    []string{"file1.txt", "file2.log"},
			shouldNotContain: []string{norm("documents/"), norm("projects/")},
		},
		{
			name:             "directory prefix matching",
			prefix:           "doc",
			currentDir:       tmpDir,
			expectedMin:      1,
			shouldContain:    []string{norm("documents/")},
			shouldNotContain: []string{"file1.txt", norm("projects/")},
		},
		{
			name:             "hidden file matching",
			prefix:           ".h",
			currentDir:       tmpDir,
			expectedMin:      1,
			shouldContain:    []string{".hidden"},
			shouldNotContain: []string{"file1.txt", norm("documents/")},
		},
		{
			name:          "subdirectory completion",
			prefix:        norm("documents/"),
			currentDir:    tmpDir,
			expectedMin:   2,
			shouldContain: []string{norm("documents/doc1.pdf"), norm("documents/doc2.txt")},
		},
		{
			name:          "nested subdirectory",
			prefix:        norm("projects/"),
			currentDir:    tmpDir,
			expectedMin:   2,
			shouldContain: []string{norm("projects/project1/"), norm("projects/main.go")},
		},
		{
			name:             "partial file in subdirectory",
			prefix:           norm("documents/doc1"),
			currentDir:       tmpDir,
			expectedMin:      1,
			shouldContain:    []string{norm("documents/doc1.pdf")},
			shouldNotContain: []string{norm("documents/doc2.txt")},
		},
		{
			name:        "absolute path completion",
			prefix:      filepath.Join(tmpDir, "file"),
			currentDir:  "/",
			expectedMin: 2,
			shouldContain: []string{
				filepath.Join(tmpDir, "file1.txt"),
				filepath.Join(tmpDir, "file2.log"),
			},
		},
		{
			name:        "absolute directory completion",
			prefix:      filepath.Join(tmpDir, "documents") + string(os.PathSeparator),
			currentDir:  "/",
			expectedMin: 2,
			shouldContain: []string{
				filepath.Join(tmpDir, "documents", "doc1.pdf"),
				filepath.Join(tmpDir, "documents", "doc2.txt"),
			},
		},
	}

	// Test home directory expansion
	if homeDir, err := os.UserHomeDir(); err == nil {
		// Create a test file in home directory for testing
		testFile := filepath.Join(homeDir, ".test_completion_file")
		_ = os.WriteFile(testFile, []byte("test"), 0644)
		t.Cleanup(func() {
			assert.NoError(t, os.Remove(testFile))
		})

		tests = append(tests, struct {
			name             string
			prefix           string
			currentDir       string
			expectedMin      int
			shouldContain    []string
			shouldNotContain []string
		}{
			name:          "home directory expansion",
			prefix:        "~/.test_completion",
			currentDir:    tmpDir,
			expectedMin:   1,
			shouldContain: []string{"~" + string(os.PathSeparator) + ".test_completion_file"},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := getFileCompletions(tt.prefix, tt.currentDir)

			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Expected at least %d completions for prefix %q, got %d: %v",
				tt.expectedMin, tt.prefix, len(completions), completions)

			for _, expected := range tt.shouldContain {
				assert.True(t, containsCompletion(completions, expected),
					"Expected completions to contain %q for prefix %q, got: %v",
					expected, tt.prefix, completions)
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.False(t, containsCompletion(completions, notExpected),
					"Expected completions to NOT contain %q for prefix %q, got: %v",
					notExpected, tt.prefix, completions)
			}
		})
	}
}

func TestGetFileCompletions_RelativePaths_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	structure := map[string][]string{
		"level1":               {"file_l1.txt"},
		"level1/level2":        {"file_l2.txt"},
		"level1/level2/level3": {"file_l3.txt"},
		"sibling":              {"sibling_file.txt"},
	}

	for dir, files := range structure {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)

		for _, file := range files {
			filePath := filepath.Join(dirPath, file)
			err := os.WriteFile(filePath, []byte("content"), 0644)
			require.NoError(t, err)
		}
	}

	norm := func(p string) string {
		return filepath.FromSlash(p)
	}

	tests := []struct {
		name             string
		workingDir       string
		prefix           string
		expectedMin      int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "relative path from root",
			workingDir:    tmpDir,
			prefix:        norm("level1/"),
			expectedMin:   2,
			shouldContain: []string{norm("level1/file_l1.txt"), norm("level1/level2/")},
		},
		{
			name:          "relative path from subdirectory",
			workingDir:    filepath.Join(tmpDir, "level1"),
			prefix:        norm("level2/"),
			expectedMin:   2,
			shouldContain: []string{norm("level2/file_l2.txt"), norm("level2/level3/")},
		},
		{
			name:          "parent directory navigation",
			workingDir:    filepath.Join(tmpDir, "level1"),
			prefix:        norm("../sibling/"),
			expectedMin:   1,
			shouldContain: []string{norm("../sibling/sibling_file.txt")},
		},
		{
			name:          "deep relative path",
			workingDir:    tmpDir,
			prefix:        norm("level1/level2/level3/"),
			expectedMin:   1,
			shouldContain: []string{norm("level1/level2/level3/file_l3.txt")},
		},
		{
			name:             "partial relative path",
			workingDir:       tmpDir,
			prefix:           norm("level1/file"),
			expectedMin:      1,
			shouldContain:    []string{norm("level1/file_l1.txt")},
			shouldNotContain: []string{norm("level1/level2/")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := getFileCompletions(tt.prefix, tt.workingDir)

			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Expected at least %d completions for prefix %q from dir %q, got %d: %v",
				tt.expectedMin, tt.prefix, tt.workingDir, len(completions), completions)

			for _, expected := range tt.shouldContain {
				assert.True(t, containsCompletion(completions, expected),
					"Expected completions to contain %q for prefix %q from dir %q, got: %v",
					expected, tt.prefix, tt.workingDir, completions)
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.False(t, containsCompletion(completions, notExpected),
					"Expected completions to NOT contain %q for prefix %q from dir %q, got: %v",
					notExpected, tt.prefix, tt.workingDir, completions)
			}
		})
	}
}

func TestGetFileCompletions_EdgeCases_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create edge case files
	edgeFiles := []string{
		"file with spaces.txt",
		"file-with-dashes.log",
		"file_with_underscores.sh",
		"file.with.dots.conf",
		"123numeric_start.txt",
		"UPPERCASE.TXT",
		"MixedCase.File",
	}

	for _, file := range edgeFiles {
		filePath := filepath.Join(tmpDir, file)
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		prefix        string
		expectedMin   int
		shouldContain []string
	}{
		{
			name:          "files with spaces",
			prefix:        "file with",
			expectedMin:   1,
			shouldContain: []string{"file with spaces.txt"},
		},
		{
			name:          "files with dashes",
			prefix:        "file-",
			expectedMin:   1,
			shouldContain: []string{"file-with-dashes.log"},
		},
		{
			name:          "files with underscores",
			prefix:        "file_",
			expectedMin:   1,
			shouldContain: []string{"file_with_underscores.sh"},
		},
		{
			name:          "files with dots",
			prefix:        "file.with",
			expectedMin:   1,
			shouldContain: []string{"file.with.dots.conf"},
		},
		{
			name:          "files starting with numbers",
			prefix:        "123",
			expectedMin:   1,
			shouldContain: []string{"123numeric_start.txt"},
		},
		{
			name:          "uppercase files",
			prefix:        "UPPER",
			expectedMin:   1,
			shouldContain: []string{"UPPERCASE.TXT"},
		},
		{
			name:          "mixed case files",
			prefix:        "Mixed",
			expectedMin:   1,
			shouldContain: []string{"MixedCase.File"},
		},
		{
			name:          "partial extension match",
			prefix:        "file",
			expectedMin:   4, // Should match multiple files starting with "file"
			shouldContain: []string{"file with spaces.txt", "file-with-dashes.log", "file_with_underscores.sh", "file.with.dots.conf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := getFileCompletions(tt.prefix, tmpDir)

			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Expected at least %d completions for prefix %q, got %d: %v",
				tt.expectedMin, tt.prefix, len(completions), completions)

			for _, expected := range tt.shouldContain {
				assert.True(t, containsCompletion(completions, expected),
					"Expected completions to contain %q for prefix %q, got: %v",
					expected, tt.prefix, completions)
			}
		})
	}
}

func TestGetFileCompletions_Permissions_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different permissions
	files := map[string]os.FileMode{
		"readable.txt":       0644,
		"writable.txt":       0200,
		"executable.sh":      0755,
		"no_permissions.txt": 0000,
	}

	for file, mode := range files {
		filePath := filepath.Join(tmpDir, file)
		err := os.WriteFile(filePath, []byte("content"), mode)
		require.NoError(t, err)
		// Clean up permissions after test
		defer func() { _ = os.Chmod(filePath, 0644) }()
	}

	// Create directories with different permissions
	dirs := map[string]os.FileMode{
		"readable_dir":   0755,
		"no_read_dir":    0000,
		"no_execute_dir": 0644,
	}

	for dir, mode := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		err := os.MkdirAll(dirPath, mode)
		require.NoError(t, err)
		// Clean up permissions after test
		defer func() { _ = os.Chmod(dirPath, 0755) }()
	}

	// Normalize helper
	norm := func(p string) string {
		return filepath.FromSlash(p)
	}

	tests := []struct {
		name          string
		prefix        string
		expectedMin   int
		shouldContain []string
	}{
		{
			name:          "all files regardless of permissions",
			prefix:        "",
			expectedMin:   7, // 4 files + 3 directories
			shouldContain: []string{"readable.txt", "writable.txt", "executable.sh", norm("readable_dir/")},
		},
		{
			name:          "executable files",
			prefix:        "executable",
			expectedMin:   1,
			shouldContain: []string{"executable.sh"},
		},
		{
			name:          "readable directory",
			prefix:        "readable_dir",
			expectedMin:   1,
			shouldContain: []string{norm("readable_dir/")},
		},
		{
			name:          "no permission files still show up in listing",
			prefix:        "no_permissions",
			expectedMin:   1,
			shouldContain: []string{"no_permissions.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := getFileCompletions(tt.prefix, tmpDir)

			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Expected at least %d completions for prefix %q, got %d: %v",
				tt.expectedMin, tt.prefix, len(completions), completions)

			for _, expected := range tt.shouldContain {
				assert.True(t, containsCompletion(completions, expected),
					"Expected completions to contain %q for prefix %q, got: %v",
					expected, tt.prefix, completions)
			}
		})
	}
}

func TestGetFileCompletions_LargeDirectory_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many files to test performance and correctness
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("file_%03d.txt", i)
		filePath := filepath.Join(tmpDir, fileName)
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)
	}

	// Also create some directories
	numDirs := 20
	for i := 0; i < numDirs; i++ {
		dirName := fmt.Sprintf("dir_%03d", i)
		dirPath := filepath.Join(tmpDir, dirName)
		err := os.MkdirAll(dirPath, 0755)
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		prefix      string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "all files and directories",
			prefix:      "",
			expectedMin: 120, // 100 files + 20 directories
			expectedMax: 120,
		},
		{
			name:        "files with prefix",
			prefix:      "file_",
			expectedMin: 100,
			expectedMax: 100,
		},
		{
			name:        "directories with prefix",
			prefix:      "dir_",
			expectedMin: 20,
			expectedMax: 20,
		},
		{
			name:        "specific range of files",
			prefix:      "file_00",
			expectedMin: 10, // file_000.txt through file_009.txt
			expectedMax: 10,
		},
		{
			name:        "single file match",
			prefix:      "file_050",
			expectedMin: 1,
			expectedMax: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := getFileCompletions(tt.prefix, tmpDir)

			assert.GreaterOrEqual(t, len(completions), tt.expectedMin,
				"Expected at least %d completions for prefix %q, got %d",
				tt.expectedMin, tt.prefix, len(completions))

			assert.LessOrEqual(t, len(completions), tt.expectedMax,
				"Expected at most %d completions for prefix %q, got %d",
				tt.expectedMax, tt.prefix, len(completions))
		})
	}
}

// TestGetFileCompletions_DotPrefixEquivalence tests that "." and "./." give equivalent results
// This was a bug where "./." would search the parent directory instead of current directory
func TestGetFileCompletions_DotPrefixEquivalence_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some hidden files and directories
	hiddenFiles := []string{".hidden1", ".hidden2", ".config"}
	regularFiles := []string{"visible1.txt", "visible2.txt"}

	for _, file := range hiddenFiles {
		filePath := filepath.Join(tmpDir, file)
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)
	}

	for _, file := range regularFiles {
		filePath := filepath.Join(tmpDir, file)
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)
	}

	// Create a hidden directory
	hiddenDir := filepath.Join(tmpDir, ".hidden_dir")
	err := os.MkdirAll(hiddenDir, 0755)
	require.NoError(t, err)

	norm := func(p string) string {
		return filepath.FromSlash(p)
	}

	t.Run("dot prefix shows hidden files", func(t *testing.T) {
		completions := getFileCompletions(".", tmpDir)

		// Should find hidden files
		assert.True(t, containsCompletion(completions, ".hidden1"),
			"Expected to find .hidden1 with '.' prefix, got: %v", completions)
		assert.True(t, containsCompletion(completions, ".hidden2"),
			"Expected to find .hidden2 with '.' prefix, got: %v", completions)
		assert.True(t, containsCompletion(completions, norm(".hidden_dir/")),
			"Expected to find .hidden_dir/ with '.' prefix, got: %v", completions)

		// Should NOT find regular files (they don't start with ".")
		assert.False(t, containsCompletion(completions, "visible1.txt"),
			"Should NOT find visible1.txt with '.' prefix, got: %v", completions)
	})

	t.Run("dot-slash-dot prefix shows hidden files with ./ prefix", func(t *testing.T) {
		completions := getFileCompletions(norm("./."), tmpDir)

		// Should find hidden files with "./" prefix (or ".\" on Windows)
		assert.True(t, containsCompletion(completions, norm("./.hidden1")),
			"Expected to find ./.hidden1 with './.' prefix, got: %v", completions)
		assert.True(t, containsCompletion(completions, norm("./.hidden2")),
			"Expected to find ./.hidden2 with './.' prefix, got: %v", completions)
		assert.True(t, containsCompletion(completions, norm("./.hidden_dir/")),
			"Expected to find ./.hidden_dir/ with './.' prefix, got: %v", completions)

		// Should NOT find regular files
		assert.False(t, containsCompletion(completions, norm("./visible1.txt")),
			"Should NOT find ./visible1.txt with './.' prefix, got: %v", completions)
	})

	t.Run("dot and dot-slash-dot give same count", func(t *testing.T) {
		dotCompletions := getFileCompletions(".", tmpDir)
		dotSlashDotCompletions := getFileCompletions(norm("./."), tmpDir)

		assert.Equal(t, len(dotCompletions), len(dotSlashDotCompletions),
			"'.' and './.' should return same number of completions: '.'=%v, './.'=%v",
			dotCompletions, dotSlashDotCompletions)
	})

	t.Run("parent-dot prefix works correctly", func(t *testing.T) {
		// Create a subdirectory and test from there
		subDir := filepath.Join(tmpDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		completions := getFileCompletions(norm("../."), subDir)

		// Should find hidden files in parent (tmpDir) with "../" prefix (or "..\" on Windows)
		assert.True(t, containsCompletion(completions, norm("../.hidden1")),
			"Expected to find ../.hidden1 with '../.' prefix from subdir, got: %v", completions)
		assert.True(t, containsCompletion(completions, norm("../.hidden2")),
			"Expected to find ../.hidden2 with '../.' prefix from subdir, got: %v", completions)
	})
}

// TestGetFileCompletions_TildePrefix tests home directory completion edge cases
func TestGetFileCompletions_TildePrefix_Integration(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get home directory")
	}

	// Create a test hidden file in home directory
	testHiddenFile := filepath.Join(homeDir, ".gsh_test_hidden_file")
	err = os.WriteFile(testHiddenFile, []byte("test"), 0644)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(testHiddenFile)
	})

	// Create a test visible file in home directory
	testVisibleFile := filepath.Join(homeDir, "gsh_test_visible_file")
	err = os.WriteFile(testVisibleFile, []byte("test"), 0644)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Remove(testVisibleFile)
	})

	norm := func(p string) string {
		return filepath.FromSlash(p)
	}

	t.Run("tilde alone lists home directory", func(t *testing.T) {
		completions := getFileCompletions("~", "/some/other/dir")

		// Should have some completions (home is not empty)
		assert.Greater(t, len(completions), 0,
			"Expected some completions for '~' prefix")

		// All completions should start with "~/"
		for _, c := range completions {
			assert.True(t, strings.HasPrefix(c.Value, "~"+string(os.PathSeparator)),
				"Expected completion to start with ~/, got: %s", c.Value)
		}

		// Should find our test visible file
		assert.True(t, containsCompletion(completions, norm("~/gsh_test_visible_file")),
			"Expected to find ~/gsh_test_visible_file with '~' prefix, got: %v", completions)
	})

	t.Run("tilde-slash lists home directory", func(t *testing.T) {
		completions := getFileCompletions("~/", "/some/other/dir")

		// Should have some completions
		assert.Greater(t, len(completions), 0,
			"Expected some completions for '~/' prefix")

		// All completions should start with "~/"
		for _, c := range completions {
			assert.True(t, strings.HasPrefix(c.Value, "~"+string(os.PathSeparator)),
				"Expected completion to start with ~/, got: %s", c.Value)
		}
	})

	t.Run("tilde-dot shows hidden files in home", func(t *testing.T) {
		completions := getFileCompletions("~/.", "/some/other/dir")

		// Should find our test hidden file
		assert.True(t, containsCompletion(completions, norm("~/.gsh_test_hidden_file")),
			"Expected to find ~/.gsh_test_hidden_file with '~/.' prefix, got: %v", completions)

		// Should NOT find visible files (they don't start with ".")
		assert.False(t, containsCompletion(completions, norm("~/gsh_test_visible_file")),
			"Should NOT find ~/gsh_test_visible_file with '~/.' prefix, got: %v", completions)
	})

	t.Run("tilde and tilde-slash give same results", func(t *testing.T) {
		tildeCompletions := getFileCompletions("~", "/some/other/dir")
		tildeSlashCompletions := getFileCompletions("~/", "/some/other/dir")

		assert.Equal(t, len(tildeCompletions), len(tildeSlashCompletions),
			"'~' and '~/' should return same number of completions")
	})
}
