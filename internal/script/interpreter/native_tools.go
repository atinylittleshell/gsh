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
