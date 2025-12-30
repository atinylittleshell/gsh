package repl

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// ErrExit is returned when the user requests to exit the REPL.
var ErrExit = fmt.Errorf("exit requested")

// handleAgentCommand handles agent chat commands (prefixed with '#').
func (r *REPL) handleAgentCommand(ctx context.Context, input string) error {
	// Check if any agents are configured
	if !r.agentManager.HasAgents() {
		fmt.Fprintf(os.Stderr, "gsh: no agents configured. Configure defaultAgentModel in .gshrc.gsh or add custom agents\n")
		return nil
	}

	// Parse input to determine if it's a command or message
	isCommand, content := parseAgentInput(input)

	if isCommand {
		// Handle agent commands
		return r.handleAgentCommandAction(content)
	}

	// Handle empty message
	if strings.TrimSpace(content) == "" {
		fmt.Println("Agent mode: type your message after # to chat with the current agent.")
		fmt.Println("Commands:")
		fmt.Println("  # /clear        - clear current agent's conversation")
		fmt.Println("  # /agents       - list all available agents")
		fmt.Println("  # /agent <name> - switch to a different agent")
		return nil
	}

	// Send message to current agent
	return r.agentManager.SendMessageToCurrentAgent(ctx, content)
}

// parseAgentInput parses input after the "#" prefix.
// Returns isCommand (true if input is a command starting with "/"),
// and the command/message content.
func parseAgentInput(input string) (isCommand bool, content string) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "/") {
		return true, trimmed[1:] // Remove "/" prefix
	}
	return false, input // Keep original spacing for messages
}

// handleAgentCommandAction handles agent commands (/clear, /agents, /agent).
func (r *REPL) handleAgentCommandAction(commandLine string) error {
	// Split command and arguments
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		fmt.Fprintf(os.Stderr, "gsh: empty command\n")
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "clear":
		return r.handleClearCommand()
	case "agents":
		return r.handleAgentsCommand()
	case "agent":
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "gsh: /agent command requires an agent name\n")
			return nil
		}
		return r.handleSwitchAgentCommand(args[0])
	default:
		fmt.Fprintf(os.Stderr, "gsh: unknown command: /%s. Try /agents or /clear\n", cmd)
		return nil
	}
}

// handleClearCommand clears the current agent's conversation.
func (r *REPL) handleClearCommand() error {
	if err := r.agentManager.ClearCurrentConversation(); err != nil {
		fmt.Fprintf(os.Stderr, "gsh: %v\n", err)
		return nil
	}
	fmt.Println("Conversation cleared")
	return nil
}

// handleAgentsCommand lists all available agents.
func (r *REPL) handleAgentsCommand() error {
	if !r.agentManager.HasAgents() {
		fmt.Println("No agents configured.")
		return nil
	}

	currentName := r.agentManager.CurrentAgentName()
	fmt.Println("Available agents:")
	for name, state := range r.agentManager.AllStates() {
		marker := " "
		if name == currentName {
			marker = "•"
		}

		msgCount := len(state.Conversation)
		status := fmt.Sprintf("(%d messages)", msgCount)
		if name == currentName {
			status = fmt.Sprintf("(current, %d messages)", msgCount)
		}

		// Try to get description from agent config
		description := ""
		if name == "default" {
			description = " - Built-in default agent"
		} else if descVal, ok := state.Agent.Config["description"]; ok {
			if descStr, ok := descVal.(*interpreter.StringValue); ok {
				description = " - " + descStr.Value
			}
		}

		fmt.Printf("  %s %-12s %s%s\n", marker, name, status, description)
	}
	return nil
}

// handleSwitchAgentCommand switches to a different agent.
func (r *REPL) handleSwitchAgentCommand(agentName string) error {
	// Check if agent exists and switch to it
	if err := r.agentManager.SetCurrentAgent(agentName); err != nil {
		fmt.Fprintf(os.Stderr, "gsh: agent '%s' not found. Use /agents to see available agents\n", agentName)
		return nil
	}

	// Get the state to show message count
	state := r.agentManager.GetAgent(agentName)
	msgCount := len(state.Conversation)
	if msgCount > 0 {
		fmt.Printf("Switched to agent '%s' (%d messages in history)\n", agentName, msgCount)
	} else {
		fmt.Printf("Switched to agent '%s'\n", agentName)
	}
	return nil
}

// handleBuiltinCommand handles built-in REPL commands.
// Returns true if the command was handled, and an error if the REPL should exit.
func (r *REPL) handleBuiltinCommand(command string) (bool, error) {
	switch command {
	case "exit":
		// Signal exit by returning ErrExit
		return true, ErrExit

	default:
		return false, nil
	}
}
