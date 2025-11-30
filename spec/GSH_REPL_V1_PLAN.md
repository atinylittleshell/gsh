# gsh REPL v1 Technical Plan

## Overview

This document outlines the plan to create a new REPL implementation based on the gsh script engine. The new implementation will:

1. **Consolidate** functionality currently spread across `pkg/` into `internal/repl/`
2. **Preserve** all key REPL features from the current implementation
3. **Enable** `.gshrc.gsh` configuration using the gsh scripting language
4. **Maintain** backward compatibility with functionality supported by existing `.gshrc` file

The goal is a clean-slate implementation that leverages the script interpreter's infrastructure while maintaining feature parity with the existing REPL.

We don't plan to remove or touch the existing REPL implementation for now. The new implementation will coexist until it is stable and fully tested.

---

## Current Architecture Analysis

### Entry Flow

```
cmd/gsh/main.go
    ├── initializeRunner()         → Creates mvdan/sh interpreter, loads .gshrc
    ├── initializeHistoryManager() → SQLite-based command history
    ├── initializeAnalyticsManager() → Usage analytics
    ├── initializeCompletionManager() → Bash completion specs
    └── run()
        └── core.RunInteractiveShell() → Main REPL loop
```

### Key Components to Consolidate

| Current Location         | Responsibility                     | New Location                |
| ------------------------ | ---------------------------------- | --------------------------- |
| `pkg/gline/`             | Line input with prediction overlay | `internal/repl/input/`      |
| `pkg/shellinput/`        | Text input widget (forked bubbles) | `internal/repl/input/`      |
| `pkg/debounce/`          | Generic debounce utility           | `internal/repl/util/`       |
| `pkg/reverse/`           | Generic slice reverse              | Inline or stdlib            |
| `internal/core/shell.go` | Interactive shell loop             | `internal/repl/`            |
| `internal/predict/`      | LLM prediction                     | `internal/repl/predict/`    |
| `internal/agent/`        | Agent chat mode                    | `internal/repl/agent/`      |
| `internal/bash/`         | Bash execution                     | Keep (shared)               |
| `internal/completion/`   | Completion system                  | `internal/repl/completion/` |
| `internal/rag/`          | Context retrieval                  | `internal/repl/context/`    |
| `internal/history/`      | Command history                    | Keep (shared)               |
| `internal/environment/`  | Config via env vars                | `internal/repl/config/`     |

### Current Dependencies on `mvdan.cc/sh/v3/interp`

The current implementation heavily relies on `interp.Runner` for:

- Environment variable storage and access
- Bash command execution
- Subshell creation
- Variable expansion

The new implementation should use the gsh script interpreter's `Environment` as the primary configuration store, with a bash executor for command execution.

---

## New Architecture Design

### Directory Structure

```
internal/repl/
├── repl.go                 # Main REPL entry point and loop
├── repl_test.go
├── config/
│   ├── config.go           # Configuration management
│   ├── config_test.go
│   ├── loader.go           # .gshrc.gsh and .gshrc loading
│   └── loader_test.go
├── input/
│   ├── input.go            # Unified line input (merges gline + shellinput)
│   ├── input_test.go
│   ├── buffer.go           # Text buffer and cursor management
│   ├── buffer_test.go
│   ├── keymap.go           # Key bindings
│   ├── completion.go       # Tab completion handling
│   ├── prediction.go       # LLM prediction integration
│   └── render.go           # View rendering (input, completions, explanations)
├── predict/
│   ├── predictor.go        # LLM prediction interface
│   ├── prefix.go           # Prefix-based prediction
│   ├── nullstate.go        # Empty input prediction
│   ├── explainer.go        # Command explanation
│   └── router.go           # Prediction routing
├── agent/
│   ├── adapter.go          # Thin wrapper adapting script engine's agent for REPL
│   └── adapter_test.go
│   # NOTE: REPL-specific tools (bash execution, file ops, permissions) - TBD
│   # These may be implemented as MCP servers rather than Go code, since tools
│   # for agents in the gsh script language should be added via MCP. This needs
│   # further investigation to determine the best approach.
├── context/
│   ├── provider.go         # Context aggregation
│   ├── retriever.go        # Retriever interface
│   ├── cwd.go              # Working directory
│   ├── git.go              # Git status
│   ├── history.go          # Command history
│   └── system.go           # System info
├── completion/
│   ├── manager.go          # Completion spec management
│   ├── provider.go         # Completion provider
│   ├── compgen.go          # compgen integration
│   ├── files.go            # File completion
│   └── words.go            # Word completion
├── executor/
│   ├── executor.go         # Command execution abstraction
│   ├── bash.go             # Bash command executor
│   └── gsh.go              # GSH script executor
└── util/
    └── debounce.go         # Debounce utility
```

