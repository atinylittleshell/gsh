# Chapter 10: Error Handling

Welcome to **resilience**! So far, your scripts have assumed that everything works perfectly: files exist, division never by zero, and tools always succeed. But in the real world, things break. Files go missing. Networks fail. Typos happen. What do your scripts do when something goes wrong?

In gsh, you handle errors gracefully with **try-catch-finally blocks**. They let you anticipate problems, respond to them, and clean up afterward. This chapter teaches you how to build scripts that don't crash when things go sideways—they recover.

## The Problem: Unhandled Errors

Let's start by seeing what happens without error handling:

```gsh
x = undefinedVariable
print(x)
```

If the variable isn't defined, this script crashes immediately. The error message appears, and nothing else runs. Your script is dead in the water.

In a real system—a data pipeline, a background job, an automation script—crashing isn't acceptable. You need to detect the error, handle it, and move on.

## The Solution: Try-Catch

Enter `try-catch`. The pattern is simple: you **try** something risky, and if it fails, you **catch** the error:

```gsh
try {
    x = undefinedVariable
    print("No error")
} catch (error) {
    print("Oops! Could not access variable: " + error.message)
}

print("Script continues here")
```

Output:

```
Oops! Could not access variable: undefined variable: undefinedVariable (line 2, column 9)
Script continues here
```

Here's what happened:

1. We entered the `try` block and accessed an undefined variable
2. An error occurred
3. We immediately jumped to the `catch` block
4. The `error` variable holds information about what went wrong
5. We printed the error message and continued

The key insight: **the script didn't crash**. It recovered.

## Understanding the Error Object

When an error is caught, the `error` variable gives you details:

```gsh
try {
    x = undefinedVariable
} catch (error) {
    print("Error message: " + error.message)
}
```

Output:

```
Error message: undefined variable: undefinedVariable
```

The `error` object has a `.message` property that describes what went wrong. Common errors include:

- "undefined variable: X" — you used a variable that wasn't defined
- "division by zero" — you tried to divide by 0
- "file not found" — a file you tried to read doesn't exist
- "array index out of bounds" — you accessed an array index that doesn't exist

## Real Example: Safe Data Processing

Let's build a practical example. Imagine you have a list of JSON strings to parse, but some might be malformed:

```gsh
jsonStrings = [
    "{\"name\": \"Alice\", \"age\": 30}",
    "{invalid json}",
    "{\"name\": \"Bob\", \"age\": 25}",
]
processed = 0
failed = 0

for (jsonStr of jsonStrings) {
    try {
        data = JSON.parse(jsonStr)
        print("Successfully parsed: " + data.name)
        processed = processed + 1
    } catch (error) {
        print("Failed to parse: " + error.message)
        failed = failed + 1
    }
}

print("Results: " + processed + " processed, " + failed + " failed")
```

Output:

```
Successfully parsed: Alice
Failed to parse: invalid character 'i' looking for beginning of value
Successfully parsed: Bob
Results: 2 processed, 1 failed
```

Notice that even though one JSON string was malformed, the script kept going and processed the others. This is the power of error handling: **robustness**.

## The `finally` Block

Sometimes you need to clean up after an operation, whether it succeeds or fails. That's what `finally` is for:

```gsh
try {
    x = 5
    print("Try block executed")
} catch (error) {
    print("Error caught: " + error.message)
} finally {
    print("Cleanup always runs")
}
```

Output:

```
Try block executed
Cleanup always runs
```

The `finally` block **always runs**, whether the try block succeeds or the catch block catches an error. It's perfect for cleanup tasks like:

- Closing database connections
- Flushing buffers
- Releasing locks
- Logging

## Try-Finally Without Catch

You can use `finally` without `catch` if you don't need to handle the error—you just want to ensure cleanup happens:

```gsh
try {
    x = undefinedVariable
} finally {
    print("Cleanup executed despite error")
}
```

Output (the finally block runs, then the error propagates):

```
Cleanup executed despite error
Error: undefined variable: undefinedVariable
```

If an error occurs, you don't catch it, but the finally block still runs to clean up. After the finally block executes, the error propagates up.

## Catch Without Finally

Similarly, you can have `catch` without `finally`:

```gsh
try {
    data = JSON.parse("{invalid json")
} catch (error) {
    print("JSON parsing failed: " + error.message)
}
```

