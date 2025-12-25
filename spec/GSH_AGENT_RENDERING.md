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

All rendering is controlled via customizable tools defined in `.gshrc.gsh`. These override the defaults from `.gshrc.default.gsh`. The renderer passes all necessary context (including terminal width) to these tools, giving users full control over formatting.

### GSH_AGENT_HEADER

Customize the agent header line. The tool receives terminal width and is responsible for the entire line including padding.

```gsh
tool GSH_AGENT_HEADER(agentName: string, terminalWidth: number): string {
    width = terminalWidth
    if (width > 80) {
        width = 80
    }
    text = "agent: " + agentName
    padding = width - 4 - len(text)  # "── " prefix (3) + " " before padding (1)
    if (padding < 3) {
        padding = 3
    }
    return "── " + text + " " + repeat("─", padding)
}
```

**Parameters:**

- `agentName`: Name of the responding agent (e.g., "default", "coder")
- `terminalWidth`: Current terminal width in columns

**Returns:** Complete header line to display

### GSH_AGENT_FOOTER

Customize the agent footer line. The tool decides what stats to show and handles all formatting.

```gsh
tool GSH_AGENT_FOOTER(inputTokens: number, outputTokens: number, durationMs: number, terminalWidth: number): string {
    width = terminalWidth
    if (width > 80) {
        width = 80
    }
    text = "" + inputTokens + " in · " + outputTokens + " out · " + durationMs + "ms"
    padding = width - 4 - len(text)
    if (padding < 3) {
        padding = 3
    }
    return "── " + text + " " + repeat("─", padding)
}
```

**Parameters:**

- `inputTokens`: Number of input/prompt tokens used
- `outputTokens`: Number of output/completion tokens used
- `durationMs`: Total duration in milliseconds
- `terminalWidth`: Current terminal width in columns

**Returns:** Complete footer line to display

### GSH_EXEC_START

Customize the exec tool start line.

```gsh
tool GSH_EXEC_START(command: string): string {
    return "▶ " + command
}
```

**Parameters:**

- `command`: The full shell command being executed

**Returns:** The start line to display

### GSH_EXEC_END

Customize the exec tool completion line.

```gsh
tool GSH_EXEC_END(commandFirstWord: string, durationMs: number, exitCode: number): string {
    if (exitCode == 0) {
        return "✓ " + commandFirstWord + " (" + durationMs + "ms)"
    }
    return "✗ " + commandFirstWord + " (" + durationMs + "ms) exit code " + exitCode
}
```

**Parameters:**

- `commandFirstWord`: First word of the command (e.g., "ls" from "ls -la")
- `durationMs`: Execution duration in milliseconds
- `exitCode`: Exit code of the command

**Returns:** The completion line to display

### GSH_TOOL_STATUS

Customize how non-exec tool status is rendered. Receives raw arguments as an object for full formatting control.

```gsh
tool GSH_TOOL_STATUS(toolName: string, status: string, args: object, durationMs: number): string {
    # Format arguments - one per line, indented
    argsStr = ""
    for (key in args) {
        value = args[key]
        # Truncate long values
        valueStr = "" + value
        if (len(valueStr) > 60) {
            valueStr = valueStr[0:57] + "..."
        }
        argsStr = argsStr + "   " + key + ": " + valueStr + "\n"
    }

    if (status == "pending") {
        return "○ " + toolName
    }
    if (status == "executing") {
        return "○ " + toolName + "\n" + argsStr
    }
    if (status == "success") {
        return "● " + toolName + " ✓ (" + durationMs + "ms)\n" + argsStr
    }
    # error
    return "● " + toolName + " ✗ (" + durationMs + "ms)\n" + argsStr
}
```

**Parameters:**

- `toolName`: Name of the tool being called (e.g., "read_file", "search")
- `status`: One of "pending", "executing", "success", or "error"
- `args`: Raw arguments object (e.g., `{path: "/config.json", encoding: "utf-8"}`)
- `durationMs`: Execution duration in milliseconds (0 for pending/executing)

**Returns:** Full status block including arguments

**Note:** The spinner character is handled by the renderer and appended for pending/executing states.

### GSH_TOOL_OUTPUT

Customize how non-exec tool output/results are displayed. Default returns empty (no output shown).

```gsh
tool GSH_TOOL_OUTPUT(toolName: string, output: string, terminalWidth: number): string {
    return ""  # Default: show nothing
}
```

**Parameters:**

- `toolName`: Name of the tool
- `output`: The tool's output/result as a string
- `terminalWidth`: Current terminal width in columns

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

### Hook Invocation

```go
func (r *Renderer) RenderAgentHeader(agentName string) {
    // Get the hook tool
    hook := r.interp.GetTool("GSH_AGENT_HEADER")
    if hook == nil {
        // Fallback to basic rendering
        fmt.Fprintf(r.writer, "── agent: %s ───\n", agentName)
        return
    }

    // Call hook with context
    result, err := r.interp.CallTool(hook, []interpreter.Value{
        &interpreter.StringValue{Value: agentName},
        &interpreter.NumberValue{Value: float64(r.termWidth())},
    })

    if err != nil {
        r.logger.Warn("GSH_AGENT_HEADER hook failed", zap.Error(err))
        fmt.Fprintf(r.writer, "── agent: %s ───\n", agentName)
        return
    }

    if str, ok := result.(*interpreter.StringValue); ok {
        fmt.Fprintln(r.writer, styles.AgentHeader(str.Value))
    }
}
```

### Spinner Implementation

