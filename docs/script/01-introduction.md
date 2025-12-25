# Chapter 01: Introduction to gsh Scripting

Welcome! If you're reading this, you're curious about gsh scripting—a way to write powerful automation scripts that blend traditional shell automation with AI capabilities. This chapter orients you to what gsh is, why it matters, and how it fits into your toolkit.

## What is gsh Scripting?

**gsh** is both an interactive shell and a scripting language. When you use gsh interactively (typing commands at a prompt), it behaves like a POSIX-compatible shell—familiar territory if you've used bash or zsh. But when you write gsh scripts in `.gsh` files, you're using a richer, more powerful language built specifically for automation.

Think of it like this:

- **gsh REPL** = Your interactive shell (POSIX-compatible, like bash)
- **gsh Scripts** = A distinct scripting language in `.gsh` files (type-safe, with AI integration)

This separation is intentional. It keeps your interactive shell predictable while giving scripts access to advanced features.

### gsh Scripts vs. Bash Scripts

Here's a quick comparison:

| Aspect           | Bash                       | gsh Scripts                                  |
| ---------------- | -------------------------- | -------------------------------------------- |
| Type safety      | No (everything is strings) | Yes (string, number, boolean, any)           |
| Type annotations | No                         | Yes (TypeScript-like syntax)                 |
| Error handling   | Limited (exit codes)       | Structured (try/catch blocks)                |
| AI integration   | Not built-in               | Native support                               |
| External tools   | Shell functions, scripts   | MCP servers                                  |
| Dependencies     | Bash binary                | Single gsh binary (no Node.js, Python, etc.) |

## Why Use gsh Scripts?

### 1. Type Safety and Clear Errors

In bash, a typo in a variable name creates a new variable (silently). In gsh, the interpreter catches these mistakes:

```gsh
userName = "Alice"
print(userName)    # ✓ Works: Alice
print(userNme)     # ✗ Error caught immediately
```

This might seem small, but it saves enormous amounts of debugging time in real scripts.

### 2. Native AI Integration

gsh scripts have first-class support for LLMs. Define an AI agent and use it in your script:

```gsh
model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}

agent DataAnalyzer {
    model: exampleModel,
    system: "You are a data analyst. Analyze provided data and extract insights.",
}

data = "Sales: $1M, Growth: 25%, Churn: 5%"
analysis = data | DataAnalyzer
print(analysis)
```

No string templating, no juggling multiple API clients. The agent is a first-class citizen in your script.

### 3. MCP Tools Integration

The Model Context Protocol (MCP) gives your scripts access to tools like filesystem access, GitHub APIs, databases, and more—without writing connector code:

```gsh
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

# Now use filesystem tools directly
files = filesystem.list_directory("/home/user")
content = filesystem.read_file("README.md")
```

### 4. Single Static Binary

No runtime dependencies. gsh scripts work anywhere the gsh binary exists:

- ✅ No Node.js required
- ✅ No Python required
- ✅ No system package manager needed
- ✅ Same behavior on macOS, Linux, Windows (where Go works)

### 5. Structured Error Handling

Bash uses exit codes; gsh uses exceptions:

```gsh
try {
    content = filesystem.read_file("data.json")
    data = JSON.parse(content)
    print("Success!")
} catch (error) {
    print(`Failed: ${error.message}`)
    # Handle gracefully
}
```

## Architecture Overview

When you run a gsh script, here's what happens under the hood:

```
your-script.gsh
      ↓
   Lexer (tokenize)
      ↓
   Parser (build AST)
      ↓
  Interpreter (execute)
      ↓
    Result
```

All of this is built in **pure Go** with no external runtime. The entire pipeline is compiled into the gsh binary.

### Design Philosophy

**Native Go Interpreter** - gsh includes a complete lexer, parser, and tree-walking interpreter written in Go. This gives gsh:

- Fast startup (no initialization overhead)
- No external dependencies (single static binary)
- Deterministic behavior across platforms
- Easy distribution

