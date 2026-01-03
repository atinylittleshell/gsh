package interpreter

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/acp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// evalACPDeclaration evaluates an ACP (Agent Client Protocol) agent declaration.
// ACP agents connect to external agent processes via the ACP protocol.
func (i *Interpreter) evalACPDeclaration(node *parser.ACPDeclaration) (Value, error) {
	acpName := node.Name.Value

	// Evaluate each config field and store as Value
	config := make(map[string]Value)

	for key, expr := range node.Config {
		value, err := i.evalExpression(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate acp config field '%s': %w", key, err)
		}

		// Validate common config fields
		switch key {
		case "command":
			if _, ok := value.(*StringValue); !ok {
				return nil, fmt.Errorf("acp config 'command' must be a string, got %s", value.Type())
			}
		case "args":
			if _, ok := value.(*ArrayValue); !ok {
				return nil, fmt.Errorf("acp config 'args' must be an array, got %s", value.Type())
			}
		case "env":
			if _, ok := value.(*ObjectValue); !ok {
				return nil, fmt.Errorf("acp config 'env' must be an object, got %s", value.Type())
			}
		case "cwd":
			if _, ok := value.(*StringValue); !ok {
				return nil, fmt.Errorf("acp config 'cwd' must be a string, got %s", value.Type())
			}
		case "mcpServers":
			if _, ok := value.(*ArrayValue); !ok {
				return nil, fmt.Errorf("acp config 'mcpServers' must be an array, got %s", value.Type())
			}
			// Allow other fields without validation for extensibility
		}

		config[key] = value
	}

	// Validate required fields
	if _, ok := config["command"]; !ok {
		return nil, fmt.Errorf("acp '%s' must have a 'command' field", acpName)
	}

	// Create the ACP value
	acpVal := &ACPValue{
		Name:   acpName,
		Config: config,
	}

	// Register the ACP agent in the environment
	i.env.Set(acpName, acpVal)

	return acpVal, nil
}

// executeACPWithString creates a new ACP session with a user message and sends the prompt.
// This is called when: "Hello" | ACPAgent
func (i *Interpreter) executeACPWithString(message string, acpVal *ACPValue) (Value, error) {
	startTime := time.Now()

	// Emit agent.start event
	i.EmitEvent(EventAgentStart, createACPAgentStartContext(acpVal.Name, message))

	// Emit agent.iteration.start event immediately to show the "Thinking..." spinner
	// This needs to happen before getOrCreateACPClient which may take time to spawn the ACP server
	i.EmitEvent(EventAgentIterationStart, createIterationStartContext(1))

	// Get or create ACP client
	client, err := i.getOrCreateACPClient(acpVal)
	if err != nil {
		i.EmitEvent(EventAgentEnd, createACPAgentEndContext(acpVal.Name, "error", time.Since(startTime).Milliseconds(), err))
		return nil, fmt.Errorf("failed to connect to ACP agent '%s': %w", acpVal.Name, err)
	}

	// Get working directory for session
	cwd := i.GetWorkingDir()
	if cwdVal, ok := acpVal.Config["cwd"]; ok {
		if cwdStr, ok := cwdVal.(*StringValue); ok {
			cwd = cwdStr.Value
		}
	}

	// Create new session
	ctx := context.Background()
	acpSession, err := client.NewSession(ctx, cwd, nil) // TODO: Support MCP servers
	if err != nil {
		i.EmitEvent(EventAgentEnd, createACPAgentEndContext(acpVal.Name, "error", time.Since(startTime).Milliseconds(), err))
		return nil, fmt.Errorf("failed to create ACP session: %w", err)
	}

	// Store session in client entry
	i.acpClientsMu.Lock()
	if entry, ok := i.acpClients[acpVal.Name]; ok {
		entry.sessions[acpSession.SessionID()] = acpSession
	}
	i.acpClientsMu.Unlock()

	// Create the session value
	session := &ACPSessionValue{
		Agent:     acpVal,
		Messages:  []ChatMessage{},
		SessionID: acpSession.SessionID(),
		Closed:    false,
	}

	// Send the prompt to the session
	return i.sendPromptToACPSessionInternal(session, acpSession, message, startTime)
}

