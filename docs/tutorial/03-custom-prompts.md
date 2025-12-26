# Chapter 03: Custom Prompts

The default `gsh>` prompt is functional but plain. In this chapter, you'll learn how to create custom prompts using the `GSH_PROMPT` tool.

## The GSH_PROMPT Tool

gsh uses a special tool called `GSH_PROMPT` to generate your shell prompt. After each command, gsh calls this tool with information about the previous command, and uses the returned string as the next prompt.

### Parameters

The `GSH_PROMPT` tool receives two parameters:

- **`exitCode`** (number) - The exit status of the last command (0 for success, non-zero for failure)
- **`durationMs`** (number) - How long the last command took in milliseconds

### Basic Example

Here's a simple custom prompt that shows the exit code:

```gsh
# ~/.gshrc.gsh

tool GSH_PROMPT(exitCode: number, durationMs: number): string {
    if (exitCode == 0) {
        return "✓ gsh> "
    }
    return "✗ gsh> "
}
```

### Including More Information

You can build richer prompts by including directory, timing, or other context:

```gsh
# ~/.gshrc.gsh

tool GSH_PROMPT(exitCode: number, durationMs: number): string {
    # Get current directory
    cwd = exec("pwd").stdout.trim()

    # Format duration
    durationSec = durationMs / 1000

    # Build prompt
    status = "✓"
    if (exitCode != 0) {
        status = "✗"
    }

    return status + " " + cwd + " (" + durationSec + "s) > "
}
```

## Automatic Starship Integration

[Starship](https://starship.rs) is a popular cross-shell prompt that provides beautiful, informative prompts out of the box. If you have Starship installed, gsh automatically detects it and uses it for prompt generation.

### How It Works

When gsh starts, it checks if `starship` is in your PATH. If found (and not disabled), gsh automatically:

- Sets up the `STARSHIP_SHELL` environment variable
- Initializes a Starship session
- Defines `GSH_PROMPT` to use Starship

### Disabling Starship Integration

If you have Starship installed but want to use your own custom prompt instead, disable the integration in your `~/.gshrc.gsh`:

```gsh
# ~/.gshrc.gsh

GSH_CONFIG.starshipIntegration = false

# Now define your own GSH_PROMPT
tool GSH_PROMPT(exitCode: number, durationMs: number): string {
    return "my-prompt> "
}
```

### Installing Starship

If you'd like to use Starship but don't have it installed, see the [official installation guide](https://starship.rs/guide/).

### Manual Starship Configuration

If you want to customize how gsh integrates with Starship, you can disable the automatic integration and configure it yourself:

```gsh
# ~/.gshrc.gsh

GSH_CONFIG.starshipIntegration = false

# Set up Starship environment
env.STARSHIP_SHELL = "gsh"
env.STARSHIP_SESSION_KEY = exec("starship session").stdout

# Define GSH_PROMPT using Starship
tool GSH_PROMPT(exitCode: number, durationMs: number): string {
    result = exec(`starship prompt --status=${exitCode} --cmd-duration=${durationMs}`)
    if (result.exitCode == 0) {
        return result.stdout
    }
    return "gsh> "
}
```

## Troubleshooting

### Prompt not updating

If your prompt isn't changing between commands:

1. Check for syntax errors in your `GSH_PROMPT` tool
2. Add debug output to see what's happening:
   ```gsh
   tool GSH_PROMPT(exitCode: number, durationMs: number): string {
       print("DEBUG: exitCode=" + exitCode)
       return "gsh> "
   }
   ```
3. Check logs: `tail -f ~/.gsh.log`

### Slow prompt

If your prompt takes a long time to appear:

1. Avoid expensive operations in `GSH_PROMPT` (network calls, heavy git operations)
2. Cache values that don't change often
3. If using Starship, configure `command_timeout` in `~/.config/starship.toml`

### Strange characters

If you see garbled text or boxes:

1. Ensure your terminal uses UTF-8 encoding
2. If using icon fonts, install a [Nerd Font](https://www.nerdfonts.com/)
3. Use simpler ASCII characters in your prompt

## Best Practices

1. **Keep it fast** - The prompt runs after every command; slow prompts are frustrating
2. **Keep it readable** - Too much information becomes clutter
3. **Use color sparingly** - Draw attention to what matters
4. **Handle errors gracefully** - Return a fallback prompt if something fails
5. **Test in different terminals** - Colors and symbols may render differently

## What's Next?

Now that you have a custom prompt, Chapter 04 covers **Command Prediction**—how to set up LLM-based command suggestions as you type.

---

**Previous Chapter:** [Chapter 02: Configuration](02-configuration.md)

**Next Chapter:** [Chapter 04: Command Prediction](04-command-prediction.md)
