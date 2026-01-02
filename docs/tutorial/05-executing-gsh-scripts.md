# Chapter 05: Executing gsh Scripts

So far, we've focused on using gsh interactively. But gsh's real power shines when you write scripts—`.gsh` files that automate tasks, call agents, and integrate with tools. This chapter shows you how to write and execute gsh scripts.

## What Are gsh Scripts?

A **gsh script** is a file with a `.gsh` extension containing code written in the gsh scripting language. Unlike your `repl.gsh` configuration file (which runs once at startup), scripts are standalone programs you execute on demand.

**gsh scripts vs. bash scripts:**

- **bash scripts** - Strings, limited types, exit codes for error handling
- **gsh scripts** - Type-safe, proper error handling, AI agents, MCP tools, structured data

## Your First gsh Script

Create a file called `hello.gsh`:

```gsh
#!/usr/bin/env gsh

# Your first gsh script
print("Hello from gsh!")
print("Current directory: " + env.PWD)
```

Make it executable:

```bash
chmod +x hello.gsh
```

Run it:

```bash
./hello.gsh
```

Output:

```
Hello from gsh!
Current directory: /Users/yourname/projects
```

Or run it explicitly with gsh:

```bash
gsh run hello.gsh
```

## Using Models in Scripts

The `gsh.models` object is available in scripts, just like in the REPL. You can configure model tiers and use them with agents:

```gsh
#!/usr/bin/env gsh

# Define a model
model localModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "qwen3-coder:30b",
}

# Assign it to the workhorse tier
gsh.models.workhorse = localModel

# Create an agent that uses the workhorse tier
agent helper {
    model: gsh.models.workhorse,
    systemPrompt: "You are a helpful assistant.",
    tools: [gsh.tools.exec],
}

# Use the agent
result = "List files in the current directory" | helper
print(result)
```

This makes it easy to write portable scripts that can use different models based on environment or configuration.

## Learning More

For deeper dives into gsh scripting, see the [Script Documentation](../script/):

## What's Next?

You've learned the full gsh tutorial! You can now:

1. ✅ Use gsh as an interactive shell
2. ✅ Configure it with `~/.gshrc` and `~/.gsh/repl.gsh`
3. ✅ Create beautiful prompts with Starship
4. ✅ Get AI command predictions
5. ✅ Use agents for complex tasks
6. ✅ Write and execute gsh scripts

Next steps:

- **Explore the [Script Documentation](../script/)** for deeper language features

---

**Previous Chapter:** [Chapter 04: Agents in the REPL](04-agents-in-the-repl.md)
