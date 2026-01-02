# GSH Keystroke Middleware Specification

This document specifies a **keystroke middleware** system that extends the existing command middleware to enable scriptable syntax highlighting, command prediction, and tab completion.

## Status: Draft / Design Phase

This spec captures design thinking and open questions for future implementation. The existing command middleware (`gsh.useCommandMiddleware`) handles input after submission; this spec covers input processing _during_ typing.

## Motivation

The gsh REPL has several "pre-submit" features currently implemented in Go:

| Feature                 | Current Implementation              | Purpose                                        |
| ----------------------- | ----------------------------------- | ---------------------------------------------- |
| **Syntax Highlighting** | `internal/repl/input/highlight.go`  | Colors commands, variables, strings, operators |
| **Command Prediction**  | `internal/repl/input/prediction.go` | Ghost text suggestions (history + LLM)         |
| **Tab Completion**      | `internal/repl/input/completion.go` | File/command/word completion                   |

These are powerful features, but they're hardcoded in Go. Users cannot:

- Add custom syntax highlighting for DSLs or custom commands
- Provide domain-specific completions (e.g., Kubernetes resources, git branches)
- Customize prediction behavior or add custom prediction sources

Following gsh's neovim-inspired extensibility model, we want to make these features scriptable while maintaining good performance.

## Relationship to Command Middleware

gsh will have **two types of middleware**:

```
┌─────────────────────────────────────────────────────────────────┐
│                     User Types in REPL                          │
└─────────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┴────────────────────┐
         │                                         │
         ▼                                         ▼
┌─────────────────────┐                 ┌─────────────────────┐
│ Keystroke Middleware│                 │ Command Middleware  │
│ (runs while typing) │                 │ (runs on Enter)     │
├─────────────────────┤                 ├─────────────────────┤
│ • Syntax highlight  │                 │ • Route # commands  │
│ • Ghost prediction  │                 │ • Custom commands   │
│ • Tab completions   │                 │ • Transform input   │
│                     │                 │ • Fall through to   │
│ Async, debounced    │                 │   shell execution   │
└─────────────────────┘                 └─────────────────────┘
```

## API Design

### Registration Functions

```gsh
// Command middleware (existing, renamed for clarity)
gsh.useCommandMiddleware(tool)
gsh.removeCommandMiddleware(tool)

// Keystroke middleware (new)
gsh.useKeystrokeMiddleware(tool)
gsh.removeKeystrokeMiddleware(tool)
```

### Keystroke Middleware Signature

```gsh
tool myKeystrokeMiddleware(ctx, next) {
    // ctx contains everything about the current input state
    // Return object can set any/all of: highlight, prediction, completions

    result = next(ctx)  // Get results from downstream middleware

    // Merge/override with our results
    return {
        highlight: result.highlight,      // Styled string or null
        prediction: result.prediction,    // Ghost text or null
        completions: result.completions,  // Array of suggestions or null
    }
}
```

### Context Object

The context provides rich information about the current input state:

```gsh
ctx = {
    // Core input state
    input: "git commit -m \"fix bug\"",
    cursorPos: 14,                    // Cursor position in input

    // Parsed word context (useful for completion)
    word: {
        text: "commit",               // Current word under cursor
        start: 4,                     // Start index of word
        end: 10,                      // End index of word
    },

    // Input classification hints (computed by Go, informational only)
    hints: {
        isAgentMode: false,           // Starts with #
        firstWord: "git",             // First token
        commandExists: true,          // Is first word a valid command?
    },

    // Trigger info - why this middleware is running
    trigger: "change",                // "change" | "tab" | "idle"
}
```

### Result Object

Middleware returns an object with any combination of:

```gsh
{
    // Syntax highlighting - styled version of input
    // Option A: Pre-rendered string with ANSI codes
    highlight: "\x1b[32mgit\x1b[0m commit -m \"fix bug\"",

    // Option B: Structured spans (easier to compose/merge)
    highlight: [
        { start: 0, end: 3, style: "command" },
        { start: 4, end: 10, style: "subcommand" },
        { start: 11, end: 13, style: "flag" },
        { start: 14, end: 23, style: "string" },
    ],

    // Command prediction - ghost text shown after cursor
    prediction: " --amend",           // Or null for no prediction

    // Tab completions - shown in completion menu
    // Option A: Rich objects
    completions: [
        { value: "commit", description: "Record changes" },
        { value: "checkout", description: "Switch branches" },
    ],
    // Option B: Simple strings
    completions: ["commit", "checkout", "cherry-pick"],
}
```

## Execution Model

### Async + Debounced Execution

Keystroke middleware runs **asynchronously** with **built-in debouncing** on the Go side:

```
User types 'g'
    → Debounce timer starts (e.g., 50ms)
User types 'i' (within 50ms)
    → Debounce timer resets
User types 't' (within 50ms)
    → Debounce timer resets
... 50ms passes with no input ...
    → Fire keystroke middleware chain async
    → When result arrives:
        - If input hasn't changed: update UI
        - If input changed: discard results (stale)
```

### Trigger Types

Different actions trigger middleware with different urgency:

| Trigger  | When               | Debounce        | Use Case                             |
| -------- | ------------------ | --------------- | ------------------------------------ |
| `change` | User types/deletes | 50ms            | Highlighting, basic predictions      |
| `tab`    | User presses Tab   | 0ms (immediate) | Tab completion must feel instant     |
| `idle`   | Extended pause     | 300ms           | Heavy operations like LLM prediction |

### Chain Execution

```
Middleware 1 (user's custom)
    ↓ next(ctx)
Middleware 2 (user's custom)
    ↓ next(ctx)
Default Middleware (from defaults/middleware.gsh)
    ↓ next(ctx)
Go Fallback (built-in highlighter, predictor, completer)
    → Returns { highlight, prediction, completions }

Results bubble back up, each middleware can override/merge
```

