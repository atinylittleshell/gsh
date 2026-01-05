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

### Statement Separation

Statements must be separated by **newlines only**. Semicolons are **not** used as statement terminators in gsh.

```gsh
# ✓ Valid: statements on separate lines
x = 5
y = 10
z = x + y

# ✗ Invalid: multiple statements on same line
x = 5 y = 10  # Error!

# ✗ Invalid: semicolons are not statement separators
x = 5; y = 10  # Error!
```

This design choice makes gsh scripts more readable and closer to Python's philosophy of "one statement per line."

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

# Early exit with return (no value needed)
tool validateInput(data) {
    if (data == null) {
        print("Error: data is required")
        return  # Returns null, exits early
    }
    # ... process data
}
```

### Return Statement

The `return` statement exits a tool and optionally returns a value:

```gsh
# Return with a value
tool add(a, b) {
    return a + b
}

# Return without a value (returns null)
tool logAndExit(message) {
    print(message)
    return  # Equivalent to: return null
}

# Implicit return - last expression value is returned if no explicit return
tool double(x) {
    x * 2  # This value is returned
}
```

**Key behaviors:**

- `return value` - Returns the specified value
- `return` - Returns `null` (useful for early exit)
- No return statement - Returns the value of the last expression in the tool body

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

# Set environment variables
env.MY_VAR = "some value"
env.PORT = 8080
env.DEBUG = true

# Unset environment variables
env.MY_VAR = null

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

# Local model via Ollama (uses OpenAI-compatible API)
model llama {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "llama3.2:3b",
}

# Model with custom headers (useful for proxies, auth, etc.)
model customModel {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-4",
    baseURL: "https://my-proxy.example.com/v1",
    headers: {
        "X-Custom-Header": "custom-value",
        "X-Auth-Token": env.CUSTOM_AUTH_TOKEN,
    },
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

    # Optional metadata for custom behavior in scripts
    metadata: {
        category: "analysis",
        priority: 1,
        tags: ["data", "reports"],
    },
}
```

### Agent Metadata

The optional `metadata` field allows you to attach arbitrary key-value pairs to an agent. This metadata can be accessed later in scripts to customize behavior:

```gsh
agent QuickHelper {
    model: gpt4,
    systemPrompt: "You are a quick helper",
    metadata: {
        timeout: 30000,
        retryCount: 3,
        features: ["fast", "concise"],
    },
}

# Access metadata properties on the agent object
print(QuickHelper.name)                    # "QuickHelper"
print(QuickHelper.metadata.timeout)        # 30000
print(QuickHelper.metadata.features)       # ["fast", "concise"]

# Use metadata in conditional logic
if (QuickHelper.metadata.timeout < 60000) {
    print("This is a fast agent")
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

### Try-Catch-Finally Blocks

```gsh
try {
    content = filesystem.read_file("data.json")
    data = JSON.parse(content)
    print(data)
} catch (error) {
    print(`Error: ${error.message}`)
    # Fallback logic
    data = getDefaultData()
} finally {
    # Always executed, regardless of success or error
    log.info("File processing completed")
}
```

The `catch` and `finally` blocks are both optional, but at least one must be present:

```gsh
# Valid: try with only catch
try {
    doSomething()
} catch (error) {
    handleError(error)
}

# Valid: try with only finally
try {
    doSomething()
} finally {
    cleanup()
}

