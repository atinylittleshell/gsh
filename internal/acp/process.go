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
)

// Process manages an ACP agent subprocess.
type Process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser

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
func SpawnProcess(ctx context.Context, config ProcessConfig) (*Process, error) {
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

	procCtx, cancel := context.WithCancel(ctx)

	p := &Process{
		cmd:            cmd,
		stdin:          stdin,
		stdout:         bufio.NewReader(stdout),
		stderr:         stderr,
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
	defer close(p.notificationCh)
	defer close(p.responseCh)

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		line, err := p.stdout.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				select {
				case p.errCh <- fmt.Errorf("read error: %w", err):
				default:
				}
			}
			return
		}

		if len(line) == 0 {
			continue
		}

		// Try to parse as a generic JSON object first
		var msg map[string]json.RawMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			// Skip malformed lines
			continue
		}

		// Check if it's a response (has id) or notification (has method, no id)
		if _, hasID := msg["id"]; hasID {
			// It's a response
			var resp JSONRPCResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				continue
			}
			select {
			case p.responseCh <- &resp:
			case <-p.ctx.Done():
				return
			}
		} else if _, hasMethod := msg["method"]; hasMethod {
			// It's a notification
			var notif JSONRPCNotification
			if err := json.Unmarshal(line, &notif); err != nil {
				continue
			}
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
	case <-done:
		// Process exited
	case <-p.ctx.Done():
		// Context cancelled, kill the process
		_ = p.cmd.Process.Kill()
	}

	return nil
}

// IsClosed returns whether the process has been closed.
func (p *Process) IsClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}
