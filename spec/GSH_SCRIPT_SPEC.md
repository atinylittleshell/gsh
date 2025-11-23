# gsh Scripting Language Specification

**Date:** November 2025

---

## 1. Introduction & Motivation

### The Problem Space

Shell users need a way to write scripts that can:

- Leverage AI agents for intelligent task execution
- Connect to external tools and services via MCP (Model Context Protocol)
- Provide type safety and clear error messages
- Work seamlessly in an interactive shell

### Introducing gsh Scripting

gsh includes a **built-in scripting language** designed specifically for agentic workflows.

**Key Design Decisions:**

- **The gsh REPL stays POSIX-compatible** - Interactive gsh remains a posix-compatible shell
- **Scripts use gsh language** - `.gsh` files use the scripting language for power features
- **Native Go interpreter** - No external dependencies (Node.js, Python, etc.)

### Quick Example

```gsh
#!/usr/bin/env gsh

# hello.gsh - A simple gsh script
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

message = "Hello from gsh!"
print(message)

# Write to a file using MCP tool
filesystem.write_file("greeting.txt", message)
print("Greeting saved!")
```

Run with: `gsh hello.gsh`

---

## 2. Fundamental Syntax Elements

### Basic Types

The gsh language uses TypeScript-like type syntax:

```gsh
name: string = "Alice"
age: number = 30
score: number = 98.6
isActive: boolean = true
data: any = {"key": "value"}  // For JSON-like dynamic data
```

### Collections

```gsh
# Arrays - literal syntax with square brackets
items: string[] = ["apple", "banana", "orange"]
numbers = [1, 2, 3, 4, 5]

# Objects - literal syntax for structured data (like JSON)
config = {
    host: "localhost",
    port: 8080,
    enabled: true,
}
user = {name: "Alice", email: "alice@example.com"}

# Sets - unique values
uniqueIds = Set([1, 2, 3, 2, 1])  # {1, 2, 3}
tags = Set(["javascript", "python", "go"])

# Maps - key-value pairs
userAges = Map([["alice", 25], ["bob", 30]])
config = Map([["host", "localhost"], ["port", 8080]])
```

### Variable Declarations

Variables are declared using assignment syntax and are always mutable:

```gsh
name = "Alice"     # type inferred as string
count = 0          # type inferred as number
count = count + 1  # reassignment (all variables are mutable)

# Explicit type annotation (optional)
port: number = 8080
```

### Control Flow

```gsh
# Conditionals (parentheses required)
if (condition) {
    # execute
} else if (otherCondition) {
    # execute
} else {
    # execute
}

# Loops
for (item of collection) {
    # process item
}

while (condition) {
    # execute
}

# Break and continue
for (item of items) {
    if (item == "skip") {
        continue
    }
    if (item == "stop") {
        break
    }
    print(item)
}
```

### Tool Declarations

Tools are the fundamental unit of composition in gsh scripts:

```gsh
tool processData(input) {
    content = filesystem.read_file(input)
    return JSON.parse(content)
}

# With type annotations (validated at runtime)
tool calculateScore(points: number, multiplier: number): number {
    return points * multiplier
}
```

### String Literals

```gsh
message = "Hello, world!"
path = '/home/user/file.txt'

# Template literals with interpolation
greeting = `Hello, ${name}!`
path = `/home/${username}/documents`

# Multi-line strings for prompts
systemPrompt = """
    You are a data analyst. Analyze the provided data
    and generate insights using the available tools.

    Always provide clear explanations.
    """
```

Triple-quoted strings automatically remove common leading whitespace on each line.

---

## 3. MCP Server Integration

### Server Declaration

MCP servers are declared using the `mcp` keyword:

```gsh
# Local process-based MCP server
mcp filesystem {
  command: "npx",
  args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/project"],
  env: {
    HOME: env.HOME,
  },
}

# With environment variables
mcp github {
  command: "npx",
  args: ["-y", "@modelcontextprotocol/server-github"],
  env: {
    GITHUB_TOKEN: env.GITHUB_TOKEN,
  },
}

# Remote HTTP/SSE server
mcp database {
  url: "http://localhost:3000/mcp",
  headers: {
    Authorization: `Bearer ${env.DB_API_KEY}`,
  },
}
```

### Tool Invocation

Once an MCP server is declared, its tools are available with dot notation:

```gsh
# Call MCP tools
content = filesystem.read_file("data.json")
files = filesystem.list_directory("/home/user")

# With parameters
github.create_issue("myorg/myrepo", {
    title: "Bug Report",
    body: "Description of the issue",
})
```

### Environment Variables

Environment variables are accessed through the `env` object:

```gsh
# Access environment variables
token = env.GITHUB_TOKEN
port = env.PORT ?? 3000  # Default if not set

# In string interpolation
message = `Token is: ${env.GITHUB_TOKEN}`
path = `/home/${env.USER}_backup`
```

---

## 4. Agent Configuration

### Model Declaration

Models define the LLM connection configuration:

```gsh
# Anthropic Claude
model claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-3-5-sonnet-20241022",
    temperature: 0.7,
}

# OpenAI GPT
model gpt4 {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-4",
    temperature: 0.5,
}

# Local model via Ollama
model llama {
    provider: "ollama",
    url: "http://localhost:11434",
    model: "llama3.2:3b",
}
```

### Agent Declaration

Agents combine a model with tools and a system prompt:

```gsh
# Define custom tools
tool analyzeData(data: string): string {
    parsed = JSON.parse(data)
    return `Found ${parsed.length} records`
}

tool formatReport(content: string): string {
    return `# Report\n\n${content}`
}

