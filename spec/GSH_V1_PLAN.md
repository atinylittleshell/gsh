# gsh v1.0 Execution Backlog

Reference: [Language Spec](./GSH_SCRIPT_SPEC.md)

## Overview

Transform gsh into a complete agentic shell with native scripting language support. The v1.0 release introduces a **native scripting language** for agent-native automation, while keeping the REPL fully POSIX bash-compatible.

**Key Deliverables:**

- gsh Scripting Language - New `.gsh` file format with MCP/agent support
- Native Go Interpreter - No external dependencies (Node.js, Python)
- Backward Compatibility - REPL stays bash-compatible, `.gshrc` unchanged

---

## Recommended Directory Structure

```
gsh/
├── cmd/
│   └── gsh/
│       └── main.go                 # Entry point
├── internal/
│   ├── script/                     # gsh scripting language implementation
│   │   ├── lexer/
│   │   │   ├── lexer.go           # Tokenizer
│   │   │   ├── lexer_test.go
│   │   │   ├── token.go           # Token types and definitions
│   │   │   └── token_test.go
│   │   ├── parser/
│   │   │   ├── parser.go          # Main parser entry point
│   │   │   ├── parser_test.go
│   │   │   ├── ast.go             # AST node definitions
│   │   │   ├── ast_test.go
│   │   │   ├── expressions.go     # Expression parsing
│   │   │   ├── expressions_test.go
│   │   │   ├── statements.go      # Statement parsing
│   │   │   ├── statements_test.go
│   │   │   ├── declarations.go    # Declaration parsing (mcp, model, agent, tool)
│   │   │   └── declarations_test.go
│   │   ├── interpreter/
│   │   │   ├── interpreter.go     # Main interpreter entry point
│   │   │   ├── interpreter_test.go
│   │   │   ├── value.go           # Value types (String, Number, etc.)
│   │   │   ├── value_test.go
│   │   │   ├── environment.go     # Scope management
│   │   │   ├── environment_test.go
│   │   │   ├── expressions.go     # Expression evaluation
│   │   │   ├── expressions_test.go
│   │   │   ├── statements.go      # Statement execution
│   │   │   ├── statements_test.go
│   │   │   ├── declarations.go    # Declaration handling
│   │   │   ├── declarations_test.go
│   │   │   ├── builtin.go         # Built-in functions
│   │   │   └── builtin_test.go
│   │   ├── mcp/
│   │   │   ├── manager.go         # MCP server manager
│   │   │   ├── client.go          # MCP client wrapper
│   │   │   └── mcp_test.go
│   │   ├── agent/
│   │   │   ├── agent.go           # Agent implementation
│   │   │   ├── conversation.go    # Conversation state
│   │   │   ├── model.go           # Model declarations
│   │   │   └── agent_test.go
│   │   └── runner/
│   │       ├── runner.go          # Script execution orchestration
│   │       └── runner_test.go
│   ├── repl/                      # New REPL implementation
│   └── ...                         # Other existing packages
├── examples/
│   ├── hello.gsh                   # Basic example
│   ├── mcp-demo.gsh               # MCP integration example
│   ├── agent-pipeline.gsh         # Agent workflow example
│   └── ...                         # More examples
├── spec/
│   ├── GSH_SCRIPT_SPEC.md         # Language specification
│   ├── GSH_V1_PLAN.md             # This document
│   └── ...
└── go.mod
```

**Key Design Principles:**

1. **Separate package for scripting** - All `.gsh` script-related code lives under `internal/script/`
2. **Clear separation of concerns** - Lexer, parser, interpreter, MCP, and agent logic in separate packages
3. **Reuse existing code** - Integration with existing gsh agent/LLM infrastructure
4. **Testability** - Each package has comprehensive tests
5. **Examples directory** - Demonstrable scripts showing all features

---

## Phase 1: Lexer & Tokenizer

**Goal:** Convert source code into tokens

### Tasks

- [x] Define token types (keywords, operators, literals, identifiers)
- [x] Implement lexer/scanner in Go
- [x] Handle whitespace, comments, line tracking
- [x] Support string literals (single, double, triple-quoted)
- [x] Support template literals with interpolation
- [x] Error reporting with line/column information
- [x] 100% unit test coverage for lexer

---

## Phase 2: Parser & AST

**Goal:** Build Abstract Syntax Tree from tokens

### Phase 2.1: Parser Foundation

- [x] Define AST node types (Statement, Expression interfaces)
- [x] Implement recursive descent parser structure
- [x] Parse basic expressions (literals, binary ops, unary ops)
- [x] Parse variable declarations and assignments
- [x] Implement operator precedence correctly

### Phase 2.2: Statements & Control Flow

- [x] Parse if/else statements
- [x] Parse while loops
- [x] Parse for-of loops
- [x] Parse break/continue
- [x] Parse try/catch blocks
- [x] Parse blocks and scoping

### Phase 2.3: Declarations & Advanced Features

- [x] Parse MCP server declarations
- [x] Parse model declarations
- [x] Parse agent declarations
- [x] Parse tool declarations with parameters and types
- [x] Parse pipe expressions (critical for agents)
- [x] Parse member access (e.g., `filesystem.read_file`)
- [x] Parse function calls with arguments

### Phase 2.4: Error Messages

- [x] Sensible and detailed error messages
- [x] Error recovery

---

## Phase 3: Basic Interpreter

**Goal:** Execute parsed AST with core language features

### Phase 3.1: Core Execution

- [x] Implement value types (String, Number, Bool, Null)
- [x] Implement environment/scope management
- [x] Execute variable declarations and assignments
- [x] Evaluate expressions (binary, unary, literals)
- [x] Execute statements (expression statements, blocks)
- [x] Implement control flow (if/else, while, for)

