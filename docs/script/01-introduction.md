# Chapter 01: Introduction to gsh Scripting

Welcome! If you're reading this, you're curious about gsh scripting—a way to write powerful automation scripts that blend traditional shell automation with AI capabilities.

## What is gsh Scripting?

**gsh** is both an interactive shell and a scripting language. When you use gsh interactively (typing commands at a prompt), it behaves like a POSIX-compatible shell—familiar territory if you've used bash or zsh. But when you write gsh scripts in `.gsh` files, you're using a richer, more powerful language built specifically for automation.

Think of it like this:

- **gsh REPL** = Your interactive shell (POSIX-compatible, like bash)
- **gsh Scripts** = A distinct scripting language in `.gsh` files (type-safe, with AI integration)

This separation is intentional. It keeps your interactive shell predictable while giving scripts access to advanced features.

## Why Use gsh Scripts?

### 1. Type Safety and Clear Errors

In bash, a typo in a variable name creates a new variable (silently). In gsh, the interpreter catches these mistakes:

```gsh
userName = "Alice"
print(userName)    # ✓ Works: Alice
print(userNme)     # ✗ Error caught immediately
```

This might seem small, but it saves debugging time in real scripts.

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

For the complete structure and learning path for this guide, see the [README](README.md).

## What's Next?

In Chapter 02, you'll set up your environment and write your first gsh script from scratch. You'll learn how to run it and understand the basic structure. By the end of that chapter, you'll have a working script and understand exactly what happens when you run it.

---

**Next Chapter:** [Chapter 02: Hello World](02-hello-world.md)
