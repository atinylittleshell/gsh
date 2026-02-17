package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Process manages an ACP agent subprocess.
type Process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser
	logger *zap.Logger

	mu       sync.Mutex
	closed   bool
	closedCh chan struct{}

	// For reading notifications
	notificationCh chan *JSONRPCNotification
	responseCh     chan *JSONRPCResponse
	errCh          chan error

	ctx    context.Context
	cancel context.CancelFunc
}

// ProcessConfig contains configuration for spawning an ACP agent process.
type ProcessConfig struct {
	Command string
	Args    []string
	Env     map[string]string
	Cwd     string
}

// SpawnProcess starts an ACP agent subprocess.
func SpawnProcess(ctx context.Context, config ProcessConfig, logger *zap.Logger) (*Process, error) {
	if config.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	cmd := exec.CommandContext(ctx, config.Command, config.Args...)

	// Set working directory if specified
	if config.Cwd != "" {
		cmd.Dir = config.Cwd
	}

	// Set environment variables
	if len(config.Env) > 0 {
		// Start with current environment
		cmd.Env = os.Environ()
		for k, v := range config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	logger.Debug("ACP process spawned",
		zap.String("command", config.Command),
		zap.Strings("args", config.Args),
		zap.Int("pid", cmd.Process.Pid))

	procCtx, cancel := context.WithCancel(ctx)

	p := &Process{
		cmd:            cmd,
		stdin:          stdin,
		stdout:         bufio.NewReader(stdout),
		stderr:         stderr,
		logger:         logger,
		closedCh:       make(chan struct{}),
		notificationCh: make(chan *JSONRPCNotification, 100),
		responseCh:     make(chan *JSONRPCResponse, 10),
		errCh:          make(chan error, 1),
		ctx:            procCtx,
		cancel:         cancel,
	}

	// Start the reader goroutine
	go p.readLoop()

	return p, nil
}

// readLoop reads JSON-RPC messages from stdout and routes them.
func (p *Process) readLoop() {
	defer func() {
		p.logger.Debug("ACP readLoop exiting, closing channels")
		close(p.notificationCh)
		close(p.responseCh)
	}()

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Debug("ACP readLoop context cancelled")
			return
		default:
		}

		line, err := p.stdout.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				p.logger.Debug("ACP readLoop read error", zap.Error(err))
				select {
				case p.errCh <- fmt.Errorf("read error: %w", err):
				default:
				}
			} else {
				p.logger.Debug("ACP readLoop EOF, agent process stdout closed")
			}
			return
		}

		if len(line) == 0 {
			continue
		}

		// Try to parse as a generic JSON object first
		var msg map[string]json.RawMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			p.logger.Debug("ACP received malformed line, skipping",
				zap.ByteString("line", line))
			continue
		}

		_, hasID := msg["id"]
		methodRaw, hasMethod := msg["method"]

		if hasID && hasMethod {
			// It's a request from the agent to the client (has both "method" and "id").
			var method string
			_ = json.Unmarshal(methodRaw, &method)
			reqID := msg["id"]

			p.handleAgentRequest(method, reqID, msg["params"])
		} else if hasID {
			// It's a response (has "id" but no "method")
			var resp JSONRPCResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				p.logger.Debug("ACP failed to parse response", zap.Error(err))
				continue
			}
			respID := -1
			if resp.ID != nil {
				respID = *resp.ID
			}
			hasError := resp.Error != nil
			p.logger.Debug("ACP received response",
				zap.Int("id", respID),
				zap.Bool("hasError", hasError))
			select {
			case p.responseCh <- &resp:
			case <-p.ctx.Done():
				return
			}
		} else if hasMethod {
			// It's a notification (has "method" but no "id")
			var notif JSONRPCNotification
			if err := json.Unmarshal(line, &notif); err != nil {
				p.logger.Debug("ACP failed to parse notification", zap.Error(err))
				continue
			}
			var method string
			_ = json.Unmarshal(methodRaw, &method)
			p.logger.Debug("ACP received notification", zap.String("method", method))
			select {
			case p.notificationCh <- &notif:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// SendRequest sends a JSON-RPC request to the agent.
func (p *Process) SendRequest(req *JSONRPCRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("process is closed")
	}

	p.logger.Debug("ACP sending request",
		zap.String("method", req.Method),
		zap.Int("id", req.ID))

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	data = append(data, '\n')

	if _, err := p.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

// Notifications returns the channel for receiving notifications.
func (p *Process) Notifications() <-chan *JSONRPCNotification {
	return p.notificationCh
}

// Responses returns the channel for receiving responses.
func (p *Process) Responses() <-chan *JSONRPCResponse {
	return p.responseCh
}

// Errors returns the channel for receiving errors.
func (p *Process) Errors() <-chan error {
	return p.errCh
}

// Close shuts down the process.
func (p *Process) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	close(p.closedCh)
	p.mu.Unlock()

	p.logger.Debug("ACP process closing")

	// Cancel the context to stop the read loop
	p.cancel()

	// Close stdin to signal the process
	p.stdin.Close()

	// Wait for the process with a timeout
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		p.logger.Debug("ACP process exited", zap.NamedError("exitError", err))
	case <-p.ctx.Done():
		p.logger.Debug("ACP process kill due to context cancellation")
		_ = p.cmd.Process.Kill()
	}

	return nil
}

