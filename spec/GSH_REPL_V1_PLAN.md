# gsh REPL v1 Technical Plan

## Overview

This document outlines the plan to create a new REPL implementation based on the gsh script engine. The new implementation will:

1. **Consolidate** functionality currently spread across `pkg/` into `internal/repl/`
2. **Preserve** all key REPL features from the current implementation
3. **Enable** `.gshrc.gsh` configuration using the gsh scripting language for all gsh-specific features
4. **Maintain** bash compatibility via `.gshrc` (pure bash file, no gsh-specific features)

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
| `pkg/debounce/`          | Generic debounce utility           | Removed (unused)            |
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
│   └── render.go           # View rendering (input, completions)
├── predict/
│   ├── predictor.go        # LLM prediction interface
│   ├── prefix.go           # Prefix-based prediction
│   ├── nullstate.go        # Empty input prediction
│   └── router.go           # Prediction routing
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
└── executor/
    ├── executor.go         # Command execution abstraction
    ├── bash.go             # Bash command executor
    └── gsh.go              # GSH script executor
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

// Config holds all REPL configuration extracted from GSH_CONFIG and declarations in .gshrc.gsh
type Config struct {
    // Prompt configuration (from GSH_CONFIG.prompt in .gshrc.gsh)
    Prompt   string
    LogLevel string

    // Agent configuration (from GSH_CONFIG.agent)
    ApprovedBashCommands []string
    Macros               map[string]string

    // All declarations from .gshrc.gsh (using gsh language syntax)
    // These are available for use in scripts and agent mode
    MCPServers map[string]*mcp.MCPServer           // from `mcp` declarations
    Models     map[string]*interpreter.ModelValue  // from `model` declarations
    Agents     map[string]*interpreter.AgentValue  // from `agent` declarations (custom agents only)
    Tools      map[string]*interpreter.ToolValue   // from `tool` declarations
    
    // NOTE: The built-in default agent is NOT in this map.
    // It's hardcoded in the REPL and always available as "default".
    // Custom agents defined in .gshrc.gsh are stored in the Agents map.
}

// Reserved tool names (looked up in Tools map):
//   - "GSH_UPDATE_PROMPT" - called before each prompt, signature: (exitCode: number, durationMs: number): string
//
// Note: .gshrc is pure bash (no GSH_* environment variables or functions). All gsh-specific
// configuration must be done in .gshrc.gsh using the GSH_CONFIG object and gsh language syntax.
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

> **Note on Phase Ordering:** Phases 3-7 were reordered from the original plan to enable
> an end-to-end working product sooner. The Main REPL Loop and Integration phases were
> moved earlier so we can have a usable shell quickly, then incrementally add prediction,
> completion, and agent features. This reduces integration risk and provides faster feedback.

### Phase 1: Foundation ✅

**Goal:** Create the basic REPL structure with configuration loading

- [x] Create `internal/repl/` directory structure
- [x] Implement `config.Config` struct with all configuration fields
- [x] Implement `config.Loader` to load `.gshrc.gsh` files
- [x] Create `executor.Executor` interface and implementations
- [x] Write comprehensive tests

### Phase 2: Input System ✅

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
- [x] Create `internal/repl/input/keymap.go` - key bindings
  - Create a configurable key binding system via `Config`
  - Emacs-style navigation (Ctrl+A/E/F/B/K/U/W etc.)
  - History navigation (Up/Down)
  - Completion triggers (Tab/Shift+Tab)
  - Special keys (Ctrl+C/D/L, Enter)
- [x] Create `internal/repl/input/completion.go` - tab completion
  - Completion provider interface
  - Multi-suggestion cycling
  - Completion info/help box state
- [x] Create `internal/repl/input/prediction.go` - LLM prediction integration
  - Use the model provider from the script engine, and a model from config named GSH_PREDICT_MODEL
  - Async prediction with state ID coordination
  - Prefer history based predictions whenever that's available. Use LLM as fallback and when input is empty
  - Debounced prediction requests
- [x] Create `internal/repl/input/render.go` - view rendering
  - Input line with cursor
  - Prediction overlay (ghost text)
  - Completion box
- [x] Create `internal/repl/input/input.go` - main unified component
  - Bubble Tea model (Init/Update/View)
  - Coordinates all sub-components
  - History value management
- [x] Remove `pkg/debounce/` (unused, prediction.go uses inline context-aware pattern)
- [x] Remove `pkg/reverse` (use `slices.Reverse` from stdlib)

### Phase 3: Main REPL Loop (Minimal E2E) ✅

**Goal:** Implement the main REPL loop with minimal dependencies to get an end-to-end working shell

This phase focuses on getting a working shell as quickly as possible. Prediction and completion
are stubbed with no-op implementations that will be filled in during later phases.

