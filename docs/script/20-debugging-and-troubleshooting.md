# Chapter 20: Debugging and Troubleshooting

When things go wrong in your gsh scripts, you need tools to find out what happened. This chapter teaches you how to diagnose problems effectively—whether you're tracking down a logic error in your script, investigating why an MCP tool call failed, or understanding what an agent is doing.

---

## Understanding Common Errors

Before we dive into debugging tools, let's understand what kinds of errors you'll encounter:

### Syntax Errors

These happen before your script even runs—when the parser encounters invalid code:

```gsh
# Syntax error: missing closing brace
tool procesData {
    return "incomplete"
# Missing }
```

When you run this, gsh will immediately report the parse error with the line number.

### Runtime Errors

These happen while your script executes. They include:

```gsh
# Type mismatch
x = "hello"
y = x + 5  # Error: can't add string + number

# Undefined variable
print(undefinedVariable)  # Error: undefinedVariable is not defined

# Failed MCP tool call
content = filesystem.read_file("/nonexistent/path.txt")  # Error: file not found

# JSON parse error
data = JSON.parse("invalid json")  # Error: invalid JSON
```

### Logic Errors

These are the hardest to find—your code runs without errors, but produces wrong results:

```gsh
total = 0
for (item of items) {
    total = item  # Wrong! Should be: total = total + item
}
print(total)  # Prints only the last item instead of the sum
```

---

## The `log` Object: Your Debugging Workhorse

The `log` object provides four levels of logging that you can use throughout your script. These are especially useful for understanding what your script is doing.

### Four Logging Levels

```gsh
log.debug("Entering processData with input: ${input}")
log.info("Processing file: ${filename}")
log.warn("Timeout approaching - only 5 seconds left")
log.error("Failed to connect to database: ${error.message}")
```

**Output (all go to stderr):**

```
[DEBUG] Entering processData with input: data.json
[INFO] Processing file: data.json
[WARN] Timeout approaching - only 5 seconds left
[ERROR] Failed to connect to database: Connection refused
```

Each level serves a purpose:

- **`log.debug()`** - Detailed diagnostic information. Use this for tracing through your code during development.
- **`log.info()`** - General informational messages about script progress.
- **`log.warn()`** - Something unexpected but not fatal. Your script continues, but something deserves attention.
- **`log.error()`** - Something went wrong. Use this in catch blocks or when an operation fails.

### Logging with Multiple Arguments

You can pass multiple arguments to log functions—they'll be joined with spaces:

```gsh
count = 42
name = "items"
log.info("Processed", count, "total", name)
```

**Output:**

```
[INFO] Processed 42 total items
```

### Practical: Adding Debug Output to a Script

Here's a realistic script with logging at key points:

```gsh
tool processFile(filename: string): string {
    log.debug(`Attempting to read file: ${filename}`)

    try {
        content = exec("cat " + filename).stdout
        log.debug(`File read successfully, size: ${content.length} bytes`)

        data = JSON.parse(content)
        log.info(`Parsed JSON with ${data.length} records`)

        return `Processed ${data.length} items from ${filename}`
    } catch (error) {
        log.error(`Failed to process ${filename}: ${error.message}`)
        return "failed"
    }
}

log.info("Starting script")
result = processFile("data.json")
print(result)
log.info("Script completed successfully")
```

When you run this, you can see exactly what's happening at each step.

---

## Error Objects and Stack Traces

When an error occurs in your script, you can catch it and inspect it. Error objects have a `.message` property:

```gsh
try {
    content = filesystem.read_file("/missing.json")
} catch (error) {
    print(`Error message: ${error.message}`)
}
```

**Output:**

```
Error message: file not found: /missing.json
```

### Reading Error Messages

Error messages usually tell you exactly what went wrong:

```gsh
# When you forget to catch an error, gsh shows the full error:
# Error: undefinedVariable is not defined
#
# Stack trace:
#   at main (line 5)

# This tells you the error occurred on line 5 of your script
```

### Multiple Errors and Stack Traces

When errors propagate through multiple tools, you get a stack trace showing the call chain:

```gsh
tool readData(path: string) {
    return filesystem.read_file(path)
}

tool processData(path: string) {
    content = readData(path)
    return JSON.parse(content)
}

try {
    result = processData("/missing.json")
} catch (error) {
    log.error(`Error: ${error.message}`)
}
```

**Output:**

```
[ERROR] Error: file not found: /missing.json
```

The error bubbles up from `filesystem.read_file()` → `readData()` → `processData()` → your `catch` block. Each caller in the chain can handle it or let it propagate further.

---

## Debugging Strategies

### Strategy 1: Instrument with Logging

Add `log.info()` calls at key decision points to trace execution:

```gsh
tool findMatches(items, pattern) {
    log.info(`Searching for pattern: ${pattern}`)
    matches = []

    for (item of items) {
        if (item == pattern) {
            log.debug(`Found match: ${item}`)
            matches = matches + [item]
        }
    }

    log.info(`Found ${matches.length} matches`)
    return matches
}
```

Run your script and look at the log output. Does it show what you expected? If not, you've found your bug!

### Strategy 2: Break Down Complex Logic

If you have a complicated expression, break it into smaller steps and log each one:

```gsh
# Before: hard to debug if it fails
result = (data.items.length > 0 && data.items[0].value > 100) ? "yes" : "no"

# After: easy to debug
itemCount = data.items.length
log.debug(`Item count: ${itemCount}`)
hasItems = itemCount > 0
log.debug(`Has items: ${hasItems}`)

if (hasItems) {
    firstValue = data.items[0].value
    log.debug(`First item value: ${firstValue}`)
    isHigh = firstValue > 100
    log.debug(`Is high: ${isHigh}`)
    result = isHigh ? "yes" : "no"
} else {
    result = "no"
}
```