### Core Interfaces

```go
// internal/repl/repl.go

// REPL is the main interactive shell interface
type REPL struct {
    config     *config.Config
    executor   executor.Executor
    history    *history.HistoryManager
    predictor  predict.Predictor
    explainer  predict.Explainer
    agent      *agent.Agent
    context    *context.Provider
    completion *completion.Manager
    logger     *zap.Logger

    // Script interpreter for .gshrc.gsh
    interpreter *interpreter.Interpreter
}

// Run starts the interactive REPL loop
func (r *REPL) Run(ctx context.Context) error

// ExecuteCommand runs a single command (bash or gsh)
func (r *REPL) ExecuteCommand(ctx context.Context, input string) error
```

```go
// internal/repl/config/config.go

// Config holds all REPL configuration extracted from GSH_CONFIG and declarations
type Config struct {
    // Prompt configuration (from GSH_CONFIG.prompt)
    Prompt   string
    LogLevel string

    // Agent configuration (from GSH_CONFIG.agent)
    ApprovedBashCommands []string
    Macros               map[string]string

    // All declarations from .gshrc.gsh (using gsh language syntax)
    // These are available for use in scripts and agent mode
    MCPServers map[string]*mcp.MCPServer           // from `mcp` declarations
    Models     map[string]*interpreter.ModelValue  // from `model` declarations
    Agents     map[string]*interpreter.AgentValue  // from `agent` declarations
    Tools      map[string]*interpreter.ToolValue   // from `tool` declarations
}

// Reserved tool names (looked up in Tools map):
//   - "GSH_UPDATE_PROMPT" - called before each prompt, signature: (exitCode: number, durationMs: number): string
```

```go
// internal/repl/executor/executor.go

// Executor abstracts command execution
type Executor interface {
    // ExecuteBash runs a bash command
    ExecuteBash(ctx context.Context, command string) (exitCode int, err error)

    // ExecuteBashInSubshell runs a bash command in a subshell, returning output
    ExecuteBashInSubshell(ctx context.Context, command string) (stdout, stderr string, err error)

    // ExecuteGsh runs a gsh script
    ExecuteGsh(ctx context.Context, script string) error

    // GetEnv gets an environment variable
    GetEnv(name string) string

    // SetEnv sets an environment variable
    SetEnv(name, value string)

    // GetPwd returns current working directory
    GetPwd() string
}
```

---

## Implementation Phases

### Phase 1: Foundation

**Goal:** Create the basic REPL structure with configuration loading

- [x] Create `internal/repl/` directory structure
- [x] Implement `config.Config` struct with all configuration fields
- [x] Implement `config.Loader` to load `.gshrc.gsh` files
- [x] Create `executor.Executor` interface and implementations
- [x] Write comprehensive tests

### Phase 2: Input System

**Goal:** Create unified input component merging `pkg/gline` and `pkg/shellinput`

The current implementation has two nested Bubble Tea models (`gline` wraps `shellinput`),
which creates unnecessary indirection. For the clean-slate implementation, these should
be merged into a single cohesive component.

**Rationale for merging:**

- `shellinput` was forked specifically for gsh, not a general-purpose library
- Echo modes (password/hidden) from shellinput are not used in gsh
- `gline` is the only consumer of `shellinput`
- Key handling is awkwardly split between both components
- Prediction-as-suggestion model is tightly coupled

**Implementation:**

- [x] Create `internal/repl/input/buffer.go` - text buffer and cursor management
  - Rune-based text storage
  - Cursor position tracking
  - Word boundary detection
  - Insert/delete operations
- [ ] Create `internal/repl/input/keymap.go` - key bindings
  - Emacs-style navigation (Ctrl+A/E/F/B/K/U/W etc.)
  - History navigation (Up/Down)
  - Completion triggers (Tab/Shift+Tab)
  - Special keys (Ctrl+C/D/L, Enter)
- [ ] Create `internal/repl/input/completion.go` - tab completion
  - Completion provider interface
  - Multi-suggestion cycling
  - Completion info/help box state
- [ ] Create `internal/repl/input/prediction.go` - LLM prediction integration
  - Async prediction with state ID coordination
  - Debounced prediction requests
  - Explanation display state
- [ ] Create `internal/repl/input/render.go` - view rendering
  - Input line with cursor
  - Prediction overlay (ghost text)
  - Completion box
  - Explanation/help box
- [ ] Create `internal/repl/input/input.go` - main unified component
  - Bubble Tea model (Init/Update/View)
  - Coordinates all sub-components
  - History value management
