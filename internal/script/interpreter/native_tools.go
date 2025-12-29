// Package interpreter provides native tool implementations that can be shared
// between the SDK (gsh.tools) and the REPL agent.
package interpreter

import (
	"time"
)

// DefaultExecTimeout is the default timeout for command execution.
const DefaultExecTimeout = 60 * time.Second

// maxViewFileOutputLen is the maximum output size (100KB) before middle truncation is applied.
const maxViewFileOutputLen = 100000

// maxGrepOutputLen is the maximum grep output size (~50KB) before truncation.
const maxGrepOutputLen = 50000

// ExecEventCallbacks provides hooks for exec lifecycle events.
// These are used by the REPL to emit agent.exec.start and agent.exec.end events.
type ExecEventCallbacks struct {
	// OnStart is called when a command starts executing.
	// Handlers that want to produce output should print directly to stdout.
	OnStart func(command string)

	// OnEnd is called when a command finishes executing.
	// Handlers that want to produce output should print directly to stdout.
	OnEnd func(command string, durationMs int64, exitCode int)
}