### Strategy 3: Test Assumptions with Assertions

Create a simple tool to verify that values are what you expect:

```gsh
tool assert(condition: boolean, message: string) {
    if (!condition) {
        log.error(`Assertion failed: ${message}`)
        # In a real scenario, you might return false or let an error propagate
        return false
    }
    return true
}

# Use it to verify your assumptions
data = JSON.parse(jsonString)
assert(data != null, "data should not be null")
assert(data.items != null, "data.items should exist")
assert(data.items.length > 0, "data.items should not be empty")
log.info("All assertions passed")
```

### Strategy 4: Print Intermediate Values

When working with complex data structures, print them to see their actual structure:

```gsh
content = filesystem.read_file("data.json")
log.debug(`Raw content: ${content}`)

data = JSON.parse(content)
log.debug(`Parsed data: ${data}`)

log.debug(`Type: ${data.type}`)
log.debug(`Items count: ${data.items.length}`)
```

---

## Debugging Specific Problems

### Problem: "Variable is not defined"

**Symptom:** Error says a variable doesn't exist

**Diagnosis:**

- Check the spelling—JavaScript is case-sensitive
- Check if the variable is in the right scope (not defined inside an if block when used outside)
- Check if the variable is actually assigned before being used

```gsh
# Wrong: variable spelled differently
userName = "Alice"
print(username)  # Error: username is not defined

# Right
userName = "Alice"
print(userName)  # Prints: Alice
```

### Problem: "Type mismatch" or "Cannot read property of null"

**Symptom:** Error about types or null values

**Diagnosis:** Use logging to check what type you actually got:

```gsh
result = exec("git branch")
log.debug(`exec result type: ${result.type}`)
log.debug(`Full result: ${result}`)

# Make sure you're accessing properties correctly
branch = result.stdout
log.debug(`Branch: ${branch}`)
```

### Problem: MCP Tool Returns Empty or Unexpected Result

**Symptom:** A tool call doesn't work as expected

**Diagnosis:** Log the parameters you're sending and the result you get:

```gsh
path = "/home/user/file.txt"
log.debug(`Reading file at: ${path}`)
content = filesystem.read_file(path)
log.debug(`File content: ${content}`)
log.debug(`Content length: ${content.length}`)
```

### Problem: Agent Doesn't Take Action You Expected

**Symptom:** You ask an agent to do something, but it does something else

**Diagnosis:** Log what you're sending to the agent and what it returns:

```gsh
prompt = `Analyze this data: ${JSON.stringify(data)}`
log.info(`Sending to agent: ${prompt}`)

result = prompt | MyAgent
log.info(`Agent response: ${result}`)
```

---

## Using `exec()` to Debug Shell Issues

When debugging shell command integration, use `exec()` to capture both success and failure:

```gsh
# Check that a command works
result = exec("git status")
log.debug(`Exit code: ${result.exitCode}`)
log.debug(`Stdout: ${result.stdout}`)
log.debug(`Stderr: ${result.stderr}`)

if (result.exitCode != 0) {
    log.error(`Command failed: ${result.stderr}`)
} else {
    log.info("Command succeeded")
}
```

This shows you exactly what the command output and whether it succeeded.

---

## Common Debugging Patterns

### Pattern: Validate Inputs at Tool Entry Points

```gsh
tool processData(data: any, options: any) {
    # Log and validate all inputs
    log.debug(`processData called`)
    log.debug(`  data type: ${data.type}`)
    log.debug(`  options: ${options}`)

    if (data == null) {
        log.error("data is null")
        throw "data is required"
    }

    # ... rest of function
}
```

### Pattern: Log Before Risky Operations

```gsh
tool deleteFiles(pattern: string) {
    log.warn(`About to delete files matching: ${pattern}`)

    files = filesystem.find_files(pattern)
    log.warn(`Found ${files.length} files to delete`)

    for (file of files) {
        log.debug(`Deleting: ${file}`)
        filesystem.delete_file(file)
    }

    log.info(`Successfully deleted ${files.length} files`)
}
```

### Pattern: Wrap External Calls with Logging

```gsh
tool safeGetUser(userId: string) {
    log.debug(`Fetching user: ${userId}`)

    try {
        # In a real script, this would be: user = api.getUser(userId)
        # For this example, we simulate the API call
        user = {id: userId, name: "Test User"}
        log.debug(`User fetched successfully: ${user.name}`)
        return user
    } catch (error) {
        log.error(`Failed to fetch user ${userId}: ${error.message}`)
        return null
    }
}
```

---

## Summary: Key Takeaways

1. **Logging is your primary debugging tool** - Use `log.debug()`, `log.info()`, `log.warn()`, and `log.error()` liberally to trace execution

2. **Error messages tell you what's wrong** - Read them carefully. They usually point directly at the problem

3. **Break problems into smaller pieces** - Complex logic is harder to debug. Decompose it into simple steps with logging

4. **Test your assumptions** - Don't assume a value is what you think it is—log it and verify

5. **Common mistakes to watch for:**
   - Variable names with wrong casing
   - Null/undefined values where you expected objects
   - Type mismatches (string vs number)
   - MCP tool paths or parameters wrong
   - Off-by-one errors in loops

---

## What's Next?

You now have practical debugging skills. In the next chapter, we'll look at the complete reference for all built-in functions so you know exactly what tools are available. After that, you'll find a quick syntax reference for the entire language.

[← Chapter 19: Conversations and the Pipe Operator](19-conversations-and-pipes.md) | [Chapter 21: Built-in Functions Reference →](21-builtin-functions.md)
