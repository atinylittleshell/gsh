package acp

import (
	"context"
	"encoding/json"
	"sync"
)

// MockSession is a mock implementation of ACPSession for testing.
type MockSession struct {
	mu        sync.Mutex
	sessionID string
	messages  []Message
	closed    bool

	// Test configuration
	SendPromptFunc func(ctx context.Context, text string, onUpdate func(*SessionUpdateParams)) (*SessionPromptResult, error)
	Updates        []*SessionUpdateParams // Updates to send during SendPrompt
	PromptResult   *SessionPromptResult   // Result to return from SendPrompt
	PromptError    error                  // Error to return from SendPrompt
}

// NewMockSession creates a new mock session for testing.
func NewMockSession(sessionID string) *MockSession {
	return &MockSession{
		sessionID: sessionID,
		messages:  make([]Message, 0),
		PromptResult: &SessionPromptResult{
			StopReason: "end_turn",
		},
	}
}

// SendPrompt implements ACPSession.SendPrompt
func (m *MockSession) SendPrompt(ctx context.Context, text string, onUpdate func(*SessionUpdateParams)) (*SessionPromptResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If custom function is set, use it
	if m.SendPromptFunc != nil {
		return m.SendPromptFunc(ctx, text, onUpdate)
	}

	// Add user message
	m.messages = append(m.messages, Message{
		Role:    "user",
		Content: text,
	})

	// Send configured updates
	if onUpdate != nil {
		for _, update := range m.Updates {
			onUpdate(update)
		}
	}

	// Add assistant message (if we have chunk updates, combine them)
	assistantContent := ""
	for _, update := range m.Updates {
		if update.Update.SessionUpdate == SessionUpdateAgentMessageChunk {
			if content := update.Update.GetMessageContent(); content != nil && content.Type == "text" {
				assistantContent += content.Text
			}
		}
	}
	if assistantContent != "" {
		m.messages = append(m.messages, Message{
			Role:    "assistant",
			Content: assistantContent,
		})
	}

	if m.PromptError != nil {
		return nil, m.PromptError
	}

	return m.PromptResult, nil
}

// GetMessages implements ACPSession.GetMessages
func (m *MockSession) GetMessages() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// GetLastMessage implements ACPSession.GetLastMessage
func (m *MockSession) GetLastMessage() *Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.messages) == 0 {
		return nil
	}
	msg := m.messages[len(m.messages)-1]
	return &msg
}

// SessionID implements ACPSession.SessionID
func (m *MockSession) SessionID() string {
	return m.sessionID
}

// Close implements ACPSession.Close
func (m *MockSession) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// IsClosed returns whether the session is closed (for testing)
func (m *MockSession) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// AddUpdate adds an update to send during the next SendPrompt call
func (m *MockSession) AddUpdate(update *SessionUpdateParams) {
	m.Updates = append(m.Updates, update)
}

// AddChunkUpdate is a helper to add a text chunk update
func (m *MockSession) AddChunkUpdate(text string) {
	contentJSON, _ := json.Marshal(&MessageContent{
		Type: "text",
		Text: text,
	})
	m.AddUpdate(&SessionUpdateParams{
		SessionID: m.sessionID,
		Update: SessionUpdatePayload{
			SessionUpdate: SessionUpdateAgentMessageChunk,
			Content:       contentJSON,
		},
	})
}

// AddToolCallUpdate is a helper to add a tool call update
func (m *MockSession) AddToolCallUpdate(toolCallID, toolName, arguments string) {
	m.AddUpdate(&SessionUpdateParams{
		SessionID: m.sessionID,
		Update: SessionUpdatePayload{
			SessionUpdate: SessionUpdateToolCall,
			ToolCallID:    toolCallID,
			Name:          toolName,
		},
	})
}

// AddToolCallEndUpdate is a helper to add a tool call completion update
func (m *MockSession) AddToolCallEndUpdate(toolCallID, toolName, status, output string) {
	contentJSON, _ := json.Marshal([]ToolCallContent{
		{
			Content: &MessageContent{
				Type: "text",
				Text: output,
			},
		},
	})
	m.AddUpdate(&SessionUpdateParams{
		SessionID: m.sessionID,
		Update: SessionUpdatePayload{
			SessionUpdate: SessionUpdateToolCallUpdate,
			ToolCallID:    toolCallID,
			Name:          toolName,
			Status:        status,
			Content:       contentJSON,
		},
	})
}

// MockClient is a mock ACP client for testing
type MockClient struct {
	mu           sync.Mutex
	connected    bool
	sessions     map[string]*MockSession
	nextSession  *MockSession // Pre-configured session to return from NewSession
	connectError error
	sessionError error
}

// NewMockClient creates a new mock client for testing
func NewMockClient() *MockClient {
	return &MockClient{
		sessions: make(map[string]*MockSession),
	}
}

// Connect implements a mock connect that succeeds by default
func (m *MockClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.connectError != nil {
		return m.connectError
	}
	m.connected = true
	return nil
}

// NewSession returns the pre-configured mock session
func (m *MockClient) NewSession(ctx context.Context, cwd string, mcpServers []MCPServer) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sessionError != nil {
		return nil, m.sessionError
	}
	// Note: This returns *Session which is the concrete type.
	// For full mocking, we need to use the interface approach in the interpreter.
	return nil, nil
}

// Close implements mock close
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// IsConnected returns the connection state
func (m *MockClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// SetConnectError sets an error to return from Connect
func (m *MockClient) SetConnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectError = err
}

// SetSessionError sets an error to return from NewSession
func (m *MockClient) SetSessionError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionError = err
}

// SetNextSession sets the session to return from the next NewSession call
func (m *MockClient) SetNextSession(session *MockSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextSession = session
}

// Ensure MockSession implements ACPSession
var _ ACPSession = (*MockSession)(nil)
