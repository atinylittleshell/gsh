package repl

import (
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

	// Set up default tools if none provided in agent config
	if _, ok := newAgent.Config["tools"]; !ok {
		agent.SetupAgentWithDefaultTools(state)
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

	// Tools are now stored in agent.Config["tools"] and don't need syncing here
	// The agent config is already the source of truth

	r.logger.Debug("synced agent modifications from SDK", zap.String("agent", modifiedAgent.Name))
}

// GetAgentNames returns all configured agent names for completion.
func (r *REPL) GetAgentNames() []string {
	return r.agentManager.GetAgentNames()
}

// GetAgentCommands returns the list of valid agent commands for completion.
func (r *REPL) GetAgentCommands() []string {
	return AgentCommands
}
