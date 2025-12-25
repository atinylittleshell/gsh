# Chapter 16: Shell Commands

You've learned how to write gsh scripts with variables, control flow, custom tools, and MCP servers. But what if you need to run shell commands? What if you want to check the current git branch, list files with `ls`, or run any other command-line tool?

In this chapter, you'll learn how to execute shell commands from within your gsh scripts using the `exec()` function. You'll see how to capture output, check exit codes, handle errors gracefully, and decide when to use `exec()` versus MCP tools.

---

## Why Shell Commands?

You might be wondering: "Isn't MCP filesystem better than shell commands?" Sometimes yes, sometimes no. Here's when each is useful:

**Use MCP servers when:**

- You need cross-platform compatibility (MCP abstracts OS differences)
- You want sandboxed access to a specific service
- The MCP tool is available (like filesystem, GitHub, etc.)

**Use `exec()` when:**

- You need to run existing command-line tools
- You want to leverage scripts you've already written
- The tool isn't available as an MCP server
- You need to chain multiple shell commands
- You're processing git, npm, docker, or other CLI tools

Think of `exec()` as your escape hatch to the wider world of command-line tools.

---

## Running Your First Shell Command

The `exec()` function is simple: pass it a command string, and it runs that command in a subshell. You get back an object with the output and exit code.

### Basic Syntax

```gsh
result = exec("command-name")
```

The result is an object with three properties:

- `stdout` - The standard output (what the command printed)
- `stderr` - The standard error (error messages)
- `exitCode` - The exit code (0 for success, non-zero for errors)

### Example 1: Echoing Text

Let's start with the simplest possible command:

```gsh
#!/usr/bin/env gsh

result = exec("echo hello")
print(result.stdout)
print(result.exitCode)
```

**Output:**

```
hello
0
```

Notice that `stdout` includes the newline that `echo` adds. The exit code is 0, which means the command succeeded.

### Example 2: Running a Git Command

Let's get the current git branch:

```gsh
#!/usr/bin/env gsh

result = exec("git branch --show-current")
branch = result.stdout
print(`Current branch: ${branch}`)
```

**Output:**

```
Current branch: main
```

The output includes a trailing newline from the git command. In a real script, you might trim it:

```gsh
#!/usr/bin/env gsh

result = exec("git branch --show-current")
branch = result.stdout.trim()
print(`Current branch: ${branch}`)
```

**Output:**

```
Current branch: main
```

Much cleaner! The `.trim()` method removes leading and trailing whitespace.

---

## Capturing Both Output and Errors

When a command fails, its error messages go to `stderr`. Let's see how to handle that:

### Example 3: Handling Command Failures

```gsh
#!/usr/bin/env gsh

result = exec("ls /nonexistent")
print(`Exit code: ${result.exitCode}`)
print(`Stderr: ${result.stderr}`)
```

**Output:**

```
Exit code: 2
Stderr: ls: cannot access '/nonexistent': No such file or directory
```

The command failed (exit code 2), and the error message is in `stderr`. Notice that `exec()` doesn't throw an error—it just returns the result. This gives you control to decide how to handle failures.

### Example 4: Checking Success with Error Handling

Here's a pattern for checking if a command succeeded:

```gsh
#!/usr/bin/env gsh

result = exec("git status")

if (result.exitCode != 0) {
    print(`Error: ${result.stderr}`)
} else {
    print(result.stdout)
}
```

**Output:**

```
On branch main
Your branch is up to date with 'origin/main'.

nothing to commit, working tree clean
```

---

## Using String Interpolation in Commands

Commands are just strings, so you can use gsh's string interpolation to build dynamic commands:

### Example 5: Dynamic File Operations

```gsh
#!/usr/bin/env gsh

filename = "data.json"
result = exec(`cat ${filename}`)

if (result.exitCode == 0) {
    content = result.stdout
    data = JSON.parse(content)
    print(`File contains: ${data}`)
} else {
    print(`Could not read file: ${result.stderr}`)
}
```

**Output:**

```
File contains: {"name": "Alice", "age": 30}
```

### Example 6: Piping Commands Together

The shell pipe operator works inside exec():

```gsh
#!/usr/bin/env gsh

result = exec("echo -e 'apple\\nbanana\\ncherry' | grep a")
print(result.stdout)
```

**Output:**

```
apple
banana
```

The grep command filtered the output to lines containing "a".

---

## Setting Timeouts

Long-running commands can hang your script. The `exec()` function accepts an optional second argument with a timeout (in milliseconds):

### Example 7: Using Timeouts

```gsh
#!/usr/bin/env gsh

try {
    result = exec("sleep 100", {timeout: 2000})
    print("Command completed")
} catch (error) {
    print(`Error: ${error.message}`)
}
```

**Output:**

```
Error: exec() command timed out after 2s
```

The timeout is specified in milliseconds. A timeout throws an error, which you can catch and handle.

### Example 8: Reasonable Timeout for Long Operations