This is the most common pattern. You handle the error and move on—no cleanup needed.

## Error Propagation: When Errors Bubble Up

What happens if you don't catch an error? It propagates up the call stack:

```gsh
tool loadData() {
    x = undefinedVariable
    return x
}

result = loadData()
```

If the error happens inside `loadData`, it propagates to the caller. Since there's no `catch` at the top level, the script crashes with an error message and a stack trace showing where the error came from.

But if you add error handling at the top level:

```gsh
tool loadData() {
    x = undefinedVariable
    return x
}

result = null
try {
    result = loadData()
} catch (error) {
    print("Could not load data: " + error.message)
    result = null
}

print("Result: " + result)
```

Now the error is caught at the top level, and you can handle it gracefully. This is a key pattern: **let tools focus on their work, and handle errors at the call site where you know how to respond**.

## Nested Try-Catch: Multiple Layers of Protection

You can nest `try-catch` blocks for fine-grained error handling:

```gsh
result = ""
try {
    try {
        x = undefinedVariable
        result = "no error"
    } catch (innerError) {
        print("Inner catch handled: " + innerError.message)
        result = "inner"
    }

    print("After inner try-catch")
} catch (outerError) {
    print("Outer catch: " + outerError.message)
    result = "outer"
}

print("Final result: " + result)
```

Output:

```
Inner catch handled: undefined variable: undefinedVariable (line 4, column 9)
After inner try-catch
Final result: inner
```

How this works:

1. The inner `try` accesses an undefined variable, causing an error
2. The inner `catch` handles it and sets a default value
3. The outer `try` continues normally with that value
4. If something else goes wrong later, the outer `catch` would handle it

This pattern is useful when different layers of your code need different error-handling strategies.

## Important: Errors That Don't Get Caught

Some control flow statements are **not** treated as errors. They pass right through `try-catch`:

### `break` and `continue`

In a loop with error handling, `break` and `continue` work normally:

```gsh
for (i of [1, 2, 3, 4, 5]) {
    try {
        if (i == 3) {
            break
        }
        print(i)
    } catch (error) {
        print("Error: " + error.message)
    }
}
```

Output:

```
1
2
```

The `break` statement doesn't get caught—it exits the loop as intended.

### `return` Statements

Similarly, `return` statements in a tool are not caught:

```gsh
tool example() {
    try {
        return 42
    } catch (error) {
        return 99
    }
}

result = example()
print(result)
```

Output:

```
42
```

The `return` executes immediately; the catch block doesn't interfere.

## Pattern: Fallbacks and Defaults

A common pattern is to provide a fallback value when something fails:

```gsh
config = {}

try {
    userInput = undefinedVariable
    config = {value: userInput}
} catch (error) {
    print("Could not get user input, using defaults")
    config = {
        host: "localhost",
        port: 8080,
        debug: false,
    }
}

print("Config host: " + config.host)
```

Output:

```
Could not get user input, using defaults
Config host: localhost
```

This pattern makes your scripts resilient: they work even when operations fail or resources are unavailable.

## Pattern: Retry Logic

Another useful pattern is to retry an operation when it fails:

```gsh
maxRetries = 3
attempt = 0
success = false

while (attempt < maxRetries && !success) {
    try {
        // Simulate an operation that might fail
        if (attempt < 2) {
            x = undefinedVariable
        }
        success = true
        print("Operation succeeded")
    } catch (error) {
        attempt = attempt + 1
        if (attempt < maxRetries) {
            print("Attempt " + attempt + " failed, retrying...")
        }
    }
}

if (success) {
    print("Success after " + attempt + " retries")
} else {
    print("All retry attempts failed")
}
```

Output:

```
Attempt 1 failed, retrying...
Attempt 2 failed, retrying...
Operation succeeded
Success after 2 retries
```

This tries an operation up to 3 times. If it fails, it retries. Only after all retries are exhausted does it give up.

## Pattern: Logging Errors

For debugging and monitoring, always log errors:

```gsh
jsonString = "{invalid json}"

try {
    data = JSON.parse(jsonString)
} catch (error) {
    log.error("JSON parsing failed: " + error.message)
    print("Note: Using log.error() ensures errors are captured for inspection")
}
```

Using `log.error()` ensures that errors are captured in logs for later inspection. This is essential for debugging production scripts that run unattended.

