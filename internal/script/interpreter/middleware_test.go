package interpreter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareManager_NewMiddlewareManager(t *testing.T) {
	mm := NewMiddlewareManager()
	assert.NotNil(t, mm)
	assert.Equal(t, 0, mm.Len())
}

func TestMiddlewareManager_Use(t *testing.T) {
	mm := NewMiddlewareManager()

	tool := &ToolValue{
		Name:       "testMiddleware",
		Parameters: []string{"ctx", "next"},
	}

	id := mm.Use(tool, nil)
	assert.NotEmpty(t, id)
	assert.Equal(t, 1, mm.Len())

	// Adding another middleware should increment count
	tool2 := &ToolValue{
		Name:       "testMiddleware2",
		Parameters: []string{"ctx", "next"},
	}
	id2 := mm.Use(tool2, nil)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id, id2)
	assert.Equal(t, 2, mm.Len())
}

func TestMiddlewareManager_Remove(t *testing.T) {
	mm := NewMiddlewareManager()

	tool := &ToolValue{
		Name:       "testMiddleware",
		Parameters: []string{"ctx", "next"},
	}

	id := mm.Use(tool, nil)
	assert.Equal(t, 1, mm.Len())

	// Remove by ID
	removed := mm.Remove(id)
	assert.True(t, removed)
	assert.Equal(t, 0, mm.Len())

	// Removing again should return false
	removed = mm.Remove(id)
	assert.False(t, removed)
}

func TestMiddlewareManager_RemoveByTool(t *testing.T) {
	mm := NewMiddlewareManager()

	tool := &ToolValue{
		Name:       "testMiddleware",
		Parameters: []string{"ctx", "next"},
	}

	mm.Use(tool, nil)
	assert.Equal(t, 1, mm.Len())

	// Remove by tool reference
	removed := mm.RemoveByTool(tool)
	assert.True(t, removed)
	assert.Equal(t, 0, mm.Len())

	// Removing again should return false
	removed = mm.RemoveByTool(tool)
	assert.False(t, removed)
}

func TestMiddlewareManager_Clear(t *testing.T) {
	mm := NewMiddlewareManager()

	for i := 0; i < 5; i++ {
		tool := &ToolValue{
			Name:       "testMiddleware",
			Parameters: []string{"ctx", "next"},
		}
		mm.Use(tool, nil)
	}
	assert.Equal(t, 5, mm.Len())

	mm.Clear()
	assert.Equal(t, 0, mm.Len())
}

func TestMiddlewareManager_ExecuteChain_NoMiddleware(t *testing.T) {
	mm := NewMiddlewareManager()
	interp := New(nil)

	result, err := mm.ExecuteChain("echo hello", interp)
	require.NoError(t, err)
	assert.False(t, result.Handled)
	assert.Equal(t, "echo hello", result.Input)
}

func TestMiddlewareManager_ExecuteChain_MiddlewareHandles(t *testing.T) {
	interp := New(nil)

	// Parse and evaluate a middleware tool that handles all input
	code := `
tool handleAllMiddleware(ctx, next) {
	return { handled: true }
}
`
	_, err := interp.EvalString(code, nil)
	require.NoError(t, err)

	toolVal, ok := interp.env.Get("handleAllMiddleware")
	require.True(t, ok)
	tool, ok := toolVal.(*ToolValue)
	require.True(t, ok)

	mm := NewMiddlewareManager()
	mm.Use(tool, interp)

	result, err := mm.ExecuteChain("any input", interp)
	require.NoError(t, err)
	assert.True(t, result.Handled)
}

func TestMiddlewareManager_ExecuteChain_MiddlewarePassesThrough(t *testing.T) {
	interp := New(nil)

	// Parse and evaluate a middleware tool that passes through
	code := `
tool passThroughMiddleware(ctx, next) {
	return next(ctx)
}
`
	_, err := interp.EvalString(code, nil)
	require.NoError(t, err)

	toolVal, ok := interp.env.Get("passThroughMiddleware")
	require.True(t, ok)
	tool, ok := toolVal.(*ToolValue)
	require.True(t, ok)

	mm := NewMiddlewareManager()
	mm.Use(tool, interp)

	result, err := mm.ExecuteChain("echo hello", interp)
	require.NoError(t, err)
	assert.False(t, result.Handled)
	assert.Equal(t, "echo hello", result.Input)
}

