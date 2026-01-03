# Events

This chapter documents the event system for customizing gsh behavior.

**Availability:** REPL and scripts

## Overview

Events in gsh use a unified middleware chain model. Every event handler receives `(ctx, next)` and can:
- **Pass through**: Call `return next(ctx)` to continue to the next handler
- **Stop chain**: Return a value without calling `next()` to stop processing
- **Transform**: Modify `ctx` before calling `next(ctx)`
- **Override**: Return `{ result: "..." }` to override default behavior

```
Event Fired
    ↓
Handler 1 → calls next(ctx) → Handler 2 → calls next(ctx) → Handler 3 → returns value
                                                                            ↓
                                                                    Chain stops, value used
```

## Event Registration

### `gsh.use(eventName, handler)`

Registers an event handler. Returns a unique handler ID.

```gsh
tool myHandler(ctx, next) {
    print("Event fired!")
    return next(ctx)
}

gsh.use("repl.ready", myHandler)
```

### `gsh.remove(eventName, handler)`

Removes a previously registered handler by reference.

```gsh
gsh.remove("repl.ready", myHandler)
```

### `gsh.removeAll(eventName)`

Removes all handlers for an event. Returns the number of handlers removed.

```gsh
# Remove all default prompt handlers and register your own
gsh.removeAll("repl.prompt")

tool myPrompt(ctx, next) {
    gsh.prompt = "$ "
    return next(ctx)
}
gsh.use("repl.prompt", myPrompt)
```

## Handler Signature

All handlers use the middleware signature `(ctx, next)`:

```gsh
tool myHandler(ctx, next) {
    # ctx contains event-specific context
    
    # Option 1: Pass through to next handler
    return next(ctx)
    
    # Option 2: Stop chain and return override
    return { result: "override value" }
    
    # Option 3: Transform context, then continue
    ctx.someProperty = "modified"
    return next(ctx)
}
```

## REPL Events

### `repl.ready`

Fired when the REPL has fully started and is ready for input.

**Context:** `null`

```gsh
tool welcome(ctx, next) {
    print("Welcome to gsh!")
    return next(ctx)
}
gsh.use("repl.ready", welcome)
```

### `repl.prompt`

Fired after each command to generate the shell prompt. Set `gsh.prompt` to customize.

**Context:** `null`

```gsh
tool myPrompt(ctx, next) {
    if (gsh.lastCommand.exitCode == 0) {
        gsh.prompt = "✓ gsh> "
    } else {
        gsh.prompt = "✗ gsh> "
    }
    return next(ctx)
}
gsh.use("repl.prompt", myPrompt)
```

### `command.input`

Fired when user submits a command. This is the unified middleware for processing user input.

**Context:**

| Property    | Type     | Description        |
| ----------- | -------- | ------------------ |
| `ctx.input` | `string` | The raw user input |

