# Chapter 05: Agents in the REPL

While command prediction helps you type faster, agents let you delegate entire tasks to AI. In this chapter, you'll learn how to configure and use agents directly in your shell.

## What is an Agent?

An **agent** is an AI-powered assistant that:

- Understands your intent
- Can reason about problems
- Has access to tools (files, commands, APIs)
- Can take multiple steps to solve problems
- Learns from context (history, git status, etc.)

Agents are more powerful than predictions because they can:

```bash
gsh> # analyze this error log and suggest fixes
# Agent reads the file, understands the error, and provides solutions

gsh> # write a script to process CSV files in parallel
# Agent generates code, potentially asking clarifying questions
```

## Setting Up Your First Agent

### Step 1: Choose an Agent Model

For agents, you want a **capable, reasoning model**:

- **Ollama models**:

  - `devstral-small-2` - Good reasoning, relatively fast
  - `neural-chat` - Conversational and helpful
  - `mistral` - Powerful and reasonably fast

- **OpenAI models**:

  - `gpt-4o-mini` - Excellent reasoning, fast
  - `gpt-4o` - Most capable (but slower)

- **Anthropic models**:
  - `claude-opus-4-mini` - Exceptional reasoning
  - `claude-opus-4` - Most capable (but expensive)

### Step 2: Configure in `.gshrc.gsh`

Create or update `~/.gshrc.gsh`:

```gsh
# ~/.gshrc.gsh

# Model for agent interactions (more capable than prediction)
model AgentModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

# Configure the default agent
GSH_CONFIG = {
    defaultAgentModel: AgentModel,
}
```

### Step 3: Use It

Start a new gsh session:

```bash
gsh
```

Invoke the agent with `#` followed by your request:

```bash
gsh> # what's in my current directory and what should I do next?
# Agent provides helpful analysis
```

## Agent Basics

### Starting an Agent Conversation

Use `#` followed by your request:

```bash
gsh> # explain what this command does: grep -r "TODO" --include="*.js"
```

The agent will:

1. Understand the command
2. Consider your context (what directory you're in, etc.)
3. Provide an explanation

### Multi-Turn Conversations

Agents remember the conversation history:

```bash
gsh> # what are the top 3 most common files in this directory?
Agent: The top 3 most common file types are...

gsh> # show me how to process these files in a script
Agent: Here's a script that processes them...

gsh> # can you make it concurrent?
Agent: Certainly! Here's a version that uses...
```

Each response builds on the previous context.

### Canceling Agent Output

If the agent is taking too long:

- **Ctrl+C** - Stop the agent
- The conversation ends and you return to the prompt

## Practical Agent Use Cases

### 1. Code Review and Explanation

```bash
gsh> # review this script for bugs and performance issues
gsh> cat myscript.sh | # review it
```

### 2. Debugging

```bash
gsh> # I'm getting this error: "connection refused on port 8080". What should I check?
```

The agent might suggest:

- Is the service running?
- Is the port in use by something else?
- Try `lsof -i :8080` to diagnose

### 3. Learning

```bash
gsh> # teach me about jq filters, starting simple
Agent: jq is a powerful JSON processor. Let me show you...

gsh> # show me a more complex example
Agent: Now here's something more advanced...
```

### 4. Task Automation

```bash
gsh> # help me write a script to backup my database daily
Agent: Here's a backup script and instructions to schedule it...

gsh> # now make it send alerts if backup fails
Agent: I'll add error handling and email notifications...
```

### 5. Problem Solving

```bash
gsh> # I need to find all node_modules folders and delete them to save space. How do I do this safely?
Agent: Here's a safe approach...

gsh> # what if I want to see which ones are the largest first?
Agent: Good idea! Here's how to sort them...
```

## Configuring Custom Agents

### Basic Agent Declaration

In `~/.gshrc.gsh`, define an agent with specific instructions:

```gsh
# Define a model
model SmartAssistant {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-4o-mini",
}

# Define an agent using that model
agent CodeReviewer {
    model: SmartAssistant,
    system: """
        You are a code review expert. When reviewing code:
        1. Look for bugs and potential issues
        2. Check for performance problems
        3. Verify security implications
        4. Suggest improvements
        5. Explain your reasoning
    """,
}
```

### Using a Custom Agent

Once defined, invoke it with `# /agent <name>` to switch to it, then use `#`:

```bash
gsh> # /agent CodeReviewer
gsh> # review this Python script
gsh> cat app.py | # review this
```

### Multiple Agents with Different Expertise

```gsh
# ~/.gshrc.gsh

model Claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-opus-4-mini",
}

agent DevOpsExpert {
    model: Claude,
    system: """
        You are a DevOps and infrastructure expert.
        Help with Kubernetes, Docker, CI/CD, infrastructure as code,
        monitoring, and deployment strategies.
    """,
}

agent SecurityReviewer {
    model: Claude,
    system: """
        You are a security expert focused on application security.
        Review code for vulnerabilities, suggest secure practices,
        and help with authentication and encryption.
    """,
}

agent DataAnalyst {
    model: Claude,
    system: """
        You are a data analyst expert.
        Help with SQL queries, data transformation, visualization,
        and statistical analysis.
    """,
}
```

Now you can ask different experts:

```bash
gsh> # /agent DevOpsExpert
gsh> # how do I set up a Kubernetes cluster?
gsh> # /agent SecurityReviewer
gsh> # review this authentication code
gsh> # /agent DataAnalyst
gsh> # write a query to find monthly trends
```

## Agent Context and Tools

### What Context Do Agents See?

Agents have access to:

- **Command history** - What you've done before
- **Current directory** - Where you are now
- **Git status** - If you're in a repo
- **Previous messages** - In the conversation
- **Environment** - Available via `env.*`

### Agents with Tools (Advanced)

Agents can use tools to perform actions. This is covered in the [Script Documentation](../script/18-agent-declarations.md), but here's a quick example:

```gsh
# Define MCP tools
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem", env.HOME],
}

# Define an agent with filesystem access
agent FileAssistant {
    model: AgentModel,
    system: "You are a file management assistant with access to the filesystem.",
    tools: [filesystem.read_file, filesystem.write_file, filesystem.list_directory],
}
```

Then:

```bash
gsh> @FileAssistant look at my project structure and summarize it
# Agent explores files and provides summary
```

## Understanding Agent Output

When you interact with an agent, gsh displays structured output to help you understand what's happening. Here's what you'll see:

### Agent Header and Footer

Each agent response is wrapped with a header and footer:

```
gsh> # what files are in this directory?

── gsh ────────────────────────────────────────────────────────────────────────
Let me check that for you.

The current directory contains:
- README.md - Project documentation
- src/ - Source code directory
- package.json - Node.js configuration

── 523 in · 324 out · 1.2s ────────────────────────────────────────────────────

gsh>
```

- **Header**: Shows the agent name (or "gsh" for the default agent)
- **Footer**: Shows token usage (input/output) and response time for your last command

### Tool Call Display

When agents use tools, you'll see their progress:

**Shell commands (exec tool):**

```
▶ ls -la
total 24
-rw-r--r--  1 user  staff  1234 file.txt
drwxr-xr-x  2 user  staff    64 src
✓ ls (0.1s)
```

The `▶` symbol indicates a shell command is running. Output streams directly, then shows a success (`✓`) or failure (`✗`) indicator with duration.

**Other tools:**

```
○ read_file
   path: "/home/user/config.json"
```

While executing, you'll see the `○` symbol with a spinner. When complete:

```
● read_file ✓ (0.02s)
   path: "/home/user/config.json"
```

The `●` symbol indicates completion, with `✓` for success or `✗` for failure.

### Status Symbols Reference

| Symbol | Meaning                |
| ------ | ---------------------- |
| `▶`   | Shell command starting |
| `○`    | Tool pending/executing |
| `●`    | Tool complete          |
| `✓`    | Success                |
| `✗`    | Error/failure          |

### Thinking Indicator

While waiting for the LLM to respond, you'll see a spinning indicator:

```
── gsh ────────────────────────────────────────────────────────────────────────
⠋ Thinking...
```

This animates to show the agent is processing your request.

## Best Practices for Agent Use

### 1. Be Specific

Instead of:

```bash
gsh> @agent help me with this
```

Try:

```bash
gsh> @agent I need to convert all JPEG files in ~/images to PNG format. Show me the command.
```

### 2. Ask Step by Step

Instead of:

```bash
gsh> @agent teach me Kubernetes
```

Try:

```bash
gsh> @agent what are the basic concepts of Kubernetes?
gsh> @agent now explain pods and containers
gsh> @agent show me how to create a pod definition
```

### 3. Provide Context

Include relevant information:

```bash
gsh> @agent I'm using Node.js 18.x and want to add TypeScript to my project. What's the best approach?
```

### 4. Verify Output

Always review agent suggestions before running them:

```bash
gsh> @agent write a script to delete old log files
# Review the script before running
gsh> bash delete_logs.sh
```

### 5. Use the Right Agent

If you defined specialized agents, use the appropriate one:

```bash
gsh> @SecurityReviewer review this Node.js API
# Better than @agent for security-specific concerns
```

## Troubleshooting

### Agent Not Responding

1. Check the model is configured:

   ```gsh
   print(GSH_CONFIG)
   ```

2. Verify the model is reachable:

   ```bash
   # For Ollama
   ollama list
   ollama serve  # Make sure it's running

   # For OpenAI
   curl -H "Authorization: Bearer $OPENAI_API_KEY" \
     https://api.openai.com/v1/models
   ```

3. Check logs:
   ```bash
   tail -f ~/.gsh.log
   ```

### Agent Responses Are Poor Quality

1. Try a more capable model:

   ```gsh
   model: "gpt-4o",  # Instead of gpt-4o-mini
   ```

2. Improve your system prompt:

   ```gsh
   agent MyAgent {
       model: BetterModel,
       system: "Be more specific about your instructions...",
   }
   ```

3. Provide more context in your request:
   ```bash
   gsh> @agent I'm in /home/user/projects/python/ml-project. I need to...
   ```

### Agent Takes Too Long

1. Switch to a faster model:

   ```gsh
   model: "gpt-4o-mini",  # Faster than gpt-4o
   ```

2. Use local Ollama if available
3. Ask simpler questions

### API Rate Limits

If you hit rate limits:

1. Wait and try again
2. Use local Ollama to avoid limits
3. Upgrade your API plan
4. Implement response caching

## Privacy and Security

### Local Agents

Using local Ollama models:

- All conversations stay on your machine
- No external API calls
- Free to use

### Cloud Agents

Using OpenAI, Anthropic, etc.:

- Conversations are sent to their servers
- Check their privacy policies
- Consider what you share (code, data, etc.)

## Advanced: Chaining Agents

Agents can work together:

```bash
gsh> @DataAnalyst write a query to get monthly revenue
# Agent generates query

gsh> @DevOpsExpert how should I schedule this query to run daily?
# Different agent helps with scheduling
```

This manual coordination is powerful but can also be automated in scripts (see [Chapter 19: Conversations and Pipes](../script/19-conversations-and-pipes.md)).

## What's Next?

You now know how to use AI agents interactively! Chapter 06 covers **Executing gsh Scripts**—how to write and run `.gsh` files that combine everything you've learned into powerful automation workflows.

---

**Previous Chapter:** [Chapter 04: Command Prediction with LLMs](04-command-prediction.md)

**Next Chapter:** [Chapter 06: Executing gsh Scripts](06-executing-gsh-scripts.md)