// sendPromptToACPSession sends a prompt to an existing ACP session.
// This is called when: session | "Follow up message"
func (i *Interpreter) sendPromptToACPSession(session *ACPSessionValue, message string) (Value, error) {
	// Check if session is closed
	if session.Closed {
		return nil, fmt.Errorf("cannot send prompt to closed ACP session")
	}

	startTime := time.Now()

	// Emit agent.start event
	i.EmitEvent(EventAgentStart, createACPAgentStartContext(session.Agent.Name, message))

	// Emit agent.iteration.start event immediately to show the "Thinking..." spinner
	i.EmitEvent(EventAgentIterationStart, createIterationStartContext(1))

	// Get the ACP session from our cache
	i.acpClientsMu.RLock()
	entry, ok := i.acpClients[session.Agent.Name]
	if !ok {
		i.acpClientsMu.RUnlock()
		err := fmt.Errorf("ACP client for '%s' not found", session.Agent.Name)
		i.EmitEvent(EventAgentEnd, createACPAgentEndContext(session.Agent.Name, "error", time.Since(startTime).Milliseconds(), err))
		return nil, err
	}
	acpSession, ok := entry.sessions[session.SessionID]
	if !ok {
		i.acpClientsMu.RUnlock()
		err := fmt.Errorf("ACP session '%s' not found", session.SessionID)
		i.EmitEvent(EventAgentEnd, createACPAgentEndContext(session.Agent.Name, "error", time.Since(startTime).Milliseconds(), err))
		return nil, err
	}
	i.acpClientsMu.RUnlock()

	return i.sendPromptToACPSessionInternal(session, acpSession, message, startTime)
}

// sendPromptToACPSessionInternal sends a prompt and handles the response with event emission.
func (i *Interpreter) sendPromptToACPSessionInternal(session *ACPSessionValue, acpSession acp.ACPSession, message string, startTime time.Time) (Value, error) {
	ctx := context.Background()

	// Track tool calls for event emission
	toolCallStarts := make(map[string]time.Time)

	// Send prompt with update callback
	result, err := acpSession.SendPrompt(ctx, message, func(update *acp.SessionUpdateParams) {
		i.handleACPSessionUpdate(session, update, toolCallStarts)
	})

	if err != nil {
		i.EmitEvent(EventAgentEnd, createACPAgentEndContext(session.Agent.Name, "error", time.Since(startTime).Milliseconds(), err))
		return nil, fmt.Errorf("ACP prompt failed: %w", err)
	}

	// Update local message history from the ACP session
	acpMessages := acpSession.GetMessages()
	session.Messages = make([]ChatMessage, len(acpMessages))
	for i, msg := range acpMessages {
		session.Messages[i] = ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}
		// Convert tool calls if present
		if len(msg.ToolCalls) > 0 {
			session.Messages[i].ToolCalls = make([]ChatToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				session.Messages[i].ToolCalls[j] = ChatToolCall{
					ID:        tc.ID,
					Name:      tc.Name,
					Arguments: tc.Arguments,
				}
			}
		}
	}

	// Emit agent.end event
	stopReason := "end_turn"
	if result != nil && result.StopReason != "" {
		stopReason = result.StopReason
	}
	i.EmitEvent(EventAgentEnd, createACPAgentEndContext(session.Agent.Name, stopReason, time.Since(startTime).Milliseconds(), nil))

	return session, nil
}

