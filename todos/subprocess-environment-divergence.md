# Subprocess Environment Divergence

## Summary

When gsh spawns subprocesses via `exec.Command` (as opposed to bash subshells via the runner), the subprocess inherits `os.Environ()` — the Go process environment frozen at startup. This misses environment modifications made by profile scripts (`.bashrc`, `.zshrc`, nvm, pyenv, etc.) that were applied to the bash runner but never synced back to the OS process environment.

The immediate symptom: ACP agents fail with `env: node: No such file or directory` when Node.js is installed via nvm, because the ACP subprocess's PATH doesn't include nvm's bin directory.

## Background: Three Disconnected Environments

The interpreter maintains three separate representations of "environment":

1. **`i.env`** — The gsh script scope chain (gsh variables, tools, closures). Managed by `internal/script/interpreter/environment.go`.
2. **`i.runner`** (bash runner from `mvdan.cc/sh`) — The bash shell environment (PATH, HOME, etc.). Modified by profile scripts sourced during gsh startup. Managed by the `interp.Runner` from `mvdan.cc/sh/v3/interp`.
3. **`os.Environ()`** — The Go process environment. Set once at process start, never updated by profile scripts.

When gsh starts, the runner is initialized with `os.Environ()` (see `interpreter.go:120`). Then profile scripts run and modify the runner's environment (e.g., nvm adds its bin dir to PATH in the runner). But `os.Environ()` is never updated — it stays frozen at the pre-profile state.

## How Subprocesses Get Their Environment

There are two paths for spawning subprocesses in gsh:

### Path A: Bash subshells (correct)

`executeBashInSubshell()` → `runner.Subshell()` → inherits runner's full environment including profile modifications.

Used by: `exec()` builtin, command resolution (`command -v`), `SetEnv()`.

### Path B: Direct `exec.Command` (broken)

`exec.CommandContext()` → inherits `os.Environ()` (stale, missing profile modifications).

Used by:
- **ACP process spawning** (`internal/acp/process.go:52-64`) — `SpawnProcess` uses `exec.CommandContext` and only sets `cmd.Env` when the ACP config has explicit `env` entries. Otherwise subprocess inherits `os.Environ()`.
- **Exec tool** (`internal/script/interpreter/exec_tool.go:88-98`) — `ExecuteCommandWithPTY` uses `os.Environ()` directly.
- **Grep tool** (`internal/script/interpreter/grep_tool.go:104-106`) — `ExecuteGrepWithBackend` uses `os.Environ()` directly.

### Why this matters

When a user has nvm/pyenv/rbenv/etc., the PATH in `os.Environ()` doesn't include the version manager's bin directory. Commands resolved via the runner work (they find the absolute path), but child processes spawned by those commands can't find dependent executables. For example:
- `npx` is resolved to `/Users/foo/.nvm/versions/node/v22.14.0/bin/npx` (correct, via runner)
- But `npx` internally runs `#!/usr/bin/env node`, which searches PATH
- The subprocess PATH (from `os.Environ()`) doesn't include nvm's bin dir
- Result: `env: node: No such file or directory`

## Current Workaround

In `internal/script/interpreter/acp.go`, the `getOrCreateACPClient` function was patched to always include `i.GetEnv("PATH")` in the env map passed to `SpawnProcess`. This fixes the ACP case for PATH specifically, but doesn't address the root cause for other env vars or other subprocess spawn sites.

## Proposed Fix

### 1. Create a centralized `SubprocessEnv()` method on the interpreter

A method that builds the complete environment for direct subprocesses by merging the runner's exported variables with `os.Environ()` as a fallback:

```go
// SubprocessEnv returns environment variables for spawning subprocesses
// via exec.Command. It merges the runner's exported variables (which
// include profile script modifications like nvm/pyenv PATH changes)
// with os.Environ() as a fallback for variables not in the runner.
func (i *Interpreter) SubprocessEnv() []string {
    // Similar to execEnv() in internal/bash/exec_unix.go
    // 1. Collect all exported vars from the runner
    // 2. Append os.Environ() for any vars not overridden by the runner
}
```

Note: A similar function already exists as `execEnv()` in `internal/bash/exec_unix.go:142-156`. That function iterates the runner's environment using `env.Each()` and appends `os.Environ()`. The new method should follow the same pattern but be accessible from the interpreter.

### 2. Use it everywhere subprocesses are spawned

Update all direct `exec.Command` sites to use the interpreter's environment:

- `internal/acp/process.go` — `SpawnProcess` should accept `[]string` env instead of `map[string]string`, or the caller (`getOrCreateACPClient`) should pass the full env.
- `internal/script/interpreter/exec_tool.go` — `ExecuteCommandWithPTY` should accept env as a parameter (it's currently a standalone function, not a method on the interpreter).
- `internal/script/interpreter/grep_tool.go` — `ExecuteGrepWithBackend` should similarly accept env.

### 3. Consider syncing env back to `os.Environ()`

An alternative or complementary approach: when the runner's environment is modified (e.g., after sourcing profile scripts), sync key variables (at minimum PATH) back to the OS process environment via `os.Setenv()`. This would fix all subprocess spawning sites automatically, but `os.Setenv` is process-global and not goroutine-safe, so it needs careful handling.

## Key Files

- `internal/script/interpreter/interpreter.go:108-130` — Runner initialization with `os.Environ()`
- `internal/script/interpreter/interpreter.go:258-269` — `GetEnv()` checks runner then os
- `internal/script/interpreter/acp.go:290-378` — `getOrCreateACPClient` builds env for ACP
- `internal/acp/process.go:47-124` — `SpawnProcess` uses `exec.CommandContext`
- `internal/script/interpreter/exec_tool.go:82-98` — `ExecuteCommandWithPTY` uses `os.Environ()`
- `internal/script/interpreter/grep_tool.go:98-106` — `ExecuteGrepWithBackend` uses `os.Environ()`
- `internal/bash/exec_unix.go:142-156` — `execEnv()` reference implementation for merging runner env with `os.Environ()`

## Testing

- Test with Node.js installed via nvm (PATH not in `os.Environ()`)
- Test ACP agent that spawns `npx` — should find `node` in subprocess
- Test exec tool running commands that depend on profile-modified PATH
- Test with explicit `env` overrides in ACP config — user overrides should take precedence
