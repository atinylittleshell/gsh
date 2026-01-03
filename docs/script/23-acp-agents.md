# Chapter 23: ACP Agents (External Agent Integration)

You've learned to create powerful agents in Chapter 18 that combine a model, system prompt, and tools. These "gsh agents" run entirely within gsh, using models you configure and tools you define.

But what about powerful AI agents that exist as standalone services? Rovo Dev, Claude Code, and custom enterprise agents are sophisticated systems with their own capabilities. Rather than reimplementing their features, gsh lets you **delegate tasks** to them using the **Agent Client Protocol (ACP)**.

Think of it this way:

- **Native gsh agents** are like employees you hire and train yourself
- **ACP agents** are like specialized contractors you bring in for specific jobs

In this chapter, you'll learn to:

- Declare and connect to external ACP agents
- Understand the differences between gsh agents and ACP agents
- Build conversations with ACP agents using the pipe operator
- Handle errors and manage sessions

---

## Core Concepts: What is ACP?

### The Agent Client Protocol

[ACP](https://zed.dev/acp) (Agent Client Protocol) is a standardized protocol for client↔agent communication. It defines how to:

- Start an agent subprocess
- Create sessions for conversations
- Send prompts and receive streaming responses
- Track tool calls the agent makes

gsh implements ACP's stdio transport, meaning it spawns ACP agents as subprocesses and communicates via JSON-RPC over stdin/stdout—similar to how MCP servers work.

### Key Differences from gsh Agents

| Aspect     | gsh Agent                    | ACP Agent         |
| ---------- | ---------------------------- | ----------------- |
| Defined in | Your gsh script              | External process  |
| Model      | You configure it             | Agent controls it |
| Tools      | You provide them             | Agent has its own |
| History    | gsh owns it                  | Agent owns it     |
| Handoffs   | Can hand off to other agents | Cannot hand off   |

The most important difference is **session ownership**. When you talk to an ACP agent, the agent maintains the conversation history internally. You can continue conversations with the same ACP agent, but you cannot hand off an ACP session to a different agent.

---

## Declaring ACP Agents

### Basic Declaration

The `acp` keyword declares an external ACP agent:

```gsh
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}
```

This tells gsh how to spawn the Rovo Dev agent when you need it. The agent won't start until you actually use it (lazy initialization).

### With Environment Variables

Pass environment variables to the agent process:

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

Specify where the agent should run:

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
    args: <string[]>,            # Optional: command-line arguments
    env: <object>,               # Optional: environment variables
    cwd: <string>,               # Optional: working directory (defaults to current)
}
```

---

## Using ACP Agents: The Pipe Operator

### Starting a Session

Pipe a string to an ACP agent to create a session and send your first prompt:

```gsh
#!/usr/bin/env gsh

acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}

# Start a session with a prompt
session = "Explain how TCP/IP works" | RovoDev

print(session)
```

**Output:**

```
<acpsession RovoDev with 2 messages>
```

The result is an **ACPSession**, not a Conversation. This is a key distinction we'll explore below.

### Continuing the Conversation

To send follow-up messages, pipe strings to the session:

```gsh
#!/usr/bin/env gsh

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

Notice the difference from gsh agents: with ACP sessions, piping a string **automatically sends the prompt** to the agent. There's no separate "add message" vs "execute agent" step—it's always both.

### Chaining in One Expression

You can chain multiple turns in a single expression:

```gsh
session = "What is Kubernetes?" | RovoDev
        | "How does pod scheduling work?"
        | "Show me a deployment YAML example"
```

Each string in the chain sends a new prompt and waits for the response.

---

## ACPSession vs Conversation: Critical Differences

### Semantic Comparison

| Operation                 | Conversation (gsh agent)        | ACPSession                     |
| ------------------------- | ------------------------------- | ------------------------------ |
| `String \| Agent`         | Create + execute → Conversation | Create + execute → ACPSession  |
| `Value \| String`         | Add message (no execution)      | Send prompt (auto-execute)     |
| `Value \| SameAgent`      | Execute agent                   | ❌ Error: already bound        |
| `Value \| DifferentAgent` | Handoff with context            | ❌ Error: cannot switch agents |

### What Works

```gsh
# ✅ Start a session
session = "Hello" | RovoDev

# ✅ Continue with more prompts
session = session | "Follow up question"
session = session | "Another question"

# ✅ Access session properties
print(session.lastMessage.content)
print(session.messages.length)
```

### What Doesn't Work

```gsh
acp Agent1 { command: "agent1", args: [] }
acp Agent2 { command: "agent2", args: [] }

session = "Hello" | Agent1

# ❌ Cannot pipe to the same agent (redundant)
session = session | Agent1  # Error: already bound to this agent

# ❌ Cannot hand off to a different agent
session = session | Agent2  # Error: cannot pipe ACPSession to different agent

# ❌ Cannot mix with gsh agents
agent LocalAgent { model: myModel, systemPrompt: "...", tools: [] }
session = session | LocalAgent  # Error: ACPSession cannot be handed off
```

