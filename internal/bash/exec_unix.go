//go:build !windows

package bash

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/term"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// NewProcessGroupExecHandler returns an ExecHandlerFunc that runs external commands
// with proper job control. The child process is placed in its own process group
// and made the foreground process group of the terminal. This means:
//
//   - When user presses Ctrl+C, SIGINT goes to the child process (not gsh)
//   - The child can handle or die from SIGINT naturally
//   - gsh remains unaffected and continues after the child exits
//
// This is the standard Unix job control model used by zsh/bash for foreground jobs.
//
// The killTimeout parameter specifies how long to wait after sending SIGINT
// before sending SIGKILL when the context is cancelled programmatically.
func NewProcessGroupExecHandler(killTimeout time.Duration) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		hc := interp.HandlerCtx(ctx)
		path, err := interp.LookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			return err
		}

		cmd := exec.Cmd{
			Path:   path,
			Args:   args,
			Dir:    hc.Dir,
			Env:    execEnv(hc.Env),
			Stdin:  hc.Stdin,
			Stdout: hc.Stdout,
			Stderr: hc.Stderr,
			// Put the child process in its own process group
			SysProcAttr: &syscall.SysProcAttr{
				Setpgid: true,
			},
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		childPgid := cmd.Process.Pid

		// Try to make the child's process group the foreground group.
		// This requires a controlling terminal (stdin must be a tty).
		// If this fails (e.g., stdin is a pipe), we fall back to the old behavior.
		var ttyFd = -1
		var originalPgrp int

		if f, ok := hc.Stdin.(*os.File); ok {
			fd := int(f.Fd())
			// Check if stdin is a terminal
			if term.IsTerminal(fd) {
				ttyFd = fd
				// Save the original foreground process group
				originalPgrp, _ = tcgetpgrp(ttyFd)
				// Make the child's process group the foreground
				_ = tcsetpgrp(ttyFd, childPgid)
			}
		}

		// Restore the original foreground process group when done
		defer func() {
			if ttyFd >= 0 && originalPgrp > 0 {
				_ = tcsetpgrp(ttyFd, originalPgrp)
			}
		}()

		// Wait for the command or context cancellation
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- cmd.Wait()
		}()

		select {
		case err := <-waitDone:
			return err
		case <-ctx.Done():
			// Context cancelled programmatically (not via Ctrl+C since child is foreground)
			// Send interrupt to the child's process group
			if cmd.Process != nil {
				_ = syscall.Kill(-childPgid, syscall.SIGINT)
			}

			// Wait for graceful shutdown or timeout
			if killTimeout >= 0 {
				select {
				case err := <-waitDone:
					return err
				case <-time.After(killTimeout):
					// Timeout - force kill the process group
					if cmd.Process != nil {
						_ = syscall.Kill(-childPgid, syscall.SIGKILL)
					}
				}
			} else {
				// Negative timeout means kill immediately
				if cmd.Process != nil {
					_ = syscall.Kill(-childPgid, syscall.SIGKILL)
				}
			}

			return <-waitDone
		}
	}
}

// tcgetpgrp returns the foreground process group ID of the terminal.
func tcgetpgrp(fd int) (int, error) {
	var pgrp int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pgrp)))
	if errno != 0 {
		return 0, errno
	}
	return int(pgrp), nil
}

// tcsetpgrp sets the foreground process group ID of the terminal.
func tcsetpgrp(fd int, pgrp int) error {
	pgrp32 := int32(pgrp)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&pgrp32)))
	if errno != 0 {
		return errno
	}
	return nil
}

// execEnv converts expand.Environ to []string for exec.Cmd.Env
func execEnv(env interface {
	Each(func(name string, vr expand.Variable) bool)
}) []string {
	var result []string
	env.Each(func(name string, vr expand.Variable) bool {
		if vr.Exported {
			result = append(result, name+"="+vr.String())
		}
		return true
	})
	// Also include current process environment for any vars not overridden
	result = append(result, os.Environ()...)
	return result
}
