# gsh Concurrency Specification

**Status:** Draft / Future Work  
**Date:** January 2026

---

## 1. Overview

This document outlines the design for adding concurrency support to the gsh scripting language. The goal is to enable non-blocking I/O operations while maintaining a simple, predictable execution model.

### Design Philosophy

gsh will follow an **implicit await model** with explicit concurrency:

- **Single-threaded execution** - One call stack, one thing executing at a time
- **Implicit await** - All operations are automatically awaited; code looks synchronous
- **Explicit concurrency** - Use `all()` for parallel operations, `go` for fire-and-forget

This approach was chosen because:

1. **Simplest user experience** - No `async`/`await` keywords cluttering code
2. **Beginner friendly** - Code just works, no async mental model needed
3. **Less boilerplate** - Don't need to mark every I/O function
4. **Matches gsh philosophy** - "Shell scripting but better"
5. **Existing code unchanged** - Current scripts continue to work

### Key Insight

In gsh, almost everything interesting is I/O-bound (MCP tools, HTTP, shell commands, agent calls). Rather than requiring explicit `async`/`await` everywhere, the runtime handles suspension transparently.

---

## 2. Preparation Work Completed

The following minimal changes have been made to prepare for concurrency:

### 2.1 Reserved Keywords

The `go` keyword is reserved in the lexer (`internal/script/lexer/token.go`):

```go
KW_GO // Reserved for future concurrency support (fire-and-forget)
```

Using `go` as a variable name produces a parse error:

```gsh
go = 5  # Error: unexpected token 'go'
```

### 2.2 Promise Value Type

A `ValueTypePromise` constant has been added to the value system (`internal/script/interpreter/value.go`):

```go
ValueTypePromise  // Reserved for future concurrency support
```

---

## 3. Proposed Syntax

### 3.1 Sequential Code (Default)

All operations are implicitly awaited. Code looks synchronous:

```gsh
# Each line completes before the next starts
content = filesystem.read_file("data.json")
data = JSON.parse(content)
print(data)
```

This is identical to current gsh behavior - **no syntax changes needed** for sequential code.

### 3.2 Concurrent Operations with `all()`

To run multiple operations in parallel, use `all()`:

```gsh
# Sequential (slow) - each waits for the previous
content1 = filesystem.read_file("file1.txt")
content2 = filesystem.read_file("file2.txt")
content3 = filesystem.read_file("file3.txt")

# Concurrent (fast) - all start at once, wait for all to complete
contents = all([
    filesystem.read_file("file1.txt"),
    filesystem.read_file("file2.txt"),
    filesystem.read_file("file3.txt"),
])
```

### 3.3 Fire-and-Forget with `go`

Use the `go` keyword to run an operation without waiting:

```gsh
tool logToServer(data) {
    http.post("/log", data)
}

tool onAgentStart(ctx) {
    go logToServer(ctx)  # Fire and forget - runs in background
    print("Agent starting")  # Executes immediately
}
```

### 3.4 Racing Operations

Use `race()` to get the first result:

```gsh
# Return whichever completes first
result = race([
    http.get("https://primary.api.com/data"),
    http.get("https://backup.api.com/data"),
])
```

---

## 4. Semantics

### 4.1 Implicit Await

When any I/O operation is called, the runtime:

1. Starts the operation
2. Suspends the current execution
3. Continues when the operation completes
4. Returns the result

From the user's perspective, it looks like a normal function call.

### 4.2 Promise Value (Internal)

Internally, async operations return a `PromiseValue` which can be in three states:

- **Pending** - Operation in progress
- **Fulfilled** - Completed successfully, has a result value
- **Rejected** - Failed, has an error

Users don't interact with Promises directly - they're an implementation detail.

### 4.3 The `go` Keyword

The `go` keyword:

1. Starts the operation
2. Returns immediately (does not wait)
3. The operation runs in the background on the event loop

```gsh
go someOperation()  # Returns immediately
print("This runs right away")
```

### 4.4 Error Handling

Errors work naturally with try/catch:

```gsh
tool safeFetch(url) {
    try {
        return http.get(url)
    } catch (error) {
        log.error(`Failed to fetch ${url}: ${error.message}`)
        return null
    }
}
```

### 4.5 Unhandled Errors in Fire-and-Forget

When a `go` operation fails, the interpreter logs a warning:

```gsh
tool mightFail() {
    http.get("/bad-url")  # Throws error
}

tool onEvent(ctx) {
    go mightFail()  # Fire and forget
    # If mightFail() throws, logs:
    # "gsh: unhandled error in background operation: connection refused"
}
```

---

## 5. Event Handler Integration

### 5.1 Event Handlers with I/O

Event handlers can perform I/O operations naturally:

```gsh
tool onAgentStart(ctx) {
    config = fetchRemoteConfig()  # Implicitly awaited
    print(`Agent starting with config: ${config.name}`)
}
gsh.on("agent.start", onAgentStart)
```

### 5.2 Handler Execution Order

Event handlers execute sequentially. Each handler completes (including any I/O) before the next starts.

### 5.3 Fire-and-Forget in Handlers

