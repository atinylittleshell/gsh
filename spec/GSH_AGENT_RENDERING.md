# GSH Agent Output Rendering Design

## Overview

This document describes the visual design system for rendering agent-related output in the gsh REPL. The goal is to create clear visual distinction between shell command output and agent interactions, while maintaining clean copy-paste behavior.

## Design Goals

1. **Visual Clarity**: Users should immediately distinguish agent output from shell output
2. **Copy-Paste Friendly**: Agent response content should be easy to copy without visual artifacts
3. **Tool Call Visibility**: Users should see what tools agents are invoking and their status
4. **Customizability**: Power users can customize rendering via `.gshrc.gsh`
5. **Consistency**: Establish a visual language that other UI components can follow

---

## Visual Language

### Output Type Distinction

| Output Type            | Visual Treatment                              |
| ---------------------- | --------------------------------------------- |
| Shell command output   | Default color, no decoration                  |
| Agent response text    | Default color, bounded by header/footer lines |
| Exec tool calls        | Triangle prefix `▶`, output streams directly |
| Non-exec tool calls    | Circle prefix `○`/`●`, multi-line arguments   |
| System/status messages | Gray color with `→` prefix                    |

### Status Symbols

| Symbol       | Meaning                         | Color                 |
| ------------ | ------------------------------- | --------------------- |
| `▶`         | Exec tool (shell command) start | Yellow (11)           |
| `○`          | Non-exec tool pending/executing | Yellow (11)           |
| `●`          | Non-exec tool complete          | Green (10) or Red (9) |
| `✓`          | Success                         | Green (10)            |
| `✗`          | Error                           | Red (9)               |
| `→`          | System message                  | Gray (8)              |
| `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` | Spinner frames                  | Context-dependent     |

### Color Palette

| Element             | ANSI Color  | Usage                             |
| ------------------- | ----------- | --------------------------------- |
| Agent header/footer | Cyan (12)   | Header/footer separator lines     |
| Tool pending        | Yellow (11) | Tool invocation prefix            |
| Success indicator   | Green (10)  | Completion checkmark              |
| Error indicator     | Red (9)     | Error X mark                      |
| Dim/secondary       | Gray (8)    | Timing, meta info                 |
| Default             | -           | Agent response text, shell output |

---

## Agent Interaction Flow

### 1. User Sends Message

```
gsh> # what files are in this directory?
```

### 2. Agent Header + Thinking Spinner

Immediately after user submits, display the agent header and a thinking spinner:

```
gsh> # what files are in this directory?

── agent: default ─────────────────────────────
⠋ Thinking...
```

The spinner animates while waiting for LLM response.

### 3. Response Streams In

The spinner is replaced with the agent's partial response text:

```
gsh> # what files are in this directory?

── agent: default ─────────────────────────────
Let me check that
```

### 4. Interaction Complete

When the agent turn is complete, display a footer with statistics:

```
gsh> # what files are in this directory?

── agent: default ─────────────────────────────
Let me check that for you.

── 523 in · 324 out · 1.2s ────────────────────

gsh>
```

The footer shows input tokens, output tokens, and total duration.

---

## Tool Call Rendering

### Exec Tool (Shell Commands) - Special Handling

The `exec` tool streams output directly to stdout and requires special treatment. Exec output is **not dimmed** - it appears in default color just like regular shell output.

**Start:**

```
▶ ls -la
```

Note: We show just the command, not "exec:".

**During (output streams directly):**

```
▶ ls -la
total 24
-rw-r--r--  1 user  staff  1234 file.txt
```

**Complete (success):**

```
▶ ls -la
total 24
-rw-r--r--  1 user  staff  1234 file.txt
✓ ls (0.1s)
```

Note: The completion line shows the first word of the command (e.g., `ls` not `ls -la`).

**Complete (failure):**

```
▶ cat /nonexistent
cat: /nonexistent: No such file or directory
✗ cat (0.1s) exit code 1
```

### Non-Exec Tools - Multi-line Arguments

All other tools use circle symbols and always display arguments on separate lines for consistency:

**Pending (streaming args from LLM):**

```
○ read_file ⠋
```

**Executing (args complete):**

