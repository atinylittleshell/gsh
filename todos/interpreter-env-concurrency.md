# Interpreter `i.env` Concurrency Issue

## Summary

The interpreter uses a single mutable field `i.env` (a pointer to `*Environment`) to track the current scope. Multiple goroutines — primarily the main REPL thread and the prediction goroutine — read and write this field concurrently without synchronization. This causes scope corruption where variables become undefined mid-execution.

A band-aid fix was applied in commit `c782ceb` ("fix: corrupted variable scope during agent execution") that adds save/restore guards at function boundaries. This prevents corruption from persisting after function returns, but does not prevent corruption during execution within those functions.

## How `i.env` Works

The interpreter uses a scope chain pattern. `i.env` always points to the "current" environment. Environments are linked: each has an optional `outer` pointer forming a chain up to the global scope. Variable lookup traverses this chain.

Every scope-creating construct saves and restores `i.env`:

```go
// Block statements, tool calls, event emission, etc.
prevEnv := i.env
i.env = NewEnclosedEnvironment(prevEnv)
defer func() { i.env = prevEnv }()
```

This pattern appears in:
- `evalBlockStatement` (`statements.go:332-336`) — if/else/for blocks
- `CallTool` (`expressions.go:612-616`) — tool/function calls
- `EmitEvent` (`interpreter.go:412-413`) — event handler dispatch
- `executeACPWithString` (`acp.go:77-78`) — ACP agent calls
- `sendPromptToACPSession` (`acp.go:134-135`) — ACP session prompts
- `executeAgentInternal` (`agent_loop.go:47-48`) — built-in agent execution
- `evalImportExpression` (`import.go:162-167`) — module imports

## The Concurrency Problem

### Goroutines involved

1. **Main REPL thread** — Runs user commands, executes middleware, calls agent tools. Reads/writes `i.env` constantly as it enters/exits scopes.

2. **Prediction goroutine** — Runs in the background to generate command predictions. Calls `interp.EmitEvent(EventReplPredict, ...)` via `EventPredictionProvider` (`internal/repl/predict/event_provider.go:86`). This triggers middleware tool calls that read/write `i.env`.

### How corruption happens

Consider the main thread running a middleware handler that calls an ACP agent inside an if block:

```
Main thread timeline:
1. CallTool(middleware):     saves env_A, sets i.env = toolEnv (has ctx, next)
2. evalBlockStatement(if):   saves toolEnv, sets i.env = ifScope
3. ACP pipe expression:      triggers EmitEvent calls for agent.start, agent.chunk, etc.
4. Each EmitEvent:           saves i.env, sets handler env, restores i.env
5. ...still inside if block, accesses ctx...
```

If the prediction goroutine fires between steps 3 and 5:

```
Prediction goroutine:
A. EmitEvent(repl.predict): saves i.env (currently ifScope), sets i.env = predictHandlerEnv
B. Runs prediction handler
C. Restores i.env = ifScope (the value it saved in step A)
```

But if step A happens AFTER step 4 has temporarily set `i.env` to a handler env, the prediction goroutine saves and later restores that temporary value, overwriting the main thread's environment. The main thread then finds `i.env` pointing to a wrong scope where `ctx` doesn't exist.

### What the c782ceb fix does

The fix adds `prevEnv := i.env; defer func() { i.env = prevEnv }()` to `EmitEvent`, `executeACPWithString`, `sendPromptToACPSession`, and `executeAgentInternal`. This ensures that when these functions return, `i.env` is restored to its pre-call value regardless of what happened during execution.

This helps because:
- After an ACP call returns from `executeACPWithString`, the defer restores `i.env` to the caller's scope
- After `EmitEvent` returns (from either goroutine), it restores `i.env`

But it doesn't fully solve the problem — corruption can still occur during execution if the interleaving is unlucky. The fix makes the window of corruption smaller and ensures recovery at function boundaries, but the underlying data race remains.

## Proposed Solutions

### Option A: Separate interpreter for predictions (recommended near-term)

Give the prediction goroutine its own interpreter instance (or a lightweight clone) so it never touches the main thread's `i.env`.

**Approach:**
- Create a `CloneForPrediction()` method on the interpreter that creates a minimal copy sharing read-only state (event handlers, provider registry) but with its own `env` field
- `EventPredictionProvider` uses this cloned interpreter for `EmitEvent` calls
- The clone's `env` would be initialized to the global scope (prediction handlers only need global-scope access, not the main thread's current nested scope)

**Pros:** Clean separation, no races possible, prediction can't affect main thread
**Cons:** Need to keep the clone's global scope in sync if new global vars are registered; prediction handlers can't see main thread's local state (but they shouldn't need to)

### Option B: Pass environment through the call chain (long-term ideal)

Instead of storing the current scope as `i.env`, pass it as a parameter through every `eval*` function:

```go
// Instead of:
func (i *Interpreter) evalExpression(node parser.Expression) (Value, error)

// Use:
func (i *Interpreter) evalExpression(env *Environment, node parser.Expression) (Value, error)
```

**Pros:** Eliminates shared mutable state entirely; each goroutine naturally has its own env; impossible to corrupt
**Cons:** Massive refactor touching every eval function in the interpreter; every `evalStatement`, `evalExpression`, `evalBlockStatement`, etc. needs an `env` parameter

### Option C: Mutex-protected `i.env` access

Wrap all reads/writes of `i.env` behind a mutex.

**Pros:** Minimal code changes
**Cons:** Serializes main thread and prediction goroutine — predictions would block during agent execution and vice versa, hurting responsiveness. Also, the granularity problem: the "read env, do work, write env" pattern isn't naturally atomic, so you'd need to hold the lock for the entire duration of scope-creating constructs, effectively single-threading the interpreter.

## Key Files

- `internal/script/interpreter/interpreter.go:411-413` — `EmitEvent` with save/restore
- `internal/script/interpreter/expressions.go:584-632` — `CallTool` with save/restore
- `internal/script/interpreter/statements.go:329-349` — `evalBlockStatement` with save/restore
- `internal/script/interpreter/acp.go:76-78, 133-135` — ACP execution save/restore
- `internal/script/interpreter/agent_loop.go:47-48` — Agent execution save/restore
- `internal/script/interpreter/environment.go` — `Environment` type and scope chain
- `internal/repl/predict/event_provider.go:63-89` — Prediction goroutine calling `EmitEvent`
- `internal/repl/input/prediction.go:241-299` — Debounced prediction running in background goroutine

## Reproduction

The test `TestACPScopePreservationInNestedBlocks` in `acp_test.go` simulates the corruption by having a mock ACP session directly overwrite `i.env` between update callbacks. This test was added in commit `c782ceb` and verifies the band-aid fix works at function boundaries.

To reproduce the actual race in practice:
1. Register a `command.input` middleware that calls an ACP agent (e.g., ClaudeCode) inside a conditional block
2. Have prediction enabled (default) so the prediction goroutine fires during agent execution
3. Access a tool parameter (e.g., `ctx`) after the ACP pipe expression returns
4. The variable may be undefined due to `i.env` pointing to a wrong scope

The race is timing-dependent and may not reproduce consistently. Using `-race` flag with Go tests or stress-testing with concurrent event emission increases the likelihood.
