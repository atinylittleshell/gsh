# gsh ACP (Agent Client Protocol) Integration Specification

**Status:** Draft  
**Date:** January 2026

---

## 1. Overview

This document specifies how gsh integrates with external agents via the [Agent Client Protocol (ACP)](https://agentclientprotocol.com/). ACP is a standardized protocol for client↔agent communication, enabling gsh scripts to delegate tasks to external AI agents.

### Motivation

gsh currently supports native agents that combine a model, system prompt, and tools:

```gsh
agent Assistant {
    model: claude,
    systemPrompt: "You are a coding assistant",
    tools: [filesystem.read_file, exec],
}
```

However, many powerful agents exist as standalone services (Rovo Dev, Claude Code, custom agents). Rather than reimplementing their capabilities, gsh should be able to delegate to them via ACP.

### Design Goals

1. **Familiar syntax** - Mirror existing `mcp` declaration pattern
2. **Clear semantics** - ACP sessions are distinct from local conversations
3. **Streaming support** - Real-time output via existing event system
4. **Stdio transport** - Spawn agents as subprocesses (like MCP servers)

### Non-Goals (for initial implementation)

- HTTP transport (remote agents)
- Passing MCP servers to external agents
- Multi-agent handoffs with shared context
- Custom authentication methods

---

## 2. ACP Protocol Background

### What is ACP?

ACP defines a JSON-RPC protocol over stdio (or HTTP) for client↔agent communication. Key concepts:

- **Client**: The application invoking the agent (gsh)
- **Agent**: An AI system that processes prompts and executes tools
- **Session**: A stateful conversation managed by the agent

### Key Endpoints

| Endpoint                        | Purpose                                   |
| ------------------------------- | ----------------------------------------- |
| `/initialize`                   | Capability negotiation                    |
| `/session/new`                  | Create a new session                      |
| `/session/prompt`               | Send a prompt, receive streaming response |
| `session/update` (notification) | Real-time updates during prompt execution |

### Critical Limitation: Session-Owned History

ACP sessions maintain their own conversation history. The client cannot inject arbitrary message history into a session. This means:

- ✅ Multi-turn conversations with the same agent work
- ❌ Agent handoffs with shared context do NOT work

This limitation fundamentally shapes our design.

---

## 3. Syntax: `acp` Declaration

### Basic Declaration

```gsh
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}
```

### With Environment Variables

```gsh
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
    env: {
        ATLASSIAN_TOKEN: env.ATLASSIAN_TOKEN,
    },
}
```

### With Working Directory

```gsh
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
    cwd: "/path/to/project",
}
```

### Full Syntax

```
acp <name> {
    command: <string>,           # Required: executable path
    args: <string[]>,            # Required: command-line arguments
    env: <object>,               # Optional: environment variables
    cwd: <string>,               # Optional: working directory (defaults to current)
}
```

### Comparison with MCP

The syntax intentionally mirrors MCP declarations:

```gsh
# MCP server (provides tools)
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

# ACP agent (processes prompts)
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}
```

---

## 4. ACPSession: A New Value Type

### Why Not Use Conversation?

gsh's `Conversation` type has these semantics:

```gsh
conv = "Hello" | Agent        # Create conversation, execute agent
conv = conv | "Follow up"     # Add message (no execution)
conv = conv | Agent           # Execute agent with full history
conv = conv | OtherAgent      # Handoff: OtherAgent sees full history
```

ACP cannot support these semantics because:

1. The agent owns the conversation history
2. We cannot pass history when switching agents
3. There's no concept of "add message without executing"

### ACPSession Semantics

`ACPSession` is a new value type with different semantics:

```gsh
session = "Hello" | RovoDev         # Create session, send prompt
session = session | "Follow up"     # Send another prompt (auto-executes)
session = session | "More"          # Continue conversation

# These are ERRORS:
session = session | OtherAgent      # ❌ Cannot pipe to different agent
session = session | RovoDev         # ❌ Redundant - already bound to RovoDev
```

### Semantic Comparison

| Operation                 | Conversation                    | ACPSession                     |
| ------------------------- | ------------------------------- | ------------------------------ |
| `String \| Agent`         | Create + execute → Conversation | Create + execute → ACPSession  |
| `Value \| String`         | Add message (no execution)      | Send prompt (auto-execute)     |
| `Value \| SameAgent`      | Execute agent                   | ❌ Error: already bound        |
| `Value \| DifferentAgent` | Handoff with context            | ❌ Error: cannot switch agents |
| History ownership         | gsh owns it                     | External agent owns it         |

### ACPSession Properties

Both `Conversation` and `ACPSession` expose consistent properties for accessing message history:

```gsh
session = "Hello" | RovoDev

# Get the last message object
print(session.lastMessage.role)      # "assistant"
print(session.lastMessage.content)   # The agent's response text

# Get message count via array length
print(session.messages.length)       # 2 (user message + assistant response)

# Get all messages as an array
for (msg of session.messages) {
    print(`${msg.role}: ${msg.content}`)
}
```

### Message Object Properties

Each message in `messages` or `lastMessage` is an object with:

| Property     | Type   | Description                                      |
| ------------ | ------ | ------------------------------------------------ |
| `role`       | string | `"user"`, `"assistant"`, `"system"`, or `"tool"` |
| `content`    | string | The message text                                 |
| `name`       | string | (Optional) Tool name for tool messages           |
| `toolCallId` | string | (Optional) ID linking tool result to tool call   |
| `toolCalls`  | array  | (Optional) Tool calls requested by assistant     |

### ACPSession Methods

```gsh
session = "Hello" | RovoDev

# Shutdown session (fully cleaned up)
session.close()
```

### Property Consistency with Conversation

`ACPSession` mirrors the same property interface as `Conversation`:

| Property              | Conversation    | ACPSession              |
| --------------------- | --------------- | ----------------------- |
| `messages`            | ✅ Full history | ✅ Locally tracked copy |
| `messages.length`     | ✅              | ✅                      |
| `lastMessage`         | ✅              | ✅                      |
| `lastMessage.role`    | ✅              | ✅                      |
| `lastMessage.content` | ✅              | ✅                      |

This consistency means code that works with gsh agents also works with ACP agents:

```gsh
# Works with both Conversation and ACPSession
tool showResponse(result) {
    print(`Agent said: ${result.lastMessage.content}`)
}

result1 = "Hello" | LocalAgent
showResponse(result1)

result2 = "Hello" | RovoDev
showResponse(result2)
```

---

## 5. Execution Flow

### 5.1 Lazy Initialization

ACP agents are initialized lazily on first use:

```gsh
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}
# Agent subprocess NOT started yet

session = "Hello" | RovoDev
# NOW the subprocess starts:
# 1. Spawn process
# 2. Send /initialize
# 3. Receive capabilities
# 4. Send /session/new
# 5. Send /session/prompt
```

### 5.2 Message Flow

When `session | "message"` executes:

```
gsh                              ACP Agent
 │                                   │
 │──── /session/prompt ─────────────>│
 │     { prompt: [...] }             │
 │                                   │
 │<─── session/update ───────────────│
 │     { type: "message_chunk" }     │
 │                                   │
 │<─── session/update ───────────────│
 │     { type: "tool_call" }         │
 │                                   │
 │<─── session/update ───────────────│
 │     { type: "message_chunk" }     │
 │                                   │
 │<─── Response ─────────────────────│
 │     { complete: true }            │
 │                                   │
```

### 5.3 Process Lifecycle

- **Start**: On first prompt to the agent
- **Keep alive**: Process persists for session lifetime
- **Shutdown**: When gsh script exits (send graceful shutdown signal)

---

## 6. Event Integration

ACP's `session/update` notifications map to gsh's existing event system:

| ACP Update              | gsh Event            | Context                                 |
| ----------------------- | -------------------- | --------------------------------------- |
| Prompt starts           | `agent.start`        | `{ agent: "RovoDev", type: "acp" }`     |
| `message_chunk`         | `agent.chunk`        | `{ content: "...", agent: "RovoDev" }`  |
| `tool_call` (pending)   | `agent.tool.pending` | `{ toolCall: {...} }`                   |
| `tool_call` (executing) | `agent.tool.start`   | `{ toolCall: {...} }`                   |
| `tool_call_result`      | `agent.tool.end`     | `{ toolCall: {...}, result: {...} }`    |
| Complete                | `agent.end`          | `{ agent: "RovoDev", response: "..." }` |

### Event Handler Compatibility

Existing event handlers work with ACP agents:

```gsh
tool onChunk(ctx) {
    write(ctx.content)  # Stream output to terminal
}
gsh.on("agent.chunk", onChunk)

# Works for both gsh agents AND ACP agents
result = "Hello" | LocalAgent
session = "Hello" | RovoDev
```

The `ctx.type` field can distinguish: `"gsh"` vs `"acp"`.

---

## 7. Error Handling

### Connection Errors

```gsh
acp BadAgent {
    command: "nonexistent-binary",
    args: [],
}

try {
    session = "Hello" | BadAgent
} catch (error) {
    print(error.message)  # "Failed to start ACP agent: executable not found"
}
```

### Protocol Errors

```gsh
try {
    session = "Hello" | RovoDev
} catch (error) {
    # ACP protocol errors surface as gsh errors
    print(error.message)  # "ACP error: session creation failed"
}
```

### Invalid Pipe Operations

```gsh
session = "Hello" | RovoDev

try {
    session = session | OtherAgent  # Error!
} catch (error) {
    print(error.message)  # "Cannot pipe ACPSession to a different agent"
}
```

---

## 8. Implementation Plan

### Existing Infrastructure

gsh already has ACP-aligned types in `internal/acp/types.go`:

| Type             | Purpose                                                         |
| ---------------- | --------------------------------------------------------------- |
| `ToolCall`       | Tool call with ID, name, arguments, status, kind                |
| `ToolCallUpdate` | Progress/result update for a tool call                          |
| `ToolCallStatus` | pending, in_progress, completed, failed                         |
| `StopReason`     | end_turn, max_tokens, max_iterations, refusal, cancelled, error |
| `ToolKind`       | read, write, execute, search, other                             |
| `SessionUpdate`  | Real-time update (message_chunk, tool_call, etc.)               |
| `AgentResult`    | Final result with stop reason, content, usage, duration         |
| `TokenUsage`     | Token consumption statistics                                    |

The `AgentCallbacks` struct in `internal/script/interpreter/conversation.go` provides hooks aligned with ACP:

- `OnChunk` - Streaming content chunks
- `OnToolPending` - Tool call starts streaming
- `OnToolCallStart` / `OnToolCallEnd` - Tool execution lifecycle
- `OnComplete` - Agent finished with `AgentResult`

**This infrastructure can be reused for ACP agent execution**, translating ACP protocol messages to these existing callback types.

### Phase 1: Parser & AST (2-3 hours)

1. **Add `acp` keyword** to lexer (`internal/script/lexer/token.go`)
2. **Add `ACPDeclaration` AST node** to parser (`internal/script/parser/ast.go`)
3. **Implement `parseACPDeclaration`** in parser (`internal/script/parser/declarations.go`)
4. **Add tests** for parsing

Files to modify:

- `internal/script/lexer/token.go` - Add `KW_ACP`
- `internal/script/lexer/lexer.go` - Add keyword mapping
- `internal/script/parser/ast.go` - Add `ACPDeclaration` struct
- `internal/script/parser/declarations.go` - Add parsing logic
- `internal/script/parser/declarations_test.go` - Add tests

### Phase 2: Value Types (2-3 hours)

1. **Add `ValueTypeACP`** and `ValueTypeACPSession` to value system
2. **Implement `ACPValue`** struct (stores declaration config)
3. **Implement `ACPSessionValue`** struct (stores session state + messages)
4. **Add `GetProperty`** to `ACPSessionValue` for `messages`/`lastMessage` access
5. **Add pipe operator support** for new types in `evalPipeExpression`

Files to modify:

- `internal/script/interpreter/value.go` - Add new value types
- `internal/script/interpreter/conversation.go` - Add pipe handling cases

### Phase 3: ACP Client (6-8 hours)

1. **Extend `internal/acp/` package** with client implementation
2. **Implement process management** (spawn, stdin/stdout, shutdown) - similar to `internal/script/mcp/manager.go`
3. **Implement JSON-RPC message handling** over stdio
4. **Implement `/initialize` handshake**
5. **Implement `/session/new`**
6. **Implement `/session/prompt` with streaming**
7. **Parse `session/update` notifications** into existing `SessionUpdate` type

New files:

- `internal/acp/client.go` - Main client implementation
- `internal/acp/process.go` - Process management (spawn/shutdown)
- `internal/acp/protocol.go` - JSON-RPC request/response types
- `internal/acp/client_test.go` - Unit tests

### Phase 4: Interpreter Integration (4-5 hours)

1. **Handle `acp` declarations** in interpreter
2. **Implement ACP execution** that translates `SessionUpdate` to `AgentCallbacks`
3. **Implement `ACPSessionValue.PipeString()`** - Sends prompt, updates local messages
4. **Emit events** during ACP execution (reuse existing event infrastructure)
5. **Handle errors** and cleanup

Files to modify:

- `internal/script/interpreter/interpreter.go` - Declaration handling
- `internal/script/interpreter/statements.go` - Statement execution
- `internal/script/interpreter/conversation.go` - Add ACP pipe handling

New file:

- `internal/script/interpreter/acp.go` - ACP-specific execution logic

### Phase 5: Testing (4-6 hours)

1. **Unit tests** for ACP client
2. **Integration tests** with mock ACP agent
3. **End-to-end tests** with real Rovo Dev (`acli rovodev acp`)
4. **Error case tests**

New files:

- `internal/acp/client_test.go`
- `internal/acp/mock_agent_test.go` - Mock ACP agent for testing
- `internal/script/interpreter/acp_test.go`

### Phase 6: Documentation (2-3 hours)

1. **Add `docs/script/XX-acp-agents.md`** chapter
2. **Update `docs/sdk/04-agents.md`** with ACP comparison
3. **Add examples** to docs

---

## 9. Example Usage

### Basic Usage

```gsh
#!/usr/bin/env gsh

# Declare an ACP agent
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}

# Start a session
session = "Write a Python function to calculate fibonacci numbers" | RovoDev

# Continue the conversation
session = session | "Add memoization for better performance"
session = session | "Now add type hints"

# Get the final response
print(session.lastMessage.content)
```

### With Event Handlers

```gsh
#!/usr/bin/env gsh

acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}

# Stream output to terminal
tool onChunk(ctx) {
    write(ctx.content)
}
gsh.on("agent.chunk", onChunk)

# This will stream the response as it's generated
session = "Explain how TCP/IP works" | RovoDev
```

### Error Handling

```gsh
#!/usr/bin/env gsh

acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}

try {
    session = "Hello" | RovoDev
    session = session | "Follow up question"
} catch (error) {
    print(`ACP error: ${error.message}`)
}
```

---

## 10. Future Enhancements

### HTTP Transport

Support connecting to remote ACP agents:

```gsh
acp RemoteAgent {
    url: "https://agent.example.com",
    headers: {
        Authorization: `Bearer ${env.API_KEY}`,
    },
}
```

### MCP Server Forwarding

Pass MCP servers to external agents:

```gsh
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
    mcpServers: [filesystem],  # Agent can use these tools
}
```

### Context Bridging

Experimental: Pass conversation context as text prefix:

```gsh
conv = "Analyze this data" | LocalAnalyst

# Bridge context to ACP agent (as text summary)
session = conv.summary() | RovoDev
```

---

## 11. Open Questions

### Q1: Should `session | SameAgent` be an error or no-op?

**Options:**

- A) Error: "ACPSession is already bound to agent 'RovoDev'"
- B) No-op: Silently ignored (redundant but harmless)

**Recommendation:** Option A (error) - makes intent clear, catches mistakes.

### Q2: What should printing an ACPSession show?

**Options:**

- A) `<acp session with RovoDev (5 turns)>`
- B) `<acp session: RovoDev>`
- C) The full conversation history (if we track it locally)

**Recommendation:** Option A - informative without being verbose.

### Q3: Should we track message history locally?

Even though the agent owns history, should gsh maintain a local copy for debugging/display?

**Decision:** Yes, track locally. This enables the `messages` and `lastMessage` properties to work consistently with `Conversation`. See "Property Consistency with Conversation" section above.

### Q4: Graceful shutdown on script exit?

Should gsh send a shutdown signal to ACP processes when the script exits?

**Recommendation:** Yes, send SIGTERM and wait briefly for clean shutdown.

---

## 12. References

- [Agent Client Protocol Specification](https://agentclientprotocol.com/)
- [ACP GitHub Repository](https://github.com/zed-industries/agent-client-protocol)
- [gsh MCP Implementation](../internal/script/mcp/) - Similar pattern for process management
- [gsh Agent Implementation](../internal/script/interpreter/agent.go) - gsh agent execution