**Minimal Implementation (must have):**

- [x] Implement `REPL` struct in `internal/repl/repl.go`
  - Config, executor, history manager, logger
  - Stub interfaces for prediction and completion (no-op providers)
- [x] Implement `NewREPL()` constructor
  - Load config via `config.Loader`
  - Initialize executor
  - Initialize history manager (reuse `internal/history/`)
  - Create no-op prediction provider
  - Create no-op completion provider
- [x] Implement `REPL.Run()` main loop
  - Display prompt (static prompt from config initially)
  - Read input using `internal/repl/input/` component
  - Execute commands via executor
  - Record commands in history
  - Handle Ctrl+C (cancel current input)
  - Handle Ctrl+D on empty line (exit shell)
  - Handle Ctrl+L (clear screen)
- [x] Implement basic control commands
  - `:exit` or `exit` - exit the shell
  - `:clear` - clear screen
- [x] Basic error handling
  - Display command exit codes on failure
  - Handle executor errors gracefully

**Stub Implementations (to be filled in later phases):**

```go
// No-op prediction provider for Phase 3
type NoOpPredictor struct{}
func (p *NoOpPredictor) Predict(ctx context.Context, input string) (string, error) {
    return "", nil // No prediction
}

// No-op completion provider for Phase 3
type NoOpCompleter struct{}
func (c *NoOpCompleter) Complete(ctx context.Context, input string, pos int) ([]string, error) {
    return nil, nil // No completions
}
```

**Testing:**

- [x] Unit tests for REPL initialization
- [x] Integration test: start REPL, run `echo hello`, verify output
- [x] Integration test: Ctrl+D exits cleanly
- [x] Integration test: Ctrl+C cancels input

### Phase 4: Integration ✅

**Goal:** Wire up the new REPL to main.go for real-world testing

- [x] Update `cmd/gsh/main.go` to use new REPL
  - Keep old implementation in codebase for now, but default to new REPL
- [x] Ensure new REPL can be tested interactively

### Phase 5: Context & Prediction

**Goal:** Port RAG and prediction systems to enable LLM-powered suggestions

- [x] Port context retrievers to `internal/repl/context/`
  - Working directory context
  - Git status context
  - System info context
  - Command history context
- [x] Port prediction system to `internal/repl/predict/`
  - Prefix predictor (predict based on typing)
  - Null-state predictor (suggest when input is empty)
  - Router (coordinate between strategies)
- [x] Integrate with new config system for LLM settings
- [x] Wire prediction into REPL (replace no-op provider)

### Phase 6: Completion System ✅

**Goal:** Port tab completion for command and file completion

- [x] Port completion manager to `internal/repl/completion/`
- [x] Port completion provider
- [x] Port compgen integration (bash completion specs)
- [x] Port file completion
- [x] Wire completion into REPL (replace no-op provider)

### Phase 7: Agent Mode ✅

**Goal:** Enable agent chat interactions in REPL with direct provider access

The script engine (`internal/script/interpreter/`) already provides:

- `agent.go` - Agent execution with tool calling loop
- `conversation.go` - Conversation state and message history
- `model.go` - Model/provider abstraction (OpenAI, Anthropic, etc.)
- `provider_openai.go` - OpenAI-compatible provider implementation

**Design Decision:** REPL directly uses providers instead of creating an adapter layer.

Initially planned to create an adapter wrapper around the script engine's agent functionality,
but this would block streaming support (adapter sits between REPL and provider, preventing
direct streaming callbacks). The REPL now:

- Loads agent config from script engine (model, systemPrompt, etc.)
- Calls `ModelProvider.ChatCompletion()` directly
- Manages conversation history in REPL state (`[]ChatMessage`)
- Can easily add streaming support in the future by calling provider's stream method

**Default Agent Design:**

- **Built-in default agent is immutable** - cannot be overridden in configuration
- Default agent is defined in code with sensible defaults (simple chat assistant, no tools)
- Users wanting custom agents must define them in `.gshrc.gsh` and explicitly switch using `#/agent <name>`
- No `GSH_CONFIG.defaultAgent` configuration option
- This ensures consistent out-of-box experience while allowing full customization

**Implementation:**

- [x] Agent configuration loading via script engine
  - ~~Uses `GSH_CONFIG.defaultAgent` or single agent fallback~~ (removed)
  - Uses built-in default agent defined in code
  - Extracts model and provider from agent's Config map
- [x] Direct provider integration in REPL
  - Store `AgentValue`, `ModelProvider`, and conversation history
  - Build chat requests with system prompt + history + new message
  - Call provider directly for responses
