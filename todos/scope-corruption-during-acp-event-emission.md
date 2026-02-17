# Scope Corruption During ACP Event Emission in Middleware Handlers

## Summary

Tool parameters (`ctx`, `next`) become undefined after an ACP pipe expression (`prompt | ACPAgent`) executes inside a middleware handler registered via `gsh.use("command.input", ...)`. The variable is accessible before the ACP call but not after, despite being in scope.

## Reproduction

Register a middleware tool for `command.input` that:
1. Has nested `if` / `else if` blocks
2. Calls an ACP agent via pipe expression inside the `else if` block
3. Accesses the `ctx` parameter after the ACP call returns

```gsh
acp ClaudeCode {
  command: "npx",
  args: ["-y", "@zed-industries/claude-code-acp"],
}

__state = null

tool myMiddleware(ctx, next) {
  if (ctx.input.startsWith("@test ")) {
    if (__state == null) {
      # First invocation - ctx works fine here
      prompt = "Hello"
      prompt | ClaudeCode
      __state = "active"
      ctx.input = "echo done"   # ctx accessible here
      return next(ctx)
    } else if (__state == "active") {
      # Second invocation - ctx fails after ACP call
      prompt = "Follow up"
      result = prompt | ClaudeCode

      # ERROR: "undefined variable: ctx" on the next line
      ctx.input = "echo done"
      return next(ctx)
    }
    return { handled: true }
  }
  return next(ctx)
}

gsh.use("command.input", myMiddleware)
```

Error message:
```
{"level":"warn","msg":"error in middleware handler","event":"command.input","handler":"myMiddleware","error":"undefined variable: ctx (line N, column N)\n\nStack trace:\n  at myMiddleware (tool 'myMiddleware')\n  at __defaultAgentMiddleware (tool '__defaultAgentMiddleware')"}
```

## Observations

- `ctx` works in the first `if` block (first invocation, `__state == null`)
- `ctx` works BEFORE the ACP pipe expression in the `else if` block (e.g., in string concatenation)
- `ctx` is undefined AFTER the ACP pipe expression returns
- The `ctx` parameter is bound in `toolEnv` via `CallTool` and should be reachable through the enclosed environment chain

## Analysis

### How scope should work

When `CallTool` executes `myMiddleware`:
1. Creates `toolEnv = NewEnclosedEnvironment(tool.Env)`
2. Binds `ctx` and `next` in `toolEnv` via `toolEnv.Set("ctx", ...)`
3. Sets `i.env = toolEnv`
4. Nested blocks (`if`, `else if`, inner `if`) create enclosed environments chaining back to `toolEnv`
5. `Environment.Get("ctx")` should traverse: `innerScope -> elseIfScope -> outerIfScope -> toolEnv (has ctx)`

### What happens during ACP execution

During `prompt | ClaudeCode`, the interpreter calls `EmitEvent()` many times for agent lifecycle events (`agent.start`, `agent.iteration.start`, `agent.chunk`, `agent.tool.pending`, `agent.tool.start`, `agent.tool.end`, `agent.iteration.end`, `agent.end`). Each event handler is invoked via `CallTool`, which:

1. Saves `prevEnv = i.env` (the current nested scope inside the middleware)
2. Sets `i.env = eventHandlerEnv`
3. Executes the event handler body
4. Restores `i.env = prevEnv` via `defer`

With many event handlers firing during a single ACP call (chunks, tool calls, etc.), there are dozens of save/restore cycles for `i.env`, all nested within the middleware handler's execution context.

### Suspected root cause

The `i.env` field on the `Interpreter` struct is not properly restored after the ACP execution completes. After `executeACPWithString` returns through `evalPipeExpression`, `i.env` should point back to `elseIfScope` (which chains to `toolEnv` containing `ctx`), but it appears to point to a different environment where `ctx` is not in the scope chain.

### Key code locations

- `internal/script/interpreter/expressions.go:584-636` - `CallTool` (env save/restore)
- `internal/script/interpreter/statements.go:326-346` - `evalBlockStatement` (env save/restore)
- `internal/script/interpreter/interpreter.go:411-482` - `EmitEvent` / `executeMiddlewareChain`
- `internal/script/interpreter/acp.go:76-212` - ACP execution (`executeACPWithString`, `sendPromptToACPSessionInternal`, `handleACPSessionUpdate`)
- `internal/script/interpreter/environment.go:38-44` - `Environment.Get` (scope chain traversal)

## Workaround

Capture `ctx` and `next` in local variables at the top of the tool body, before any ACP calls:

```gsh
tool myMiddleware(ctx, next) {
  _ctx = ctx
  _next = next
  # Use _ctx and _next throughout the function body
}
```

This creates a second reference in `toolEnv`'s scope that survives the ACP execution.

## Suggested debugging approach

1. Add logging to `CallTool` to print the `i.env` pointer value before save and after restore
2. Add logging to `evalIdentifier` when a lookup fails, printing the full scope chain (each environment's address and keys)
3. Run the reproduction case and compare the `i.env` pointer after `evalPipeExpression` returns vs. what it was before
4. This should reveal whether `i.env` is pointing to the wrong environment or if the scope chain's `outer` pointers are broken