## Default Implementation

The default middleware in `defaults/middleware.gsh` would delegate to Go's implementation:

```gsh
tool __defaultKeystrokeMiddleware(ctx, next) {
    result = next(ctx)  // Get Go fallback results

    // Could enhance/override here in the future
    // For now, pass through Go's implementation

    return result
}

gsh.useKeystrokeMiddleware(__defaultKeystrokeMiddleware)
```

## Example: Custom Middleware

### Custom Syntax Highlighting for a DSL

```gsh
tool queryHighlighter(ctx, next) {
    result = next(ctx)

    if (ctx.input.startsWith("@query")) {
        // Custom highlighting for query DSL
        result.highlight = highlightQueryDSL(ctx.input)
    }

    return result
}

gsh.useKeystrokeMiddleware(queryHighlighter)
```

### Custom Completions for kubectl

```gsh
tool k8sCompleter(ctx, next) {
    result = next(ctx)

    if (ctx.trigger == "tab" && ctx.hints.firstWord == "kubectl") {
        // Add kubernetes-specific completions
        pods = exec("kubectl get pods -o name").split("\n")
        result.completions = pods.map(p => p.replace("pod/", ""))
    }

    return result
}

gsh.useKeystrokeMiddleware(k8sCompleter)
```

### Custom Prediction Source

```gsh
tool historyPredictor(ctx, next) {
    result = next(ctx)

    if (ctx.trigger == "idle" && result.prediction == null) {
        // Add prediction from command history matching prefix
        match = gsh.history.findPrefix(ctx.input)
        if (match != null) {
            result.prediction = match.substring(ctx.input.length())
        }
    }

    return result
}

gsh.useKeystrokeMiddleware(historyPredictor)
```

## Open Design Questions

### 1. Highlight Format

Should highlight return:

- **Option A: Styled string** (with ANSI codes)

  - Pros: Simpler, direct output
  - Cons: Harder to compose, middleware can't easily merge highlights

- **Option B: Span array**

  - Pros: Structured, easy to merge, can define standard style names
  - Cons: More complex, needs Go-side rendering

- **Option C: Both** - accept either format
  - Pros: Flexibility
  - Cons: Complexity in handling

**Leaning toward:** Option C with span array as the "preferred" format.

### 2. Trigger Granularity

Current triggers: `change`, `tab`, `idle`

Should we add more?

- `enter` - Before command middleware runs (for last-second transforms)?
- `cursor` - Cursor moved but input unchanged?
- `paste` - Large input pasted (might want different debounce)?

**Leaning toward:** Start minimal (`change`, `tab`, `idle`), add more if needed.

### 3. Go Fallback Behavior

Options:

- **Option A:** Go fallback always runs at end of chain, middleware wraps it
- **Option B:** Go fallback only runs if no middleware is registered
- **Option C:** Go fallback runs, but middleware can return `{ useDefault: false }` to suppress

**Leaning toward:** Option A - most flexible, matches event system.

### 4. Debounce Configuration

Should users be able to configure debounce timing?

```gsh
gsh.config.keystroke = {
    debounceMs: 50,        // For regular changes
    idleMs: 300,           // Before firing "idle" trigger
    tabDebounceMs: 0,      // Tab is immediate
}
```

**Leaning toward:** Yes, but with sensible defaults.

### 5. Cancellation

If user keeps typing while middleware is running, should we:

- **Option A:** Let it finish but discard results (simpler)
- **Option B:** Actually cancel the execution (more complex, needs interpreter support)

**Leaning toward:** Option A for simplicity. The debounce + stale-check should handle most cases.

### 6. Performance Budget

What's an acceptable latency for keystroke middleware?

- Highlighting: < 5ms (feels instant)
- Completion: < 50ms (feels responsive)
- Prediction: < 200ms (can be slightly delayed)

Should we have timeouts that fall back to Go implementation if script is too slow?

### 7. Style Names for Spans

If using span-based highlighting, what standard style names should we support?

```gsh
// Potential standard styles
"command"      // Valid command (green)
"invalid"      // Invalid command (red/underline)
"argument"     // Command arguments
"flag"         // --flags
"string"       // "quoted strings"
"variable"     // $VAR
"operator"     // | && || etc.
"comment"      // # comments
```

## Implementation Plan

### Phase 1: API Rename (Current)

Rename existing middleware API to be more specific:

- `gsh.use()` → `gsh.useCommandMiddleware()`
- `gsh.remove()` → `gsh.removeCommandMiddleware()`

### Phase 2: Keystroke Middleware Infrastructure

1. Create `KeystrokeMiddlewareManager` in interpreter
2. Add `gsh.useKeystrokeMiddleware()` / `gsh.removeKeystrokeMiddleware()`
3. Implement debounce + async execution in Go
4. Wire up to REPL input handling

### Phase 3: Migrate Existing Features

1. Create Go-side "fallback" that wraps existing highlighter/predictor/completer
2. Add default keystroke middleware to `defaults/middleware.gsh`
3. Test that existing behavior is preserved

### Phase 4: Documentation & Examples

1. Document keystroke middleware in tutorial
2. Provide example middleware for common use cases
3. Document performance considerations

## Related Files

- `spec/GSH_MIDDLEWARE_SPEC.md` - Command middleware specification
- `internal/script/interpreter/middleware.go` - Current middleware implementation
- `internal/repl/input/highlight.go` - Current syntax highlighter
- `internal/repl/input/prediction.go` - Current prediction system
- `internal/repl/input/completion.go` - Current completion system
