# Chapter 21: Built-in Functions Reference

Welcome to the reference chapter! By now you've learned the gsh language inside and out—variables, control flow, agents, MCP tools. But there's one more piece: the **built-in functions** that ship with gsh. These are your everyday utilities for printing output, working with JSON, executing shell commands, and handling user input.

This chapter is both a learning guide and a quick reference. We'll walk through each built-in function with concrete examples, then you'll have everything you need to look up quickly.

## Output: `print()`

The most basic built-in: **print to stdout**.

### Basic Usage

```gsh
print("Hello, world!")
```

**Output:**

```
Hello, world!
```

Print always adds a newline at the end. If you pass multiple arguments, they're separated by spaces:

```gsh
name = "Alice"
age = 30
print("User:", name, "Age:", age)
```

**Output:**

```
User: Alice Age: 30
```

Print works with any value type:

```gsh
print(42)
print(true)
print([1, 2, 3])
print({name: "Bob", role: "admin"})
```

**Output:**

```
42
true
[1, 2, 3]
{name: "Bob", role: "admin"}
```

### Use Cases

- Display results to the user
- Debug by printing variable state
- Output formatted messages and reports

---

## Logging: `log.debug()`, `log.info()`, `log.warn()`, `log.error()`

When you need **structured, level-based logging**, use the `log` object. It has four methods matching severity levels:

### Basic Logging

```gsh
log.debug("Starting process")
log.info("Processing file: data.json")
log.warn("Memory usage at 80%")
log.error("Failed to connect to database")
```

**Output (to stderr):**

```
[DEBUG] Starting process
[INFO] Processing file: data.json
[WARN] Memory usage at 80%
[ERROR] Failed to connect to database
```

Each log level serves a purpose:

- **`log.debug()`** - Detailed information for troubleshooting (usually disabled in production)
- **`log.info()`** - General informational messages about script progress
- **`log.warn()`** - Warning about something unexpected but not a failure
- **`log.error()`** - Error messages when something goes wrong

### Multiple Arguments

Like `print()`, log methods accept multiple arguments:

```gsh
statusCode = 200
message = "OK"
log.info("HTTP response:", statusCode, message)
```

**Output:**

```
[INFO] HTTP response: 200 OK
```

### Practical Example

Here's a script that uses logging throughout its lifecycle:

```gsh
tool processFile(filename: string) {
    log.info("Starting to process:", filename)

    try {
        log.debug("Reading file...")
        content = exec(`cat ${filename}`).stdout

        log.debug("Parsing JSON...")
        data = JSON.parse(content)

        log.info("Successfully processed", filename)
        return data
    } catch (error) {
        log.error("Failed to process file:", error.message)
        return null
    }
}

log.info("Script started")
result = processFile("data.json")
if (result != null) {
    log.info("Result:", result)
} else {
    log.warn("No result returned from processing")
}
log.info("Script completed")
```

**Output:**

```
[INFO] Script started
[INFO] Starting to process: data.json
[INFO] Successfully processed data.json
[INFO] Result: {...}
[INFO] Script completed
```

### Notes

- Logs go to **stderr**, not stdout, so you can pipe stdout separately
- The `[PREFIX]` format is used when no structured logger is configured
- In production environments with a structured logger, the output format may vary

---

## User Input: `input()`

Read lines from **stdin** with an optional prompt:

### Basic Input

```gsh
name = input()
print("Hello, " + name)
```

If you run this and type `Alice`, you get:

**Output:**

```
Hello, Alice
```

### With Prompt

Pass a prompt string (displayed without a newline):

```gsh
age = input("How old are you? ")
print("You are", age, "years old")
```

Type `30` and you see:

**Output:**

```
How old are you? 30
You are 30 years old
```

Notice the prompt appears on the same line as your input—that's because the prompt string has no trailing newline.

### Trimming Line Endings

`input()` automatically trims trailing newlines (`\n` and `\r\n`), so you always get clean strings:

```gsh
value = input("Enter value: ")
# Even if user types "hello\n", value is exactly "hello"
print("Length:", value.length())
```

### Complete Example: Interactive Calculator

Here's an interactive program that uses `input()`:

```gsh
tool add(a: number, b: number): number {
    return a + b
}

print("=== Simple Calculator ===")
x = input("Enter first number: ")
y = input("Enter second number: ")

x_num = JSON.parse(x)
y_num = JSON.parse(y)

result = add(x_num, y_num)
print("Result:", result)
```

**Interaction:**

```
=== Simple Calculator ===
Enter first number: 5
Enter second number: 3
Result: 8
```

### Function Signature

```gsh
input(prompt?: string): string
```

- **`prompt`** (optional) - String to display before reading input
- **Returns** - User's input as a string, with trailing whitespace trimmed

---

## JSON Utilities: `JSON.parse()` and `JSON.stringify()`

Work with JSON data—the lingua franca of modern APIs and data exchange.

### Parsing: `JSON.parse()`

Convert a JSON string into a gsh value:

```gsh
jsonStr = '{"name": "Alice", "age": 30}'
data = JSON.parse(jsonStr)

print(data.name)
print(data.age)
```

**Output:**

```
Alice
30
```

JSON arrays become gsh arrays:

```gsh
jsonStr = '[1, 2, 3, 4, 5]'
numbers = JSON.parse(jsonStr)

for (num of numbers) {
    print(num)
}
```

**Output:**

```
1
2
3
4
5
```

Nested structures work too:

```gsh
jsonStr = '''
{
    "user": {
        "name": "Bob",
        "email": "bob@example.com"
    },
    "posts": [
        {"id": 1, "title": "First Post"},
        {"id": 2, "title": "Second Post"}
    ]
}
'''

data = JSON.parse(jsonStr)
print("User:", data.user.name)
print("Email:", data.user.email)
print("Post count:", data.posts.length())
```

**Output:**

```
User: Bob
Email: bob@example.com
Post count: 2
```

### Stringifying: `JSON.stringify()`

Convert gsh values into JSON strings:

```gsh
user = {
    name: "Charlie",
    age: 25,
    tags: ["developer", "open-source"]
}

jsonStr = JSON.stringify(user)
print(jsonStr)
```

**Output:**

```
{"name":"Charlie","age":25,"tags":["developer","open-source"]}
```

Works with all value types:

```gsh
print(JSON.stringify(true))
print(JSON.stringify(42))
print(JSON.stringify("hello"))
print(JSON.stringify(null))
print(JSON.stringify([1, 2, 3]))
```

**Output:**

```
true
42
"hello"
null
[1,2,3]
```

### Practical Example: API Integration

Here's how you'd work with a JSON API:

```gsh
mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

tool loadConfig(filename: string) {
    try {
        jsonStr = filesystem.read_file(filename)
        config = JSON.parse(jsonStr)
        return config
    } catch (error) {
        log.error("Failed to load config:", error.message)
        return null
    }
}

tool saveConfig(filename: string, config: any) {
    try {
        jsonStr = JSON.stringify(config)
        filesystem.write_file(filename, jsonStr)
        log.info("Config saved to", filename)
    } catch (error) {
        log.error("Failed to save config:", error.message)
    }
}

# Load config
config = loadConfig("settings.json")
if (config != null) {
    print("Loaded config:", config)

    # Modify it
    config.debug = true

    # Save it back
    saveConfig("settings.json", config)
}
```

### Function Signatures

```gsh
JSON.parse(jsonString: string): any
JSON.stringify(value: any): string
```

### Error Handling

If the JSON is malformed, `JSON.parse()` throws an error:

```gsh
try {
    data = JSON.parse("not valid json")
} catch (error) {
    log.error("Parse error:", error.message)
}
```

---

## Shell Commands: `exec()`

Execute shell commands and capture their output. This is your bridge to the entire Unix toolchain.

### Basic Execution

```gsh
result = exec("echo hello")
print("stdout:", result.stdout)
print("stderr:", result.stderr)
print("exit code:", result.exitCode)
```

**Output:**

```
stdout: hello

stderr:
exit code: 0
```

### Checking Exit Codes

