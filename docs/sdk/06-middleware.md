# Middleware

This chapter documents the command middleware system for intercepting user input.

**Availability:** REPL only

## Overview

Middleware functions process user commands before execution. They run in registration order, and each can either handle the command or pass it to the next middleware.

```
User Input
    ↓
Middleware Chain (first registered = first to run)
    ↓ (each middleware either handles or calls next())
    ↓
Fall-through: Execute as shell command
```

## API

### `gsh.useCommandMiddleware(middleware)`

Registers a middleware function. Returns an ID for later removal.

```gsh
tool myMiddleware(ctx, next) {
    if (ctx.input.trim() == "/hello") {
        print("Hello, world!")
        return { handled: true }
    }
    return next(ctx)
}

middlewareId = gsh.useCommandMiddleware(myMiddleware)
```

### `gsh.removeCommandMiddleware(middleware)`

Removes a previously registered middleware by reference.

```gsh
gsh.removeCommandMiddleware(myMiddleware)
```

## Middleware Signature

```gsh
tool myMiddleware(ctx, next) {
    # ctx.input contains the raw user input

    if (shouldHandle(ctx.input)) {
        # Handle the command and stop the chain
        return { handled: true }
    }

    # Optionally modify input before passing to next middleware
    ctx.input = transformedInput

    # Continue to next middleware in chain
    return next(ctx)
}
```

### Key Behaviors

| Action                       | Effect                                     |
| ---------------------------- | ------------------------------------------ |
| `return { handled: true }`   | Stop the chain; command is fully processed |
| `return next(ctx)`           | Continue to the next middleware            |
| Modify `ctx.input`           | Transform input for downstream middleware  |
| All middleware call `next()` | Falls through to shell execution           |

## Examples

### Custom Command

Add a `/time` command:

```gsh
tool timeMiddleware(ctx, next) {
    if (ctx.input.trim() == "/time") {
        result = exec("date")
        print(result.stdout)
        return { handled: true }
    }
    return next(ctx)
}

gsh.useCommandMiddleware(timeMiddleware)
```

Usage:

```bash
gsh> /time
Thu Jan 01 09:58:30 UTC 2026
```

### Command Aliases

Create shortcuts for common commands:

```gsh
tool aliasMiddleware(ctx, next) {
    aliases = {
        "ll": "ls -la",
        "gs": "git status",
        "gd": "git diff",
    }

    trimmed = ctx.input.trim()
    if (aliases[trimmed] != null) {
        ctx.input = aliases[trimmed]
    }

    return next(ctx)
}

gsh.useCommandMiddleware(aliasMiddleware)
```

### Input Logging

Log all commands:

```gsh
tool loggingMiddleware(ctx, next) {
    print("[LOG] " + ctx.input)
    return next(ctx)
}

gsh.useCommandMiddleware(loggingMiddleware)
```

## Default Middleware

The default middleware in `cmd/gsh/defaults/middleware/agent.gsh` handles the `#` prefix for agent chat. You can add your own middleware to extend or replace this behavior.

## Best Practices

1. **Keep middleware simple** - Complex logic makes debugging harder
2. **Always call `next(ctx)`** - Unless you're handling the command
3. **Consider order** - Place critical middleware first
4. **Handle edge cases** - Empty strings, special characters
5. **Document behavior** - Add comments explaining what each middleware does

## Troubleshooting

### Middleware not being called

- Check registration with `gsh.useCommandMiddleware()`
- Verify no earlier middleware returns `{ handled: true }`
- Enable debug logging: `gsh.logging.level = "debug"`

### Middleware interfering with other commands

- Ensure you call `next(ctx)` for commands you don't handle
- Check condition logic is correct
- Test with various input types

---

**Next:** [UI](07-ui.md) - Styling helpers and spinner API
