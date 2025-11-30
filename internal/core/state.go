package core

import (
	"bytes"
	"io"
	"sync"
)

// ShellState holds the state of the shell execution
type ShellState struct {
	LastCommand  string
	LastExitCode int
	LastStderr   string
}

// StderrCapturer wraps an io.Writer and captures the output into a buffer
type StderrCapturer struct {
	original  io.Writer
	buffer    *bytes.Buffer
	mu        sync.Mutex
	capturing bool
}

func NewStderrCapturer(original io.Writer) *StderrCapturer {
	return &StderrCapturer{
		original: original,
	}
}

func (c *StderrCapturer) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	if c.capturing {
		if c.buffer == nil {
			c.buffer = new(bytes.Buffer)
		}
		// Limit buffer size to avoid memory issues (e.g., 64KB is enough for error messages)
		remaining := 64*1024 - c.buffer.Len()
		if remaining > 0 {
			toWrite := p
			if len(toWrite) > remaining {
				toWrite = toWrite[:remaining]
			}
			c.buffer.Write(toWrite)
		}
	}
	c.mu.Unlock()
	return c.original.Write(p)
}

func (c *StderrCapturer) StartCapture() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.capturing = true
	c.buffer = new(bytes.Buffer)
}

func (c *StderrCapturer) StopCapture() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.capturing = false
	if c.buffer == nil {
		return ""
	}
	res := c.buffer.String()
	c.buffer = nil
	return res
}
