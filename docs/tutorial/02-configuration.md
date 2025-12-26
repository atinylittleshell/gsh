# Chapter 02: Configuration

gsh uses a two-file configuration system that separates bash compatibility from gsh-specific features. In this chapter, you'll learn how to configure your gsh environment.

## The Two Configuration Files

### `.gshrc` - Bash Compatible Configuration

The `.gshrc` file in your home directory is executed as **pure bash** before the REPL starts. Use this for:

- Shell aliases
- Shell functions
- PATH modifications
- Environment variables
- Any standard bash code

**Important**: This file is NOT executed by the gsh interpreter, so you cannot use gsh-specific syntax here.

### `.gshrc.gsh` - gsh-Specific Configuration

The `.gshrc.gsh` file in your home directory is executed using the **gsh scripting language**. Use this for:

- gsh-specific configuration
- Model declarations
- Agent declarations
- MCP server declarations
- Custom tools
- gsh language features

Both files are optional. If they don't exist, gsh uses sensible defaults.

## Your First `.gshrc`

Create `~/.gshrc` with some basic shell configuration:

```bash
# ~/.gshrc
# Aliases
alias ll='ls -la'
alias gs='git status'
alias gca='git commit -am'

# Shell functions
function greet() {
    echo "Hello, $1!"
}

# PATH modifications
export PATH="$HOME/.local/bin:$PATH"

# Environment variables
export EDITOR=vim
export LANG=en_US.UTF-8
```

After saving, start a new gsh session:

```bash
gsh
```

Your aliases and functions are now available:

```bash
gsh> ll
total 128
drwxr-xr-x  user  group    4096 Dec 25 00:58 Documents
drwxr-xr-x  user  group    4096 Dec 25 00:58 Downloads
...

gsh> greet Alice
Hello, Alice!
```

## Your First `.gshrc.gsh`

Now create `~/.gshrc.gsh` with gsh-specific configuration:

```gsh
# ~/.gshrc.gsh

# Define a custom tool
tool greet_formal(name: string, title: string): string {
    return `Good day, ${title} ${name}. It is a pleasure to meet you.`
}

# Define a variable
GREETING = "Welcome to gsh!"

# Print a welcome message
print(GREETING)
```

After saving, start a new gsh session. You should see:

```
Welcome to gsh!
gsh>
```

Inside the REPL, you can use the tool:

```bash
gsh> greet_formal "Alice" "Dr."
Good day, Dr. Alice. It is a pleasure to meet you.
```

## Understanding gsh Configuration

The gsh REPL has built-in configuration through a `GSH_CONFIG` object. This is defined in the default configuration but can be overridden in your `.gshrc.gsh`.

### Default Configuration

gsh comes with sensible defaults defined in an embedded `.gshrc.default.gsh`. These include:

- Default models for predictions and agents
- Default prompt string
- Log level setting
- Starship integration
- Welcome screen display

You typically don't need to modify these, but you can override them in your `~/.gshrc.gsh`.

### Configuration Merging

When you define `GSH_CONFIG` in your `~/.gshrc.gsh`, your settings are **merged** with the defaults rather than replacing them entirely. This means you only need to specify the settings you want to change—all other settings retain their default values.

For example, if you define in your `~/.gshrc.gsh`:

```gsh
GSH_CONFIG = {
    logLevel: "debug",
}
```

The final configuration will be:

```gsh
{
    prompt: "gsh> ",              # preserved from default
    logLevel: "debug",            # your override
    starshipIntegration: true,    # preserved from default
    showWelcome: true,            # preserved from default
    predictModel: GSH_PREDICT_MODEL,      # preserved from default
    defaultAgentModel: GSH_AGENT_MODEL,   # preserved from default
}
```

### Custom Configuration Example

Here's a more complete `~/.gshrc.gsh` example:

```gsh
# ~/.gshrc.gsh

# Override specific settings (other defaults are preserved)
GSH_CONFIG = {
    prompt: "my-shell> ",
    logLevel: "info",
}

# Define custom tools
tool uppercase(text: string): string {
    return text.toUpperCase()
}

tool lowercase(text: string): string {
    return text.toLowerCase()
}

# Define useful variables
API_BASE_URL = "https://api.example.com"
```

## Learning the gsh Language

Your `.gshrc.gsh` files can use the full gsh scripting language. This includes:

- Variables and assignments
- Arrays and objects
- Functions (called "tools")
- Control flow (if/else, loops)
- Error handling (try/catch)
- String interpolation
- And much more!

For a complete guide to the gsh scripting language, see the [Script Documentation](../script/). Here's a quick roadmap:

- **[Chapter 03: Values and Types](../script/03-values-and-types.md)** - Learn about strings, numbers, booleans, arrays, and objects
- **[Chapter 04: Variables and Assignment](../script/04-variables-and-assignment.md)** - How to store and use data
- **[Chapter 05: Operators and Expressions](../script/05-operators-and-expressions.md)** - Math, comparisons, and logic
- **[Chapter 06: Arrays and Objects](../script/06-arrays-and-objects.md)** - Working with collections
- **[Chapter 07: String Manipulation](../script/07-string-manipulation.md)** - String methods and operations
- **[Chapter 08: Conditionals](../script/08-conditionals.md)** - if/else statements
- **[Chapter 09: Loops](../script/09-loops.md)** - for and while loops
- **[Chapter 10: Error Handling](../script/10-error-handling.md)** - try/catch blocks
- **[Chapter 11: Tool Declarations](../script/11-tool-declarations.md)** - Defining custom functions
- **[Chapter 21: Built-in Functions](../script/21-builtin-functions.md)** - Functions gsh provides

## Common Configuration Patterns

### Pattern 1: Simple Bash Aliases and Functions

**~/.gshrc**