// ReadStderr reads available stderr content from the process with a short timeout.
// This is useful for capturing diagnostic output when the process fails.
// The read is non-blocking to avoid hanging when the process has no stderr output.
func (p *Process) ReadStderr() string {
	if p.stderr == nil {
		return ""
	}
	type readResult struct {
		data []byte
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 4096)
		n, err := p.stderr.Read(buf)
		ch <- readResult{data: buf[:n], err: err}
	}()
	select {
	case result := <-ch:
		if len(result.data) == 0 {
			return ""
		}
		return string(result.data)
	case <-time.After(500 * time.Millisecond):
		return ""
	}
}

// handleAgentRequest handles JSON-RPC requests from the agent to the client.
func (p *Process) handleAgentRequest(method string, reqID json.RawMessage, params json.RawMessage) {
	var result interface{}

	switch method {
	case "session/request_permission":
		// Auto-approve permission requests by selecting the first option.
		result = p.handleRequestPermission(params)
	default:
		p.logger.Warn("ACP received unsupported agent-to-client request, rejecting",
			zap.String("method", method),
			zap.ByteString("id", reqID))
		p.sendAgentResponse(reqID, nil, &JSONRPCError{
			Code:    -32601,
			Message: fmt.Sprintf("method not supported by client: %s", method),
		})
		return
	}

	p.sendAgentResponse(reqID, result, nil)
}

// handleRequestPermission auto-approves a permission request by selecting the first option.
func (p *Process) handleRequestPermission(params json.RawMessage) interface{} {
	// Parse just enough to find the first option ID
	var req struct {
		Options []struct {
			ID string `json:"id"`
		} `json:"options"`
	}
	if err := json.Unmarshal(params, &req); err != nil || len(req.Options) == 0 {
		p.logger.Debug("ACP permission request: no options found, approving with empty optionId")
		return map[string]interface{}{
			"outcome":  "selected",
			"optionId": "",
		}
	}

	optionID := req.Options[0].ID
	p.logger.Debug("ACP auto-approving permission request",
		zap.String("optionId", optionID))

	return map[string]interface{}{
		"outcome":  "selected",
		"optionId": optionID,
	}
}

// sendAgentResponse sends a JSON-RPC response back to the agent.
func (p *Process) sendAgentResponse(reqID json.RawMessage, result interface{}, rpcErr *JSONRPCError) {
	resp := map[string]interface{}{
		"jsonrpc": JSONRPCVersion,
		"id":      json.RawMessage(reqID),
	}
	if rpcErr != nil {
		resp["error"] = map[string]interface{}{
			"code":    rpcErr.Code,
			"message": rpcErr.Message,
		}
	} else {
		resp["result"] = result
	}

	data, err := json.Marshal(resp)
	if err != nil {
		p.logger.Debug("ACP failed to marshal agent response", zap.Error(err))
		return
	}
	data = append(data, '\n')

	p.mu.Lock()
	if !p.closed {
		_, _ = p.stdin.Write(data)
	}
	p.mu.Unlock()
}

// IsClosed returns whether the process has been closed.
func (p *Process) IsClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}