- [x] Hook agent into REPL - "#" prefix triggers agent mode
- [x] Conversation state management
  - System prompt included in each request (not stored in history)
  - User/assistant messages tracked across interactions
  - `# reset` command to clear conversation
- [x] Write comprehensive tests for agent initialization and fallback logic
- [x] Add `SetVariable()` method to interpreter (for future script-based agent usage)

**Benefits of Direct Provider Approach:**

- Simpler architecture (no adapter layer to maintain)
- Streaming support becomes straightforward (add `ChatCompletionStream` to provider interface)
- Full control over conversation management
- Cleaner separation of concerns: script engine for config, provider for execution

### Phase 8: Agent Switching ✅

**Goal:** Allow users to dynamically switch between configured agents

Users can switch agents on-the-fly without editing config files, with each agent maintaining
isolated conversation history. The shell always starts with the **built-in default agent**, and
users must explicitly switch to custom agents defined in `.gshrc.gsh`.

**Syntax:**

```bash
#<message>              # Send message to current agent (starts with built-in default)
#/clear                 # Clear current agent's conversation
#/agents                # List all available agents (built-in + custom from .gshrc.gsh)
#/agent <name>          # Switch to a different agent (custom agents only)
#/agent default         # Switch back to the built-in default agent
```

**Agent Types:**

- **Built-in default agent**: Hardcoded, immutable, always available as "default"
  - Simple chat assistant with no tools
  - Consistent out-of-box experience
  - Cannot be overridden in configuration
- **Custom agents**: Defined in `.gshrc.gsh` using `agent` declarations
  - Can have custom system prompts, models, and tools
  - Must be explicitly activated with `#/agent <name>`
  - Each maintains isolated conversation history

**Implementation:**

- [x] Multi-agent state management with `map[string]*AgentState`
- [x] Command parsing to distinguish between messages and commands (`parseAgentInput`)
- [x] Agent command handlers (`/clear`, `/agents`, `/agent`)
- [x] Completion support for agent commands and names
- [x] Single source of truth for agent commands in `AgentCommands` variable
- [x] Each agent maintains isolated conversation history
- [x] Switching preserves conversation state for all agents
- [x] Comprehensive tests for all agent switching scenarios
- [ ] **TODO**: Implement built-in default agent that can't be overridden
- [ ] **TODO**: Remove `GSH_CONFIG.defaultAgent` configuration option
- [ ] **TODO**: Update agent initialization to always start with built-in default
- [ ] **TODO**: Add "default" as reserved agent name in command completion

### Phase 9: Migration & Cleanup ✅

**Goal:** Complete the transition

- [x] Ensure bash compatibility for `.gshrc` files
- [x] Remove gsh specific features from `.gshrc.default`
- [x] Remove old implementation once new one is stable
- [x] Update all imports
- [x] Update documentation
- [x] Final testing

### Phase 10: Default Configuration

**Goal:** Provide comprehensive default gsh configuration and built-in default agent

- [ ] **Implement built-in default agent in code**
  - Define immutable default agent with simple chat assistant prompt
  - No tools, no special capabilities - just basic chat
  - Use default model (e.g., Ollama qwen2.5 or first available model)
  - Cannot be overridden by user configuration
  - Always available as "default" agent
- [ ] Create `.gshrc.default.gsh` with example configurations
  - Example model configurations (Ollama with qwen2.5, OpenAI, Anthropic)
  - Example custom agent configurations (coder, reviewer, etc.)
  - Default GSH_CONFIG settings (prompt, logLevel, etc.)
  - Example macros (gitdiff, gitpush, gitreview)
  - Context configuration examples for RAG
  - Example MCP server declarations
  - **NOTE**: No `defaultAgent` configuration - users must explicitly switch to custom agents