### Phase 3.2: Functions & Error Handling

- [x] Implement tool declarations and tool parameters and return values
- [x] Implement tool calls
- [x] Implement try/catch error handling
- [x] Implement break/continue
- [x] Add built-in functions (print, log._, JSON._)
- [x] Environment variable access (env object)

---

## Phase 4: MCP Integration

**Goal:** Add MCP server and tool support

### Phase 4.1: MCP SDK Integration

- [x] Add Go MCP SDK dependency (use context7 for docs): `github.com/modelcontextprotocol/go-sdk/mcp`
- [x] Implement MCP manager to handle multiple servers
- [x] Start MCP servers as subprocesses (stdio transport)
- [x] Initialize and connect to MCP servers
- [x] List available tools from MCP servers

### Phase 4.2: MCP Tool Execution

- [x] Parse MCP declarations in interpreter
- [x] Make MCP tools available in environment (e.g., `filesystem.read_file`)
- [x] Implement MCP tool call execution
- [x] Handle MCP tool parameters and results
- [x] Error handling for MCP failures
- [x] Resource cleanup (close connections on exit)

### Phase 4.3: Testing

- [x] Start filesystem MCP server
- [x] Call `read_file`, `write_file` tools
- [x] Test with multiple MCP servers
- [x] Error handling for missing servers/tools
- [x] Test with remote MCP servers (HTTP/SSE)

---

## Phase 5: Agent Integration

**Goal:** Add agent declarations and pipe operator

### Phase 5.1: Agent Infrastructure

- [x] Parse model declarations
- [x] Parse agent declarations
- [x] Implement model provider abstraction
- [x] Implement OpenAI provider with ChatCompletion API
- [x] Register agents in interpreter environment

### Phase 5.2: Pipe Operator & Conversations

- [x] Implement Conversation object
- [x] Implement pipe operator evaluation
- [x] Implement conversation state management
- [x] Handle `String | Agent` (create conversation)
- [x] Handle `Conversation | String` (add user message)
- [x] Handle `Conversation | Agent` (execute with context)
- [x] Tool calling from agents (MCP + user-defined tools)

### Phase 5.3: Create E2E Test

- [x] Declare models (use OpenAI provider but through local ollama endpoint)
- [x] Declare agents with tools
- [x] Basic pipe: `"prompt" | Agent`
- [x] Multi-turn: `conv | "message" | Agent`
- [x] Agent handoff: `conv | Agent1 | "message" | Agent2`
- [x] Agents calling user-defined tools

---

## Phase 6: Advanced Features

**Goal:** Polish and complete remaining features

### Phase 6.1: Collections

- [x] Array operations (indexing, methods)
- [x] Object operations (member access, methods)
- [x] String operations
- [x] Map and Set support
- [x] Template literal interpolation

### Phase 6.2: Error Messages

- [x] Add stack traces for runtime errors

---

## Phase 7: Integration & Polish

**Goal:** Integrate with gsh, finalize v1.0

### Phase 7.1: Integration

- [x] Add `.gsh` file execution to gsh CLI
- [x] Shebang support for `.gsh` files
- [ ] Add tests for clear error messages E2E through gsh CLI executing `.gsh` scripts
- [ ] Add inline help in CLI
- [ ] Integrate log.\\* functions with zap logger (currently outputs to stderr)
- [ ] Pass logger context to interpreter for proper log file integration

### Phase 7.2: Polish

- [ ] Write comprehensive documentation
- [ ] Create example scripts (10+ examples)
- [ ] End-to-end testing with real workflows
- [ ] Performance optimization
- [ ] Memory leak detection and fixing
- [ ] Binary size optimization
- [ ] Cross-platform testing (Linux, macOS, Windows)

### Phase 7.3: Release Preparation

- [ ] Update README with gsh scripting features
- [ ] Write migration guide (bash → gsh scripts)
- [ ] Create tutorial/quickstart
- [ ] Update CHANGELOG
- [ ] Tag release: v1.0.0

---

## Post-v1.0 Roadmap

### v1.1 - Type System

- [ ] Parse time type checks
- [ ] Runtime type validation for tool parameters
- [ ] Type checking for tool return values
- [ ] Type annotations for variables and functions
- [ ] Type inference improvements

### v1.2 - Module System

- [ ] Import/export between `.gsh` files
- [ ] Code reuse and libraries
- [ ] Package management (optional)

### v1.3 - Standard Library

- [ ] HTTP client
- [ ] JSON/CSV parsing
- [ ] Date/time utilities
- [ ] File I/O helpers

### v1.4 - Developer Experience

- [ ] Syntax highlighting
- [ ] Language Server Protocol (LSP)
- [ ] Debugger
- [ ] REPL improvements

### v1.5 - Performance

- [ ] Bytecode compilation
- [ ] VM optimization
- [ ] Caching and precompilation

### v2.0 - Advanced Features

- [ ] Async/await explicit syntax (if needed)
- [ ] Pattern matching
- [ ] Generics/parametric types
- [ ] More advanced type system

---

## Success Metrics

### Technical Metrics

- [ ] All unit tests passing (target: 80%+ coverage)
- [ ] All integration tests passing
- [ ] Binary size < 50 MB
- [ ] Script execution time < 2x Python equivalent
- [ ] Startup time < 100ms

### User Metrics

- [ ] 10+ example scripts demonstrating features
- [ ] Documentation complete and clear
- [ ] Zero breaking changes for existing gsh users
- [ ] Can execute real-world automation tasks

### Release Criteria

- [ ] All phases complete
- [ ] All tests passing
- [ ] Documentation complete
- [ ] Examples working
- [ ] Cross-platform builds successful
- [ ] Performance acceptable
- [ ] No known critical bugs