```go
func (r *Renderer) spinnerLoop(ctx context.Context, prefix string) {
    frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
    ticker := time.NewTicker(80 * time.Millisecond)
    defer ticker.Stop()

    i := 0
    for {
        select {
        case <-ctx.Done():
            fmt.Print("\r\033[K")  // Clear line
            return
        case <-ticker.C:
            fmt.Printf("\r%s %s", prefix, frames[i%len(frames)])
            i++
        }
    }
}
```

### Passing Arguments to Hooks

The renderer converts Go `map[string]any` to a gsh object value before passing to hooks:

```go
func (r *Renderer) argsToValue(args map[string]any) *interpreter.ObjectValue {
    obj := &interpreter.ObjectValue{
        Properties: make(map[string]interpreter.Value),
    }
    for key, value := range args {
        obj.Properties[key] = toInterpreterValue(value)
    }
    return obj
}

func toInterpreterValue(v any) interpreter.Value {
    switch val := v.(type) {
    case string:
        return &interpreter.StringValue{Value: val}
    case float64:
        return &interpreter.NumberValue{Value: val}
    case bool:
        return &interpreter.BoolValue{Value: val}
    // ... handle arrays, nested objects, etc.
    default:
        return &interpreter.StringValue{Value: fmt.Sprintf("%v", v)}
    }
}
```

---

## Config Loading Changes

To support tool overrides, the config loader must use a **single interpreter** for both default and user configs:

```go
func (l *Loader) LoadDefaultConfigPath(defaultContent string) (*LoadResult, error) {
    // Create ONE interpreter for both configs
    interp := interpreter.NewWithLogger(l.logger)

    // 1. Load default config first
    if defaultContent != "" {
        l.evalIntoInterpreter(interp, defaultContent)
    }

    // 2. Load user config into SAME interpreter (shadows defaults)
    userContent, _ := os.ReadFile(userConfigPath)
    l.evalIntoInterpreter(interp, string(userContent))

    // 3. Extract final state
    result.Interpreter = interp
    l.extractConfigFromInterpreter(interp, result)

    return result, nil
}
```

This ensures that if a user defines `GSH_AGENT_HEADER` in `.gshrc.gsh`, it shadows the default definition from `.gshrc.default.gsh`.

---

## Reserved Names

The following names are reserved for REPL configuration:

| Name                | Type   | Purpose                             |
| ------------------- | ------ | ----------------------------------- |
| `GSH_CONFIG`        | Object | REPL configuration settings         |
| `GSH_UPDATE_PROMPT` | Tool   | Dynamic prompt generation           |
| `GSH_AGENT_HEADER`  | Tool   | Agent header line rendering         |
| `GSH_AGENT_FOOTER`  | Tool   | Agent footer line rendering         |
| `GSH_EXEC_START`    | Tool   | Exec tool start line rendering      |
| `GSH_EXEC_END`      | Tool   | Exec tool completion line rendering |
| `GSH_TOOL_STATUS`   | Tool   | Non-exec tool status rendering      |
| `GSH_TOOL_OUTPUT`   | Tool   | Non-exec tool output display        |

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

1. **DONE**: Fix config loader to use single interpreter\*\*

   - Modify `internal/repl/config/loader.go` to load `.gshrc.default.gsh` and `.gshrc.gsh` into the same interpreter
   - Ensure user-defined tools shadow default tools
   - Add tests for override behavior

2. **TODO: Add default render tool definitions to `.gshrc.default.gsh`**

   - `GSH_AGENT_HEADER`
   - `GSH_AGENT_FOOTER`
   - `GSH_EXEC_START`
   - `GSH_EXEC_END`
   - `GSH_TOOL_STATUS`
   - `GSH_TOOL_OUTPUT`

3. **TODO: Create `internal/repl/render` package**

   - `renderer.go` - Core `Renderer` struct with hook invocation logic
   - `styles.go` - Lip Gloss style definitions for colors
   - `spinner.go` - Spinner animation with goroutine/ticker

4. **TODO: Update config to extract render hooks**

   - Add fields to `Config` struct for each render tool
   - Extract tools from interpreter in loader

5. **TODO: Integrate renderer with agent execution**

   - Inject `Renderer` into `agent.Manager`
   - Call `RenderAgentHeader` when agent message starts
   - Call `StartThinkingSpinner` while waiting for LLM
   - Call `RenderAgentText` as response streams
   - Call `RenderAgentFooter` when turn completes

6. **TODO: Implement exec tool rendering**

   - Call `RenderExecStart` before command execution
   - Let output stream directly to stdout (no capture)
   - Call `RenderExecEnd` after command completes

7. **TODO: Implement non-exec tool rendering**

   - Call `RenderToolPending` when tool call starts streaming
   - Call `StartToolSpinner` during argument streaming
   - Call `RenderToolExecuting` when args complete
   - Call `RenderToolComplete` when tool finishes
   - Call `RenderToolOutput` if hook returns non-empty

8. **TODO: Add tests**

   - Unit tests for renderer with mocked hooks
   - Integration tests for full agent interaction flow
   - Tests for hook override behavior

9. **TODO: Update documentation**
   - Update `docs/tutorial/` with agent rendering info
   - Add examples to `.gshrc.default.gsh` comments

---

## Future Enhancements

1. **Streaming support**: Render tokens as they arrive from the LLM
2. **Collapsible tool output**: Allow expanding/collapsing verbose tool output
3. **Theme customization**: User-defined color schemes
4. **Custom spinner frames**: Configurable via hook or config
5. **Markdown rendering**: Rich formatting for agent responses
