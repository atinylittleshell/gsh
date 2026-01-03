package acp

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNewInitializeRequest(t *testing.T) {
	req := NewInitializeRequest(1)

	if req.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", req.JSONRPC)
	}
	if req.ID != 1 {
		t.Errorf("expected ID 1, got %d", req.ID)
	}
	if req.Method != "initialize" {
		t.Errorf("expected method 'initialize', got %s", req.Method)
	}

	params, ok := req.Params.(InitializeParams)
	if !ok {
		t.Fatalf("expected params to be InitializeParams")
	}
	if params.ProtocolVersion != 1 {
		t.Errorf("expected protocol version 1, got %d", params.ProtocolVersion)
	}
}

func TestNewSessionNewRequest(t *testing.T) {
	mcpServers := []MCPServer{
		{
			Name:    "filesystem",
			Command: "/path/to/mcp-server",
			Args:    []string{"--stdio"},
		},
	}
	req := NewSessionNewRequest(2, "/home/user/project", mcpServers)

	if req.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", req.JSONRPC)
	}
	if req.ID != 2 {
		t.Errorf("expected ID 2, got %d", req.ID)
	}
	if req.Method != "session/new" {
		t.Errorf("expected method 'session/new', got %s", req.Method)
	}

	params, ok := req.Params.(SessionNewParams)
	if !ok {
		t.Fatalf("expected params to be SessionNewParams")
	}
	if params.Cwd != "/home/user/project" {
		t.Errorf("expected cwd '/home/user/project', got %s", params.Cwd)
	}
	if len(params.MCPServers) != 1 {
		t.Errorf("expected 1 MCP server, got %d", len(params.MCPServers))
	}
}

func TestNewSessionPromptRequest(t *testing.T) {
	req := NewSessionPromptRequest(3, "sess_abc123", "Hello, world!")

	if req.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", req.JSONRPC)
	}
	if req.ID != 3 {
		t.Errorf("expected ID 3, got %d", req.ID)
	}
	if req.Method != "session/prompt" {
		t.Errorf("expected method 'session/prompt', got %s", req.Method)
	}

	params, ok := req.Params.(SessionPromptParams)
	if !ok {
		t.Fatalf("expected params to be SessionPromptParams")
	}
	if params.SessionID != "sess_abc123" {
		t.Errorf("expected session ID 'sess_abc123', got %s", params.SessionID)
	}
	if len(params.Prompt) != 1 {
		t.Fatalf("expected 1 prompt content, got %d", len(params.Prompt))
	}
	if params.Prompt[0].Type != "text" {
		t.Errorf("expected prompt type 'text', got %s", params.Prompt[0].Type)
	}
	if params.Prompt[0].Text != "Hello, world!" {
		t.Errorf("expected prompt text 'Hello, world!', got %s", params.Prompt[0].Text)
	}
}

func TestJSONRPCRequestSerialization(t *testing.T) {
	req := NewInitializeRequest(0)

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Verify it can be unmarshaled back
	var parsed JSONRPCRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if parsed.JSONRPC != req.JSONRPC {
		t.Errorf("JSONRPC mismatch: %s != %s", parsed.JSONRPC, req.JSONRPC)
	}
	if parsed.ID != req.ID {
		t.Errorf("ID mismatch: %d != %d", parsed.ID, req.ID)
	}
	if parsed.Method != req.Method {
		t.Errorf("Method mismatch: %s != %s", parsed.Method, req.Method)
	}
}

func TestJSONRPCResponseParsing(t *testing.T) {
	responseJSON := `{
		"jsonrpc": "2.0",
		"id": 1,
		"result": {
			"protocolVersion": 1,
			"agentCapabilities": {
				"loadSession": true
			}
		}
	}`

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}
	if resp.ID == nil || *resp.ID != 1 {
		t.Errorf("expected ID 1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}

	// Parse the result
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result.ProtocolVersion != 1 {
		t.Errorf("expected protocol version 1, got %d", result.ProtocolVersion)
	}
	if !result.AgentCapabilities.LoadSession {
		t.Error("expected loadSession capability to be true")
	}
}

func TestJSONRPCErrorParsing(t *testing.T) {
	responseJSON := `{
		"jsonrpc": "2.0",
		"id": 1,
		"error": {
			"code": -32600,
			"message": "Invalid Request"
		}
	}`

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("expected error code -32600, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("expected error message 'Invalid Request', got %s", resp.Error.Message)
	}
}

func TestSessionUpdateNotificationParsing(t *testing.T) {
	// Test agent_message_chunk
	notifJSON := `{
		"jsonrpc": "2.0",
		"method": "session/update",
		"params": {
			"sessionId": "sess_abc123",
			"update": {
				"sessionUpdate": "agent_message_chunk",
				"content": {
					"type": "text",
					"text": "Hello, I am analyzing your code..."
				}
			}
		}
	}`

	var notif JSONRPCNotification
	if err := json.Unmarshal([]byte(notifJSON), &notif); err != nil {
		t.Fatalf("failed to unmarshal notification: %v", err)
	}

	if notif.Method != "session/update" {
		t.Errorf("expected method 'session/update', got %s", notif.Method)
	}

	var params SessionUpdateParams
	if err := json.Unmarshal(notif.Params, &params); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}

	if params.SessionID != "sess_abc123" {
		t.Errorf("expected session ID 'sess_abc123', got %s", params.SessionID)
	}
	if params.Update.SessionUpdate != "agent_message_chunk" {
		t.Errorf("expected sessionUpdate 'agent_message_chunk', got %s", params.Update.SessionUpdate)
	}
	content := params.Update.GetMessageContent()
	if content == nil {
		t.Fatal("expected content, got nil")
	}
	if content.Type != "text" {
		t.Errorf("expected content type 'text', got %s", content.Type)
	}
	if content.Text != "Hello, I am analyzing your code..." {
		t.Errorf("unexpected content text: %s", content.Text)
	}
}