# Declare agent
agent DataAnalyst {
    model: claude,

    systemPrompt: """
        You are a data analyst. Analyze the provided data
        and generate insights using the available tools.
    """,

    tools: [
        filesystem.read_file,
        filesystem.write_file,
        analyzeData,
        formatReport,
    ],

    temperature: 0.5,  # Override model default
}
```

### Agent Invocation

Agents are invoked using the **pipe operator** (`|`):

```gsh
# Basic invocation
result = "Analyze customer trends for Q3" | DataAnalyst

# Multi-turn conversation
conv = "Here's the data: ${data}" | DataAnalyst
     | "What trends do you see?"
     | DataAnalyst
     | "Be more specific about Q3"
     | DataAnalyst

# Agent handoffs
conv = "Analyze this: ${data}" | DataAnalyst
     | "Write a report based on the analysis"
     | ReportWriter
```

**Pipe Operator Semantics:**

- `String | Agent` → Creates conversation, returns conversation object
- `Conversation | String` → Adds user message, returns conversation
- `Conversation | Agent` → Executes agent with context, returns conversation

---

## 5. Error Handling

### Try-Catch Blocks

```gsh
try {
    content = filesystem.read_file("data.json")
    data = JSON.parse(content)
    print(data)
} catch (error) {
    print(`Error: ${error.message}`)
    # Fallback logic
    data = getDefaultData()
}
```

### Error Propagation

Errors propagate up the call stack until caught:

```gsh
tool processFile(path: string): any {
    # If this fails, error propagates to caller
    content = filesystem.read_file(path)
    return JSON.parse(content)
}

tool safeProcess(path: string): any {
    try {
        return processFile(path)
    } catch (error) {
        log.error(`Failed to process ${path}: ${error.message}`)
        return null
    }
}
```

---

## 6. Built-in Functions

### Logging

```gsh
log.debug("Debug message", {data: value})
log.info("Info message")
log.warn("Warning message")
log.error("Error message")
```

### Output

```gsh
print("Hello, world!")
print(`Value: ${count}`)
```

### JSON Utilities

```gsh
# Parse JSON
data = JSON.parse('{"key": "value"}')

# Stringify
json = JSON.stringify({name: "Alice", age: 30})
```

---

## 7. Execution Model

### Native Go Interpreter

gsh uses a **native Go interpreter** built into the gsh binary:

```
.gsh file → Lexer → Tokens → Parser → AST → Interpreter → Execute
```

**Architecture:**

1. **Lexer** - Tokenizes source code (pure Go)
2. **Parser** - Recursive descent parser (pure Go)
3. **AST** - Abstract syntax tree representation (Go structs)
4. **Interpreter** - Tree-walking interpreter (pure Go)

**No external dependencies:**

- ✅ No Node.js required
- ✅ No Python required
- ✅ Single static binary
- ✅ Cross-platform (works everywhere Go works)

### MCP Integration

gsh uses the official Go MCP SDK:

```go
import "github.com/modelcontextprotocol/go-sdk/mcp"
```

MCP servers run as subprocesses and communicate via stdio/JSON-RPC.

### Agent Integration

Agent execution should be a new implementation, not reusing existing gsh REPL agent code.

```go
# Do NOT reuse:
internal/agent/agent.go
internal/llm/client.go
```

### File Execution

```bash
# Run a .gsh script
gsh script.gsh

# Shebang support
chmod +x script.gsh
./script.gsh
```

---

### Standard Library

Built-in utilities for common tasks:

- HTTP requests
- JSON/CSV parsing
- Date/time manipulation
- String utilities

### IDE Support

- Syntax highlighting
- Language server protocol (LSP)
- Debugging tools

---

## Appendix A: Complete Example

```gsh
#!/usr/bin/env gsh

# PR Analyzer - Automated GitHub PR analysis

# MCP servers
mcp github {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-github"],
    env: {
        GITHUB_TOKEN: env.GITHUB_TOKEN,
    },
}

# Models
model claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-3-5-sonnet-20241022",
}

# Agents
agent PRAnalyzer {
    model: claude,
    systemPrompt: "You analyze pull requests for code quality and issues",
    tools: [github.get_pull_request_diff],
}

agent Summarizer {
    model: claude,
    systemPrompt: "You create concise summaries",
}

# Tool to analyze a single PR
tool analyzePR(repo: string, prNumber: number) {
    log.info(`Analyzing PR #${prNumber}`)

    try {
        # Get PR diff
        diff = github.get_pull_request_diff(repo, prNumber)

        # Analyze with agent
        analysis = `Review this PR:\n${diff}` | PRAnalyzer

        # Summarize
        summary = analysis | "Summarize in 2 sentences" | Summarizer

        # Post comment
        github.create_comment(repo, prNumber, {
            body: `🤖 Automated Analysis:\n\n${summary}`,
        })

        log.info(`✓ Posted analysis for PR #${prNumber}`)
        return true
    } catch (error) {
        log.error(`Failed to analyze PR #${prNumber}: ${error.message}`)
        return false
    }
}

# Main script execution
log.info("Starting PR analysis")

repo = "myorg/myrepo"
prs = github.list_pull_requests(repo, {state: "open"})

count = 0
for (pr of prs) {
    if (analyzePR(repo, pr.number)) {
        count = count + 1
    }
}

print(`Analyzed ${count} pull requests`)
```

Run with: `gsh pr-analyzer.gsh`

---

**End of Specification**
