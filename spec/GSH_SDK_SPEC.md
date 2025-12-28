# GSH SDK Specification

This document specifies the design for making gsh highly customizable through a neovim-inspired SDK exposed to gsh scripts.

## Design Goals

1. **Neovim-like extensibility** - Configuration IS code, not special syntax
2. **Clear mode separation** - `gsh` (core) vs `gsh.repl` (REPL-only)
3. **Unified event system** - `gsh.on()`
4. **Direct object manipulation** - Modify agents, settings directly
5. **Deprecate GSH_CONFIG** - Replace with proper SDK objects

## Architecture Overview

```
┌───────────────────────────────────────────────────────────────┐
│                      gsh Script Environment                   │
├───────────────────────────────────────────────────────────────┤
│  gsh.*                │  gsh.repl.*                           │
│  (both modes)         │  (REPL mode only, null in script)     │
├───────────────────────┼───────────────────────────────────────┤
│  gsh.version          │  gsh.repl.agents                      │
│  gsh.logging          │  gsh.repl.currentAgent                │
│  gsh.terminal         │  gsh.repl.models                      │
│  gsh.tools            │    .lite / .workhorse / .premium      │
│  gsh.integrations     │  gsh.repl.lastCommand                 │
│  gsh.on()             │    .exitCode / .durationMs            │
│  gsh.off()            │  gsh.repl.history                     │
│  gsh.lastAgentRequest │                                       │
└───────────────────────┴───────────────────────────────────────┘
```

## Core SDK: `gsh` Object

Available in both script mode (`gsh script.gsh`) and REPL mode.

### `gsh.version`

```gsh
# Read-only string
print(gsh.version)  # "1.0.0"
```

### `gsh.logging`

Logging configuration object.

```gsh
gsh.logging.level = "info"      # "debug", "info", "warn", "error" (read/write, changes take effect immediately)
gsh.logging.file                # Log file path (read-only)
```

- `gsh.logging.level` is read/write. Changes take effect immediately via AtomicLevel.
- `gsh.logging.file` is read-only and set at logger initialization.

### `gsh.terminal`

Terminal information, useful for formatting output in both modes.

```gsh
gsh.terminal.width   # int - current terminal width (80 if no TTY)
gsh.terminal.height  # int - current terminal height (24 if no TTY)
gsh.terminal.isTTY   # bool - true if running in interactive terminal
```

All properties are read-only.

### `gsh.tools`

Object containing references to built-in native tools. These are `ToolValue` wrappers around Go implementations.

```gsh
gsh.tools.exec        # Shell command execution
gsh.tools.grep        # File content search
gsh.tools.view_file   # View file contents
gsh.tools.edit_file   # Edit file contents
```

These can be used when configuring agent tools:

```gsh
gsh.repl.agents[0].tools = [gsh.tools.exec, gsh.tools.grep, myCustomTool]
```

