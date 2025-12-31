# Core Properties

This chapter documents the core properties of the `gsh` object.

## `gsh.version`

**Type:** `string` (read-only)  
**Availability:** REPL + Script

Returns the current gsh version.

```gsh
print("Running gsh " + gsh.version)
# Output: Running gsh 1.0.0
```

## `gsh.terminal`

**Type:** `object` (read-only)  
**Availability:** REPL + Script

Provides information about the current terminal.

### Properties

| Property              | Type      | Description                   |
| --------------------- | --------- | ----------------------------- |
| `gsh.terminal.width`  | `number`  | Terminal width in columns     |
| `gsh.terminal.height` | `number`  | Terminal height in rows       |
| `gsh.terminal.isTTY`  | `boolean` | Whether running in a terminal |

### Example

```gsh
if (gsh.terminal.isTTY) {
    print("Terminal: " + gsh.terminal.width + "x" + gsh.terminal.height)
} else {
    print("Running in non-interactive mode")
}
```

Use terminal dimensions to format output appropriately for the user's screen size.

## `gsh.logging`

**Type:** `object`  
**Availability:** REPL + Script

Controls logging behavior.

### Properties

| Property            | Type                  | Description                                         |
| ------------------- | --------------------- | --------------------------------------------------- |
| `gsh.logging.level` | `string` (read/write) | Log level: `"debug"`, `"info"`, `"warn"`, `"error"` |
| `gsh.logging.file`  | `string` (read-only)  | Path to the log file                                |

### Log Levels

```gsh
gsh.logging.level = "debug"    # Most verbose - shows all debug info
gsh.logging.level = "info"     # Normal operation (default)
gsh.logging.level = "warn"     # Warnings and errors only
gsh.logging.level = "error"    # Errors only
```

### Example

```gsh
# Enable debug logging for troubleshooting
gsh.logging.level = "debug"

# Check where logs are written
print("Logs written to: " + gsh.logging.file)
```

View logs with:

```bash
tail -f ~/.gsh/gsh.log
```

## `gsh.prompt`

**Type:** `string` (write-only)  
**Availability:** REPL only

Sets the shell prompt string. Typically used in a `repl.prompt` event handler.

### Example

```gsh
tool myPrompt() {
    gsh.prompt = "my-shell> "
}
gsh.on("repl.prompt", myPrompt)
```

### Dynamic Prompts

Build prompts that reflect the current state:

```gsh
tool dynamicPrompt() {
    cwd = exec("pwd").stdout.trim()

    if (gsh.lastCommand.exitCode == 0) {
        gsh.prompt = "✓ " + cwd + " > "
    } else {
        gsh.prompt = "✗ " + cwd + " > "
    }
}
gsh.on("repl.prompt", dynamicPrompt)
```

For more prompt customization options including Starship integration, see the [Tutorial](../tutorial/02-configuration.md).

## `gsh.lastCommand`

**Type:** `object` (read-only)  
**Availability:** REPL only

Information about the most recently executed command.

### Properties

| Property                     | Type     | Description                              |
| ---------------------------- | -------- | ---------------------------------------- |
| `gsh.lastCommand.exitCode`   | `number` | Exit code of last command (0 = success)  |
| `gsh.lastCommand.durationMs` | `number` | Duration of last command in milliseconds |

### Example

```gsh
tool showStats() {
    exitCode = gsh.lastCommand.exitCode
    durationSec = gsh.lastCommand.durationMs / 1000

    if (exitCode != 0) {
        print("Command failed with exit code: " + exitCode)
    }
    print("Duration: " + durationSec + "s")
}
gsh.on("repl.prompt", showStats)
```

---

**Next:** [Models](02-models.md) - Model tiers and configuration
