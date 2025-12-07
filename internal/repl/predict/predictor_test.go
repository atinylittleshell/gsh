package predict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultContextFormatter_FormatContext(t *testing.T) {
	formatter := &DefaultContextFormatter{}

	t.Run("empty context", func(t *testing.T) {
		result := formatter.FormatContext(nil)
		assert.Equal(t, "", result)

		result = formatter.FormatContext(map[string]string{})
		assert.Equal(t, "", result)
	})

	t.Run("single entry", func(t *testing.T) {
		result := formatter.FormatContext(map[string]string{
			"cwd": "/home/user",
		})
		assert.Contains(t, result, "## cwd")
		assert.Contains(t, result, "/home/user")
	})

	t.Run("multiple entries", func(t *testing.T) {
		result := formatter.FormatContext(map[string]string{
			"cwd": "/home/user",
			"git": "branch: main",
		})
		assert.Contains(t, result, "## cwd")
		assert.Contains(t, result, "/home/user")
		assert.Contains(t, result, "## git")
		assert.Contains(t, result, "branch: main")
	})

	t.Run("skips empty values", func(t *testing.T) {
		result := formatter.FormatContext(map[string]string{
			"cwd":   "/home/user",
			"empty": "",
		})
		assert.Contains(t, result, "## cwd")
		assert.NotContains(t, result, "## empty")
	})
}

func TestBestPractices(t *testing.T) {
	// Ensure BestPractices constant is defined and not empty
	assert.NotEmpty(t, BestPractices)
	assert.Contains(t, BestPractices, "Git commit")
}
