//go:build windows

package bash

import (
	"time"

	"mvdan.cc/sh/v3/interp"
)

// NewProcessGroupExecHandler on Windows falls back to the default exec handler
// since Windows doesn't support Unix process groups.
func NewProcessGroupExecHandler(killTimeout time.Duration) interp.ExecHandlerFunc {
	return interp.DefaultExecHandler(killTimeout)
}