```gsh
#!/usr/bin/env gsh

# Timeout after 30 seconds (30000 ms)
try {
    result = exec("npm install", {timeout: 30000})
    if (result.exitCode == 0) {
        print("Installation succeeded")
    } else {
        print(`Installation failed: ${result.stderr}`)
    }
} catch (error) {
    print(`Command timed out or failed: ${error.message}`)
}
```

**Output:**

```
Installation succeeded
```

---

## Practical Examples: Real-World Tasks

### Example 9: Getting System Information

```gsh
#!/usr/bin/env gsh

# Get hostname
hostname = exec("hostname").stdout.trim()
print(`Hostname: ${hostname}`)

# Get current user
user = exec("whoami").stdout.trim()
print(`User: ${user}`)

# Get current working directory
cwd = exec("pwd").stdout.trim()
print(`Working directory: ${cwd}`)
```

**Output:**

```
Hostname: mycomputer
User: alice
Working directory: /home/alice/projects
```

### Example 10: Counting Files in a Directory

```gsh
#!/usr/bin/env gsh

# Count files in current directory
result = exec("find . -type f | wc -l")
count = result.stdout.trim()
print(`This directory contains ${count} files`)
```

**Output:**

```
This directory contains 42 files
```

### Example 11: Processing Command Output

```gsh
#!/usr/bin/env gsh

# Get list of all Go files
result = exec("find . -name '*.go' -type f")

if (result.exitCode == 0) {
    files = result.stdout.split("\n")

    # Filter out empty lines
    goFiles = []
    for (file of files) {
        if (file.length > 0) {
            goFiles.push(file)
        }
    }

    print(`Found ${goFiles.length} Go files`)
    for (file of goFiles) {
        print(`  ${file}`)
    }
} else {
    print(`Error: ${result.stderr}`)
}
```

**Output:**

```
Found 3 Go files
  ./cmd/gsh/main.go
  ./cmd/gsh/main_test.go
  ./internal/script/parser/parser.go
```

---

## When to Use exec() vs. MCP Tools

Let's compare the two approaches:

### Example 12: Reading Files with exec() vs. MCP

**Using exec():**

```gsh
#!/usr/bin/env gsh

result = exec("cat config.json")
if (result.exitCode == 0) {
    config = JSON.parse(result.stdout)
    print(config)
}
```

**Using MCP (from Chapter 15):**

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

result = filesystem.read_file({path: "config.json"})
config = JSON.parse(result.content)
print(config)
```

Both work, but:

- **exec()** is simpler if you just need to run `cat` once
- **MCP** is better if you need cross-platform compatibility or multiple file operations
- **MCP** is safer (sandboxed) and more explicit

### Example 13: Combining exec() and MCP

You can use both in the same script:

```gsh
#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

# Use exec() for git operations
result = exec("git log --oneline -n 5")
print("Recent commits:")
print(result.stdout)

# Use MCP for file operations
content = filesystem.read_file({path: "README.md"})
print("README.md:")
print(content.content)
```

**Output:**

```
Recent commits:
a1b2c3d Fix bug in parser
d4e5f6g Add shell command support
h7i8j9k Refactor type system
l0m1n2o Initial commit
README.md:
# My Project

This is a great project.
```

---

## Error Handling Patterns

### Pattern 1: Check Exit Code

```gsh
#!/usr/bin/env gsh

result = exec("some-command")
if (result.exitCode != 0) {
    print(`Command failed: ${result.stderr}`)
}
```

### Pattern 2: Try-Catch for Timeouts

```gsh
#!/usr/bin/env gsh

try {
    result = exec("long-running-command", {timeout: 5000})
    if (result.exitCode != 0) {
        print(`Command failed: ${result.stderr}`)
    }
} catch (error) {
    print(`Error: ${error.message}`)
}
```

### Pattern 3: Fallback Values

```gsh
#!/usr/bin/env gsh

result = exec("git branch --show-current")
branch = result.exitCode == 0 ? result.stdout.trim() : "unknown"
print(`Branch: ${branch}`)
```

---

## Key Takeaways

- **`exec()` executes shell commands** and returns `{stdout, stderr, exitCode}`
- **Non-zero exit codes don't throw errors** — you need to check the `exitCode`
- **Timeouts are specified in milliseconds** and will throw errors if exceeded
- **String interpolation works in commands** — pass dynamic values with `${variable}`
- **Use exec() for ad-hoc shell commands**, MCP for structured tool integration
- **Always check exit codes** or handle potential errors with try-catch for timeouts
- **Combine stdout/stderr handling** with error checking for robust scripts

---

## What's Next

You've now mastered executing shell commands and integrating them with your gsh scripts. In the next chapter (Chapter 17), you'll learn about **Model Declarations** — how to configure LLM providers like OpenAI, Anthropic, and Ollama so your scripts can use AI agents to solve complex problems.

Once you can declare models, you'll be ready to define agents and unlock the true power of gsh: intelligent, agentic workflows.

---

**Previous Chapter:** [Chapter 15: MCP Tool Invocation](15-mcp-tool-invocation.md)

**Next Chapter:** [Chapter 17: Model Declarations](17-model-declarations.md)