```
○ read_file ⠹
   path: "/home/user/config.json"
```

**Complete (success):**

```
● read_file ✓ (0.02s)
   path: "/home/user/config.json"
```

**Complete (failure):**

```
● read_file ✗ (0.01s)
   path: "/missing.txt"
```

### Multi-Argument Example

```
○ search ⠹
   query: "error handling"
   directory: "/src"
   max_results: 10
```

Becomes:

```
● search ✓ (0.2s)
   query: "error handling"
   directory: "/src"
   max_results: 10
```

### Long Argument Values

Long string values are truncated with `...`:

```
● write_file ✓ (0.05s)
   path: "/output.go"
   content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt..."
```

---

## Customization Hooks

All rendering is controlled via customizable tools defined in `.gshrc.gsh`. These override the defaults from `.gshrc.default.gsh`. Every hook receives a single `ctx` parameter - a consistent **RenderContext** object containing all available context. This design provides maximum flexibility: hooks can access any information they need without being constrained by a specific parameter list.

### RenderContext Structure

The RenderContext is an object with the following top-level groups:

```gsh
ctx = {
    terminal: {
        width: 120,      # Current terminal width in columns
        height: 40       # Current terminal height in rows
    },
    agent: {
        name: "default"  # Name of the current agent (null if no agent active)
    },
    repl: {
        lastExitCode: 0,     # Exit code of the last command
        lastDurationMs: 150  # Duration of the last command in milliseconds
    },
    query: {
        durationMs: 1234,    # Overall query/turn duration in milliseconds
        inputTokens: 523,    # Number of input/prompt tokens used
        outputTokens: 324    # Number of output/completion tokens used
    },
    exec: {
        command: "ls -la",       # Full shell command being executed
        commandFirstWord: "ls",  # First word of the command
        durationMs: 100,         # Execution duration in milliseconds
        exitCode: 0              # Exit code of the command
    },
    toolCall: {
        name: "read_file",           # Name of the tool being called
        status: "success",           # One of: "pending", "executing", "success", "error"
        args: { path: "/config.json" },  # Tool arguments object
        durationMs: 20,              # Tool execution duration in milliseconds
        output: "file contents..."   # Tool output/result as a string
    }
}
```

**Nullability by Hook:**

| Hook               | `terminal` | `agent`   | `repl` | `query` | `exec`      | `toolCall` |
| ------------------ | ---------- | --------- | ------ | ------- | ----------- | ---------- |
| `GSH_PROMPT`       | ✓          | ✓ or null | ✓      | null    | null        | null       |
| `GSH_AGENT_HEADER` | ✓          | ✓         | null   | null    | null        | null       |
| `GSH_AGENT_FOOTER` | ✓          | ✓         | null   | ✓       | null        | null       |
| `GSH_EXEC_START`   | ✓          | ✓         | null   | null    | ✓ (partial) | null       |
| `GSH_EXEC_END`     | ✓          | ✓         | null   | null    | ✓           | null       |
| `GSH_TOOL_STATUS`  | ✓          | ✓         | null   | null    | null        | ✓          |
| `GSH_TOOL_OUTPUT`  | ✓          | ✓         | null   | null    | null        | ✓          |

Fields are `null` when not applicable to the current hook context.

### GSH_PROMPT

Customize the shell prompt. Called before each command input.

```gsh
tool GSH_PROMPT(ctx: object): string {
    # Access ctx.repl.lastExitCode, ctx.repl.lastDurationMs
    # Access ctx.terminal.width, ctx["agent"].name, etc.

    # Show a different prompt symbol based on last exit code
    symbol = "❯"
    if (ctx.repl.lastExitCode != 0) {
        symbol = "✗"
    }

    return symbol + " "
}
```

**Returns:** The prompt string to display

### GSH_AGENT_HEADER

Customize the agent header line displayed when an agent starts responding.

```gsh
tool GSH_AGENT_HEADER(ctx: object): string {
    width = ctx.terminal.width
    if (width > 80) {
        width = 80
    }
    text = "agent: " + ctx.agent.name
    padding = width - 4 - len(text)  # "── " prefix (3) + " " before padding (1)
    if (padding < 3) {
        padding = 3
    }
    return "── " + text + " " + repeat("─", padding)
}
```

