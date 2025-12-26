package render

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRenderer(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()
	termWidth := func() int { return 80 }

	renderer := New(interp, &buf, termWidth)

	assert.NotNil(t, renderer)
	assert.Equal(t, interp, renderer.interp)
	assert.Equal(t, &buf, renderer.writer)
}

func TestRenderAgentHeader_Fallback(t *testing.T) {
	var buf bytes.Buffer
	// No interpreter - should use fallback
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderAgentHeader("test-agent")

	output := buf.String()
	assert.Contains(t, output, "agent: test-agent")
}

func TestRenderAgentHeader_WithHook(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	// Define a custom hook
	_, err := interp.EvalString(`
		tool GSH_AGENT_HEADER(agentName: string, terminalWidth: number): string {
			return "== " + agentName + " =="
		}
	`)
	require.NoError(t, err)

	renderer := New(interp, &buf, func() int { return 80 })
	renderer.RenderAgentHeader("custom")

	output := buf.String()
	assert.Contains(t, output, "== custom ==")
}

func TestRenderAgentFooter_Fallback(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderAgentFooter(100, 50, 2*time.Second)

	output := buf.String()
	assert.Contains(t, output, "100")
	assert.Contains(t, output, "50")
}

func TestRenderAgentFooter_WithHook(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	_, err := interp.EvalString(`
		tool GSH_AGENT_FOOTER(inputTokens: number, outputTokens: number, durationMs: number, terminalWidth: number): string {
			return "tokens: " + inputTokens + "/" + outputTokens
		}
	`)
	require.NoError(t, err)

	renderer := New(interp, &buf, func() int { return 80 })
	renderer.RenderAgentFooter(100, 50, time.Second)

	output := buf.String()
	assert.Contains(t, output, "tokens: 100/50")
}

func TestStartThinkingSpinner(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	ctx := context.Background()
	cancel := renderer.StartThinkingSpinner(ctx)

	// Let spinner run briefly
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)

	output := buf.String()
	assert.Contains(t, output, "Thinking...")
}

func TestRenderAgentText(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderAgentText("Hello, world!")

	assert.Equal(t, "Hello, world!", buf.String())
}

func TestRenderExecStart_Fallback(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderExecStart("ls -la")

	output := buf.String()
	assert.Contains(t, output, SymbolExec)
	assert.Contains(t, output, "ls -la")
}

func TestRenderExecStart_WithHook(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	_, err := interp.EvalString(`
		tool GSH_EXEC_START(command: string): string {
			return "▶ running: " + command
		}
	`)
	require.NoError(t, err)

	renderer := New(interp, &buf, func() int { return 80 })
	renderer.RenderExecStart("echo hello")

	output := buf.String()
	assert.Contains(t, output, "running: echo hello")
}

func TestRenderExecEnd_Success(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderExecEnd("ls -la", 100*time.Millisecond, 0)

	output := buf.String()
	assert.Contains(t, output, SymbolSuccess)
	assert.Contains(t, output, "ls") // First word of command
}

func TestRenderExecEnd_Failure(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderExecEnd("cat /nonexistent", 50*time.Millisecond, 1)

	output := buf.String()
	assert.Contains(t, output, SymbolError)
	assert.Contains(t, output, "cat")
	assert.Contains(t, output, "exit code 1")
}

func TestRenderExecEnd_WithHook(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	_, err := interp.EvalString(`
		tool GSH_EXEC_END(commandFirstWord: string, durationMs: number, exitCode: number): string {
			if (exitCode == 0) {
				return "✓ " + commandFirstWord + " ok"
			}
			return "✗ " + commandFirstWord + " failed"
		}
	`)
	require.NoError(t, err)

	renderer := New(interp, &buf, func() int { return 80 })
	renderer.RenderExecEnd("echo hello", time.Second, 0)

	output := buf.String()
	assert.Contains(t, output, "echo ok")
}

func TestRenderToolPending_Fallback(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderToolPending("read_file")

	output := buf.String()
	assert.Contains(t, output, SymbolToolPending)
	assert.Contains(t, output, "read_file")
}

func TestRenderToolExecuting_Fallback(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	args := map[string]interface{}{
		"path": "/home/user/file.txt",
	}
	renderer.RenderToolExecuting("read_file", args)

	output := buf.String()
	assert.Contains(t, output, SymbolToolPending)
	assert.Contains(t, output, "read_file")
	assert.Contains(t, output, "path")
}

func TestRenderToolComplete_Success(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	args := map[string]interface{}{
		"path": "/home/user/file.txt",
	}
	renderer.RenderToolComplete("read_file", args, 100*time.Millisecond, true)

	output := buf.String()
	assert.Contains(t, output, SymbolToolComplete)
	assert.Contains(t, output, "read_file")
	assert.Contains(t, output, SymbolSuccess)
}