See [Native Tool Interop](#native-tool-interop) for implementation details.

### `gsh.on(event, handler)`

Register an event handler. The handler must be a named tool. Returns a handler ID that can be used with `gsh.off()`.

```gsh
tool onToolEnd(ctx) {
    print("Tool " + ctx.toolCall.name + " completed")
}
handlerId = gsh.on("agent.tool.end", onToolEnd)
```

### `gsh.off(event, handlerId)`

Unregister an event handler.

```gsh
gsh.off("agent.tool.end", handlerId)
```

### `gsh.off(event)`

Unregister all handlers for an event.

```gsh
gsh.off("agent.start")  # Remove all agent.start handlers
```

### `gsh.lastAgentRequest`

Reference to the last model request made by any model provider. This is useful for debugging, analytics, and token usage tracking.

```gsh
gsh.lastAgentRequest.model        # The model that was used
gsh.lastAgentRequest.agent        # The agent that initiated the request (or null)
gsh.lastAgentRequest.iteration    # Iteration number within the agent conversation
gsh.lastAgentRequest.durationMs   # Request duration in milliseconds
gsh.lastAgentRequest.usage        # Token usage object
gsh.lastAgentRequest.usage.inputTokens
gsh.lastAgentRequest.usage.outputTokens
gsh.lastAgentRequest.usage.cachedTokens
gsh.lastAgentRequest.stopReason   # "end_turn", "tool_use", "max_tokens", etc.
gsh.lastAgentRequest.error        # Error string if failed, null otherwise
```

This object is updated after every model request completes, in both script and REPL mode.

**Example: Token usage tracking**

```gsh
tool logTokenUsage() {
    req = gsh.lastAgentRequest
    print("Tokens: " + req.usage.inputTokens + " in, " + req.usage.outputTokens + " out")
    if (req.usage.cachedTokens > 0) {
        ratio = (req.usage.cachedTokens / req.usage.inputTokens) * 100
        print("Cache hit: " + ratio.toFixed(0) + "%")
    }
}
gsh.on("agent.iteration.end", logTokenUsage)
```

**Example: Debug failed requests**

```gsh
tool logModelErrors() {
    if (gsh.lastAgentRequest.error != null) {
        print("Model error: " + gsh.lastAgentRequest.error)
        print("Model: " + gsh.lastAgentRequest.model.name)
    }
}
gsh.on("agent.iteration.end", logModelErrors)
```

## REPL SDK: `gsh.repl` Object

Only available in REPL mode. In script mode (`gsh script.gsh`), `gsh.repl` is `null`.

```gsh
# In script mode:
if (gsh.repl != null) {
    # REPL-specific code
}
```

### `gsh.repl.agents`

Array of agent configurations. The first element (`agents[0]`) is always the default/built-in agent.

```gsh
# Agent object structure
{
    name: string,           # Agent name (e.g., "default", "reviewer")
    model: ModelValue,      # Reference to a model declaration
    systemPrompt: string,   # System prompt for the agent
    tools: ToolValue[],     # Array of tools available to the agent
}
```

#### Modifying the default agent

```gsh
# Change the model
gsh.repl.agents[0].model = myLocalModel

# Change the system prompt
gsh.repl.agents[0].systemPrompt = "You are a helpful shell assistant."

# Add a custom tool
gsh.repl.agents[0].tools.push(myCustomTool)

# Replace all tools
gsh.repl.agents[0].tools = [gsh.tools.exec, gsh.tools.grep, myTool]
```

#### Adding custom agents

```gsh
gsh.repl.agents.push({
    name: "reviewer",
    model: gpt4Model,
    systemPrompt: "You are a code reviewer. Be concise.",
    tools: [gsh.tools.exec, gsh.tools.grep, gsh.tools.view_file],
})
```

#### Agent name constraints

- `agents[0].name` is always `"default"` and cannot be changed
- Custom agent names must be unique and non-empty
- Names are used with `/agent <name>` command in REPL

### `gsh.repl.currentAgent`

Reference to the currently active agent from the `agents` array. Read/write.

```gsh
# Get current agent name
print(gsh.repl.currentAgent.name)  # "default"

# Switch to another agent
gsh.repl.currentAgent = gsh.repl.agents[1]

# Or find by name
for (agent of gsh.repl.agents) {
    if (agent.name == "reviewer") {
        gsh.repl.currentAgent = agent
    }
}
```

### `gsh.integrations`

Configuration for external tool integrations. Available in both modes.

#### `gsh.integrations.starship`

Boolean. When `true`, gsh attempts to use [Starship](https://starship.rs/) for the prompt in REPL mode.

```gsh
gsh.integrations.starship = true  # Default
```

If Starship is not available or fails, falls back to the `repl.prompt` event handler.

### `gsh.repl.models`

Pre-configured model tiers for different use cases. This provides a standardized way to reference models by purpose rather than specific configuration.

```gsh
gsh.repl.models.lite       # Fast, lightweight model for predictions and quick tasks
gsh.repl.models.workhorse  # Capable model for general agent work (default agent uses this)
gsh.repl.models.premium    # Most capable model, reserved for high-value tasks
```

Example configuration:

```gsh
model gemma {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

model devstral {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

model gpt4 {
    provider: "openai",
    model: "gpt-4o",
}

gsh.repl.models.lite = gemma
gsh.repl.models.workhorse = devstral
gsh.repl.models.premium = gpt4
```

**Usage:**

- **Prediction**: Always uses `gsh.repl.models.lite`
- **Default agent**: Uses `gsh.repl.models.workhorse` by default (can be overridden via `gsh.repl.agents[0].model`)
- **Premium**: Reserved for future high-value features (e.g., complex multi-step reasoning)

### `gsh.repl.lastCommand`

Information about the last executed shell command. Read-only object.

```gsh
gsh.repl.lastCommand.exitCode    # Exit code of the last command
gsh.repl.lastCommand.durationMs  # Duration in milliseconds

if (gsh.repl.lastCommand.exitCode != 0) {
    print("Last command failed with exit code: " + gsh.repl.lastCommand.exitCode)
}
print("Last command took " + gsh.repl.lastCommand.durationMs + "ms")
```

### `gsh.repl.history` (Future)

History access API.

```gsh
gsh.repl.history.recent(10)      # Get last 10 commands
gsh.repl.history.search("git")   # Search history
gsh.repl.history.add("command")  # Add to history
```

### `gsh.repl.completion` (Future)

Custom completion registration.

```gsh
tool completeMyApp(ctx) {
    # Return completion suggestions
    return ["subcommand1", "subcommand2"]
}
gsh.repl.completion.register({
    command: "myapp",
    complete: completeMyApp
})
```

### `gsh.repl.keymap` (Future)

Custom key bindings.

```gsh
tool customReverseSearch(ctx) {
    # Custom reverse search behavior
}
gsh.repl.keymap.set("ctrl+r", customReverseSearch)
```

## Event System

### Event Handler Behavior

1. **Registration order** - Handlers are called in registration order
2. **No handler = no output** - If no handler is registered for an event that expects a return value, nothing is rendered
3. **Default handlers** - `.gshrc.default.gsh` registers default handlers, users can override or remove them
4. **Return values** - Some events expect string returns for rendering; others are purely for side effects

### Data Access Pattern

Event handlers use a **hybrid approach** for accessing data:

- **Static/global state** → Access via SDK objects (`gsh.terminal.width`, `gsh.repl.currentAgent`, `gsh.version`)
- **Event-specific transient data** → Passed as ctx parameter (`ctx.message`, `ctx.toolCall`, `ctx.exec`)

This keeps ctx objects minimal and focused on data unique to each event invocation.

### Event Naming Convention

Events follow a hierarchical naming pattern based on the lifecycle they belong to:

- `repl.*` - REPL-specific events
- `agent.*` - Agent execution events (both modes, since scripts can run agents)

### REPL Events

These events fire during REPL operation.

#### `repl.ready`

Called when REPL is fully initialized and ready for input.

```gsh
tool onReplReady() {
    print("Welcome to gsh " + gsh.version + "!")
}
gsh.on("repl.ready", onReplReady)
```

#### `repl.prompt`

Called to render the shell prompt. Handler should return the prompt string.

```gsh
tool onReplPrompt() {
    if (gsh.repl.lastCommand.exitCode != 0) {
        return "✗ gsh> "
    }
    return "gsh> "
}
gsh.on("repl.prompt", onReplPrompt)
```

#### `repl.command.before`

Called before a shell command executes.

```gsh
tool onCommandBefore(ctx) {
    # ctx.command: string (the command to execute)
}
gsh.on("repl.command.before", onCommandBefore)
```

#### `repl.command.after`

Called after a shell command completes.

```gsh
tool onCommandAfter(ctx) {
    # ctx.command: string
    # ctx.exitCode: number
    # ctx.durationMs: number
}
gsh.on("repl.command.after", onCommandAfter)
```

#### `repl.exit`

Called when REPL is shutting down.

```gsh
tool onReplExit() {
    print("Goodbye!")
}
gsh.on("repl.exit", onReplExit)
```

### Agent Events

These events fire during agent execution. They work in both REPL and script mode (since scripts can run agents too).

#### `agent.start`

Called when an agent conversation begins. Handler can return a string to render as a header.

```gsh
tool onAgentStart(ctx) {
    # ctx.message: string (user's message)
    # Use gsh.repl.currentAgent for agent info, gsh.terminal for dimensions
    name = gsh.repl.currentAgent.name
    if (name == "default") { name = "gsh" }
    width = gsh.terminal.width
    if (width > 80) { width = 80 }
    return "── " + name + " " + "─".repeat(width - 4 - name.length)
}
gsh.on("agent.start", onAgentStart)
```

#### `agent.iteration.start`

Called at the start of each LLM request/response cycle.

```gsh
tool onIterationStart(ctx) {
    # ctx.iteration: number (1-based)
}
gsh.on("agent.iteration.start", onIterationStart)
```

#### `agent.chunk`

Called for each streaming text chunk received from the LLM.

```gsh
tool onChunk(ctx) {
    # ctx.content: string (the text chunk)
}
gsh.on("agent.chunk", onChunk)
```

#### `agent.tool.start`

Called when a non-exec tool call begins. Handler can return a string to render.

```gsh
tool onToolStart(ctx) {
    # ctx.toolCall: { id, name, args }
    return "○ " + ctx.toolCall.name
}
gsh.on("agent.tool.start", onToolStart)
```

#### `agent.tool.end`

Called when a non-exec tool call completes. Handler can return a string to render.

```gsh
tool onToolEnd(ctx) {
    # ctx.toolCall: { id, name, args, durationMs, output, error }
    duration = (ctx.toolCall.durationMs / 1000).toFixed(2)
    if (ctx.toolCall.error != null) {
        return "● " + ctx.toolCall.name + " ✗ (" + duration + "s)"
    }
    return "● " + ctx.toolCall.name + " ✓ (" + duration + "s)"
}
gsh.on("agent.tool.end", onToolEnd)
```

#### `agent.exec.start`

Called when an exec (shell command) tool starts. Handler can return a string to render.

```gsh
tool onExecStart(ctx) {
    # ctx.exec: { command, commandFirstWord }
    return "▶ " + ctx.exec.command
}
gsh.on("agent.exec.start", onExecStart)
```

#### `agent.exec.end`

Called when an exec tool completes. Handler can return a string to render.

```gsh
tool onExecEnd(ctx) {
    # ctx.exec: { command, commandFirstWord, durationMs, exitCode }
    duration = (ctx.exec.durationMs / 1000).toFixed(1)
    if (ctx.exec.exitCode == 0) {
        return "● " + ctx.exec.commandFirstWord + " ✓ (" + duration + "s)"
    }
    return "● " + ctx.exec.commandFirstWord + " ✗ (" + duration + "s) exit " + ctx.exec.exitCode
}
gsh.on("agent.exec.end", onExecEnd)
```

#### `agent.iteration.end`

Called at the end of each LLM request/response cycle.

```gsh
tool onIterationEnd(ctx) {
    # ctx.iteration: number
    # ctx.usage: { inputTokens, outputTokens, cachedTokens }
}
gsh.on("agent.iteration.end", onIterationEnd)
```

#### `agent.end`

Called when the agent conversation completes. Handler can return a string to render as a footer.

```gsh
tool onAgentEnd(ctx) {
    # ctx.result: { stopReason, durationMs, totalInputTokens, totalOutputTokens, error }
    return "── done in " + ctx.result.durationMs + "ms ──"
}
gsh.on("agent.end", onAgentEnd)
```

## Native Tool Interop

Built-in tools (exec, grep, view_file, edit_file) are implemented in Go but need to be exposed as `ToolValue` objects in the script environment.

### Implementation Approach

1. **NativeToolValue** - New value type that wraps a Go function
2. **Registration** - Built-in tools registered at interpreter initialization
3. **Invocation** - When called from gsh script, delegates to Go implementation
4. **Agent integration** - Agents treat native tools the same as script-defined tools

```go
// In interpreter/value.go
type NativeToolValue struct {
    Name        string
    Description string
    Parameters  []ToolParameter
    Invoke      func(args map[string]interface{}) (interface{}, error)
}

func (n *NativeToolValue) Type() string { return "tool" }
```

### Exposing via `gsh.tools`

```go
// In interpreter/builtin.go
func (i *Interpreter) registerGshTools() {
    gshTools := &ObjectValue{
        Properties: map[string]Value{
            "exec":      i.createNativeTool("exec", execDescription, execParams, execInvoke),
            "grep":      i.createNativeTool("grep", grepDescription, grepParams, grepInvoke),
            "view_file": i.createNativeTool("view_file", viewFileDescription, viewFileParams, viewFileInvoke),
            "edit_file": i.createNativeTool("edit_file", editFileDescription, editFileParams, editFileInvoke),
        },
    }
    // Add to gsh object
}
```

## Migration Reference

| Old                              | New                                                                                         |
| -------------------------------- | ------------------------------------------------------------------------------------------- |
| `GSH_CONFIG.prompt`              | `gsh.on("repl.prompt", myPrompt)`                                                           |
| `GSH_CONFIG.starshipIntegration` | `gsh.integrations.starship`                                                                 |
| `GSH_CONFIG.showWelcome`         | `gsh.on("repl.ready", ...)` to show custom welcome                                          |
| `GSH_CONFIG.logLevel`            | `gsh.logging.level`                                                                         |
| `GSH_CONFIG.predictModel`        | `gsh.repl.models.lite`                                                                      |
| `GSH_CONFIG.defaultAgentModel`   | `gsh.repl.models.workhorse` or `gsh.repl.agents[0].model`                                   |
| `tool GSH_PROMPT(ctx)`           | `tool myPrompt() {...}` then `gsh.on("repl.prompt", myPrompt)`                              |
| `tool GSH_AGENT_HEADER(ctx)`     | `tool myHeader(ctx) {...}` then `gsh.on("agent.start", myHeader)` (ctx has message)         |
| `tool GSH_AGENT_FOOTER(ctx)`     | `tool myFooter(ctx) {...}` then `gsh.on("agent.end", myFooter)` (ctx has result)            |
| `tool GSH_EXEC_START(ctx)`       | `tool myExecStart(ctx) {...}` then `gsh.on("agent.exec.start", myExecStart)` (ctx has exec) |
| `tool GSH_EXEC_END(ctx)`         | `tool myExecEnd(ctx) {...}` then `gsh.on("agent.exec.end", myExecEnd)` (ctx has exec)       |
| `tool GSH_TOOL_STATUS(ctx)`      | `tool myTool(ctx) {...}` then `gsh.on("agent.tool.start/end", myTool)` (ctx has toolCall)   |
| `tool GSH_TOOL_OUTPUT(ctx)`      | Handle in `agent.tool.end` handler                                                          |

## Implementation Phases

Since gsh is not yet released, we can implement everything without backward compatibility concerns. The phases below are organized by logical dependency, not migration order.

### Phase 1 (DONE): Core `gsh` Object & Event System

**Files to modify:**

- `internal/script/interpreter/builtin.go` - Add `gsh` object registration
- `internal/script/interpreter/value.go` - Add event handler storage
- `internal/script/interpreter/interpreter.go` - Initialize `gsh` object

**Deliverables:**

- `gsh.version` (read-only string)
- `gsh.logging.level` (read/write, changes take effect immediately via AtomicLevel)
- `gsh.logging.file` (read-only)
- `gsh.terminal.width`, `gsh.terminal.height`, `gsh.terminal.isTTY`
- `gsh.integrations.starship`
- `gsh.on(event, handler)`, `gsh.off(event, handlerId)`
- `gsh.lastAgentRequest` (updated after each agent iteration)
- Event emission infrastructure
- Distinguish read-only vs read/write properties (writing to a read-only property raises error)
- Updating properties should take effect immediately (logging for example via AtomicLevel)

### Phase 2: `gsh.repl` Object & REPL Events

**Files to modify:**

- `internal/script/interpreter/builtin.go` - Add `gsh.repl` object
- `internal/repl/repl.go` - Initialize `gsh.repl` when in REPL mode
- `internal/repl/render/renderer.go` - Use event system for rendering

**Deliverables:**

- `gsh.repl` object (null in script mode)
- `gsh.repl.models.lite`, `gsh.repl.models.workhorse`, `gsh.repl.models.premium`
- `gsh.repl.lastCommand.exitCode`, `gsh.repl.lastCommand.durationMs`
- REPL events: `repl.ready`, `repl.prompt`, `repl.command.before`, `repl.command.after`, `repl.exit`

### Phase 3: Native Tool Interop

**Files to modify:**

- `internal/script/interpreter/value.go` - Add `NativeToolValue` type
- `internal/script/interpreter/builtin.go` - Register native tools
- `internal/repl/agent/tools.go` - Refactor to support both native and script tools

**Deliverables:**

- `NativeToolValue` type
- `gsh.tools.exec`, `gsh.tools.grep`, `gsh.tools.view_file`, `gsh.tools.edit_file`
- Native tools callable from gsh scripts
- Native tools usable in agent tool arrays

### Phase 4: `gsh.repl.agents` Array

**Files to modify:**

- `internal/repl/agent/agent.go` - Read agent config from `gsh.repl.agents`
- `internal/repl/repl.go` - Initialize `gsh.repl.agents[0]` with defaults
- `internal/script/interpreter/agent.go` - Support agents from the array

**Deliverables:**

- `gsh.repl.agents` array
- `gsh.repl.agents[0]` initialized with default config and `gsh.repl.models.workhorse`
- `gsh.repl.agents[0].model`, `.systemPrompt`, `.tools` modifiable
- Custom agents addable via `gsh.repl.agents.push()`
- `gsh.repl.currentAgent` read/write

### Phase 5: Agent Lifecycle Events

**Files to modify:**

- `internal/script/interpreter/conversation.go` - Emit events during agent execution
- `internal/repl/agent/agent.go` - Wire up event emission

**Deliverables:**

- `agent.start`, `agent.end`
- `agent.iteration.start`, `agent.iteration.end`
- `agent.chunk`
- `agent.tool.start`, `agent.tool.end`
- `agent.exec.start`, `agent.exec.end`

### Phase 6: Remove Legacy Code & Update Defaults

**Files to modify:**

- `cmd/gsh/.gshrc.default.gsh` - Rewrite using new SDK and events
- `internal/repl/config/` - Remove `GSH_CONFIG` support
- `internal/repl/render/renderer.go` - Remove old `GSH_*` hook support

**Deliverables:**

- Remove `GSH_CONFIG` entirely
- Remove `tool GSH_*` hook pattern
- New `.gshrc.default.gsh` using SDK
- Update all documentation in `docs/`

## Future Considerations

### Plugin System

Once the SDK is stable, consider a plugin system (with lock file):

```gsh
# Load a plugin from a URL or local path
gsh.plugin.load("https://github.com/user/gsh-plugin-git")
gsh.plugin.load("~/.gsh/plugins/my-plugin.gsh")
```

### Theming

Centralized color/style configuration:

```gsh
gsh.theme = {
    colors: {
        primary: "yellow",
        error: "red",
        dim: "gray",
    },
}
```
