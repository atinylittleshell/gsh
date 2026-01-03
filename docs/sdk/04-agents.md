# Agents

This chapter documents how to define and use custom agents.

**Availability:** REPL + Script

## Agent Declaration

Define agents using the `agent` keyword:

```gsh
agent myAgent {
    model: gsh.models.workhorse,
    systemPrompt: "You are a helpful assistant.",
    tools: [gsh.tools.exec, gsh.tools.grep, gsh.tools.view_file],
}
```

### Required Fields

| Field          | Type            | Description                                     |
| -------------- | --------------- | ----------------------------------------------- |
| `model`        | model reference | A model object or `gsh.models.*` tier reference |
| `systemPrompt` | `string`        | Defines the agent's behavior and personality    |
| `tools`        | `array`         | Array of tools the agent can use                |

## Using Agents

### Pipe Expressions

The primary way to interact with agents is through pipe expressions:

```gsh
# Simple query
result = "What files are in the current directory?" | myAgent

# Chain with context
result = "Explain this code" | myAgent
followup = "Now refactor it for readability" | result
```

### Conversations

Pipe expressions return conversation objects that maintain context:

```gsh
conv = "What is 2 + 2?" | myAgent
# conv now contains the conversation history

# Continue the conversation
conv = "Multiply that by 3" | conv
```

## Model Resolution

When using model tier references (`gsh.models.lite`, `gsh.models.workhorse`, `gsh.models.premium`), the model is resolved dynamically at runtime:

```gsh
agent myAgent {
    model: gsh.models.workhorse,  # Resolved when agent is used
    systemPrompt: "You are helpful.",
    tools: [],
}

# Change the workhorse model
gsh.models.workhorse = differentModel

# Now myAgent uses differentModel
result = "Hello" | myAgent
```

Direct model assignments always use the specified model:

```gsh
agent fixedAgent {
    model: specificModel,  # Always uses this exact model
    systemPrompt: "You are helpful.",
    tools: [],
}
```

## Example Agents

### Code Reviewer

```gsh
agent codeReviewer {
    model: gsh.models.workhorse,
    systemPrompt: `You are an expert code reviewer. When reviewing code:
- Check for bugs and potential issues
- Suggest improvements for readability
- Point out security concerns
- Be constructive and specific`,
    tools: [gsh.tools.grep, gsh.tools.view_file],
}

# Usage
review = "Review the main.go file" | codeReviewer
```

### Shell Expert

```gsh
agent shellExpert {
    model: gsh.models.workhorse,
    systemPrompt: `You are a shell scripting expert. Help users with:
- Writing shell commands
- Debugging shell scripts
- Explaining command output
Execute commands to verify your suggestions work.`,
    tools: [gsh.tools.exec],
}

# Usage
help = "How do I find all .py files modified in the last week?" | shellExpert
```

### File Organizer

```gsh
agent fileOrganizer {
    model: gsh.models.workhorse,
    systemPrompt: `You help organize files and directories. You can:
- Search for files matching patterns
- View file contents to understand them
- Execute commands to move/rename files
Always confirm before making changes.`,
    tools: [gsh.tools.exec, gsh.tools.grep, gsh.tools.view_file],
}
```

## Default Agent

In REPL mode, the `#` prefix invokes the default agent. You can customize this by modifying the default middleware in your `~/.gsh/repl.gsh`. See the default configuration in `cmd/gsh/defaults/middleware/agent.gsh` for reference.

## Agent Events

When agents run, they emit events that you can hook into for customization:

- `agent.start` - Agent begins processing
- `agent.iteration.start` - Each reasoning iteration starts
- `agent.chunk` - Text chunk received (streaming)
- `agent.tool.pending` - Tool call streaming (args incomplete)
- `agent.tool.start` - Tool execution begins
- `agent.tool.end` - Tool execution completes
- `agent.end` - Agent finishes

See [Events](05-events.md) for details on handling these events.

## ACP Agents (External Agents)

In addition to gsh agents defined in scripts, you can delegate to external agents via the **Agent Client Protocol (ACP)**. ACP agents are powerful standalone AI systems like Rovo Dev that run as separate processes.

### Declaration

```gsh
acp RovoDev {
    command: "acli",
    args: ["rovodev", "acp"],
    cwd: "/path/to/project",  # Optional
    env: {                     # Optional
        API_KEY: env.MY_KEY,
    },
}
```

### Usage

```gsh
# Start a session
session = "Analyze this codebase" | RovoDev

# Continue the conversation
session = session | "Focus on error handling"
session = session | "Show me specific improvements"

# Access the response
print(session.lastMessage.content)

# Clean up
session.close()
```

### Key Differences from gsh Agents

| Aspect            | gsh Agent           | ACP Agent                |
| ----------------- | ------------------- | ------------------------ |
| Type              | `agent` declaration | `acp` declaration        |
| Result            | `Conversation`      | `ACPSession`             |
| History           | gsh owns it         | Agent owns it            |
| Handoffs          | ✅ Can hand off     | ❌ Cannot hand off       |
| `Value \| String` | Adds message only   | Sends prompt immediately |

### ACPSession Properties

| Property      | Type      | Description                 |
| ------------- | --------- | --------------------------- |
| `messages`    | array     | All messages in the session |
| `lastMessage` | object    | The most recent message     |
| `agent`       | ACP value | The bound agent             |
| `sessionId`   | string    | Unique session identifier   |
| `closed`      | boolean   | Whether session is closed   |

### Events

ACP agents emit the same events as gsh agents:

- `agent.start`, `agent.end`
- `agent.chunk`
- `agent.tool.pending`, `agent.tool.start`, `agent.tool.end`

The `ctx.type` field distinguishes them: `"acp"` vs `"gsh"`.

For comprehensive documentation, see **[Chapter 23: ACP Agents](../script/23-acp-agents.md)** in the Scripting Guide.

---

**Next:** [Events](05-events.md) - Event system for customization