Use `go` for non-blocking operations in handlers:

```gsh
tool onAgentEnd(ctx) {
    go logToRemoteServer(ctx)  # Don't block the event
    print("Agent finished")
}
```

### 5.4 Performance Considerations

For latency-sensitive events (like `agent.chunk` for streaming), handlers should avoid I/O or use `go`:

```gsh
tool onAgentChunk(ctx) {
    # DON'T: This blocks streaming
    # logToServer(ctx.content)

    # DO: Fire and forget
    go logToServer(ctx.content)

    # Or just do fast, synchronous work
    print(ctx.content)
}
```

---

## 6. Built-in Concurrent Operations

### 6.1 `all(operations)` - Parallel Execution

Wait for all operations to complete:

```gsh
results = all([
    http.get("/api/users"),
    http.get("/api/posts"),
    http.get("/api/comments"),
])
# results = [usersData, postsData, commentsData]
```

If any operation fails, `all()` throws the first error.

### 6.2 `race(operations)` - First Wins

Return the first result:

```gsh
result = race([
    http.get("https://fast-server.com/data"),
    http.get("https://slow-server.com/data"),
])
# result = whichever completed first
```

### 6.3 `sleep(ms)` - Delay

Pause execution:

```gsh
print("Starting...")
sleep(1000)  # Wait 1 second
print("Done!")
```

### 6.4 `timeout(operation, ms)` - Deadline

Run with a timeout:

```gsh
try {
    result = timeout(http.get("/slow-endpoint"), 5000)
} catch (error) {
    print("Request timed out")
}
```

---

## 7. Implementation Plan

### Phase 1: Core Infrastructure

1. **PromiseValue struct** - Implement with state, result, error fields
2. **Event loop** - Add to Interpreter for managing pending operations
3. **Implicit await** - Modify expression evaluation to await Promises automatically

### Phase 2: Concurrency Primitives

1. **`go` statement** - Parser and interpreter support
2. **`all()` builtin** - Concurrent execution with result collection
3. **`race()` builtin** - First-to-complete semantics

### Phase 3: Utilities

1. **`sleep()` builtin** - Timer-based delay
2. **`timeout()` builtin** - Deadline wrapper
3. **Error handling** - Unhandled rejection warnings

### Phase 4: Integration

1. **MCP tools** - Ensure all MCP calls work with implicit await
2. **Agent pipes** - Make agent execution work with concurrency
3. **Event handlers** - Update EmitEvent for proper awaiting

### Phase 5: Polish

1. **Error messages** - Clear errors for concurrency issues
2. **Documentation** - Update all docs with concurrency examples
3. **Testing** - Comprehensive concurrency test suite

---

## 8. Comparison with Alternatives

### JavaScript-style async/await

```javascript
// JavaScript
async function fetchData() {
  const result = await http.get(url);
  return result;
}
```

```gsh
# gsh (implicit await)
tool fetchData() {
    result = http.get(url)
    return result
}
```

**gsh advantage:** No `async`/`await` boilerplate.

### Go-style goroutines

```go
// Go
go func() {
    result := fetchData()
    ch <- result
}()
data := <-ch
```

```gsh
# gsh
go fetchData()  # fire-and-forget

# or for results:
results = all([fetchData(), fetchData()])
```

**gsh advantage:** Simpler syntax, no channels needed for common cases.

---

## 9. Open Questions

### 9.1 `go` with Return Value

Should `go` return anything?

```gsh
# Option A: Returns nothing (void)
go someOperation()

# Option B: Returns a handle for later
handle = go someOperation()
result = wait(handle)  # Explicit wait if needed
```

**Recommendation:** Start with Option A (simpler). Add handles later if needed.

### 9.2 Nested `all()`

Should nested `all()` calls flatten or nest?

```gsh
results = all([
    all([op1(), op2()]),
    all([op3(), op4()]),
])
# Is results [[r1,r2], [r3,r4]] or [r1,r2,r3,r4]?
```

**Recommendation:** Nest (preserve structure), matching JavaScript `Promise.all`.

### 9.3 Error Aggregation in `all()`

When multiple operations fail, return first error or all errors?

**Recommendation:** First error (simpler, matches JavaScript). Consider `allSettled()` variant later.

---

## 10. Why Not async/await?

We initially considered JavaScript-style `async`/`await` but chose implicit await because:

| Aspect          | async/await                    | Implicit await  |
| --------------- | ------------------------------ | --------------- |
| Sequential code | `result = await fn()`          | `result = fn()` |
| Boilerplate     | Need `async` on every function | None            |
| Learning curve  | Must understand Promises       | Just works      |
| Fire-and-forget | `fn()` (no await)              | `go fn()`       |
| Concurrent ops  | `await all([...])`             | `all([...])`    |

The only thing `async`/`await` gives you is fire-and-forget by omitting `await`. We achieve the same with the explicit `go` keyword, which is clearer about intent.

---

## 11. References

- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [JavaScript Promise.all](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/all)
- [Structured Concurrency](https://vorpus.org/blog/notes-on-structured-concurrency-or-go-statement-considered-harmful/)