// handleACPSessionUpdate processes ACP session updates and emits corresponding events.
func (i *Interpreter) handleACPSessionUpdate(session *ACPSessionValue, update *acp.SessionUpdateParams, toolCallStarts map[string]time.Time) {
	switch update.Update.SessionUpdate {
	case acp.SessionUpdateAgentMessageChunk:
		// Emit agent.chunk event
		if content := update.Update.GetMessageContent(); content != nil && content.Type == "text" {
			i.EmitEvent(EventAgentChunk, createChunkContext(content.Text))
		}

	case acp.SessionUpdateToolCall:
		// Tool call started - emit agent.tool.pending and agent.tool.start events
		toolCallID := update.Update.ToolCallID
		toolName := update.Update.GetToolName()

		// Emit pending event (tool call detected, may not have full args yet)
		i.EmitEvent(EventAgentToolPending, createToolPendingContext(toolCallID, toolName))

		// Emit start event with arguments
		toolCallStarts[toolCallID] = time.Now()
		i.EmitEvent(EventAgentToolStart, createToolCallContext(toolCallID, toolName, update.Update.Arguments, nil, nil, nil))

	case acp.SessionUpdateToolCallUpdate:
		// Tool call update - check status and emit agent.tool.end if completed/failed
		toolCallID := update.Update.ToolCallID
		status := update.Update.Status

		if status == "completed" || status == "failed" {
			toolName := update.Update.GetToolName()
			var durationMs int64
			if start, ok := toolCallStarts[toolCallID]; ok {
				durationMs = time.Since(start).Milliseconds()
				delete(toolCallStarts, toolCallID)
			}

			// Get output from content if available
			var output string
			if contents := update.Update.GetToolCallContent(); len(contents) > 0 {
				for _, c := range contents {
					if c.Content != nil && c.Content.Type == "text" {
						output += c.Content.Text
					}
				}
			}

			var toolErr error
			if status == "failed" {
				toolErr = fmt.Errorf("tool call failed")
			}

			i.EmitEvent(EventAgentToolEnd, createToolCallContext(toolCallID, toolName, nil, &durationMs, &output, toolErr))
		}
	}
}

// getOrCreateACPClient gets an existing ACP client or creates a new one.
func (i *Interpreter) getOrCreateACPClient(acpVal *ACPValue) (*acp.Client, error) {
	i.acpClientsMu.Lock()
	defer i.acpClientsMu.Unlock()

	// Check if client already exists and is connected
	if entry, ok := i.acpClients[acpVal.Name]; ok {
		if entry.client.IsConnected() {
			return entry.client, nil
		}
		// Client exists but disconnected, close it and create new one
		entry.client.Close()
	}

	// Extract config from ACPValue
	command := ""
	if cmdVal, ok := acpVal.Config["command"]; ok {
		if cmdStr, ok := cmdVal.(*StringValue); ok {
			command = cmdStr.Value
		}
	}

	// Resolve command to absolute path using shell's PATH resolution
	// This ensures we find executables even if PATH was modified after gsh started
	if command != "" && !strings.Contains(command, string(os.PathSeparator)) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stdout, _, exitCode, err := i.executeBashInSubshell(ctx, fmt.Sprintf("command -v %s", command))
		if err == nil && exitCode == 0 {
			resolved := strings.TrimSpace(stdout)
			if resolved != "" {
				command = resolved
			}
		}
	}

	var args []string
	if argsVal, ok := acpVal.Config["args"]; ok {
		if argsArr, ok := argsVal.(*ArrayValue); ok {
			for _, elem := range argsArr.Elements {
				if str, ok := elem.(*StringValue); ok {
					args = append(args, str.Value)
				}
			}
		}
	}

	env := make(map[string]string)
	if envVal, ok := acpVal.Config["env"]; ok {
		if envObj, ok := envVal.(*ObjectValue); ok {
			for key := range envObj.Properties {
				val := envObj.GetPropertyValue(key)
				if str, ok := val.(*StringValue); ok {
					env[key] = str.Value
				}
			}
		}
	}

	cwd := ""
	if cwdVal, ok := acpVal.Config["cwd"]; ok {
		if cwdStr, ok := cwdVal.(*StringValue); ok {
			cwd = cwdStr.Value
		}
	}

	// Create client config
	config := acp.ClientConfig{
		Command:     command,
		Args:        args,
		Env:         env,
		Cwd:         cwd,
		InitTimeout: 30 * time.Second,
	}

	// Create and connect client using the factory
	client, err := i.acpClientFactory(config)
	if err != nil {
		return nil, err
	}

	// Store client
	i.acpClients[acpVal.Name] = &acpClientEntry{
		client:   client,
		sessions: make(map[string]acp.ACPSession),
	}

	return client, nil
}

