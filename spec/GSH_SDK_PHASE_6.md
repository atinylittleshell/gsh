# Phase 6 Implementation Plan: Remove Legacy Code & Update Defaults

## Overview

Phase 6 is the final cleanup phase that removes the legacy `GSH_CONFIG` object and `tool GSH_*` hook pattern, replacing them entirely with the new SDK-based event system (`gsh.on()`, `gsh.off()`, and `gsh.*` properties).

## Key Steps

### 1. (DONE) `cmd/gsh/.gshrc.default.gsh` - Complete Rewrite

**Current State:** Uses `GSH_CONFIG` object and `tool GSH_*` hooks pattern.

**Changes Required:**

- Remove `GSH_CONFIG = { ... }` object entirely
- Replace with SDK-based configuration:
  - `gsh.integrations.starship = true`
  - `gsh.logging.level = "info"`
- Replace `tool GSH_AGENT_HEADER(ctx)` → `tool onAgentStart(ctx)` + `gsh.on("agent.start", onAgentStart)`
- Replace `tool GSH_AGENT_FOOTER(ctx)` → `tool onAgentEnd(ctx)` + `gsh.on("agent.end", onAgentEnd)`
- Replace `tool GSH_EXEC_START(ctx)` → `tool onExecStart(ctx)` + `gsh.on("agent.exec.start", onExecStart)`
- Replace `tool GSH_EXEC_END(ctx)` → `tool onExecEnd(ctx)` + `gsh.on("agent.exec.end", onExecEnd)`
- Replace `tool GSH_TOOL_STATUS(ctx)` → Split into `onToolStart` + `onToolEnd` handlers registered via `gsh.on("agent.tool.start/end", ...)`
- Replace `tool GSH_TOOL_OUTPUT(ctx)` → Handle in `agent.tool.end` handler
- Set up model tiers: `gsh.repl.models.lite`, `gsh.repl.models.workhorse`, `gsh.repl.models.premium`
- Register welcome message via `gsh.on("repl.ready", ...)`
- Register prompt via `gsh.on("repl.prompt", ...)` (replaces `tool GSH_PROMPT`)

**New Structure:**

```gsh
# Model declarations (unchanged)
model GSH_PREDICT_MODEL { ... }
model GSH_AGENT_MODEL { ... }

# SDK configuration (replaces GSH_CONFIG)
gsh.integrations.starship = true
gsh.logging.level = "info"

# Model tier configuration
gsh.repl.models.lite = GSH_PREDICT_MODEL
gsh.repl.models.workhorse = GSH_AGENT_MODEL

# Event handlers (replaces GSH_* hooks)
tool onAgentStart(ctx) { ... }
gsh.on("agent.start", onAgentStart)

tool onAgentEnd(ctx) { ... }
gsh.on("agent.end", onAgentEnd)

# ... etc for other events
```

---

### 2. `internal/repl/config/config.go` - Remove GSH_CONFIG Fields

**Current State:** Has many fields extracted from `GSH_CONFIG`.

**Changes Required:**

