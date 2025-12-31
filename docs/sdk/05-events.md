# Events

This chapter documents the event system for customizing gsh behavior.

**Availability:** REPL only

## Event Registration

### `gsh.on(eventName, handler)`

Registers an event handler.

```gsh
tool myHandler(ctx) {
    print("Event fired!")
}

gsh.on("repl.ready", myHandler)
```

### `gsh.off(eventName)`

Unregisters all handlers for an event.

```gsh
gsh.off("repl.ready")
```

### `gsh.removeEventHandler(eventName, handler)`

Removes a specific handler by reference.

```gsh
gsh.removeEventHandler("repl.ready", myHandler)
```

## REPL Events

### `repl.ready`

Fired when the REPL has fully started and is ready for input.

**Context:** None

```gsh
tool welcome() {
    print("Welcome to gsh!")
}
gsh.on("repl.ready", welcome)
```

### `repl.prompt`

Fired after each command to generate the shell prompt. Set `gsh.prompt` to customize.

**Context:** None

```gsh
tool myPrompt() {
    if (gsh.lastCommand.exitCode == 0) {
        gsh.prompt = "✓ gsh> "
    } else {
        gsh.prompt = "✗ gsh> "
    }
}
gsh.on("repl.prompt", myPrompt)
```

## Agent Events

These events fire during agent interactions.

### `agent.start`

Fired when an agent begins responding to a query.

**Context:** None

```gsh
tool agentStarted() {
    print("🤖 Agent is thinking...")
}
gsh.on("agent.start", agentStarted)
```

### `agent.iteration.start`

Fired at the beginning of each agent iteration. An iteration ends when the agent gives a final response without tool calls.

**Context:** None

```gsh
tool iterationStart() {
    print("Starting agent iteration...")
}
gsh.on("agent.iteration.start", iterationStart)
```

### `agent.chunk`

Fired when a chunk of agent output is received (streaming).

**Context:**

| Property      | Type     | Description             |
| ------------- | -------- | ----------------------- |
| `ctx.content` | `string` | The text chunk received |

```gsh
tool chunkReceived(ctx) {
    # Chunks are printed automatically; use for custom handling
}
gsh.on("agent.chunk", chunkReceived)
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
tool agentFinished(ctx) {
    if (ctx.error != null) {
        print("Error: " + ctx.error)
        return
    }

    durationSec = (ctx.query.durationMs / 1000).toFixed(1)
    print("── " + ctx.query.inputTokens + " in, " + ctx.query.outputTokens + " out (" + durationSec + "s) ──")
}
gsh.on("agent.end", agentFinished)
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
tool toolPending(ctx) {
    print("► Calling " + ctx.toolCall.name + "...")
}
gsh.on("agent.tool.pending", toolPending)
```

### `agent.tool.start`

Fired when a tool begins execution (arguments are complete).

**Context:**

| Property            | Type     | Description                          |
| ------------------- | -------- | ------------------------------------ |
| `ctx.toolCall.id`   | `string` | Unique identifier for this tool call |
| `ctx.toolCall.name` | `string` | Name of the tool being called        |
| `ctx.toolCall.args` | `object` | Tool arguments                       |

**Return Value:** Returning `{ result: "..." }` skips tool execution and uses the returned result. Add `error: "..."` to mark as failed.

```gsh
# Permission system example
tool toolPermissions(ctx) {
    if (ctx.toolCall.name == "exec") {
        command = ctx.toolCall.args.command
        if (command.includes("rm -rf")) {
            return {
                result: "Permission denied: This command is not allowed.",
                error: "Blocked by permission system"
            }
        }
    }
    # No return = allow normal execution
}
gsh.on("agent.tool.start", toolPermissions)
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

**Return Value:** Returning `{ result: "..." }` overrides the tool output passed to the agent.

```gsh
# Redact sensitive output
tool redactSecrets(ctx) {
    if (ctx.toolCall.output != null) {
        if (ctx.toolCall.output.includes("API_KEY")) {
            return { result: "[OUTPUT REDACTED]" }
        }
    }
}
gsh.on("agent.tool.end", redactSecrets)
```

## Event Handler Return Values

Some events support return values to override default behavior:

| Event              | Return Value                      | Effect                          |
| ------------------ | --------------------------------- | ------------------------------- |
| `agent.tool.start` | `{ result: "..." }`               | Skip execution, use this result |
| `agent.tool.start` | `{ result: "...", error: "..." }` | Skip execution, mark as error   |
| `agent.tool.end`   | `{ result: "..." }`               | Override the tool output        |

### Multiple Handlers

When multiple handlers are registered, they run in registration order. The **first handler that returns a non-null value** determines the override.

```gsh
tool handler1(ctx) {
    if (ctx.toolCall.name == "exec") {
        return { result: "blocked" }  # This wins
    }
}

tool handler2(ctx) {
    return { result: "also blocked" }  # Ignored if handler1 returned
}

gsh.on("agent.tool.start", handler1)
gsh.on("agent.tool.start", handler2)
```

## Best Practices

1. **Keep handlers fast** - They run frequently; avoid expensive operations
2. **Handle null gracefully** - Not all context properties are always present
3. **Use return values sparingly** - Only when you need to override behavior
4. **Check event names** - Typos silently fail to register
5. **Test incrementally** - Add one handler at a time

---

**Next:** [Middleware](06-middleware.md) - Command middleware system