**Return Value:** Return `{ handled: true }` to stop processing (command won't execute as shell command).

```gsh
# Handle # prefix for agent chat
tool agentMiddleware(ctx, next) {
    if (ctx.input.startsWith("#")) {
        message = ctx.input.substring(1).trim()
        # ... process agent message ...
        return { handled: true }
    }
    return next(ctx)
}
gsh.use("command.input", agentMiddleware)
```

## Agent Events

These events fire during agent interactions.

### `agent.start`

Fired when an agent begins responding to a query.

**Context:**

| Property       | Type     | Description                      |
| -------------- | -------- | -------------------------------- |
| `ctx.agent`    | `string` | Name of the agent                |
| `ctx.message`  | `string` | The user's message to the agent  |

```gsh
tool agentStarted(ctx, next) {
    print("🤖 Agent is thinking...")
    return next(ctx)
}
gsh.use("agent.start", agentStarted)
```

### `agent.iteration.start`

Fired at the beginning of each agent iteration.

**Context:**

| Property        | Type     | Description           |
| --------------- | -------- | --------------------- |
| `ctx.iteration` | `number` | Current iteration (1-based) |

```gsh
tool iterationStart(ctx, next) {
    print("Starting iteration " + ctx.iteration)
    return next(ctx)
}
gsh.use("agent.iteration.start", iterationStart)
```

### `agent.chunk`

Fired when a chunk of agent output is received (streaming).

**Context:**

| Property      | Type     | Description             |
| ------------- | -------- | ----------------------- |
| `ctx.content` | `string` | The text chunk received |

```gsh
tool chunkReceived(ctx, next) {
    # Custom chunk handling
    gsh.ui.write(ctx.content)
    return next(ctx)
}
gsh.use("agent.chunk", chunkReceived)
```

### `agent.end`

Fired when the agent finishes responding.

**Context:**

| Property                 | Type               | Description                    |
| ------------------------ | ------------------ | ------------------------------ |
| `ctx.query.inputTokens`  | `number`           | Input tokens used              |
| `ctx.query.outputTokens` | `number`           | Output tokens used             |
| `ctx.query.cachedTokens` | `number`           | Cached tokens (if supported)   |
| `ctx.query.durationMs`   | `number`           | Total duration in milliseconds |
| `ctx.error`              | `string` or `null` | Error message if failed        |

```gsh
tool agentFinished(ctx, next) {
    if (ctx.error != null) {
        print("Error: " + ctx.error)
        return next(ctx)
    }

    durationSec = (ctx.query.durationMs / 1000).toFixed(1)
    print("── " + ctx.query.inputTokens + " in, " + ctx.query.outputTokens + " out (" + durationSec + "s) ──")
    return next(ctx)
}
gsh.use("agent.end", agentFinished)
```

## Tool Events

These events fire when agents call tools.

### `agent.tool.pending`

Fired when a tool call is streaming from the model (arguments not yet complete).

**Context:**

| Property            | Type     | Description                          |
| ------------------- | -------- | ------------------------------------ |
| `ctx.toolCall.id`   | `string` | Unique identifier for this tool call |
| `ctx.toolCall.name` | `string` | Name of the tool being called        |

```gsh
tool toolPending(ctx, next) {
    gsh.ui.spinner.start(ctx.toolCall.name, ctx.toolCall.id)
    return next(ctx)
}
gsh.use("agent.tool.pending", toolPending)
```

### `agent.tool.start`

Fired when a tool begins execution (arguments are complete).

**Context:**

| Property            | Type     | Description                          |
| ------------------- | -------- | ------------------------------------ |
| `ctx.toolCall.id`   | `string` | Unique identifier for this tool call |
| `ctx.toolCall.name` | `string` | Name of the tool being called        |
| `ctx.toolCall.args` | `object` | Tool arguments                       |

**Return Value:** Return `{ result: "..." }` to skip execution and use the returned result. Add `error: "..."` to mark as failed.

```gsh
# Permission system example
tool toolPermissions(ctx, next) {
    if (ctx.toolCall.name == "exec") {
        command = ctx.toolCall.args.command
        if (command.includes("rm -rf")) {
            return {
                result: "Permission denied: This command is not allowed.",
                error: "Blocked by permission system"
            }
        }
    }
    # Continue to normal execution
    return next(ctx)
}
gsh.use("agent.tool.start", toolPermissions)
```

### `agent.tool.end`

Fired when a tool finishes execution.

**Context:**

| Property                  | Type               | Description                          |
| ------------------------- | ------------------ | ------------------------------------ |
| `ctx.toolCall.id`         | `string`           | Unique identifier for this tool call |
| `ctx.toolCall.name`       | `string`           | Name of the tool being called        |
| `ctx.toolCall.args`       | `object`           | Tool arguments                       |
| `ctx.toolCall.output`     | `string`           | Tool output (if successful)          |
| `ctx.toolCall.error`      | `string` or `null` | Error message (if failed)            |
| `ctx.toolCall.durationMs` | `number`           | Execution time in milliseconds       |

**Return Value:** Return `{ result: "..." }` to override the tool output passed to the agent.

```gsh
# Redact sensitive output
tool redactSecrets(ctx, next) {
    if (ctx.toolCall.output != null) {
        if (ctx.toolCall.output.includes("API_KEY")) {
            return { result: "[OUTPUT REDACTED]" }
        }
    }
    return next(ctx)
}
gsh.use("agent.tool.end", redactSecrets)
```

## Handler Chain Behavior

When multiple handlers are registered, they run in registration order. Each handler can:

1. **Continue the chain**: `return next(ctx)` - passes control to next handler
2. **Stop the chain**: Return without calling `next()` - no more handlers run

The **return value propagates back** through the chain:

```gsh
tool handler1(ctx, next) {
    print("handler1 before")
    result = next(ctx)  # Call next handler
    print("handler1 after")
    return result
}

tool handler2(ctx, next) {
    print("handler2")
    return { result: "from handler2" }  # Stop chain, return value
}

gsh.use("some.event", handler1)
gsh.use("some.event", handler2)

# Output:
# handler1 before
# handler2
# handler1 after
# Final result: { result: "from handler2" }
```

## Best Practices

1. **Always call `next(ctx)`** unless you're intentionally stopping the chain
2. **Keep handlers fast** - They run frequently; avoid expensive operations
3. **Handle null gracefully** - Not all context properties are always present
4. **Use return values sparingly** - Only when you need to override behavior
5. **Check event names** - Typos silently fail to register
6. **Test incrementally** - Add one handler at a time

---

**Next:** [UI](06-ui.md) - Styling helpers and spinner API