func TestRenderToolComplete_Error(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	args := map[string]interface{}{
		"path": "/missing.txt",
	}
	renderer.RenderToolComplete("read_file", args, 50*time.Millisecond, false)

	output := buf.String()
	assert.Contains(t, output, SymbolToolComplete)
	assert.Contains(t, output, "read_file")
	assert.Contains(t, output, SymbolError)
}

func TestRenderToolOutput_DefaultEmpty(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	// Define default hook that returns empty
	_, err := interp.EvalString(`
		tool GSH_TOOL_OUTPUT(toolName: string, output: string, terminalWidth: number): string {
			return ""
		}
	`)
	require.NoError(t, err)

	renderer := New(interp, &buf, func() int { return 80 })
	renderer.RenderToolOutput("test", "some output")

	// Should be empty since hook returns empty
	assert.Empty(t, buf.String())
}

func TestRenderToolOutput_CustomHook(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	_, err := interp.EvalString(`
		tool GSH_TOOL_OUTPUT(toolName: string, output: string, terminalWidth: number): string {
			return "Output: " + output
		}
	`)
	require.NoError(t, err)

	renderer := New(interp, &buf, func() int { return 80 })
	renderer.RenderToolOutput("test", "hello")

	output := buf.String()
	assert.Contains(t, output, "Output: hello")
}

func TestRenderSystemMessage(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	renderer.RenderSystemMessage("Connecting to server...")

	output := buf.String()
	assert.Contains(t, output, SymbolSystemMessage)
	assert.Contains(t, output, "Connecting to server...")
}

func TestStartToolSpinner(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	ctx := context.Background()
	cancel := renderer.StartToolSpinner(ctx, "search")

	// Let spinner run briefly
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)

	output := buf.String()
	assert.Contains(t, output, "search")
}

func TestGetTerminalWidth_Default(t *testing.T) {
	var buf bytes.Buffer
	// No termWidth function
	renderer := New(nil, &buf, nil)

	width := renderer.getTerminalWidth()
	assert.Equal(t, 80, width) // Default fallback
}

func TestGetTerminalWidth_Custom(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 120 })

	width := renderer.getTerminalWidth()
	assert.Equal(t, 120, width)
}

func TestGetTerminalWidth_ZeroReturnsDefault(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 0 })

	width := renderer.getTerminalWidth()
	assert.Equal(t, 80, width) // Fallback when function returns 0
}

func TestGetVariable(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	_, err := interp.EvalString(`testVar = "hello"`)
	require.NoError(t, err)

	renderer := New(interp, &buf, func() int { return 80 })
	val := renderer.GetVariable("testVar")

	require.NotNil(t, val)
	strVal, ok := val.(*interpreter.StringValue)
	require.True(t, ok)
	assert.Equal(t, "hello", strVal.Value)
}

func TestGetVariable_NotFound(t *testing.T) {
	var buf bytes.Buffer
	interp := interpreter.New()

	renderer := New(interp, &buf, func() int { return 80 })
	val := renderer.GetVariable("nonexistent")

	assert.Nil(t, val)
}

func TestGetVariable_NilInterpreter(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	val := renderer.GetVariable("anything")
	assert.Nil(t, val)
}

func TestToInterpreterValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType interpreter.ValueType
	}{
		{"nil", nil, interpreter.ValueTypeNull},
		{"string", "hello", interpreter.ValueTypeString},
		{"float64", 3.14, interpreter.ValueTypeNumber},
		{"int", 42, interpreter.ValueTypeNumber},
		{"int64", int64(100), interpreter.ValueTypeNumber},
		{"bool true", true, interpreter.ValueTypeBool},
		{"bool false", false, interpreter.ValueTypeBool},
		{"map", map[string]interface{}{"key": "value"}, interpreter.ValueTypeObject},
		{"slice", []interface{}{"a", "b"}, interpreter.ValueTypeArray},
		{"unknown", struct{}{}, interpreter.ValueTypeString}, // Falls back to string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toInterpreterValue(tt.input)
			assert.Equal(t, tt.wantType, result.Type())
		})
	}
}

func TestFormatArgs_Truncation(t *testing.T) {
	var buf bytes.Buffer
	renderer := New(nil, &buf, func() int { return 80 })

	// Create a long value that should be truncated
	longValue := "This is a very long string that exceeds sixty characters and should be truncated"
	args := map[string]interface{}{
		"content": longValue,
	}

	result := renderer.formatArgs(args)
	assert.Contains(t, result, "...")
	assert.Contains(t, result, "content:")
}
