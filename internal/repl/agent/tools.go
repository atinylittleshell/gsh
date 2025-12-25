// Package agent provides agent state management and messaging functionality for the REPL.
package agent

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// DefaultToolExecutor creates a tool executor function that handles the built-in tools.
// It writes live output to the provided writer (typically os.Stdout).
func DefaultToolExecutor(liveOutput io.Writer) ToolExecutor {
	return func(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
		switch toolName {
		case "exec":
			return ExecuteExecTool(ctx, args, liveOutput)
		default:
			return "", fmt.Errorf("unknown tool: %s", toolName)
		}
	}
}

// DefaultTools returns the default set of tools available to agents.
func DefaultTools() []interpreter.ChatTool {
	return []interpreter.ChatTool{
		ExecToolDefinition(),
	}
}

// SetupAgentWithDefaultTools configures an agent state with the default tools and executor.
// This is a convenience function for setting up agents with default tool support.
func SetupAgentWithDefaultTools(state *State) {
	if state.Tools == nil {
		state.Tools = DefaultTools()
	}
	if state.ToolExecutor == nil {
		state.ToolExecutor = DefaultToolExecutor(os.Stdout)
	}
}
