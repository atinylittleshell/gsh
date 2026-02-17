package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Client is an ACP client that manages communication with an ACP agent.
type Client struct {
	process *Process
	config  ClientConfig
	logger  *zap.Logger

	// Protocol state
	initialized       bool
	agentCapabilities AgentCapabilities
	protocolVersion   int

	// Request ID counter
	requestID int64

	// Synchronization
	mu sync.RWMutex

	// Context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
}

// ClientConfig contains configuration for an ACP client.
type ClientConfig struct {
	Command string
	Args    []string
	Env     map[string]string
	Cwd     string

	// Timeout for initialization
	InitTimeout time.Duration

	// Logger for ACP activity logging. If nil, a no-op logger is used.
	Logger *zap.Logger
}

// Session represents an active ACP session.
type Session struct {
	client    *Client
	sessionID string

	// Local copy of messages for debugging/display
	messages []Message
	mu       sync.RWMutex

	// Callback for handling updates
	onUpdate func(update *SessionUpdateParams)
}

// Message represents a message in the session history.
type Message struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	Name       string         `json:"name,omitempty"`
	ToolCallID string         `json:"toolCallId,omitempty"`
	ToolCalls  []ToolCallInfo `json:"toolCalls,omitempty"`
}

// ToolCallInfo represents information about a tool call in a message.
type ToolCallInfo struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// NewClient creates a new ACP client but does not start the agent process.
func NewClient(config ClientConfig) *Client {
	if config.InitTimeout == 0 {
		config.InitTimeout = 30 * time.Second
	}

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Connect starts the agent process and performs the initialization handshake.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.process != nil {
		return fmt.Errorf("client already connected")
	}

	c.logger.Debug("ACP connecting",
		zap.String("command", c.config.Command),
		zap.Strings("args", c.config.Args))

	// Spawn the process
	proc, err := SpawnProcess(ctx, ProcessConfig{
		Command: c.config.Command,
		Args:    c.config.Args,
		Env:     c.config.Env,
		Cwd:     c.config.Cwd,
	}, c.logger)
	if err != nil {
		c.logger.Debug("ACP failed to spawn process", zap.Error(err))
		return fmt.Errorf("failed to spawn ACP agent: %w", err)
	}

	c.process = proc

	// Perform initialization handshake
	if err := c.initialize(ctx); err != nil {
		stderrOutput := proc.ReadStderr()
		proc.Close()
		c.process = nil
		c.logger.Debug("ACP initialization failed",
			zap.Error(err),
			zap.String("stderr", stderrOutput))
		if stderrOutput != "" {
			return fmt.Errorf("initialization failed: %w\nagent stderr: %s", err, stderrOutput)
		}
		return fmt.Errorf("initialization failed: %w", err)
	}

	c.logger.Debug("ACP connected successfully",
		zap.Int("protocolVersion", c.protocolVersion))
	c.initialized = true
	return nil
}

// initialize performs the ACP initialization handshake.
func (c *Client) initialize(ctx context.Context) error {
	initCtx, cancel := context.WithTimeout(ctx, c.config.InitTimeout)
	defer cancel()

	c.logger.Debug("ACP sending initialize handshake")

	// Send initialize request
	reqID := c.nextRequestID()
	req := NewInitializeRequest(int(reqID))

	if err := c.process.SendRequest(req); err != nil {
		return fmt.Errorf("failed to send initialize request: %w", err)
	}

	// Wait for response
	resp, err := c.waitForResponse(initCtx, int(reqID))
	if err != nil {
		return fmt.Errorf("failed to receive initialize response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
	}

	// Parse the result
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		c.logger.Debug("ACP initialize result parse error",
			zap.Error(err),
			zap.ByteString("rawResult", resp.Result))
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	c.logger.Debug("ACP initialize handshake complete",
		zap.Int("protocolVersion", result.ProtocolVersion))

	c.protocolVersion = result.ProtocolVersion
	c.agentCapabilities = result.AgentCapabilities

	return nil
}

// waitForResponse waits for a response with the given ID.
func (c *Client) waitForResponse(ctx context.Context, id int) (*JSONRPCResponse, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case resp, ok := <-c.process.Responses():
			if !ok {
				return nil, fmt.Errorf("response channel closed")
			}
			if resp.ID != nil && *resp.ID == id {
				return resp, nil
			}
			// Not our response, could be from a different request
			// In a more sophisticated implementation, we'd have a response router
		case err := <-c.process.Errors():
			return nil, err
		}
	}
}

