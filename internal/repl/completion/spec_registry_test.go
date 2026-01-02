package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpecRegistry(t *testing.T) {
	r := NewSpecRegistry()
	require.NotNil(t, r)
	assert.NotNil(t, r.specs)
	assert.Empty(t, r.specs)
}

func TestSpecRegistryAddSpec(t *testing.T) {
	r := NewSpecRegistry()

	spec := CompletionSpec{
		Command: "git",
		Type:    WordListCompletion,
		Value:   "add commit push pull",
	}

	r.AddSpec(spec)

	got, ok := r.GetSpec("git")
	assert.True(t, ok)
	assert.Equal(t, spec, got)
}

func TestSpecRegistryRemoveSpec(t *testing.T) {
	r := NewSpecRegistry()

	spec := CompletionSpec{
		Command: "git",
		Type:    WordListCompletion,
		Value:   "add commit push pull",
	}

	r.AddSpec(spec)
	r.RemoveSpec("git")

	_, ok := r.GetSpec("git")
	assert.False(t, ok)
}

func TestSpecRegistryGetSpecNotFound(t *testing.T) {
	r := NewSpecRegistry()

	_, ok := r.GetSpec("nonexistent")
	assert.False(t, ok)
}

func TestSpecRegistryListSpecs(t *testing.T) {
	r := NewSpecRegistry()

	specs := []CompletionSpec{
		{Command: "git", Type: WordListCompletion, Value: "add commit"},
		{Command: "docker", Type: WordListCompletion, Value: "run build"},
	}

	for _, spec := range specs {
		r.AddSpec(spec)
	}

	listed := r.ListSpecs()
	assert.Len(t, listed, 2)

	// Check that both specs are present (order may vary)
	commands := make(map[string]bool)
	for _, spec := range listed {
		commands[spec.Command] = true
	}
	assert.True(t, commands["git"])
	assert.True(t, commands["docker"])
}

func TestSpecRegistryUpdateSpec(t *testing.T) {
	r := NewSpecRegistry()

	spec1 := CompletionSpec{
		Command: "git",
		Type:    WordListCompletion,
		Value:   "add commit",
	}

	spec2 := CompletionSpec{
		Command: "git",
		Type:    WordListCompletion,
		Value:   "add commit push pull",
	}

	r.AddSpec(spec1)
	r.AddSpec(spec2)

	got, ok := r.GetSpec("git")
	assert.True(t, ok)
	assert.Equal(t, spec2.Value, got.Value)
}

func TestCompletionTypeConstants(t *testing.T) {
	assert.Equal(t, CompletionType("W"), WordListCompletion)
	assert.Equal(t, CompletionType("F"), FunctionCompletion)
}