**Returns:** Complete header line to display

### GSH_AGENT_FOOTER

Customize the agent footer line displayed when an agent finishes responding.

```gsh
tool GSH_AGENT_FOOTER(ctx: object): string {
    width = ctx.terminal.width
    if (width > 80) {
        width = 80
    }
    durationSec = ctx.query.durationMs / 1000
    text = "" + ctx.query.inputTokens + " in · " + ctx.query.outputTokens + " out · " + durationSec + "s"
    padding = width - 4 - len(text)
    if (padding < 3) {
        padding = 3
    }
    return "── " + text + " " + repeat("─", padding)
}
```

**Returns:** Complete footer line to display

### GSH_EXEC_START

Customize the exec tool start line.

```gsh
tool GSH_EXEC_START(ctx: object): string {
    return "▶ " + ctx.exec.command
}
```

**Returns:** The start line to display

### GSH_EXEC_END

Customize the exec tool completion line.

```gsh
tool GSH_EXEC_END(ctx: object): string {
    durationSec = ctx.exec.durationMs / 1000
    if (ctx.exec.exitCode == 0) {
        return "✓ " + ctx.exec.commandFirstWord + " (" + durationSec + "s)"
    }
    return "✗ " + ctx.exec.commandFirstWord + " (" + durationSec + "s) exit code " + ctx.exec.exitCode
}
```

**Returns:** The completion line to display

### GSH_TOOL_STATUS

Customize how non-exec tool status is rendered.

```gsh
tool GSH_TOOL_STATUS(ctx: object): string {
    # Format arguments - one per line, indented
    argsStr = ""
    keys = ctx.toolCall.args.keys()
    for (key of keys) {
        value = ctx.toolCall.args[key]
        # Truncate long values
        valueStr = "" + value
        if (len(valueStr) > 60) {
            valueStr = valueStr[0:57] + "..."
        }
        argsStr = argsStr + "   " + key + ": " + valueStr + "\n"
    }

    durationSec = ctx.toolCall.durationMs / 1000

    if (ctx.toolCall.status == "pending") {
        return "○ " + ctx.toolCall.name
    }
    if (ctx.toolCall.status == "executing") {
        return "○ " + ctx.toolCall.name + "\n" + argsStr
    }
    if (ctx.toolCall.status == "success") {
        return "● " + ctx.toolCall.name + " ✓ (" + durationSec + "s)\n" + argsStr
    }
    # error
    return "● " + ctx.toolCall.name + " ✗ (" + durationSec + "s)\n" + argsStr
}
```

**Returns:** Full status block including arguments

**Note:** The spinner character is handled by the renderer and appended for pending/executing states.

### GSH_TOOL_OUTPUT

Customize how non-exec tool output/results are displayed. Default returns empty (no output shown).

```gsh
tool GSH_TOOL_OUTPUT(ctx: object): string {
    return ""  # Default: show nothing
}
```

**Returns:** String to display (empty = show nothing)

---

## Implementation Architecture

### Package Structure

```
internal/repl/render/
├── renderer.go      # Core renderer, public API
├── header.go        # Header/footer rendering via hooks
├── spinner.go       # Spinner animation
├── tool.go          # Tool status rendering (exec and non-exec)
└── styles.go        # Lip Gloss style definitions
```

### Renderer Interface

```go
type Renderer struct {
    interp     *interpreter.Interpreter  // For calling custom hooks
    writer     io.Writer
    termWidth  func() int                // Function to get current terminal width
}

// Header/Footer
func (r *Renderer) RenderAgentHeader(agentName string)
func (r *Renderer) RenderAgentFooter(inputTokens, outputTokens int, duration time.Duration)

// Thinking state
func (r *Renderer) StartThinkingSpinner(ctx context.Context) context.CancelFunc

// Agent text
func (r *Renderer) RenderAgentText(text string)

// Exec tool lifecycle
func (r *Renderer) RenderExecStart(command string)
func (r *Renderer) RenderExecEnd(command string, duration time.Duration, exitCode int)

// Non-exec tool lifecycle
func (r *Renderer) RenderToolPending(toolName string)
func (r *Renderer) StartToolSpinner(toolName string) context.CancelFunc
func (r *Renderer) RenderToolExecuting(toolName string, args map[string]any)
func (r *Renderer) RenderToolComplete(toolName string, args map[string]any, duration time.Duration, success bool)
func (r *Renderer) RenderToolOutput(toolName string, output string)
```