The reason for these restrictions: the ACP agent owns the conversation history. gsh cannot inject that history into a different agent.

---

## ACPSession Properties and Methods

### Accessing Messages

ACPSession provides the same property interface as Conversation for consistency:

```gsh
session = "What is 2 + 2?" | RovoDev

# Get the last message
print(session.lastMessage.role)      # "assistant"
print(session.lastMessage.content)   # The agent's response

# Get message count
print(session.messages.length)       # 2 (user + assistant)

# Iterate over messages
for (msg of session.messages) {
    print(`${msg.role}: ${msg.content}`)
}
```

### Session Properties

| Property      | Type      | Description                         |
| ------------- | --------- | ----------------------------------- |
| `messages`    | array     | All messages in the session         |
| `lastMessage` | object    | The most recent message             |
| `agent`       | ACP value | The agent this session is bound to  |
| `sessionId`   | string    | Unique identifier for the session   |
| `closed`      | boolean   | Whether the session has been closed |

### Closing Sessions

When you're done with a session, close it to clean up resources:

```gsh
session = "Hello" | RovoDev

# Do work...

# Close the session
session.close()

# Verify it's closed
print(session.closed)  # true
```

After closing, you cannot send more prompts to the session.

---

## Event Integration

ACP agents emit the same events as gsh agents, so your existing event handlers work seamlessly:

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

# Track tool calls
tool onToolStart(ctx) {
    print(`Tool: ${ctx.toolCall.name}`)
}
gsh.on("agent.tool.start", onToolStart)

# This will stream the response as it's generated
session = "List the files in this directory" | RovoDev
```

### Event Mapping

| ACP Update        | gsh Event            | Context                                 |
| ----------------- | -------------------- | --------------------------------------- |
| Prompt starts     | `agent.start`        | `{ agent: "RovoDev", type: "acp" }`     |
| Message chunk     | `agent.chunk`        | `{ content: "...", agent: "RovoDev" }`  |
| Tool call pending | `agent.tool.pending` | `{ toolCall: {...} }`                   |
| Tool executing    | `agent.tool.start`   | `{ toolCall: {...} }`                   |
| Tool complete     | `agent.tool.end`     | `{ toolCall: {...}, result: {...} }`    |
| Prompt complete   | `agent.end`          | `{ agent: "RovoDev", response: "..." }` |

The `ctx.type` field distinguishes ACP agents (`"acp"`) from gsh agents (`"gsh"`).

---

## Error Handling

### Connection Errors

If the ACP agent executable doesn't exist or fails to start:

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

If the agent fails during communication:

```gsh
try {
    session = "Hello" | RovoDev
} catch (error) {
    print(error.message)  # "ACP error: session creation failed"
}
```

### Closed Session Errors

Sending to a closed session fails:

```gsh
session = "Hello" | RovoDev
session.close()

try {
    session = session | "More questions"  # Error!
} catch (error) {
    print(error.message)  # "Cannot send prompt to closed ACP session"
}
```

---

## When to Use ACP vs gsh Agents

### Use ACP Agents When:

- You have access to a powerful external agent (Rovo Dev, Claude Code, etc.)
- The agent has specialized capabilities you don't want to replicate
- You need the agent's built-in tools and integrations
- Session continuity with the same agent is sufficient

### Use gsh Agents When:

- You need full control over the model and prompts
- You want to hand off conversations between different agents
- You're building custom workflows with your own tools
- You need to inspect or modify conversation history directly

### Combining Both

You can use both in the same script:

```gsh
#!/usr/bin/env gsh

# External agent for complex analysis
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
}

# Native agent for simple tasks
model lite {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gpt-oss:20b",
}

agent Summarizer {
    model: lite,
    systemPrompt: "Summarize text concisely.",
    tools: [],
}

# Use Rovo Dev for complex code analysis
codeSession = "Analyze the error handling in this codebase" | RovoDev

# Use native gsh agent for quick summary
summary = `Summarize this analysis: ${codeSession.lastMessage.content}` | Summarizer

print(summary.lastMessage.content)
```

---

## Key Takeaways

1. **ACP connects to external agents** - Use the `acp` keyword to declare agents that run as separate processes
2. **Sessions are bound to one agent** - Unlike gsh conversations, you cannot hand off ACP sessions between agents
3. **Piping strings auto-executes** - `session | "question"` both adds the message and sends it to the agent
4. **Same events, same handlers** - Your existing event handlers work with both gsh and ACP agents
5. **Close sessions when done** - Use `session.close()` to clean up resources
6. **Use the right tool for the job** - ACP for powerful external agents, gsh agents for control and flexibility

---

## What's Next

You've now learned to integrate with external AI agents via ACP. Combined with gsh agents from Chapter 18, you have a complete toolkit for building sophisticated AI-powered workflows.

For more details on customizing agent behavior in the REPL, see the **[SDK Guide](../sdk/README.md)**.

---

**Previous Chapter:** [Chapter 22: Imports and Modules](22-imports-and-modules.md)

**Next Chapter:** Return to [Scripting Guide Home](README.md)