# Valid: try with both catch and finally
try {
    doSomething()
} catch (error) {
    handleError(error)
} finally {
    cleanup()
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

### Input

Read user input from stdin:

```gsh
# With prompt
name = input("What is your name? ")
print(`Hello, ${name}!`)

# Without prompt
print("Enter a value:")
value = input()
```

**Function signature:**

```gsh
input(prompt?: string): string
```

**Parameters:**

- `prompt` (optional) - A string to display before waiting for input

**Returns:** The user's input as a string (without trailing newline)

**Notes:**

- The prompt is printed to stdout without a trailing newline
- Input is read until a newline character
- Both `\n` and `\r\n` line endings are trimmed from the result

### JSON Utilities

```gsh
# Parse JSON
data = JSON.parse('{"key": "value"}')

# Stringify
json = JSON.stringify({name: "Alice", age: 30})
```

### Shell Command Execution

Execute shell commands and capture their output:

```gsh
# Basic usage - execute a command and get result
result = exec("echo hello")
print(result.stdout)      # "hello\n"
print(result.stderr)      # ""
print(result.exitCode)    # 0

# With timeout (in milliseconds)
result = exec("sleep 10", {timeout: 5000})  # Times out after 5 seconds

# Check exit code for errors
result = exec("ls /nonexistent")
if (result.exitCode != 0) {
    log.error(`Command failed: ${result.stderr}`)
}

# Practical examples
branch = exec("git branch --show-current").stdout
files = exec("ls -la").stdout
hostname = exec("hostname").stdout
```

**Function signature:**

```gsh
exec(command: string, options?: {timeout?: number}): {stdout: string, stderr: string, exitCode: number}
```

**Options:**

- `timeout` - Maximum execution time in milliseconds (default: 60000ms / 60 seconds)

**Returns:** An object containing:

- `stdout` - Standard output as a string
- `stderr` - Standard error as a string
- `exitCode` - Exit code of the command (0 for success)

**Notes:**

- Commands are executed in a subshell (isolated from the main shell environment)
- Non-zero exit codes do not throw errors; check `exitCode` in the result
- Timeouts throw an error
- Use string interpolation for dynamic commands: `exec("echo ${variable}")`

---

## 7. Import and Export

### Module System Overview

gsh supports a module system for organizing code into reusable components. This allows you to:

- Break large scripts into focused modules
- Share tools and utilities across scripts
- Organize default handlers and configuration
- Create libraries of reusable helpers

### Import Syntax

gsh provides two import patterns:

#### Side-Effect Import

Execute a file for its side effects (like registering event handlers). No symbols are imported:

```gsh
import "./events/agent.gsh"
```

This is useful for modules that self-register:

```gsh
# file: events/agent.gsh
tool onAgentStart(ctx) {
    print("Agent started")
}
gsh.on("agent.start", onAgentStart)  # Self-registers
```

```gsh
# file: main.gsh
import "./events/agent.gsh"  # File runs, handler is registered
```

#### Selective Import

Import only specific exported symbols:

```gsh
import { helper, config } from "./helpers.gsh"
```

This is useful for importing tools and utilities:

```gsh
# file: helpers.gsh
export tool processData(input) {
    return JSON.parse(input)
}

export config = {
    timeout: 5000,
    retries: 3,
}
```

```gsh
# file: main.gsh
import { processData, config } from "./helpers.gsh"

data = processData('{"key": "value"}')
print(config.timeout)
```

### Export Declaration

Mark symbols as available for import using the `export` keyword:

```gsh
# Export a variable
export myVariable = 42

# Export a tool
export tool myHelper(x) {
    return x * 2
}

# Export a model
export model myModel {
    provider: "openai",
    model: "gpt-4",
}

# Export an agent
export agent myAgent {
    model: myModel,
    tools: [myHelper],
}

# Export an MCP server declaration
export mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}
```

Only symbols marked with `export` are visible to importers. Non-exported symbols remain private to the module.

### Path Resolution

Import paths are resolved relative to the **current script's directory**:

#### Path Prefixes

- `./` - Relative to current script's directory
- `../` - Parent directory (can stack: `../../`)
- Absolute paths (e.g., `/home/user/scripts/foo.gsh`) - Filesystem only

#### Origin Types

gsh distinguishes between two script origins:

**Embedded** - Scripts bundled into the gsh binary (e.g., default configuration):

```gsh
# From: cmd/gsh/defaults/init.gsh
import "./models.gsh"        # Resolves to: cmd/gsh/defaults/models.gsh
import "./events/agent.gsh"  # Resolves to: cmd/gsh/defaults/events/agent.gsh
```

**Filesystem** - Scripts on the user's filesystem (e.g., `~/.gsh/repl.gsh`):

```gsh
# From: ~/.gsh/repl.gsh
import "./my-tools.gsh"     # Resolves to: ~/.gsh/my-tools.gsh
import "../shared/lib.gsh"  # Resolves to: ~/shared/lib.gsh
```

Each import is resolved relative to its origin type. Embedded imports stay within embedded defaults; filesystem imports stay on the filesystem.

### Module Scope

Each imported file has its own **module scope**:

```gsh
# file: helpers.gsh
privateVar = "not visible outside"      # Private to this module
export publicVar = "visible to importers"  # Exported

export tool publicTool() {
    return privateVar  # Can access private vars internally
}
```

```gsh
# file: main.gsh
import { publicVar, publicTool } from "./helpers.gsh"

print(publicVar)      # Works: "visible to importers"
print(publicTool())   # Works, returns "not visible outside"
print(privateVar)     # Error: undefined variable
```

### Circular Import Prevention

Circular imports are detected and result in an error:

```gsh
# file: a.gsh
import "./b.gsh"  # OK

# file: b.gsh
import "./a.gsh"  # Error: circular import detected
```

Each unique file path can only be imported once per interpreter session. Subsequent imports of the same file return cached exports without re-executing the module.

### Module Caching

The interpreter caches imported modules to:

- Prevent re-execution of the same file
- Enable circular import detection
- Improve performance for repeated imports

Each unique file path is only executed once per interpreter session. If you import the same module in multiple places, it runs once and subsequent imports use the cached exports:

```gsh
# file: config.gsh
export defaultTimeout = 5000

# file: module-a.gsh
import { defaultTimeout } from "./config.gsh"
# config.gsh executes here

# file: module-b.gsh
import { defaultTimeout } from "./config.gsh"
# config.gsh NOT re-executed, uses cache

# file: main.gsh
import "./module-a.gsh"
import "./module-b.gsh"
# config.gsh runs only once, during module-a import
```

### Example: Organizing Code into Modules

You can break scripts into focused modules:

```gsh
# file: helpers.gsh
export tool analyzeData(input) {
    return JSON.parse(input)
}

export tool formatOutput(text: string): string {
    return text.toUpperCase()
}

export config = {
    timeout: 10000,
    retries: 3,
}
```

```gsh
# file: main.gsh
import { analyzeData, formatOutput, config } from "./helpers.gsh"

data = analyzeData('{"values": [1, 2, 3]}')
result = formatOutput(JSON.stringify(data))
print(result)
```

This approach helps organize larger scripts into reusable, focused modules.

---

## 8. Execution Model

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
