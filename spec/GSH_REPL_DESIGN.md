# gsh REPL Design Document

## Overview

This document describes the technical design of the gsh REPL implementation based on the gsh script engine. The REPL consolidates functionality from multiple packages into a cohesive `internal/repl/` structure and enables `.gshrc.gsh` configuration using the gsh scripting language.

**Key Design Goals:**

1. **Consolidation**: Move REPL-specific code from `pkg/` into `internal/repl/`
2. **Script-based Configuration**: Enable `.gshrc.gsh` for all gsh-specific features
3. **Bash Compatibility**: Maintain pure bash support via `.gshrc` files
4. **Clean Architecture**: Leverage script interpreter infrastructure while maintaining feature parity

---

## Architecture

### Directory Structure

```
internal/repl/
├── repl.go                 # Main REPL entry point and loop
├── config/
│   ├── config.go           # Configuration management
│   └── loader.go           # .gshrc.gsh and .gshrc loading
├── input/
│   ├── input.go            # Unified line input (merges gline + shellinput)
│   ├── buffer.go           # Text buffer and cursor management
│   ├── keymap.go           # Key bindings
│   ├── completion.go       # Tab completion handling
│   ├── prediction.go       # LLM prediction integration
│   └── render.go           # View rendering
├── predict/
│   ├── predictor.go        # LLM prediction interface
│   ├── prefix.go           # Prefix-based prediction
│   ├── nullstate.go        # Empty input prediction
│   └── router.go           # Prediction routing
├── context/
│   ├── provider.go         # Context aggregation
│   ├── retriever.go        # Retriever interface
│   ├── cwd.go              # Working directory context
│   ├── git.go              # Git status context
│   ├── history.go          # Command history context
│   └── system.go           # System info context
├── completion/
│   ├── manager.go          # Completion spec management
│   ├── provider.go         # Completion provider
│   ├── compgen.go          # compgen integration
│   ├── files.go            # File completion
│   └── words.go            # Word completion
└── executor/
    └── executor.go         # Command execution abstraction
```

### Core Interfaces

```go
// REPL is the main interactive shell interface
type REPL struct {
    config             *config.Config
    executor           *executor.REPLExecutor
    history            *history.HistoryManager
    predictor          *predict.Router
    contextProvider    *replcontext.Provider
    completionProvider *completion.Provider
    logger             *zap.Logger
    agentStates        map[string]*AgentState    // Multi-agent state management
    currentAgentName   string
}

// REPLExecutor handles command execution (concrete implementation, not interface)
type REPLExecutor struct {
    runner      *interp.Runner              // mvdan/sh bash executor
    interpreter *interpreter.Interpreter   // gsh script executor
    logger      *zap.Logger
}

// Methods on REPLExecutor:
// - ExecuteBash(ctx context.Context, command string) (int, error)
// - ExecuteBashInSubshell(ctx context.Context, command string) (string, string, int, error)
// - ExecuteGsh(ctx context.Context, script string) error
// - GetEnv(name string) string
// - SetEnv(name, value string)
// - GetPwd() string
// - Interpreter() *interpreter.Interpreter  // For accessing the gsh interpreter
```

---

## Configuration Design

### Two-File Configuration System

**`.gshrc` - Pure Bash Compatibility**

- Executed as standard bash for backward compatibility
- Use for: aliases, PATH modifications, bash functions
- No gsh-specific features allowed
- Enables easy migration from bash/zsh

**`.gshrc.gsh` - GSH-Specific Configuration**

- Executed using gsh script interpreter
- Use for: REPL config, model/agent/MCP declarations, tools
- Provides full access to gsh language features

### Configuration Schema

```gsh
# ~/.gshrc.gsh

# Model declarations
model claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-sonnet-4-20250514",
}

# MCP server declarations
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem", env.HOME],
}

# Agent declarations
agent coder {
    model: claude,
    systemPrompt: "You are a coding assistant.",
    tools: [filesystem.read_file, filesystem.write_file],
}

# REPL configuration object
GSH_CONFIG = {
    prompt: "gsh> ",
    logLevel: "info",
    predictModel: ollamaPredict,        # Model reference for command prediction
    defaultAgentModel: ollamaAgent,     # Model reference for built-in default agent
}

# Custom prompt update tool
tool GSH_UPDATE_PROMPT(exitCode: number, durationMs: number): string {
    if (exitCode == 0) {
        return "✓ gsh> "
    }
    return `✗ [${exitCode}] gsh> `
}
```

### Configuration Loading

```go
type Config struct {
    // REPL settings
    Prompt   string
    LogLevel string

    // Prediction/Agent models (store model name, not model object)
    PredictModel      string  // Name of model to use for command prediction
    DefaultAgentModel string  // Name of model to use for built-in default agent

    // Declarations from .gshrc.gsh (map of name -> value)
    MCPServers map[string]*mcp.MCPServer
    Models     map[string]*interpreter.ModelValue
    Agents     map[string]*interpreter.AgentValue
    Tools      map[string]*interpreter.ToolValue
}
```