- [ ] Load `.gshrc.default.gsh` during REPL initialization (before user's `.gshrc.gsh`)
  - Default file provides examples only, not active configuration
  - Users copy/modify examples into their own `.gshrc.gsh`
- [ ] Update documentation to reference both default files and built-in agent

---

## Configuration Design

### .gshrc vs .gshrc.gsh

**`.gshrc` - Bash Compatibility Only**

The `.gshrc` file is executed as pure bash for backward compatibility. It exists to make migration from bash/zsh easy - users can copy-paste their existing `.bashrc` content without modifications.

Use `.gshrc` for:

- Aliases: `alias ll='ls -la'`
- PATH modifications: `export PATH="$HOME/bin:$PATH"`
- Standard bash functions and environment variables
- Anything that would work in a regular bash shell

**Do NOT use `.gshrc` for:**

- gsh-specific configuration (prompt customization, log levels, etc.)
- Model, agent, or MCP server declarations
- Any `GSH_*` variables or functions

**`.gshrc.gsh` - All GSH-Specific Configuration**

The `.gshrc.gsh` file is where all gsh-specific features are configured using the gsh scripting language.

Use `.gshrc.gsh` for:

- REPL configuration via `GSH_CONFIG` object
- Model declarations: `model claude { ... }`
- Agent declarations: `agent coder { ... }`
- MCP server declarations: `mcp filesystem { ... }`
- Tool declarations: `tool GSH_UPDATE_PROMPT(...) { ... }`

### .gshrc.gsh Configuration Schema

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
    
    # NOTE: No defaultAgent configuration
    # The built-in default agent is always used on startup
    # To use custom agents, switch explicitly with: #/agent <name>
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

1. Load `/etc/profile` and `~/.gsh_profile` (if login shell)
2. **Load `~/.gshrc` (pure bash - for bash/zsh compatibility)**
3. Load `~/.gshenv` (bash format)
4. **Load `~/.gshrc.gsh` (gsh-specific configuration)**

The `.gshrc` file is loaded first via bash executor, then `.gshrc.gsh` is loaded via gsh interpreter. This means:

- Standard shell setup (aliases, PATH) happens in bash
- gsh-specific features are configured in gsh
- `.gshrc.gsh` can access environment variables set in `.gshrc`

### Config Extraction

The loader will:

1. Execute `.gshrc` using bash executor (skip if file doesn't exist)
2. Execute `.gshrc.gsh` using the gsh interpreter (skip if file doesn't exist)
3. Look for the `GSH_CONFIG` variable in the interpreter environment (reserved name)
4. Convert the `GSH_CONFIG` object to `Config` struct
5. Look for `GSH_UPDATE_PROMPT` tool and store reference
6. Collect all `mcp`, `model`, and `agent` declarations (using gsh language syntax)

---

## Backward Compatibility

### Migration from Bash/Zsh

Users migrating from bash or zsh can copy their existing configuration directly:

```bash
# Simple migration:
cp ~/.bashrc ~/.gshrc

# Then optionally create ~/.gshrc.gsh for gsh-specific features
```

The `.gshrc` file will work exactly like `.bashrc` - it's pure bash with no special gsh behavior.

### Prompt Customization

**Static Prompts:**

```gsh
# ~/.gshrc.gsh
GSH_CONFIG = {
    prompt: "gsh> ",
}
```

**Dynamic Prompts:**

```gsh
# ~/.gshrc.gsh
tool GSH_UPDATE_PROMPT(exitCode: number, durationMs: number): string {
    if (exitCode == 0) {
        return "✓ gsh> "
    }
    return `✗ [${exitCode}] gsh> `
}
```

**Note:** Prompt customization requires `.gshrc.gsh`. The old `GSH_PROMPT` environment variable and `GSH_UPDATE_PROMPT()` bash function are not supported. This keeps the implementation clean and encourages users to adopt the more powerful gsh scripting approach.

### Migration Path

1. **Start:** Copy your `.bashrc` to `.gshrc`
2. **Basic gsh features:** Create `.gshrc.gsh` with `GSH_CONFIG` for prompt customization
3. **Advanced features:** Add model/agent/MCP declarations to `.gshrc.gsh` as needed

No need to "port" bash functions to gsh - they continue to work in `.gshrc`. Only gsh-specific features require `.gshrc.gsh`.

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

- Test with bash-style `.gshrc` files (aliases, PATH, env vars)
- Test with gsh-specific `.gshrc.gsh` files
- Test combined `.gshrc` + `.gshrc.gsh` configurations
- Test that bash features in `.gshrc` work identically to standard shells
- Test that gsh-specific features require `.gshrc.gsh`

---

## Success Criteria

- [x] All existing REPL features work identically
- [x] `.gshrc.gsh` configuration is fully supported
- [x] Backward compatibility with `.gshrc` maintained
- [x] `pkg/` directory can be removed (no external dependencies on it)
- [x] All tests pass
- [x] No performance regression
- [x] Documentation updated

## Open Questions

1. **Prompt Update Mechanism**: Should `updatePrompt` be a gsh tool or support calling external commands like starship?

   - Proposal: Support both - gsh tool takes precedence, fall back to `GSH_UPDATE_PROMPT` bash function

2. **MCP Server Lifecycle**: Should MCP servers declared in `.gshrc.gsh` stay running for the entire session?

   - Proposal: Yes, lazy-start on first use, keep running until shell exit

3. **Agent Integration**: Should agents from `.gshrc.gsh` be usable in the REPL's agent mode?

   - Proposal: Yes, allow selecting which agent to use for chat mode

4. **Error Handling**: How to handle `.gshrc.gsh` parse/runtime errors?
   - Proposal: Print warning, continue with defaults, don't block shell startup