Non-zero exit codes indicate failure (but don't throw an error):

```gsh
result = exec("ls /nonexistent")
if (result.exitCode != 0) {
    print("Command failed!")
    print("Error:", result.stderr)
}
```

**Output:**

```
Command failed!
Error: ls: cannot access '/nonexistent': No such file or directory
```

### Using Command Results

Capture output and process it:

```gsh
result = exec("git branch --show-current")
currentBranch = result.stdout
# stdout includes trailing newline, so trim it
currentBranch = currentBranch.split("\n")[0]
print("Current branch:", currentBranch)
```

**Output (if on main branch):**

```
Current branch: main
```

### String Interpolation

Use dynamic commands with string interpolation:

```gsh
filename = "data.json"
result = exec(`cat ${filename}`)
print(result.stdout)
```

### Timeouts

By default, commands time out after **60 seconds**. You can customize this:

```gsh
# Command times out after 5 seconds
result = exec("sleep 10", {timeout: 5000})
```

If a timeout occurs, an error is thrown:

```gsh
try {
    result = exec("sleep 30", {timeout: 5000})
} catch (error) {
    log.error("Command timed out:", error.message)
}
```

### Practical Example: Git Integration

Here's a script that uses exec() to interact with git:

```gsh
tool getCurrentBranch(): string {
    result = exec("git branch --show-current")
    return result.stdout.split("\n")[0]
}

tool getCommitCount(): number {
    result = exec("git rev-list --all --count")
    return JSON.parse(result.stdout.split("\n")[0])
}

tool getStatus(): string {
    result = exec("git status --porcelain")
    return result.stdout
}

branch = getCurrentBranch()
count = getCommitCount()
status = getStatus()

print("Branch:", branch)
print("Total commits:", count)
print("Status:\n" + status)
```

**Output (example):**

```
Branch: feature/new-feature
Total commits: 245
Status:
 M docs/script/21-builtin-functions.md
?? tmp_file.txt
```

### Function Signature

```gsh
exec(command: string, options?: {timeout?: number}): {stdout: string, stderr: string, exitCode: number}
```

**Options:**

- **`timeout`** (milliseconds, default: 60000) - Maximum time to wait for the command

**Returns an object with:**

- **`stdout`** - Standard output as a string
- **`stderr`** - Standard error as a string
- **`exitCode`** - Exit code (0 for success, non-zero for failure)

### Important Notes

- Commands run in an **isolated subshell**, not in the current shell environment
- Non-zero exit codes don't throw errors—check `exitCode` in the result
- **Timeouts throw errors**—wrap in try/catch if needed
- Use string interpolation for dynamic commands: ``exec(`command ${variable}`)``

---

## Environment Variables: `env` Object

Access and modify system environment variables through the `env` object (covered more thoroughly in Chapter 13, but repeated here for completeness).

### Reading Variables

```gsh
print("Home:", env.HOME)
print("User:", env.USER)
print("Path:", env.PATH)
```

### Non-existent Variables

Accessing a variable that doesn't exist returns `null`:

```gsh
debug = env.DEBUG_MODE
if (debug == null) {
    print("Debug mode not set")
}
```

### Default Values with `??`

Use the nullish coalescing operator to provide defaults:

```gsh
port = env.PORT ?? 3000
print("Server running on port:", port)
```

### Setting Variables

```gsh
env.MY_VAR = "some value"
env.PORT = 8080
env.DEBUG = true
```

These changes are visible to any `exec()` calls that follow:

```gsh
env.GREETING = "Hello from gsh"
exec("echo $GREETING")
```

### Unsetting Variables

Set a variable to `null` to unset it:

```gsh
env.TEMPORARY = "will remove"
# ... later ...
env.TEMPORARY = null
```

---

## Collections: `Map()` and `Set()`

Create specialized collection types beyond arrays and objects.

### Maps: Key-Value Storage

Create a map from an array of `[key, value]` pairs:

```gsh
userAges = Map([
    ["alice", 25],
    ["bob", 30],
    ["charlie", 35],
])

print(userAges.get("alice"))
print(userAges.get("bob"))
```

**Output:**

```
25
30
```

Or create an empty map and add entries:

```gsh
config = Map()
config.set("host", "localhost")
config.set("port", 8080)
config.set("debug", true)

print(config.get("host"))
```

**Output:**

```
localhost
```

### Sets: Unique Values

Create a set from an array (duplicates are removed):

```gsh
tags = Set(["javascript", "python", "go", "python", "javascript"])
print("Set size:", tags.size)
```

**Output:**

```
Set size: 3
```

Empty set:

```gsh
colors = Set()
colors.add("red")
colors.add("blue")
colors.add("red")  # ignored (already exists)

print(colors.has("red"))
print(colors.has("green"))
```

**Output:**

```
true
false
```

### Practical Example: Deduplication

```gsh
# Get all users from a log file and deduplicate
users = []
for (i = 0; i < 100; i = i + 1) {
    users = users + ["user_" + (i % 10)]
}

uniqueUsers = Set(users)
print("Total entries:", users.length())
print("Unique users:", uniqueUsers.size)
```

**Output:**

```
Total entries: 100
Unique users: 10
```

---

## Date and Time: `DateTime`

Work with dates and times using the `DateTime` object. Similar to dayjs, but with a static methods API.

### Getting Current Time: `DateTime.now()`

Get the current timestamp in milliseconds since Unix epoch:

```gsh
timestamp = DateTime.now()
print("Current timestamp:", timestamp)
```

**Output:**

```
Current timestamp: 1704067200000
```

### Parsing Dates: `DateTime.parse()`

Parse a date string into a timestamp (milliseconds):

```gsh
# Parse ISO 8601 format (auto-detected)
ts = DateTime.parse("2024-01-15T10:30:00Z")
print(ts)

# Parse date-only format (auto-detected)
ts = DateTime.parse("2024-01-15")
print(ts)

# Parse with custom format
ts = DateTime.parse("15/01/2024", "DD/MM/YYYY")
print(ts)
```

**Supported auto-detected formats:**

- ISO 8601: `2024-01-15T10:30:00Z`, `2024-01-15T10:30:00.000Z`
- Date only: `2024-01-15`, `2024/01/15`
- US format: `01/15/2024`
- EU format: `15/01/2024`
- Named: `Jan 15, 2024`, `January 15, 2024`

### Formatting Dates: `DateTime.format()`

Format a timestamp into a human-readable string:

```gsh
ts = DateTime.now()

# Default ISO 8601 format
print(DateTime.format(ts))

# Custom formats
print(DateTime.format(ts, "YYYY-MM-DD"))
print(DateTime.format(ts, "DD/MM/YYYY"))
print(DateTime.format(ts, "MMM DD, YYYY"))
print(DateTime.format(ts, "HH:mm:ss"))
```

**Output (example):**

```
2024-01-15T10:30:00.000-05:00
2024-01-15
15/01/2024
Jan 15, 2024
10:30:00
```

**Format tokens (dayjs-compatible):**

| Token  | Output             | Example |
| ------ | ------------------ | ------- |
| `YYYY` | 4-digit year       | 2024    |
| `YY`   | 2-digit year       | 24      |
| `MMMM` | Full month name    | January |
| `MMM`  | Short month name   | Jan     |
| `MM`   | Month (2-digit)    | 01      |
| `M`    | Month              | 1       |
| `DD`   | Day (2-digit)      | 05      |
| `D`    | Day                | 5       |
| `dddd` | Full weekday       | Monday  |
| `ddd`  | Short weekday      | Mon     |
| `HH`   | Hour 24h (2-digit) | 09      |
| `hh`   | Hour 12h (2-digit) | 09      |
| `mm`   | Minutes (2-digit)  | 05      |
| `ss`   | Seconds (2-digit)  | 05      |
| `SSS`  | Milliseconds       | 123     |
| `A`    | AM/PM              | PM      |
| `a`    | am/pm              | pm      |
| `Z`    | Timezone offset    | -07:00  |
| `ZZ`   | Timezone offset    | -0700   |

### Calculating Differences: `DateTime.diff()`

Calculate the difference between two timestamps:

```gsh
start = DateTime.parse("2024-01-01")
end = DateTime.parse("2024-01-15")

# Default unit is milliseconds
diffMs = DateTime.diff(end, start)
print("Milliseconds:", diffMs)

# Specify unit
diffDays = DateTime.diff(end, start, "days")
print("Days:", diffDays)

diffHours = DateTime.diff(end, start, "hours")
print("Hours:", diffHours)
```

**Output:**

```
Milliseconds: 1209600000
Days: 14
Hours: 336
```

**Supported units:**

- `milliseconds` or `ms`
- `seconds` or `s`
- `minutes` or `m`
- `hours` or `h`
- `days` or `d`

### Practical Example: Timing Operations

```gsh
tool timeExecution(name: string, fn: tool) {
    start = DateTime.now()
    result = fn()
    end = DateTime.now()

    duration = DateTime.diff(end, start, "seconds")
    log.info(name, "completed in", duration, "seconds")

    return result
}

tool slowOperation() {
    exec("sleep 2")
    return "done"
}

result = timeExecution("Slow operation", slowOperation)
print("Result:", result)
```

**Output:**

```
[INFO] Slow operation completed in 2 seconds
Result: done
```

### Practical Example: Date Formatting in Logs

```gsh
tool logWithTimestamp(message: string) {
    ts = DateTime.now()
    formatted = DateTime.format(ts, "YYYY-MM-DD HH:mm:ss")
    print("[" + formatted + "]", message)
}

logWithTimestamp("Application started")
logWithTimestamp("Processing data...")
logWithTimestamp("Done!")
```

**Output:**

```
[2024-01-15 10:30:00] Application started
[2024-01-15 10:30:01] Processing data...
[2024-01-15 10:30:02] Done!
```

### Function Signatures

```gsh
DateTime.now(): number
DateTime.parse(dateString: string, format?: string): number
DateTime.format(timestamp: number, format?: string): string
DateTime.diff(timestamp1: number, timestamp2: number, unit?: string): number
```

---

## Summary: When to Use Each Built-in

| Function            | Purpose                            | Example                             |
| ------------------- | ---------------------------------- | ----------------------------------- |
| `print()`           | Output to stdout                   | `print("Result:", value)`           |
| `log.info()`        | Structured logging (info level)    | `log.info("Processing started")`    |
| `log.warn()`        | Structured logging (warning level) | `log.warn("Deprecated API")`        |
| `log.error()`       | Structured logging (error level)   | `log.error("Failed:", err.message)` |
| `input()`           | Read user input                    | `name = input("Enter name: ")`      |
| `JSON.parse()`      | Parse JSON strings                 | `data = JSON.parse(jsonStr)`        |
| `JSON.stringify()`  | Convert to JSON                    | `jsonStr = JSON.stringify(data)`    |
| `exec()`            | Run shell commands                 | `result = exec("git status")`       |
| `env`               | Access environment variables       | `token = env.API_KEY`               |
| `Map()`             | Key-value collections              | `config = Map([["key", "value"]])`  |
| `Set()`             | Unique value collections           | `unique = Set([1, 2, 2, 3])`        |
| `DateTime.now()`    | Current timestamp (ms)             | `ts = DateTime.now()`               |
| `DateTime.parse()`  | Parse date strings                 | `ts = DateTime.parse("2024-01-15")` |
| `DateTime.format()` | Format timestamps                  | `DateTime.format(ts, "YYYY-MM-DD")` |
| `DateTime.diff()`   | Calculate time differences         | `DateTime.diff(end, start, "days")` |

---

## Key Takeaways

1. **`print()` outputs to stdout** with automatic newlines—use it for user-facing messages
2. **`log.*` methods output to stderr** with severity levels—use them for structured logging
3. **`input()` reads from stdin** with optional prompts—use it for interactive scripts
4. **JSON utilities handle parsing and serialization**—essential for working with APIs and config files
5. **`exec()` runs shell commands** and captures output—your gateway to Unix tools
6. **`env` accesses environment variables**—bridge between gsh and the system
7. **`Map()` and `Set()` provide specialized collections**—maps for lookups, sets for uniqueness
8. **`DateTime` provides date/time utilities**—parsing, formatting, and calculating differences

---

## What's Next

You've now completed the full language reference! You understand:

- ✅ Core syntax and types (Chapters 3-7)
- ✅ Control flow (Chapters 8-10)
- ✅ Functions and composition (Chapters 11-12)
- ✅ External integration (Chapters 13-16)
- ✅ AI and agents (Chapters 17-19)
- ✅ Debugging (Chapter 20)
- ✅ Built-in functions (Chapter 21 - this chapter)

---

**Previous Chapter:** [Chapter 20: Debugging and Troubleshooting](20-debugging-and-troubleshooting.md) | **Next Chapter:** [Chapter 22: Imports and Modules](22-imports-and-modules.md)