// closeACPSession closes an ACP session and cleans up resources.
func (i *Interpreter) closeACPSession(session *ACPSessionValue) error {
	if session.Closed {
		return nil
	}

	session.Closed = true

	// Remove session from client entry
	i.acpClientsMu.Lock()
	defer i.acpClientsMu.Unlock()

	if entry, ok := i.acpClients[session.Agent.Name]; ok {
		if acpSession, ok := entry.sessions[session.SessionID]; ok {
			acpSession.Close()
			delete(entry.sessions, session.SessionID)
		}
	}

	return nil
}

// ACPSessionMethodValue represents a method bound to an ACPSessionValue instance
type ACPSessionMethodValue struct {
	Name    string
	Session *ACPSessionValue
	Interp  *Interpreter
}

func (m *ACPSessionMethodValue) Type() ValueType { return ValueTypeTool }
func (m *ACPSessionMethodValue) String() string {
	return fmt.Sprintf("<acpsession method: %s>", m.Name)
}
func (m *ACPSessionMethodValue) IsTruthy() bool          { return true }
func (m *ACPSessionMethodValue) Equals(other Value) bool { return false }

// getACPSessionProperty returns ACPSession properties and methods
func (i *Interpreter) getACPSessionProperty(session *ACPSessionValue, property string) (Value, error) {
	switch property {
	case "close":
		return &ACPSessionMethodValue{Name: "close", Session: session, Interp: i}, nil
	case "messages":
		return session.GetProperty("messages"), nil
	case "lastMessage":
		return session.GetProperty("lastMessage"), nil
	case "agent":
		return session.GetProperty("agent"), nil
	case "sessionId":
		return session.GetProperty("sessionId"), nil
	case "closed":
		return session.GetProperty("closed"), nil
	default:
		return &NullValue{}, nil
	}
}

// callACPSessionMethod executes an ACPSession method
func (i *Interpreter) callACPSessionMethod(method *ACPSessionMethodValue, args []Value) (Value, error) {
	switch method.Name {
	case "close":
		if err := method.Interp.closeACPSession(method.Session); err != nil {
			return nil, err
		}
		return &NullValue{}, nil
	default:
		return nil, fmt.Errorf("unknown ACPSession method: %s", method.Name)
	}
}

// Helper functions for ACP-specific event contexts

// createACPAgentStartContext creates the context object for agent.start event for ACP agents
func createACPAgentStartContext(agentName, message string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"name": {Value: &StringValue{Value: agentName}},
					"type": {Value: &StringValue{Value: "acp"}},
				},
			}},
			"message": {Value: &StringValue{Value: message}},
		},
	}
}

// createACPAgentEndContext creates the context object for agent.end event for ACP agents
// Note: ACP agents don't provide token counts, so we set them to 0
func createACPAgentEndContext(agentName string, stopReason string, durationMs int64, err error) Value {
	var errorVal Value = &NullValue{}
	if err != nil {
		errorVal = &StringValue{Value: err.Error()}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"name": {Value: &StringValue{Value: agentName}},
					"type": {Value: &StringValue{Value: "acp"}},
				},
			}},
			"query": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"inputTokens":  {Value: &NumberValue{Value: 0}},
					"outputTokens": {Value: &NumberValue{Value: 0}},
					"cachedTokens": {Value: &NumberValue{Value: 0}},
					"durationMs":   {Value: &NumberValue{Value: float64(durationMs)}},
				},
			}},
			"stopReason": {Value: &StringValue{Value: stopReason}},
			"error":      {Value: errorVal},
		},
	}
}
