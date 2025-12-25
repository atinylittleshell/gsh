# Chapter 13: Environment Variables

Your gsh script doesn't live in isolation. It runs within a system that has configuration, secrets, and settings stored in **environment variables**. In this chapter, you'll learn to read from and write to your system's environment—connecting your script to the wider world.

## What Are Environment Variables?

Environment variables are key-value pairs that exist in your system environment. They're set by your operating system, shell, or other programs. Examples include:

- `HOME` - Your home directory
- `USER` - Your username
- `PATH` - Where the system looks for executables
- `PORT` - A port number your application should listen on
- `API_KEY` - A secret token for an external service

In gsh, you access these through the special `env` object.

## Reading Environment Variables

The simplest case: read an environment variable.

```gsh
home = env.HOME
print(`Your home directory is: ${home}`)
```

Output (example):

```
Your home directory is: /Users/alice
```

That's it. Use dot notation to access any environment variable by name.

### What If the Variable Doesn't Exist?

If you try to read a variable that's not set, gsh returns `null`:

```gsh
result = env.NONEXISTENT_VARIABLE
print(result)
```

Output:

```
null
```

This is different from bash, where accessing a nonexistent variable gives you an empty string. In gsh, you get `null`, which is more precise—it distinguishes between "not set" and "set to empty string."

### Providing Default Values

When a variable might not be set, use the **nullish coalescing operator** (`??`) to provide a default:

```gsh
port = env.PORT ?? 3000
print(`Server listening on port: ${port}`)
```

Output (if `PORT` is not set):

```
Server listening on port: 3000
```

Output (if `PORT=8080`):

```
Server listening on port: 8080
```

The `??` operator says: "Use the left value if it's not null, otherwise use the right value." This is perfect for optional configuration.

### Reading Multiple Variables

Here's a practical example that reads several variables:

```gsh
# Read API configuration from environment
apiKey = env.API_KEY ?? ""
apiHost = env.API_HOST ?? "api.example.com"
timeout = env.API_TIMEOUT ?? 30

print(`API Key set: ${apiKey != ""}`)
print(`API Host: ${apiHost}`)
print(`Timeout: ${timeout}`)
```

Output (with defaults):

```
API Key set: false
API Host: api.example.com
Timeout: 30
```

Notice that `env.API_TIMEOUT` returns a string (all environment variables are strings in the OS). If you need a number, convert it:

```gsh
timeoutStr = env.API_TIMEOUT ?? "30"
timeout = parseFloat(timeoutStr)
print(`Timeout (as number): ${timeout}`)
```

Actually, gsh doesn't have a `parseFloat` built-in yet, so for now, know that environment variables are always strings.

## Setting Environment Variables

You can also set environment variables from within your script. This affects the script's environment and any subprocesses (like shell commands) it spawns.

```gsh
# Set an environment variable
env.MY_VAR = "hello"
print(env.MY_VAR)
```

Output:

```
hello
```

### Setting Different Types

You can set strings, numbers, and booleans. gsh converts them to strings (since that's what the OS environment stores):

```gsh
# Set as string
env.NAME = "Alice"

# Set as number
env.PORT = 8080

# Set as boolean
env.DEBUG = true

# Read them back
print(env.NAME)    # "Alice"
print(env.PORT)    # "8080"
print(env.DEBUG)   # "true"
```

Output:

```
Alice
8080
true
```

Notice that numbers and booleans are converted to strings. If you read them back and need them as numbers or booleans, you'll need to convert.

## Unsetting Environment Variables

Set a variable to `null` to unset (remove) it from the environment:

```gsh
env.TEMP_VAR = "value"
print(env.TEMP_VAR)

env.TEMP_VAR = null
print(env.TEMP_VAR)
```

Output:

```
value
null
```

## Using Environment Variables in String Interpolation

Environment variables work in template literals just like any other variable:

```gsh
username = env.USER ?? "guest"
greeting = `Welcome, ${username}!`
print(greeting)
```

Output (example):

```
Welcome, alice!
```

This is useful for constructing paths, URLs, or other dynamic strings.

## A Practical Example: Configuration

Here's a realistic script that reads configuration from environment variables:

```gsh
#!/usr/bin/env gsh

# Read configuration
dbHost = env.DB_HOST ?? "localhost"
dbPort = env.DB_PORT ?? "5432"
dbUser = env.DB_USER ?? "postgres"
dbPassword = env.DB_PASSWORD ?? "password"

# Validate required variables
if (env.API_KEY == null) {
    print("ERROR: API_KEY environment variable is required")
}

# Build connection string (example)
connectionString = `postgres://${dbUser}:${dbPassword}@${dbHost}:${dbPort}/mydb`
print(`Connecting to: ${connectionString}`)

# Set internal variables
env.INTERNAL_CONFIG = "loaded"
print("Configuration loaded successfully")
```

Output (with some env vars set):

```
Connecting to: postgres://user:pass@db.example.com:5432/mydb
Configuration loaded successfully
```

## Environment Variables and Shell Commands

When you run shell commands with `exec()`, the environment variables set in your script are inherited by the subshell:

```gsh
# Set a variable
env.GREETING = "Hello from gsh"

# Use it in a shell command
result = exec("echo $GREETING")
print(result.stdout)
```

Output:

```
Hello from gsh
```

This is powerful because it lets your script pass configuration to external tools:

```gsh
# Configure a tool via environment
env.LOG_LEVEL = "debug"
env.TIMEOUT = "60"

# Run a tool that respects these variables
result = exec("./my-tool")
print(result.stdout)
```

## Key Takeaways

- **Access environment variables** with `env.VARIABLE_NAME`
- **Non-existent variables return `null`**, not empty string
- **Use `??` for defaults** when a variable might not be set
- **Set variables** with `env.VARIABLE_NAME = value`
- **Unset variables** by setting them to `null`
- **All environment variables are strings** (the OS stores them that way)
- **Variables set in your script** are inherited by shell commands you run
- **Use in string interpolation** just like regular variables

## What's Next?

Now that you can read and write environment variables, you're ready to integrate with external systems. In Chapter 14, you'll learn about **MCP Servers**—the gateway to powerful external tools. MCP servers often use environment variables for configuration (like API keys), so the patterns you learned here will come in handy.

Ready to connect to external tools? → [Chapter 14: MCP Servers](14-mcp-servers.md)

---

**Previous:** [Chapter 12: Tool Calls and Composition](12-tool-calls-and-composition.md)
