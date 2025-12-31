# gsh SDK Reference

This reference documents the built-in `gsh` SDK object and all its capabilities.
The `gsh` object is the primary API for configuring and extending gsh, available in both REPL mode and script execution.

## Quick Reference

| Property/Method              | Description                                  | Availability  |
| ---------------------------- | -------------------------------------------- | ------------- |
| `gsh.version`                | Current gsh version                          | REPL + Script |
| `gsh.terminal`               | Terminal dimensions and TTY info             | REPL + Script |
| `gsh.logging`                | Log level and file configuration             | REPL + Script |
| `gsh.models`                 | Model tier system (lite, workhorse, premium) | REPL + Script |
| `gsh.tools`                  | Built-in tools for agents                    | REPL + Script |
| `gsh.prompt`                 | Set the shell prompt                         | REPL only     |
| `gsh.lastCommand`            | Exit code and duration of last command       | REPL only     |
| `gsh.on()` / `gsh.off()`     | Event handler registration                   | REPL only     |
| `gsh.useCommandMiddleware()` | Command middleware registration              | REPL only     |
| `gsh.ui.styles`              | Text styling helpers                         | REPL + Script |
| `gsh.ui.spinner`             | Loading spinner API                          | REPL + Script |

## Configuration File

The `~/.gsh/repl.gsh` file uses the gsh scripting language to configure your environment. This file is optional—gsh uses sensible defaults if it doesn't exist.

```gsh
# ~/.gsh/repl.gsh

# Configure models
model myModel {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-5.2",
}
gsh.models.workhorse = myModel

# Set logging level
gsh.logging.level = "info"
```

You can study the default configuration in `cmd/gsh/defaults/` as a reference.

## Chapters

1. **[Core Properties](01-gsh-object.md)** - Version, terminal, logging, prompt, lastCommand
2. **[Models](02-models.md)** - Model tiers and model declaration syntax
3. **[Tools](03-tools.md)** - Built-in tools for agents (exec, grep, view_file, edit_file)
4. **[Agents](04-agents.md)** - Defining and using custom agents
5. **[Events](05-events.md)** - Event system with gsh.on() and gsh.off()
6. **[Middleware](06-middleware.md)** - Command middleware for intercepting user input
7. **[UI](07-ui.md)** - Styling helpers and spinner API

## Related Resources

- **[Tutorial](../tutorial/README.md)** - Guided introduction to gsh
- **[Script Guide](../script/README.md)** - Full gsh scripting language reference
- **[Main README](../../README.md)** - Installation and overview

## Debugging

Enable debug logging to troubleshoot configuration issues:

```gsh
gsh.logging.level = "debug"
```

Then view the logs:

```bash
tail -f ~/.gsh/gsh.log
```