func TestToolCallUpdateParsing(t *testing.T) {
	notifJSON := `{
		"jsonrpc": "2.0",
		"method": "session/update",
		"params": {
			"sessionId": "sess_abc123",
			"update": {
				"sessionUpdate": "tool_call_update",
				"toolCallId": "call_001",
				"status": "completed",
				"content": [
					{
						"type": "content",
						"content": {
							"type": "text",
							"text": "Analysis complete"
						}
					}
				]
			}
		}
	}`

	var notif JSONRPCNotification
	if err := json.Unmarshal([]byte(notifJSON), &notif); err != nil {
		t.Fatalf("failed to unmarshal notification: %v", err)
	}

	var params SessionUpdateParams
	if err := json.Unmarshal(notif.Params, &params); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}

	if params.Update.SessionUpdate != "tool_call_update" {
		t.Errorf("expected sessionUpdate 'tool_call_update', got %s", params.Update.SessionUpdate)
	}
	if params.Update.ToolCallID != "call_001" {
		t.Errorf("expected toolCallId 'call_001', got %s", params.Update.ToolCallID)
	}
	if params.Update.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", params.Update.Status)
	}
}

func TestClientConfig(t *testing.T) {
	config := ClientConfig{
		Command:     "test-agent",
		Args:        []string{"--mode", "acp"},
		Env:         map[string]string{"API_KEY": "secret"},
		Cwd:         "/path/to/project",
		InitTimeout: 10 * time.Second,
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("expected client, got nil")
	}

	// Client should not be connected yet
	if client.IsConnected() {
		t.Error("expected client to not be connected")
	}
}

func TestClientConnectFailsWithBadCommand(t *testing.T) {
	config := ClientConfig{
		Command:     "nonexistent-command-that-does-not-exist",
		Args:        []string{},
		InitTimeout: 2 * time.Second,
	}

	client := NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Error("expected error connecting with bad command, got nil")
	}
}

func TestSessionMessageTracking(t *testing.T) {
	// Create a session with a nil client (we're just testing the message tracking)
	session := &Session{
		sessionID: "test_session",
		messages:  make([]Message, 0),
	}

	// Simulate adding messages
	session.mu.Lock()
	session.messages = append(session.messages, Message{
		Role:    "user",
		Content: "Hello",
	})
	session.messages = append(session.messages, Message{
		Role:    "assistant",
		Content: "Hi there!",
	})
	session.mu.Unlock()

	// Test GetMessages
	msgs := session.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected first message role 'user', got %s", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected second message role 'assistant', got %s", msgs[1].Role)
	}

	// Test GetLastMessage
	lastMsg := session.GetLastMessage()
	if lastMsg == nil {
		t.Fatal("expected last message, got nil")
	}
	if lastMsg.Role != "assistant" {
		t.Errorf("expected last message role 'assistant', got %s", lastMsg.Role)
	}
	if lastMsg.Content != "Hi there!" {
		t.Errorf("expected last message content 'Hi there!', got %s", lastMsg.Content)
	}

	// Test SessionID
	if session.SessionID() != "test_session" {
		t.Errorf("expected session ID 'test_session', got %s", session.SessionID())
	}
}

func TestProcessConfigValidation(t *testing.T) {
	// Test that empty command returns error
	_, err := SpawnProcess(context.Background(), ProcessConfig{
		Command: "",
	})
	if err == nil {
		t.Error("expected error for empty command, got nil")
	}
}

func TestGetToolName(t *testing.T) {
	tests := []struct {
		name     string
		payload  SessionUpdatePayload
		expected string
	}{
		{
			name: "prefers Name when both are set",
			payload: SessionUpdatePayload{
				Name:  "read_file",
				Title: "cat /path/to/file",
			},
			expected: "read_file",
		},
		{
			name: "falls back to Title when Name is empty",
			payload: SessionUpdatePayload{
				Name:  "",
				Title: "cat go.mod",
			},
			expected: "cat go.mod",
		},
		{
			name: "returns empty string when both are empty",
			payload: SessionUpdatePayload{
				Name:  "",
				Title: "",
			},
			expected: "",
		},
		{
			name: "returns Name when Title is empty",
			payload: SessionUpdatePayload{
				Name:  "exec",
				Title: "",
			},
			expected: "exec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payload.GetToolName()
			if result != tt.expected {
				t.Errorf("GetToolName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