### Rendering Approach

- **Styling**: Lip Gloss for colors and text formatting
- **Animation**: DIY goroutine + ticker for spinners, ANSI escape codes for line updates
- **No Bubble Tea**: Agent output rendering happens outside the input TUI loop
- **Hook-driven**: All formatting decisions delegated to customizable tools

---

## Reserved Names

The following names are reserved for REPL configuration:

| Name               | Type   | Purpose                             |
| ------------------ | ------ | ----------------------------------- |
| `GSH_CONFIG`       | Object | REPL configuration settings         |
| `GSH_PROMPT`       | Tool   | Dynamic prompt generation           |
| `GSH_AGENT_HEADER` | Tool   | Agent header line rendering         |
| `GSH_AGENT_FOOTER` | Tool   | Agent footer line rendering         |
| `GSH_EXEC_START`   | Tool   | Exec tool start line rendering      |
| `GSH_EXEC_END`     | Tool   | Exec tool completion line rendering |
| `GSH_TOOL_STATUS`  | Tool   | Non-exec tool status rendering      |
| `GSH_TOOL_OUTPUT`  | Tool   | Non-exec tool output display        |

---

## Copy-Paste Considerations

The design prioritizes clean copy-paste:

1. **Agent response text**: No prefixes, no borders - just clean text
2. **Exec tool output**: No indentation, no dimming - default color like shell output
3. **Header/footer lines**: Clearly "meta" - users naturally exclude when selecting
4. **Tool status lines**: Prefix-based but typically not content users want to copy
5. **Tool arguments**: Indented with `   ` - meta information, not typically copied

Users can easily select just the content they need without capturing visual artifacts.

---

## Execution Plan

Implementation should proceed in this sequence:

1. **DONE: Fix config loader to use single interpreter**

   - Modify `internal/repl/config/loader.go` to load `.gshrc.default.gsh` and `.gshrc.gsh` into the same interpreter
   - Ensure user-defined tools shadow default tools
   - Add tests for override behavior

2. **DONE: Add default render tool definitions to `.gshrc.default.gsh`**

   - `GSH_AGENT_HEADER`
   - `GSH_AGENT_FOOTER`
   - `GSH_EXEC_START`
   - `GSH_EXEC_END`
   - `GSH_TOOL_STATUS`
   - `GSH_TOOL_OUTPUT`

3. **DONE: Create `internal/repl/render` package**

   - `renderer.go` - Core `Renderer` struct with hook invocation logic
   - `styles.go` - Lip Gloss style definitions for colors
   - `spinner.go` - Spinner animation with goroutine/ticker

4. **DONE: Integrate renderer with agent execution**

   - Inject `Renderer` into `agent.Manager`
   - Call `RenderAgentHeader` when agent message starts
   - Call `StartThinkingSpinner` while waiting for LLM
   - Call `RenderAgentText` as response streams
   - Call `RenderAgentFooter` when turn completes

5. **DONE: Implement exec tool rendering**

   - Call `RenderExecStart` before command execution
   - Let output stream directly to stdout (no capture)
   - Call `RenderExecEnd` after command completes

6. **DONE: Implement non-exec tool rendering**

   - Call `RenderToolPending` when tool call starts streaming
   - Call `StartToolSpinner` during argument streaming
   - Call `RenderToolExecuting` when args complete
   - Call `RenderToolComplete` when tool finishes
   - Call `RenderToolOutput` if hook returns non-empty

7. **TODO: Update documentation**
   - Update `docs/tutorial/` with agent rendering info

---

## Future Enhancements

1. **Streaming support**: Render tokens as they arrive from the LLM
2. **Collapsible tool output**: Allow expanding/collapsing verbose tool output
3. **Theme customization**: User-defined color schemes
4. **Custom spinner frames**: Configurable via hook or config
5. **Markdown rendering**: Rich formatting for agent responses
