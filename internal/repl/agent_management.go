package repl

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/atinylittleshell/gsh/internal/repl/agent"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// handleAgentAddedFromSDK is called when an agent is added via gsh.repl.agents.push()
func (r *REPL) handleAgentAddedFromSDK(newAgent *interpreter.AgentValue) {
	if newAgent == nil {
		r.logger.Warn("cannot add nil agent")
		return
	}

	// Get the model from the agent's config
	modelVal, ok := newAgent.Config["model"]
	if !ok {
		r.logger.Warn("cannot add agent without model", zap.String("agent", newAgent.Name))
		return
	}
	model, ok := modelVal.(*interpreter.ModelValue)
	if !ok || model.Provider == nil {
		r.logger.Warn("agent model has no provider", zap.String("agent", newAgent.Name))
		return
	}

	// Create agent state using the AgentValue directly
	state := &agent.State{
		Agent:        newAgent,
		Provider:     model.Provider,
		Conversation: []interpreter.ChatMessage{},
		Interpreter:  r.executor.Interpreter(),
	}

	// Convert tools from Config["tools"] to ChatTool if present
	if toolsVal, ok := newAgent.Config["tools"]; ok {
		if toolsArr, ok := toolsVal.(*interpreter.ArrayValue); ok && len(toolsArr.Elements) > 0 {
			state.Tools = make([]interpreter.ChatTool, 0, len(toolsArr.Elements))
			for _, toolVal := range toolsArr.Elements {
				if chatTool := valueToTool(toolVal); chatTool != nil {
					state.Tools = append(state.Tools, *chatTool)
				}
			}
		}
	}

	// Set up default tools if none provided
	if len(state.Tools) == 0 {
		agent.SetupAgentWithDefaultTools(state)
	} else {
		// Still need tool executor
		state.ToolExecutor = agent.DefaultToolExecutor(os.Stdout)
	}

	r.agentManager.AddAgent(newAgent.Name, state)
	r.logger.Info("added agent from SDK", zap.String("agent", newAgent.Name))
}

// handleAgentSwitchFromSDK is called when gsh.repl.currentAgent is set
func (r *REPL) handleAgentSwitchFromSDK(switchedAgent *interpreter.AgentValue) {
	if switchedAgent == nil {
		return
	}
	if err := r.agentManager.SetCurrentAgent(switchedAgent.Name); err != nil {
		r.logger.Warn("failed to switch agent from SDK", zap.String("agent", switchedAgent.Name), zap.Error(err))
	}
}

// handleAgentModifiedFromSDK is called when an agent's properties are modified via SDK
func (r *REPL) handleAgentModifiedFromSDK(modifiedAgent *interpreter.AgentValue) {
	if modifiedAgent == nil {
		return
	}

	// Get the existing state for this agent
	state := r.agentManager.GetAgent(modifiedAgent.Name)
	if state == nil {
		r.logger.Warn("modified agent not found in manager", zap.String("agent", modifiedAgent.Name))
		return
	}

	// Sync model/provider if changed
	if modelVal, ok := modifiedAgent.Config["model"]; ok {
		if model, ok := modelVal.(*interpreter.ModelValue); ok && model.Provider != nil {
			state.Provider = model.Provider
		}
	}

	// Sync tools if changed
	if toolsVal, ok := modifiedAgent.Config["tools"]; ok {
		if toolsArr, ok := toolsVal.(*interpreter.ArrayValue); ok {
			state.Tools = make([]interpreter.ChatTool, 0, len(toolsArr.Elements))
			for _, toolVal := range toolsArr.Elements {
				if chatTool := valueToTool(toolVal); chatTool != nil {
					state.Tools = append(state.Tools, *chatTool)
				}
			}
		}
	}

	r.logger.Debug("synced agent modifications from SDK", zap.String("agent", modifiedAgent.Name))
}

// valueToTool converts a Value (ToolValue or NativeToolValue) to a ChatTool
func valueToTool(v interpreter.Value) *interpreter.ChatTool {
	switch tool := v.(type) {
	case *interpreter.NativeToolValue:
		return &interpreter.ChatTool{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		}
	case *interpreter.ToolValue:
		// For script-defined tools, create a ChatTool with the tool's parameter info
		params := make(map[string]interface{})
		props := make(map[string]interface{})
		required := make([]string, 0)

		for _, param := range tool.Parameters {
			props[param] = map[string]interface{}{
				"type":        "string",
				"description": param,
			}
			required = append(required, param)
		}
		params["type"] = "object"
		params["properties"] = props
		params["required"] = required

		return &interpreter.ChatTool{
			Name:        tool.Name,
			Description: fmt.Sprintf("Script tool: %s", tool.Name),
			Parameters:  params,
		}
	default:
		return nil
	}
}

// GetAgentNames returns all configured agent names for completion.
func (r *REPL) GetAgentNames() []string {
	return r.agentManager.GetAgentNames()
}

// GetAgentCommands returns the list of valid agent commands for completion.
func (r *REPL) GetAgentCommands() []string {
	return AgentCommands
}