- [ ] Consolidate debounce utility to `internal/repl/util/debounce.go`
- [ ] Remove `pkg/reverse` (use `slices.Reverse` from stdlib)
- [ ] Write comprehensive tests

### Phase 3: Context & Prediction

**Goal:** Port RAG and prediction systems

- [ ] Port context retrievers to `internal/repl/context/`
  - Working directory
  - Git status
  - System info
  - Command history
- [ ] Port prediction system to `internal/repl/predict/`
  - Prefix predictor
  - Null-state predictor
  - Explainer
  - Router
- [ ] Integrate with new config system for LLM settings
- [ ] Write comprehensive tests

### Phase 4: Completion System

**Goal:** Port tab completion

- [ ] Port completion manager to `internal/repl/completion/`
- [ ] Port completion provider
- [ ] Port compgen integration
- [ ] Port file completion
- [ ] Write comprehensive tests

### Phase 5: Agent Mode

**Goal:** Create thin adapter for script engine's agent in REPL context

**Pre-requisite:** Add `maxInputTokens` as a property on `ModelValue` in the script engine (`internal/script/interpreter/model.go`)

The script engine (`internal/script/interpreter/`) already provides:

- `agent.go` - Agent execution with tool calling loop
- `conversation.go` - Conversation state and message history
- `model.go` - Model/provider abstraction (OpenAI, Anthropic, etc.)

The REPL agent module should be a **minimal adapter** that:

- Provides REPL-specific tools (bash execution with permissions, file ops with confirmation)
- Handles interactive I/O (streaming responses, user confirmations)
- Manages the agent session lifecycle in REPL context

**NOTE:** REPL-specific agent tools (bash execution with permissions, file operations
with confirmations, done signal, permissions menu) - implementation approach TBD.
These may be better implemented as MCP servers rather than Go code, since the gsh
script language uses MCP for tool integration. This needs further investigation.

### Phase 6: Main REPL Loop

**Goal:** Implement the main REPL

- [ ] Implement `REPL.Run()` main loop
  - Prompt display with update function support
  - Input handling
  - Command execution (bash for now)
  - History recording
  - Signal handling (Ctrl+C, Ctrl+D)
  - Control commands (`:clear`, `:exit`, etc.)
- [ ] Create `internal/repl/agent/adapter.go` - thin wrapper around `interpreter.Agent`
  - Extract GSH_DEFAULT_AGENT from config - use that for REPL
  - Manage a Conversation instance
  - Hook it up to REPL - "#" is agent mode
- [ ] Handle agent streaming output to terminal (this may need changes in the core model provider and agent logic)
- [ ] Integrate all subsystems
- [ ] Write integration tests

### Phase 7: Integration

**Goal:** Wire up the new REPL to main.go

- [ ] Create `NewREPL()` constructor in `internal/repl/`
- [ ] Update `cmd/gsh/main.go` to use new REPL
  - Add flag or detection logic to choose implementation
  - Initially run both in parallel for testing
- [ ] Ensure all existing tests pass
- [ ] Add new integration tests

### Phase 8: Migration & Cleanup

**Goal:** Complete the transition

- [ ] Implement backward compatibility layer for `.gshrc` bash files
  - Run `.gshrc` through bash executor
  - Map `GSH_*` environment variables to `Config` fields
- [ ] Remove old implementation once new one is stable
  - Remove `pkg/gline/`
  - Remove `pkg/shellinput/`
  - Remove `pkg/debounce/`
  - Remove `pkg/reverse/`
  - Remove `internal/core/shell.go` (keep paths.go, prompter.go)
  - Remove `internal/predict/`
  - Remove old agent code if fully replaced
- [ ] Update all imports
- [ ] Update documentation
- [ ] Final testing

---

## .gshrc.gsh Configuration Design

### Configuration Schema

```gsh
# ~/.gshrc.gsh

# Define models using the gsh language syntax
model claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-sonnet-4-20250514",
}

# Optional: Pre-configure MCP servers for agent mode
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem", env.HOME],
}

mcp github {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-github"],
    env: {
        GITHUB_TOKEN: env.GITHUB_TOKEN,
    },
}

# Optional: Pre-configure agents using the gsh language syntax
agent coder {
    model: claude,
    systemPrompt: "You are a coding assistant.",
    tools: [filesystem.read_file, filesystem.write_file],
}

# GSH_CONFIG - Reserved configuration object for REPL settings
GSH_CONFIG = {
    # Prompt settings
    prompt: "gsh> ",
    logLevel: "info",

    # Agent settings
    agent: {
        approvedBashCommands: [
            "^ls.*$",
            "^pwd$",
            "^cd\\s+.*$",
            "^git\\s+status.*$",
            "^git\\s+diff.*$",
        ],
        macros: {
            gitdiff: "Review all staged and unstaged changes and suggest improvements",
            gitpush: "Review changes, create a commit with a good message, and push",
        },
    },
}

# Optional: Custom prompt update tool (reserved name)
tool GSH_UPDATE_PROMPT(exitCode: number, durationMs: number): string {
    if (exitCode == 0) {
        return "gsh> "
    }
    return `gsh [${exitCode}]> `
}
```