```bash
alias dc='docker compose'
alias k='kubectl'

function kpods() {
    kubectl get pods -n "$1"
}
```

### Pattern 2: Custom gsh Tools

**~/.gshrc.gsh**

```gsh
# Tool to format timestamps
tool format_time(seconds: number): string {
    days = seconds / (24 * 3600)
    hours = (seconds % (24 * 3600)) / 3600
    mins = (seconds % 3600) / 60
    secs = seconds % 60

    result = ""
    if (days > 0) {
        result = result + `${days}d `
    }
    if (hours > 0) {
        result = result + `${hours}h `
    }
    if (mins > 0) {
        result = result + `${mins}m `
    }
    result = result + `${secs}s`

    return result
}
```

### Pattern 3: Environment-Based Configuration

**~/.gshrc.gsh**

```gsh
# Check if we're in a development environment
ENV = env.NODE_ENV || "development"

if (ENV == "development") {
    DEBUG_MODE = true
    print("🔧 Development mode enabled")
} else {
    DEBUG_MODE = false
    print("✅ Production mode")
}
```

## Customizing Agent Rendering

gsh provides customizable hooks that control how agent interactions are displayed. These are defined as tools in `.gshrc.default.gsh` and can be overridden in your `~/.gshrc.gsh`.

### Available Rendering Hooks

| Hook               | Purpose                                  |
| ------------------ | ---------------------------------------- |
| `GSH_AGENT_HEADER` | Header line when agent starts responding |
| `GSH_AGENT_FOOTER` | Footer line with token usage and timing  |
| `GSH_EXEC_START`   | Start line for shell command execution   |
| `GSH_EXEC_END`     | Completion line for shell commands       |
| `GSH_TOOL_STATUS`  | Status display for non-exec tool calls   |
| `GSH_TOOL_OUTPUT`  | Tool output display (empty by default)   |

### Example: Custom Agent Header

Override the header to show a custom format:

```gsh
# ~/.gshrc.gsh

tool GSH_AGENT_HEADER(agentName: string, terminalWidth: number): string {
    # Simple centered header
    text = "🤖 " + agentName + " 🤖"
    return text
}
```

### Example: Minimal Footer

Show only the duration, not token counts:

```gsh
# ~/.gshrc.gsh

tool GSH_AGENT_FOOTER(inputTokens: number, outputTokens: number, durationMs: number, terminalWidth: number): string {
    durationSec = durationMs / 1000
    return "── completed in " + durationSec + "s ──"
}
```

### Example: Custom Exec Display

Change how shell commands are displayed:

```gsh
# ~/.gshrc.gsh

tool GSH_EXEC_START(command: string): string {
    return "$ " + command
}

tool GSH_EXEC_END(commandFirstWord: string, durationMs: number, exitCode: number): string {
    durationSec = durationMs / 1000
    if (exitCode == 0) {
        return "[done] " + commandFirstWord + " (" + durationSec + "s)"
    }
    return "[failed] " + commandFirstWord + " (exit " + exitCode + ")"
}
```

### Example: Show Tool Output

By default, tool output is hidden. Enable it for debugging:

```gsh
# ~/.gshrc.gsh

tool GSH_TOOL_OUTPUT(toolName: string, output: string, terminalWidth: number): string {
    # Show first 200 characters of output
    if (output.length > 200) {
        return "   → " + output.substring(0, 200) + "..."
    }
    return "   → " + output
}
```

For more details on the rendering system, see [Chapter 05: Agents in the REPL](05-agents-in-the-repl.md#understanding-agent-output).

## Debugging Configuration

If something goes wrong with your configuration, you can see what's happening:

### Check if files exist

```bash
gsh> ls -la ~/.gshrc ~/.gshrc.gsh
```

### Enable debug logging

In your `~/.gshrc.gsh`, you only need to set the `logLevel` field—other settings will be preserved from the defaults:

```gsh
GSH_CONFIG = {
    logLevel: "debug",
}
```

Then look at the debug logs:

```bash
gsh> tail -f ~/.gsh.log
```

### Test configuration in isolation

Create a temporary test file to check your configuration:

```bash
# Save your configuration code to a test file
cat > /tmp/test_config.gsh << 'EOF'
# Your configuration here
tool test_tool(): string {
    return "It works!"
}
EOF

# Test it by sourcing in a script (we'll cover this in Chapter 06)
gsh /tmp/test_config.gsh
```

## Configuration Best Practices

1. **Keep `.gshrc` minimal** - Use bash-compatible code only
2. **Use `.gshrc.gsh` for complex logic** - Take advantage of gsh features
3. **Avoid side effects** - Don't execute expensive operations during config loading
4. **Document your tools** - Add comments explaining what your custom tools do
5. **Test incrementally** - Add configuration features one at a time and test after each change

## Troubleshooting

### Configuration not loading

If your configuration doesn't seem to be loading:

1. Check file permissions: `ls -la ~/.gshrc*`
2. Check for syntax errors in `.gshrc.gsh` - gsh will report them at startup
3. Try renaming files temporarily to isolate the problem

### Aliases not working

If bash aliases defined in `.gshrc` aren't available:

- They only work in interactive shells
- They don't work in scripts
- Check that you exported variables if needed

### Custom tools not available

If tools defined in `.gshrc.gsh` aren't working:

- They're only available within the REPL, not in scripts
- Check for syntax errors in your tool definitions
- Try running the tool with explicit arguments

## What's Next?

Your shell is now configured! Chapter 03 covers **Custom Prompts with Starship**—how to create sophisticated, informative prompts that show git status, exit codes, and more.

---

**Previous Chapter:** [Chapter 01: Getting Started](01-getting-started-with-gsh.md)

**Next Chapter:** [Chapter 03: Custom Prompts](03-custom-prompts.md)