**Important:** `GSH_CONFIG.predictModel` and `GSH_CONFIG.defaultAgentModel` must be **model references** (e.g., `predictModel: myModel`), not strings. The loader extracts the model's name and stores it in the Config struct.

**Loading Priority:**

1. Load embedded default config from `~/.gshrc.default.gsh` (system defaults, if provided)
2. Load user config from `~/.gshrc.gsh` (user overrides, if exists)
3. Load `~/.gshrc` via bash executor (pure bash compatibility, if exists)
4. Load `~/.gshenv` via bash executor (if exists)
5. Extract `GSH_CONFIG` variable from gsh interpreter environment
6. Extract `GSH_UPDATE_PROMPT` tool if defined
7. Collect all declarations (models, agents, tools, MCP servers)

**Reserved Names:**

- `GSH_CONFIG` - Configuration object for REPL settings (fields: `prompt`, `logLevel`, `predictModel`, `defaultAgentModel`)
- `GSH_UPDATE_PROMPT` - Tool called before each prompt, signature: `(exitCode: number, durationMs: number): string`

---

## Input System Design

### Design Rationale

The previous implementation used two nested Bubble Tea models (`gline` wrapping `shellinput`), creating unnecessary indirection. The new implementation merges these into a single cohesive component.

**Why merge?**

- `shellinput` was forked specifically for gsh, not a general library
- Unused features (echo modes for passwords)
- `gline` was the only consumer
- Key handling split awkwardly between components
- Tight coupling with prediction model

### Input Component Architecture

```
input.Model (Bubble Tea model)
├── buffer.Buffer           # Text storage, cursor management
├── keymap.KeyMap           # Configurable key bindings
├── completion.State        # Tab completion state
├── prediction.State        # LLM prediction state
└── render.View             # Visual rendering
```

**Key Features:**

- Rune-based text buffer with efficient cursor operations
- Emacs-style key bindings (Ctrl+A/E/F/B/K/U/W)
- Tab completion with multi-suggestion cycling
- Async LLM prediction with debouncing
- History-based suggestions preferred over LLM
- Ghost text overlay for predictions

---

## Agent Mode Design

### Direct Provider Integration

**Design Decision:** REPL calls model providers directly instead of wrapping script engine's agent functionality.

**Rationale:**

- Enables future streaming support (adapter layer would block streaming callbacks)
- Simpler architecture (no adapter to maintain)
- Full control over conversation management
- Clean separation: script engine for config, provider for execution

### Built-in Default Agent

**Design Decision:** Default agent is immutable and hardcoded.

**Rationale:**

- Ensures consistent out-of-box experience
- Cannot be accidentally misconfigured
- Users wanting customization define new agents in `.gshrc.gsh`
- Must explicitly switch to custom agents

**Default Agent Properties:**

- Simple chat assistant with no tools
- Uses model from `GSH_CONFIG.defaultAgentModel` (or sensible fallback)
- Always available as "default"
- Cannot be overridden or removed

### Agent Commands

```bash
#<message>           # Send message to current agent
#/clear              # Clear current agent's conversation
#/agents             # List available agents
#/agent <name>       # Switch to different agent
#/agent default      # Switch to built-in default agent
```

### Multi-Agent State Management

```go
type AgentState struct {
    Agent        *interpreter.AgentValue
    Provider     interpreter.ModelProvider
    Conversation []interpreter.ChatMessage
}

// REPL maintains map of all agent states
type REPL struct {
    // ... other fields
    agentStates      map[string]*AgentState
    currentAgentName string
}
```

**Key Properties:**

- Each agent maintains isolated conversation history
- Switching agents preserves all conversation state
- Built-in default agent always initialized on startup if a default agent model is configured
- Custom agents are initialized when loaded from `.gshrc.gsh`
- System prompt is included in each request but NOT stored in conversation history

---

## Prediction System Design

### Two-Strategy Router

```
Router
├── PrefixPredictor     # Predict next command based on what user is typing
└── NullStatePredictor  # Suggest command when input is empty
```

**Strategy Selection:**

- Empty input → NullStatePredictor
- Non-empty input → PrefixPredictor

### Context-Aware Prediction

Prediction includes contextual information:

- Current working directory
- Git status (branch, modified files)
- Recent command history
- System information (OS, shell)

### History Preference

**Design Decision:** Prefer history-based predictions over LLM.

When user is typing, the prediction system checks in this order:

1. **Empty input**: Use NullStatePredictor (LLM-only)
2. **Non-empty input**:
   - First, check command history for entries starting with the input prefix
   - If found in history, use the most recent match (instant, free)
   - If not found in history, fall back to LLM prefix prediction
3. **Agent messages** (input starting with `#`): No prediction

This provides instant feedback for common commands while enabling intelligent suggestions for novel situations, reducing LLM API costs and improving offline experience.

---

## Completion System Design

### Completion Sources

The completion system checks sources in this order:

1. **Agent Commands** - Completions for `#/clear`, `#/agents`, `#/agent <name>`
2. **Special Prefixes** - Completions for `#/` and `#!` prefixes (agent mode)
3. **Command Position** - First word (command name) completions
   - Built-in shell commands
   - External commands from PATH
4. **Argument Position** - Subsequent word completions
   - Bash completion specs (via `compgen`)
   - File/directory completion
   - Previous argument history
5. **Macro Expansion** - Macro completions if applicable

### Completion Provider Architecture

The `Provider` is initialized with a `RunnerProvider` (typically the REPL executor) and an optional `AgentProvider` for agent-related completions:

```go
type Provider struct {
    specRegistry     *SpecRegistry              // Bash completion specs
    runnerProvider   RunnerProvider             // Access to bash runner/pwd
    macroCompleter   *completers.MacroCompleter
    builtinCompleter *completers.BuiltinCompleter
    commandCompleter *completers.CommandCompleter
    agentProvider    AgentProvider              // For agent name/command completions
}

// GetCompletions returns completion suggestions for the current input line
func (p *Provider) GetCompletions(line string, pos int) []string
```

### Spec Registry

Manages bash completion specifications:

- Load from system completion directories
- Parse bash completion functions
- Translate to internal spec format
- Cache for performance

---

## Key Technical Decisions

### 1. Configuration File Separation

**Decision:** Separate `.gshrc` (bash) and `.gshrc.gsh` (gsh-specific).

**Alternatives Considered:**

- Single file with special gsh syntax blocks
- Environment variable-based configuration

**Why This Approach:**

- Clear separation of concerns
- Easy migration from bash/zsh (copy `.bashrc` → `.gshrc`)
- No ambiguity about which features work where
- Leverages full power of gsh scripting language

### 2. Input Component Merger

**Decision:** Merge `gline` and `shellinput` into single component.

**Alternatives Considered:**

- Keep nested model structure
- Create adapter layer

**Why This Approach:**

- Eliminates unnecessary indirection
- Simplifies key handling logic
- Better performance (one update pass instead of two)
- Easier to maintain and test

### 3. Direct Provider Access for Agents

**Decision:** REPL calls model providers directly, not via script engine adapter.

**Alternatives Considered:**

- Wrap script engine's agent execution
- Create adapter layer

**Why This Approach:**

- Enables streaming in the future
- Simpler architecture
- Better control over conversation state
- Script engine still used for configuration

### 4. Immutable Default Agent

**Decision:** Built-in default agent cannot be overridden in config.

**Alternatives Considered:**

- Allow `GSH_CONFIG.defaultAgent` to override
- No default agent at all

**Why This Approach:**

- Consistent out-of-box experience
- Prevents misconfiguration
- Users can still define custom agents
- Clear distinction between default and custom

### 5. History-First Prediction

**Decision:** Check command history before calling LLM predictor.

**Alternatives Considered:**

- Always use LLM
- Hybrid scoring (combine history and LLM)

**Why This Approach:**

- Instant feedback for common commands
- Reduces LLM API costs
- Better offline experience
- LLM still available for novel situations

---

## Migration Path

### From Bash/Zsh

```bash
# 1. Copy existing config
cp ~/.bashrc ~/.gshrc

# 2. Optionally create gsh-specific config
cat > ~/.gshrc.gsh << 'EOF'
GSH_CONFIG = {
    prompt: "gsh> ",
}
EOF
```

### Prompt Customization

**Old (not supported):**

```bash
# .gshrc
export GSH_PROMPT="$ "
GSH_UPDATE_PROMPT() {
    echo "$ "
}
```

**New:**

```gsh
# .gshrc.gsh
GSH_CONFIG = {
    prompt: "$ ",
}

tool GSH_UPDATE_PROMPT(exitCode: number, durationMs: number): string {
    return "$ "
}
```

---

## Performance Considerations

1. **Prediction Debouncing**: LLM calls debounced to avoid API spam
2. **Completion Caching**: Bash completion specs cached in memory
3. **Context Throttling**: Context retrieval throttled during rapid typing
4. **History Indexing**: Command history indexed for fast prefix matching
5. **Lazy Agent Loading**: Custom agents only initialized when first accessed

---

## Testing Strategy

### Unit Tests

- Component isolation with mocked dependencies
- Configuration loading edge cases
- Input buffer operations
- Prediction routing logic
- Agent switching scenarios

### Integration Tests

- End-to-end REPL with mocked terminal
- Configuration file loading
- Command execution
- Agent conversations with mocked LLM
- Completion system integration

### Compatibility Tests

- Bash `.gshrc` files work identically to `.bashrc`
- GSH-specific features require `.gshrc.gsh`
- Combined `.gshrc` + `.gshrc.gsh` configurations
- Migration from existing setups

---

## Future Enhancements

1. **Streaming Agent Responses**: Leverage direct provider access for streaming
2. **Custom Completion Providers**: Plugin system for completion sources
3. **Advanced Context Retrieval**: Semantic search over command history
4. **Multi-line Editing**: Support for complex scripts in REPL
5. **Session Persistence**: Save/restore agent conversations across sessions