func TestMiddlewareManager_ExecuteChain_MiddlewareModifiesInput(t *testing.T) {
	interp := New(nil)

	// Parse and evaluate a middleware tool that modifies input
	code := `
tool modifyInputMiddleware(ctx, next) {
	ctx.input = "modified: " + ctx.input
	return next(ctx)
}
`
	_, err := interp.EvalString(code, nil)
	require.NoError(t, err)

	toolVal, ok := interp.env.Get("modifyInputMiddleware")
	require.True(t, ok)
	tool, ok := toolVal.(*ToolValue)
	require.True(t, ok)

	mm := NewMiddlewareManager()
	mm.Use(tool, interp)

	result, err := mm.ExecuteChain("original", interp)
	require.NoError(t, err)
	assert.False(t, result.Handled)
	assert.Equal(t, "modified: original", result.Input)
}

func TestMiddlewareManager_ExecuteChain_ChainOrder(t *testing.T) {
	interp := New(nil)

	// Create two middleware that append to input to verify order
	code := `
tool first(ctx, next) {
	ctx.input = ctx.input + " -> first"
	return next(ctx)
}

tool second(ctx, next) {
	ctx.input = ctx.input + " -> second"
	return next(ctx)
}
`
	_, err := interp.EvalString(code, nil)
	require.NoError(t, err)

	firstVal, _ := interp.env.Get("first")
	first, _ := firstVal.(*ToolValue)
	secondVal, _ := interp.env.Get("second")
	second, _ := secondVal.(*ToolValue)

	mm := NewMiddlewareManager()
	mm.Use(first, interp)
	mm.Use(second, interp)

	result, err := mm.ExecuteChain("start", interp)
	require.NoError(t, err)
	assert.False(t, result.Handled)
	// First registered = first to run
	assert.Equal(t, "start -> first -> second", result.Input)
}

func TestMiddlewareManager_ExecuteChain_ChainStopsOnHandle(t *testing.T) {
	interp := New(nil)

	code := `
__secondCalled = false

tool first(ctx, next) {
	return { handled: true }
}

tool second(ctx, next) {
	__secondCalled = true
	return next(ctx)
}
`
	_, err := interp.EvalString(code, nil)
	require.NoError(t, err)

	firstVal, _ := interp.env.Get("first")
	first, _ := firstVal.(*ToolValue)
	secondVal, _ := interp.env.Get("second")
	second, _ := secondVal.(*ToolValue)

	mm := NewMiddlewareManager()
	mm.Use(first, interp)
	mm.Use(second, interp)

	result, err := mm.ExecuteChain("input", interp)
	require.NoError(t, err)
	assert.True(t, result.Handled)

	// Verify second middleware was NOT called
	secondCalledVal, _ := interp.env.Get("__secondCalled")
	secondCalled, _ := secondCalledVal.(*BoolValue)
	assert.False(t, secondCalled.Value)
}

func TestMiddlewareManager_ExecuteChain_ConditionalHandling(t *testing.T) {
	interp := New(nil)

	code := `
tool conditionalMiddleware(ctx, next) {
	if (ctx.input.startsWith("#")) {
		return { handled: true }
	}
	return next(ctx)
}
`
	_, err := interp.EvalString(code, nil)
	require.NoError(t, err)

	toolVal, _ := interp.env.Get("conditionalMiddleware")
	tool, _ := toolVal.(*ToolValue)

	mm := NewMiddlewareManager()
	mm.Use(tool, interp)

	// Input starting with # should be handled
	result, err := mm.ExecuteChain("# hello", interp)
	require.NoError(t, err)
	assert.True(t, result.Handled)

	// Input not starting with # should pass through
	result, err = mm.ExecuteChain("echo hello", interp)
	require.NoError(t, err)
	assert.False(t, result.Handled)
	assert.Equal(t, "echo hello", result.Input)
}