- Remove these fields entirely (they're now accessed via SDK):
  - `Prompt string` → Access via `gsh.on("repl.prompt", ...)` event
  - `LogLevel string` → Access via `gsh.logging.level`
  - `StarshipIntegration *bool` → Access via `gsh.integrations.starship`
  - `ShowWelcome *bool` → Access via `gsh.on("repl.ready", ...)` event
  - `PredictModel string` → Access via `gsh.repl.models.lite`
  - `DefaultAgentModel string` → Access via `gsh.repl.models.workhorse`
- Remove associated helper methods:
  - `GetUpdatePromptTool()`
  - `GetPredictModel()`
  - `GetDefaultAgentModel()`
  - `StarshipIntegrationEnabled()`
  - `ShowWelcomeEnabled()`
- **Keep:** `MCPServers`, `Models`, `Agents`, `Tools` maps (these are still needed for declarations)

**New Config struct:**

```go
type Config struct {
    // Declarations from .gshrc.gsh (still needed)
    MCPServers map[string]*mcp.MCPServer
    Models     map[string]*interpreter.ModelValue
    Agents     map[string]*interpreter.AgentValue
    Tools      map[string]*interpreter.ToolValue
}
```

---

### 3. `internal/repl/config/loader.go` - Remove GSH_CONFIG Extraction

**Current State:** Has `extractGSHConfig()` function that parses `GSH_CONFIG` object.

**Changes Required:**

- Remove `extractGSHConfig()` function entirely
- Remove references to `GSH_CONFIG` variable extraction
- Remove merging logic for `GSH_CONFIG` objects
- Keep `extractConfigFromInterpreter()` but only extract declarations (models, agents, tools, MCP servers)
- Remove starship content loading logic (starship integration now handled via SDK)

---

### 4. `internal/repl/render/renderer.go` - Replace Hook Calls with Event System

**Current State:** Calls hooks like `GSH_AGENT_HEADER`, `GSH_AGENT_FOOTER`, `GSH_EXEC_START`, etc.

**Changes Required:**

- Replace `callHookWithContext("GSH_AGENT_HEADER", ctx)` → Emit `agent.start` event and use return value
- Replace `callHookWithContext("GSH_AGENT_FOOTER", ctx)` → Emit `agent.end` event
- Replace `callHookWithContext("GSH_EXEC_START", ctx)` → Emit `agent.exec.start` event
- Replace `callHookWithContext("GSH_EXEC_END", ctx)` → Emit `agent.exec.end` event
- Replace `callHookWithContext("GSH_TOOL_STATUS", ctx)` → Emit `agent.tool.start` or `agent.tool.end` events
- Replace `callHookWithContext("GSH_TOOL_OUTPUT", ctx)` → Handle in `agent.tool.end`
- Replace `callHookWithContext("GSH_PROMPT", ctx)` → Emit `repl.prompt` event

**Key Implementation Detail:**
The event system needs to support return values from handlers for rendering. The `EmitEvent` function should:

1. Call all registered handlers in order
2. Return the string result from the **last** handler that returns a non-empty string
3. Fall back to built-in defaults if no handlers are registered or all return empty

**New Method Needed:**

```go
// EmitEventWithReturn emits an event and returns the last non-empty string result
func (r *Renderer) EmitEventWithReturn(eventName string, ctx Value) string {
    handlers := r.interp.GetEventHandlers(eventName)
    var result string
    for _, handler := range handlers {
        val, err := r.interp.CallTool(handler, []Value{ctx})
        if err == nil {
            if strVal, ok := val.(*StringValue); ok && strVal.Value != "" {
                result = strVal.Value
            }
        }
    }
    return result
}
```

---

### 5. `internal/repl/repl.go` - Update Initialization and Event Usage

**Current State:** Uses `r.config.ShowWelcomeEnabled()`, `r.config.GetUpdatePromptTool()`, etc.

**Changes Required:**

- Remove `showWelcomeScreen()` call based on config; instead the welcome is shown via `repl.ready` event handler in `.gshrc.default.gsh`
- Update `getPrompt()` to emit `repl.prompt` event instead of calling `GSH_PROMPT` tool
- Update model initialization to use `gsh.repl.models.workhorse` and `gsh.repl.models.lite` via SDK
- Remove references to `r.config.StarshipIntegrationEnabled()` (now via SDK)
- Update predictor initialization to use `gsh.repl.models.lite` via SDK

**Prompt Logic Change:**

```go
func (r *REPL) getPrompt() string {
    // Emit repl.prompt event and get result
    interp := r.executor.Interpreter()
    ctx := createPromptContext(r.lastExitCode, r.lastDurationMs)

    handlers := interp.GetEventHandlers("repl.prompt")
    for _, handler := range handlers {
        result, err := interp.CallTool(handler, []interpreter.Value{ctx})
        if err == nil {
            if strVal, ok := result.(*interpreter.StringValue); ok && strVal.Value != "" {
                return strVal.Value
            }
        }
    }

    // Fallback to static prompt
    return "gsh> "
}
```

---

### 6. `internal/repl/config/config_test.go` - Update Tests

**Changes Required:**

- Remove tests for `GetUpdatePromptTool()`
- Remove tests for `StarshipIntegrationEnabled()`
- Remove tests for `ShowWelcomeEnabled()`
- Remove tests for `GetPredictModel()`
- Remove tests for `GetDefaultAgentModel()`
- Keep tests for declaration extraction (models, agents, tools)

---

### 7. `internal/repl/config/loader_test.go` - Update Tests

**Changes Required:**

- Remove tests related to `GSH_CONFIG` parsing
- Remove tests for starship integration flag
- Update tests to verify SDK-based configuration works
- Add tests for event handler registration from default config

---

### 8. `internal/repl/render/renderer_test.go` - Update Tests

**Changes Required:**

- Update tests from `tool GSH_AGENT_HEADER` pattern to event-based pattern
- Update assertions to match new event system behavior

---

### 9. Documentation Updates (`docs/` folder)

**Files to update:**

- `docs/tutorial/02-configuration.md` - Remove all `GSH_CONFIG` references, document new SDK approach
- `docs/tutorial/03-custom-prompts.md` - Replace `tool GSH_PROMPT` with `gsh.on("repl.prompt", ...)`
- `docs/tutorial/05-agents-in-the-repl.md` - Update rendering hook examples

**Key Changes:**

- All `GSH_CONFIG = { ... }` examples → SDK property assignments
- All `tool GSH_*` hook examples → `gsh.on()` event handler registrations
- Migration guide from old to new approach (brief section for existing users)

---

## Implementation Order

1. **Start with `cmd/gsh/.gshrc.default.gsh`** - Rewrite to use new SDK patterns. This establishes the new default behavior.

2. **Update `internal/repl/render/renderer.go`** - Add `EmitEventWithReturn` method and update all render methods to use event system instead of direct hook calls.

3. **Update `internal/repl/repl.go`** - Change prompt generation and welcome screen to use events.

4. **Clean up `internal/repl/config/config.go`** - Remove `GSH_CONFIG` fields.

5. **Clean up `internal/repl/config/loader.go`** - Remove `extractGSHConfig()` and related logic.

6. **Update tests** - Fix all broken tests.

7. **Update documentation** - Rewrite docs to reflect new SDK approach.

---

## Backward Compatibility Considerations

Since the spec says "gsh is not yet released, we can implement everything without backward compatibility concerns," we can do a clean break:

1. **No deprecation warnings** - Just remove the old code
2. **No migration helpers** - Users need to update their config files
3. **Clean API** - The new SDK is the only way to configure gsh

---

## Testing Strategy

1. **Unit tests** for event emission and return value handling
2. **Integration tests** that load the new `.gshrc.default.gsh` and verify:
   - Agent headers/footers render correctly
   - Exec start/end events fire and render
   - Tool events fire and render
   - Prompt events work correctly
3. **Manual testing** of the full REPL experience

---

## Risks and Mitigations

| Risk                                            | Mitigation                                                |
| ----------------------------------------------- | --------------------------------------------------------- |
| Event handlers not called in correct order      | Ensure `eventManager.On()` maintains registration order   |
| Return values from events not properly captured | Add dedicated `EmitEventWithReturn` method                |
| Performance regression from event overhead      | Event handlers are lightweight tool calls, minimal impact |
| Breaking existing user configs                  | Document migration clearly in release notes               |

---

## Migration Reference

This table shows the mapping from old patterns to new SDK patterns:

| Old Pattern                      | New Pattern                                                                  |
| -------------------------------- | ---------------------------------------------------------------------------- |
| `GSH_CONFIG.prompt`              | `gsh.on("repl.prompt", myPrompt)`                                            |
| `GSH_CONFIG.starshipIntegration` | `gsh.integrations.starship`                                                  |
| `GSH_CONFIG.showWelcome`         | `gsh.on("repl.ready", ...)` to show custom welcome                           |
| `GSH_CONFIG.logLevel`            | `gsh.logging.level`                                                          |
| `GSH_CONFIG.predictModel`        | `gsh.repl.models.lite`                                                       |
| `GSH_CONFIG.defaultAgentModel`   | `gsh.repl.models.workhorse` or `gsh.repl.agents[0].model`                    |
| `tool GSH_PROMPT(ctx)`           | `tool myPrompt() {...}` then `gsh.on("repl.prompt", myPrompt)`               |
| `tool GSH_AGENT_HEADER(ctx)`     | `tool myHeader(ctx) {...}` then `gsh.on("agent.start", myHeader)`            |
| `tool GSH_AGENT_FOOTER(ctx)`     | `tool myFooter(ctx) {...}` then `gsh.on("agent.end", myFooter)`              |
| `tool GSH_EXEC_START(ctx)`       | `tool myExecStart(ctx) {...}` then `gsh.on("agent.exec.start", myExecStart)` |
| `tool GSH_EXEC_END(ctx)`         | `tool myExecEnd(ctx) {...}` then `gsh.on("agent.exec.end", myExecEnd)`       |
| `tool GSH_TOOL_STATUS(ctx)`      | `tool myTool(ctx) {...}` then `gsh.on("agent.tool.start/end", myTool)`       |
| `tool GSH_TOOL_OUTPUT(ctx)`      | Handle in `agent.tool.end` handler                                           |

---

## Event Context Objects

Each event receives a context object with specific properties:

### `repl.prompt`

```gsh
# ctx: { repl: { lastExitCode, lastDurationMs }, terminal: { width, height } }
tool onPrompt(ctx) {
    if (ctx.repl.lastExitCode != 0) {
        return "✗ gsh> "
    }
    return "gsh> "
}
gsh.on("repl.prompt", onPrompt)
```

### `agent.start`

```gsh
# ctx: { message: string }
# Access agent info via: gsh.repl.currentAgent
# Access terminal info via: gsh.terminal
tool onAgentStart(ctx) {
    name = gsh.repl.currentAgent.name
    if (name == "default") { name = "gsh" }
    width = gsh.terminal.width
    if (width > 80) { width = 80 }
    return "── " + name + " " + "─".repeat(width - 4 - name.length)
}
gsh.on("agent.start", onAgentStart)
```

### `agent.end`

```gsh
# ctx: { result: { stopReason, durationMs, totalInputTokens, totalOutputTokens, error } }
tool onAgentEnd(ctx) {
    return "── done in " + ctx.result.durationMs + "ms ──"
}
gsh.on("agent.end", onAgentEnd)
```

### `agent.exec.start`

```gsh
# ctx: { exec: { command, commandFirstWord } }
tool onExecStart(ctx) {
    return "▶ " + ctx.exec.command
}
gsh.on("agent.exec.start", onExecStart)
```

### `agent.exec.end`

```gsh
# ctx: { exec: { command, commandFirstWord, durationMs, exitCode } }
tool onExecEnd(ctx) {
    duration = (ctx.exec.durationMs / 1000).toFixed(1)
    if (ctx.exec.exitCode == 0) {
        return "● " + ctx.exec.commandFirstWord + " ✓ (" + duration + "s)"
    }
    return "● " + ctx.exec.commandFirstWord + " ✗ (" + duration + "s) exit " + ctx.exec.exitCode
}
gsh.on("agent.exec.end", onExecEnd)
```

### `agent.tool.pending`

```gsh
# ctx: { toolCall: { id, name, status } }
# Fires when tool call enters pending state (streaming from LLM, before args are complete)
# Aligns with ACP's pending status
tool onToolPending(ctx) {
    # Show spinner with tool name while streaming
    return "○ " + ctx.toolCall.name + " ⠋"
}
gsh.on("agent.tool.pending", onToolPending)
```

### `agent.tool.start`

```gsh
# ctx: { toolCall: { id, name, args } }
# Fires when tool execution begins (after streaming is complete)
tool onToolStart(ctx) {
    return "○ " + ctx.toolCall.name
}
gsh.on("agent.tool.start", onToolStart)
```

### `agent.tool.end`

```gsh
# ctx: { toolCall: { id, name, args, durationMs, output, error } }
tool onToolEnd(ctx) {
    duration = (ctx.toolCall.durationMs / 1000).toFixed(2)
    if (ctx.toolCall.error != null) {
        return "● " + ctx.toolCall.name + " ✗ (" + duration + "s)"
    }
    return "● " + ctx.toolCall.name + " ✓ (" + duration + "s)"
}
gsh.on("agent.tool.end", onToolEnd)
```

### `agent.iteration.start`

```gsh
# ctx: { iteration: number }
tool onIterationStart(ctx) {
    # Called at start of each LLM request/response cycle
}
gsh.on("agent.iteration.start", onIterationStart)
```

### `agent.iteration.end`

```gsh
# ctx: { iteration: number, usage: { inputTokens, outputTokens, cachedTokens } }
tool onIterationEnd(ctx) {
    # Called at end of each LLM request/response cycle
}
gsh.on("agent.iteration.end", onIterationEnd)
```

### `agent.chunk`

```gsh
# ctx: { content: string }
tool onChunk(ctx) {
    # Called for each streaming text chunk from LLM
}
gsh.on("agent.chunk", onChunk)
```

### `repl.ready`

```gsh
# No ctx parameter
tool onReplReady() {
    print("Welcome to gsh " + gsh.version + "!")
}
gsh.on("repl.ready", onReplReady)
```

### `repl.command.before`

```gsh
# ctx: { command: string }
tool onCommandBefore(ctx) {
    # Called before shell command executes
}
gsh.on("repl.command.before", onCommandBefore)
```

### `repl.command.after`

```gsh
# ctx: { command: string, exitCode: number, durationMs: number }
tool onCommandAfter(ctx) {
    # Called after shell command completes
}
gsh.on("repl.command.after", onCommandAfter)
```

### `repl.exit`

```gsh
# No ctx parameter
tool onReplExit() {
    print("Goodbye!")
}
gsh.on("repl.exit", onReplExit)
```