### Loading Priority

1. Load embedded `.gshrc.default` (sets baseline defaults)
2. Load `/etc/profile` and `~/.gsh_profile` (if login shell)
3. Load `~/.gshrc` (bash format, for backward compatibility)
4. Load `~/.gshenv` (bash format)
5. Load `~/.gshrc.gsh` (gsh format, overrides previous settings)

### Config Extraction

The loader will:

1. Execute `.gshrc.gsh` using the gsh interpreter
2. Look for the `GSH_CONFIG` variable in the environment (reserved name)
3. Convert the `GSH_CONFIG` object to `Config` struct
4. Look for `GSH_UPDATE_PROMPT` tool and store reference
5. Collect all `mcp`, `model`, and `agent` declarations (using gsh language syntax)

---

## Backward Compatibility

### Environment Variable Mapping

For users who prefer bash-style configuration, we maintain full backward compatibility:

| Env Variable          | .gshrc.gsh Equivalent                 |
| --------------------- | ------------------------------------- |
| `GSH_PROMPT`          | `GSH_CONFIG.prompt`                   |
| `GSH_LOG_LEVEL`       | `GSH_CONFIG.logLevel`                 |
| `GSH_UPDATE_PROMPT()` | `tool GSH_UPDATE_PROMPT(...) { ... }` |

### Migration Path

Users can migrate incrementally:

1. Keep using `.gshrc` - everything works as before
2. Create `.gshrc.gsh` with partial config - overrides specific settings
3. Eventually move all config to `.gshrc.gsh`

---

## Testing Strategy

### Unit Tests

- Each component has its own `_test.go` file
- Mock external dependencies (LLM clients, file system)
- Test configuration loading with various scenarios
- Test input handling edge cases

### Integration Tests

- End-to-end REPL tests with mocked terminal
- Configuration loading from actual files
- Command execution tests
- Agent mode tests with mocked LLM

### Compatibility Tests

- Test with existing `.gshrc` files
- Test with new `.gshrc.gsh` files
- Test mixed configurations
- Test all existing features work identically

---

## Success Criteria

- [ ] All existing REPL features work identically
- [ ] `.gshrc.gsh` configuration is fully supported
- [ ] Backward compatibility with `.gshrc` maintained
- [ ] `pkg/` directory can be removed (no external dependencies on it)
- [ ] All tests pass
- [ ] No performance regression
- [ ] Documentation updated

---

## Risk Mitigation

1. **Feature Parity**: Comprehensive test suite comparing old vs new behavior
2. **Performance**: Benchmark critical paths (startup, prediction, completion)
3. **User Disruption**: Keep old implementation until new one is stable
4. **Complexity**: Incremental implementation with clear phase boundaries

---

## Timeline Estimate

| Phase                         | Estimated Effort |
| ----------------------------- | ---------------- |
| Phase 1: Foundation           | 2-3 days         |
| Phase 2: Input System         | 3-4 days         |
| Phase 3: Context & Prediction | 2-3 days         |
| Phase 4: Completion System    | 2-3 days         |
| Phase 5: Agent Mode           | 3-4 days         |
| Phase 6: Main REPL Loop       | 2-3 days         |
| Phase 7: Integration          | 2-3 days         |
| Phase 8: Migration & Cleanup  | 1-2 days         |
| **Total**                     | **17-25 days**   |

---

## Open Questions

1. **Prompt Update Mechanism**: Should `updatePrompt` be a gsh tool or support calling external commands like starship?

   - Proposal: Support both - gsh tool takes precedence, fall back to `GSH_UPDATE_PROMPT` bash function

2. **MCP Server Lifecycle**: Should MCP servers declared in `.gshrc.gsh` stay running for the entire session?

   - Proposal: Yes, lazy-start on first use, keep running until shell exit

3. **Agent Integration**: Should agents from `.gshrc.gsh` be usable in the REPL's agent mode?

   - Proposal: Yes, allow selecting which agent to use for chat mode

4. **Error Handling**: How to handle `.gshrc.gsh` parse/runtime errors?
   - Proposal: Print warning, continue with defaults, don't block shell startup

---

**Document Version:** 1.0
**Last Updated:** 2025-01-13
