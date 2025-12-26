# Chapter 03: Custom Prompts with Starship

The default `gsh>` prompt is functional but plain. In this chapter, you'll learn how to create sophisticated, informative prompts using **Starship**, a popular cross-shell prompt generator.

## Why Customize Your Prompt?

A good prompt shows you:

- **Current directory** - Where am I right now?
- **Git status** - Am I in a repo? What branch?
- **Exit code** - Did the last command succeed?
- **Execution time** - How long did that take?
- **Programming language** - What runtime/version is active?
- **System status** - CPU, memory, or other metrics

Starship makes it easy to add all of this without complex shell scripting.

## Installing Starship

First, install Starship following its [official documentation](https://starship.rs/guide/#🚀-installation).

Quick install on macOS:

```bash
brew install starship
```

On Linux:

```bash
curl -sS https://starship.rs/install.sh | sh
```

Verify installation:

```bash
starship --version
```

## Basic gsh Starship Integration

To use Starship with gsh, add this to your `~/.gshrc.gsh`:

```gsh
# ~/.gshrc.gsh

# Initialize Starship environment variables
STARSHIP_SHELL = "gsh"
STARSHIP_SESSION_KEY = `starship session`
STARSHIP_START_TIME = `starship time`

# Update prompt after each command (gsh calls this tool automatically)
tool GSH_PROMPT(exitCode: number, durationMs: number): string {
    return `starship prompt --status=${exitCode} --cmd-duration=${durationMs}`
}
```

Now start a new gsh session:

```bash
gsh
```

You should see a much fancier prompt showing your current directory, git branch, and more!

## Understanding the Prompt Update Mechanism

Here's how gsh's dynamic prompt works:

1. **At startup**, the REPL loads your `.gshrc.gsh` configuration
2. **After each command execution**, gsh calls the `GSH_PROMPT()` tool with:
   - `exitCode` - The exit status of the command (0 for success, non-zero for failure)
   - `durationMs` - How long the command took in milliseconds
3. **The returned string** becomes the next prompt

The tool receives the exit code and duration as parameters and can use them to create a dynamic prompt that reflects the result of the previous command.

## Configuring Starship

Starship is highly configurable via `~/.config/starship.toml`. Here's a practical configuration:

```toml
# ~/.config/starship.toml

# Configure the command timeout
command_timeout = 500

# Shows exit code of last command if non-zero
[status]
disabled = false
success_symbol = "[✓](bold green)"
error_symbol = "[✗](bold red)"
format = '[$symbol $common_meaning $signal_name]($style) '

# Shows current directory
[directory]
truncation_length = 3
truncate_to_repo = true

# Shows git branch and status
[git_branch]
symbol = " "
format = "on [$symbol$branch(:$remote_branch)]($style) "

[git_status]
format = '([\[$all_status$ahead_behind\]]($style) )'
conflicted = "🏳"
up_to_date = "✓"
untracked = "?"
stashed = "$"
modified = "!"
staged = "+"
renamed = "»"
deleted = "✘"

# Language versions (shown when relevant)
[nodejs]
symbol = " "
format = "[$symbol($version )]($style)"

[python]
symbol = " "
format = "[$symbol($version )]($style)"

[rust]
symbol = "🦀 "
format = "[$symbol($version )]($style)"

# Time display (optional)
[time]
disabled = false
format = "[$time]($style) "
style = "dimmed white"
```

## Advanced: Full Featured Prompt

Here's a more complete example matching the reference configuration at `cmd/gsh/.gshrc.starship`:

**~/.gshrc.gsh**

```gsh
# Initialize Starship environment
STARSHIP_SHELL = "gsh"
STARSHIP_SESSION_KEY = `starship session`
STARSHIP_START_TIME = `starship time`

# Prompt update function (called after each command)
tool GSH_PROMPT(exitCode: number, durationMs: number): string {
    prompt = `starship prompt --status=${exitCode} --cmd-duration=${durationMs}`
    return prompt
}

# Optional: Custom greeting on startup
print("🚀 Welcome to gsh")
```

**~/.config/starship.toml**

```toml
format = """
$username\
$hostname\
$directory\
$git_branch\
$git_status\
$nodejs\
$python\
$rust\
$line_break\
$status\
$character"""

# User info (optional)
[username]
show_always = true
format = "[$user]($style) "
style_user = "white bold"
style_root = "red bold"

# Hostname (optional)
[hostname]
ssh_only = true
format = "on [$hostname]($style) "
style = "bold blue"

# Directory
[directory]
truncation_length = 3
truncate_to_repo = true
format = "[$path]($style) "
style = "cyan bold"

[directory.substitutions]
"Documents" = " "
"Downloads" = " "
"Music" = " "
"Pictures" = " "

# Git integration
[git_branch]
symbol = " "
format = "on [$symbol$branch]($style) "
style = "purple bold"

[git_status]
format = '([\[$all_status$ahead_behind\]]($style) )'
style = "red bold"
conflicted = "🏳"
up_to_date = "✓"
untracked = "?"
stashed = "$"
modified = "!"
staged = "+"
renamed = "»"
deleted = "✘"

# Languages
[nodejs]
symbol = " "
format = "[$symbol($version )]($style)"
style = "green bold"

[python]
symbol = " "
format = "[$symbol($version )]($style)"
style = "yellow bold"

[rust]
symbol = "🦀 "
format = "[$symbol($version )]($style)"
style = "red bold"

# Status line
[status]
disabled = false
success_symbol = "[✓](bold green)"
error_symbol = "[✗](bold red)"
format = '[$symbol $common_meaning]($style) '
style = "red bold"

# Character (the actual prompt symbol)
[character]
success_symbol = "[➜](bold green)"
error_symbol = "[➜](bold red)"
```

## Troubleshooting

### Starship not appearing

If your prompt still looks like `gsh>`:

1. Check Starship is installed: `starship --version`
2. Check `.gshrc.gsh` has no syntax errors (look for parse errors at startup)
3. Ensure `GSH_PROMPT` tool is defined in your `.gshrc.gsh`
4. Check logs for errors: `tail -f ~/.gsh.log`
5. Try running Starship manually to verify it works: `starship prompt --status=0 --cmd-duration=0`

### Strange characters or rendering issues

This usually means your terminal doesn't support the Nerd Font symbols. Solutions:

1. **Install a Nerd Font** - Download from [nerdfonts.com](https://www.nerdfonts.com/)
2. **Use simpler symbols** in your Starship config:
   ```toml
   [git_branch]
   symbol = "[branch]"
   format = "on [$symbol $branch]($style) "
   ```
3. **Check terminal encoding** - Ensure it's set to UTF-8

### Prompt not updating between commands

This might mean `GSH_PROMPT` has an error:

1. Add debug output to see what's happening:

   ```gsh
   tool GSH_PROMPT(exitCode: number, durationMs: number): string {
       print(`DEBUG: exitCode=${exitCode}, durationMs=${durationMs}`)
       return `gsh [${exitCode}]> `
   }
   ```

2. Check logs: `tail -f ~/.gsh.log`

### Slow prompt rendering

If your prompt takes a long time to appear:

1. Increase `command_timeout` in `~/.config/starship.toml`
2. Disable git status for large repositories
3. Disable language detection if not needed

## Prompt Best Practices

1. **Keep it readable** - Too much information becomes clutter
2. **Use colors wisely** - Draw attention to important info
3. **Test with different terminal themes** - Some colors work better with light backgrounds
4. **Profile performance** - A slow prompt is worse than a plain one
5. **Keep `.gshrc.gsh` simple** - Don't do expensive operations in `GSH_PROMPT`

## Examples

### Minimal Prompt

Just directory and git status:

```toml
format = "$directory$git_branch$git_status$character"

[character]
success_symbol = "[→](bold green)"
error_symbol = "[→](bold red)"
```

### Developer-Focused Prompt

Shows languages, git, and execution time:

```toml
format = """
$directory\
$git_branch\
$git_status\
$nodejs\
$python\
$rust\
$line_break\
$status\
$character"""
```

### Minimal but Informative

Just the essentials:

```toml
format = "$directory $character "

[character]
success_symbol = "[→](green)"
error_symbol = "[→](red)"
```

## What's Next?

Your shell now has a beautiful, informative prompt! Chapter 04 covers **Configuring the Predict Model**—how to set up LLM-based command prediction so gsh suggests commands as you type.

---

**Previous Chapter:** [Chapter 02: Configuration](02-configuration.md)

**Next Chapter:** [Chapter 04: Command Prediction with LLMs](04-command-prediction.md)