// NewSession creates a new ACP session.
func (c *Client) NewSession(ctx context.Context, cwd string, mcpServers []MCPServer) (*Session, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	// Ensure mcpServers is not nil (required by ACP protocol)
	if mcpServers == nil {
		mcpServers = []MCPServer{}
	}

	c.logger.Debug("ACP creating new session", zap.String("cwd", cwd))

	// Send session/new request
	reqID := c.nextRequestID()
	req := NewSessionNewRequest(int(reqID), cwd, mcpServers)

	if err := c.process.SendRequest(req); err != nil {
		return nil, fmt.Errorf("failed to send session/new request: %w", err)
	}

	// Wait for response
	resp, err := c.waitForResponse(ctx, int(reqID))
	if err != nil {
		return nil, fmt.Errorf("failed to receive session/new response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("session/new error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
	}

	// Parse the result
	var result SessionNewResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse session/new result: %w", err)
	}

	c.logger.Debug("ACP session created", zap.String("sessionId", result.SessionID))

	return &Session{
		client:    c,
		sessionID: result.SessionID,
		messages:  make([]Message, 0),
	}, nil
}

// SendPrompt sends a prompt to the session and streams updates via the callback.
func (s *Session) SendPrompt(ctx context.Context, text string, onUpdate func(*SessionUpdateParams)) (*SessionPromptResult, error) {
	s.mu.Lock()
	// Add user message to local history
	s.messages = append(s.messages, Message{
		Role:    "user",
		Content: text,
	})
	s.onUpdate = onUpdate
	s.mu.Unlock()

	logger := s.client.logger

	// Truncate prompt text for logging
	promptPreview := text
	if len(promptPreview) > 100 {
		promptPreview = promptPreview[:100] + "..."
	}
	logger.Debug("ACP sending prompt",
		zap.String("sessionId", s.sessionID),
		zap.String("prompt", promptPreview))

	// Send session/prompt request
	reqID := s.client.nextRequestID()
	req := NewSessionPromptRequest(int(reqID), s.sessionID, text)

	if err := s.client.process.SendRequest(req); err != nil {
		return nil, fmt.Errorf("failed to send session/prompt request: %w", err)
	}

	// Process notifications and wait for response
	var assistantContent string
	var currentToolCalls []ToolCallInfo

	for {
		select {
		case <-ctx.Done():
			logger.Debug("ACP prompt context cancelled",
				zap.String("sessionId", s.sessionID))
			return nil, ctx.Err()

		case notif, ok := <-s.client.process.Notifications():
			if !ok {
				logger.Debug("ACP notification channel closed during prompt",
					zap.String("sessionId", s.sessionID))
				return nil, fmt.Errorf("notification channel closed")
			}

			// Parse the notification params
			var params SessionUpdateParams
			if err := json.Unmarshal(notif.Params, &params); err != nil {
				continue // Skip malformed notifications
			}

			// Only process notifications for our session
			if params.SessionID != s.sessionID {
				continue
			}

			// Handle different update types
			switch params.Update.SessionUpdate {
			case SessionUpdateAgentMessageChunk:
				if content := params.Update.GetMessageContent(); content != nil && content.Type == "text" {
					assistantContent += content.Text
				}
				logger.Debug("ACP session update",
					zap.String("sessionId", s.sessionID),
					zap.String("type", SessionUpdateAgentMessageChunk))
			case SessionUpdateToolCall:
				currentToolCalls = append(currentToolCalls, ToolCallInfo{
					ID:        params.Update.ToolCallID,
					Name:      params.Update.GetToolName(),
					Arguments: params.Update.Arguments,
				})
				logger.Debug("ACP session update",
					zap.String("sessionId", s.sessionID),
					zap.String("type", SessionUpdateToolCall),
					zap.String("toolName", params.Update.GetToolName()),
					zap.String("toolCallId", params.Update.ToolCallID))
			case SessionUpdateToolCallUpdate:
				logger.Debug("ACP session update",
					zap.String("sessionId", s.sessionID),
					zap.String("type", SessionUpdateToolCallUpdate),
					zap.String("toolCallId", params.Update.ToolCallID),
					zap.String("status", params.Update.Status))
			default:
				logger.Debug("ACP session update",
					zap.String("sessionId", s.sessionID),
					zap.String("type", params.Update.SessionUpdate))
			}

			// Call the update callback if provided
			if onUpdate != nil {
				onUpdate(&params)
			}

		case resp, ok := <-s.client.process.Responses():
			if !ok {
				logger.Debug("ACP response channel closed during prompt",
					zap.String("sessionId", s.sessionID))
				return nil, fmt.Errorf("response channel closed")
			}

			if resp.ID != nil && *resp.ID == int(reqID) {
				// This is our response
				if resp.Error != nil {
					logger.Debug("ACP prompt error response",
						zap.String("sessionId", s.sessionID),
						zap.String("error", resp.Error.Message),
						zap.Int("code", resp.Error.Code))
					return nil, fmt.Errorf("session/prompt error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
				}

				// Parse the result
				var result SessionPromptResult
				if err := json.Unmarshal(resp.Result, &result); err != nil {
					return nil, fmt.Errorf("failed to parse session/prompt result: %w", err)
				}

				logger.Debug("ACP prompt completed",
					zap.String("sessionId", s.sessionID),
					zap.String("stopReason", result.StopReason))

				// Add assistant message to local history
				s.mu.Lock()
				s.messages = append(s.messages, Message{
					Role:      "assistant",
					Content:   assistantContent,
					ToolCalls: currentToolCalls,
				})
				s.mu.Unlock()

				return &result, nil
			}

		case err := <-s.client.process.Errors():
			logger.Debug("ACP error during prompt",
				zap.String("sessionId", s.sessionID),
				zap.Error(err))
			return nil, err
		}
	}
}

// GetMessages returns a copy of the local message history.
func (s *Session) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages := make([]Message, len(s.messages))
	copy(messages, s.messages)
	return messages
}

// GetLastMessage returns the last message in the history.
func (s *Session) GetLastMessage() *Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.messages) == 0 {
		return nil
	}
	msg := s.messages[len(s.messages)-1]
	return &msg
}

// SessionID returns the session ID.
func (s *Session) SessionID() string {
	return s.sessionID
}

// Close closes the session. Note: ACP doesn't have a session/close method,
// so this just marks the session as closed locally.
func (s *Session) Close() error {
	// ACP protocol doesn't have a session/close method
	// The session is cleaned up when the client is closed
	return nil
}

// nextRequestID returns the next request ID.
func (c *Client) nextRequestID() int64 {
	return atomic.AddInt64(&c.requestID, 1)
}

// Close shuts down the client and the agent process.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("ACP client closing")

	c.cancel()

	if c.process != nil {
		return c.process.Close()
	}
	return nil
}

// IsConnected returns whether the client is connected and initialized.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized && c.process != nil && !c.process.IsClosed()
}

// AgentCapabilities returns the agent's capabilities after initialization.
func (c *Client) AgentCapabilities() AgentCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agentCapabilities
}

// ProtocolVersion returns the negotiated protocol version.
func (c *Client) ProtocolVersion() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.protocolVersion
}