**No Special Runtime** - Unlike scripts that require Node.js, Python, or Ruby installed, gsh scripts run in any environment with the gsh binary. This is crucial for deployment, CI/CD, and production systems.

## How This Ebook is Organized

This ebook is designed to teach you gsh scripting in the order you need each concept, not alphabetical order. It follows this progression:

**Part 1: Getting Started** (2 chapters)

- [Chapter 01](01-introduction.md) (you are here): Orient yourself
- [Chapter 02: Write your first script](02-hello-world.md)

**Part 2: Core Language Fundamentals** (5 chapters)

- [Chapter 03: Values and Types](03-values-and-types.md)
- [Chapter 04: Variables and Assignment](04-variables-and-assignment.md)
- [Chapter 05: Operators and Expressions](05-operators-and-expressions.md)
- [Chapter 06: Arrays and Objects](06-arrays-and-objects.md)
- [Chapter 07: String Manipulation](07-string-manipulation.md)

**Part 3: Control Flow** (3 chapters)

- [Chapter 08: Conditionals](08-conditionals.md)
- [Chapter 09: Loops](09-loops.md)
- [Chapter 10: Error Handling](10-error-handling.md)

**Part 4: Functions & Reusability** (2 chapters)

- [Chapter 11: Tool Declarations](11-tool-declarations.md)
- [Chapter 12: Tool Calls and Composition](12-tool-calls-and-composition.md)

**Part 5: External Integration** (4 chapters)

- [Chapter 13: Environment Variables](13-environment-variables.md)
- [Chapter 14: MCP Servers](14-mcp-servers.md)
- [Chapter 15: MCP Tool Invocation](15-mcp-tool-invocation.md)
- [Chapter 16: Shell Commands](16-shell-commands.md)

**Part 6: AI Agents** (3 chapters)

- [Chapter 17: Model Declarations](17-model-declarations.md)
- [Chapter 18: Agent Declarations](18-agent-declarations.md)
- [Chapter 19: Conversations and Pipes](19-conversations-and-pipes.md)

**Part 7: Real-World Patterns** (3 chapters)

- [Chapter 20: Common Patterns](20-common-patterns.md)
- [Chapter 21: Case Studies](21-case-studies.md)
- [Chapter 22: Debugging and Troubleshooting](22-debugging-and-troubleshooting.md)

**Part 8: Reference** (3 chapters)

- [Chapter 23: Built-in Functions](23-builtin-functions.md)
- [Chapter 24: Syntax Quick Reference](24-syntax-quick-reference.md)
- [Chapter 25: MCP Ecosystem](25-mcp-ecosystem.md)

Each chapter builds on previous ones. You'll write your first script in Chapter 02, and by Chapter 19 you'll be orchestrating multi-agent workflows with MCP tools and AI models.

## A Real Example

Here's a complete (simple) script that showcases several concepts:

```gsh
#!/usr/bin/env gsh

# Declare a tool to format a greeting
tool makeGreeting(name: string, emoji: string): string {
    return `${emoji} Hello, ${name}!`
}

# Use the tool
greeting = makeGreeting("Alice", "👋")
print(greeting)

# Access environment
home = env.HOME
print(`Your home directory: ${home}`)

# Handle errors
try {
    result = exec("whoami")
    user = result.stdout
    print(`Current user: ${user}`)
} catch (error) {
    print(`Error: ${error.message}`)
}
```

When you run this script with `gsh example.gsh`, it will:

1. Define a tool
2. Call it and print the result
3. Access an environment variable
4. Execute a shell command and use its output

All with type safety and clear error handling.

## What's Next?

In Chapter 02, you'll set up your environment and write your first gsh script from scratch. You'll learn how to run it and understand the basic structure. By the end of that chapter, you'll have a working script and understand exactly what happens when you run it.

Ready? Let's write some code! → [Chapter 02: Hello World](02-hello-world.md)
