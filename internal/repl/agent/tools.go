// Package agent provides agent state management and messaging functionality for the REPL.
package agent

import (
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// SetupAgentWithDefaultTools configures an agent state with the default tools.
// This is a convenience function for setting up agents with default tool support.
func SetupAgentWithDefaultTools(state *State) {
	// Put default tools in the agent config
	toolValues := make([]interpreter.Value, 0, 4)
	toolValues = append(toolValues,
		interpreter.CreateExecNativeTool(),
		interpreter.CreateGrepNativeTool(),
		interpreter.CreateViewFileNativeTool(),
		interpreter.CreateEditFileNativeTool(),
	)
	state.Agent.Config["tools"] = &interpreter.ArrayValue{Elements: toolValues}
}