## Best Practices

Here are proven strategies for error handling in gsh:

### 1. Catch Where You Can Handle It

Don't catch errors everywhere. Catch them where you know how to respond:

```gsh
# Bad: Catches too broadly without context
try {
    # 50 lines of code
} catch (error) {
    print("Something went wrong")
}

# Good: Catches specific operations with clear recovery
try {
    config = filesystem.read_file("config.json")
} catch (error) {
    config = getDefaultConfig()
}
```

### 2. Use Specific Error Messages

When you catch an error, use its message to decide what to do:

```gsh
try {
    result = exec("npm install")
} catch (error) {
    if (error.message.contains("permission denied")) {
        print("You don't have permission to install packages")
    } else {
        print("Installation failed: " + error.message)
    }
}
```

### 3. Always Provide Context

When you catch and re-report an error, add context:

```gsh
try {
    content = filesystem.read_file(filename)
} catch (error) {
    log.error("Failed to load config from " + filename + ": " + error.message)
}
```

Instead of just "file not found", you've now logged "Failed to load config from config.json: file not found". Much more helpful!

### 4. Use Finally for Cleanup

If you're allocating resources (even logical resources), use `finally` to ensure cleanup:

```gsh
try {
    conn = getConnection()
    data = conn.query("SELECT * FROM users")
} catch (error) {
    log.error("Query failed: " + error.message)
} finally {
    conn.close()
}
```

The connection is always closed, even if the query fails.

## Real-World Example: Robust Data Processing

Let's bring it all together with a complete example—a data processing pipeline that handles errors gracefully:

```gsh
tool processData(jsonStr: string) {
    try {
        data = JSON.parse(jsonStr)

        if (data == null) {
            return {
                success: false,
                error: "Data is empty",
            }
        }

        return {
            success: true,
            value: data,
        }
    } catch (error) {
        return {
            success: false,
            error: error.message,
        }
    }
}

dataItems = [
    "[1, 2, 3]",
    "{invalid json}",
    "[4, 5]",
    "null",
]
results = []
totalCount = 0

for (item of dataItems) {
    try {
        result = processData(item)
        results.push(result)

        if (result.success) {
            totalCount = totalCount + 1
            print("✓ Parsed successfully")
        } else {
            print("✗ Failed: " + result.error)
        }
    } catch (error) {
        print("✗ Unexpected error: " + error.message)
    }
}

print("")
print("Pipeline Summary")
print("===============")
successful = 0
for (r of results) {
    if (r.success) {
        successful = successful + 1
    }
}
print("Total parsed: " + totalCount + " items")
print("Successful: " + successful + " / " + results.length)
```

Output:

```
✓ Parsed successfully
✗ Failed: JSON.parse error: invalid character 'i' looking for beginning of object key string
✓ Parsed successfully
✗ Failed: JSON.parse error: unexpected end of JSON input

Pipeline Summary
===============
Total parsed: 2 items
Successful: 2 / 4
```

Notice how the pipeline:

- Processes items one by one
- Catches errors and logs them
- Continues processing even when some items fail
- Provides a summary at the end

This is a production-ready pattern you can use in real scripts.

## Key Takeaways

- **Try-catch blocks let your scripts survive errors** instead of crashing
- **The `error` object has a `.message` property** describing what went wrong
- **`catch` is optional if you have `finally`**—you can clean up without handling the error
- **`finally` always runs**, regardless of success or failure
- **Errors propagate up the call stack** until caught—use this to your advantage
- **Catch where you can respond**—don't catch errors you can't handle
- **Log errors for debugging** using `log.error()` and `log.warn()`
- **Use error handling to build resilient pipelines** that keep going when individual operations fail

## What's Next?

Now that you can handle what goes wrong, it's time to learn how to organize your code. The next chapter is **Chapter 11: Tool Declarations**, where you'll learn to define reusable tools that other parts of your script can call. Tools are the foundation of composition and code organization.

But first, try writing a script that:

1. Attempts to read multiple files
2. Catches errors gracefully
3. Logs what succeeded and what failed

This reinforces the patterns you've learned.

---

**Previous Chapter:** [Chapter 09: Loops](09-loops.md)

**Next Chapter:** [Chapter 11: Tool Declarations](11-tool-declarations.md)
