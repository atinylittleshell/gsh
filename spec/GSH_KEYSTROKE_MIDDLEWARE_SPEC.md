# GSH Keystroke Middleware Specification

This document specifies a **keystroke middleware** system that extends the existing command middleware to enable scriptable syntax highlighting, command prediction, and tab completion.

## Status: Draft / Partially Implemented

This spec captures design thinking for future implementation. The existing command middleware (`gsh.useCommandMiddleware`) handles input after submission; this spec covers input processing _during_ typing.

**Current implementation status:**

- ✅ `repl.predict` - Fully implemented with trigger context
- 🔲 `repl.highlight` - Not yet implemented
- 🔲 `repl.completion` - Not yet implemented

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

## Architecture: Separate Events Model

Rather than a single unified `repl.keystroke` event, gsh uses **separate events** for each feature. This allows:

1. **Independent timing** - Each feature can have its own debounce/trigger behavior
2. **Simpler middleware** - Each handler focuses on one concern
3. **Gradual migration** - Features can be moved to scripts incrementally
4. **Better performance** - Only run the middleware needed for each trigger

```
┌─────────────────────────────────────────────────────────────────┐
│                     User Types in REPL                          │
└─────────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  repl.predict   │  │  repl.highlight │  │ repl.completion │
│  (ghost text)   │  │  (syntax color) │  │  (tab menu)     │
├─────────────────┤  ├─────────────────┤  ├─────────────────┤
│ trigger:        │  │ trigger:        │  │ trigger:        │
│ • "instant"     │  │ • "change"      │  │ • "tab"         │
│ • "debounced"   │  │                 │  │                 │
│                 │  │ debounce: 50ms  │  │ debounce: 0ms   │
│ instant: 0ms    │  │                 │  │ (immediate)     │
│ debounced: 200ms│  │                 │  │                 │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

## Event: repl.predict (Implemented)

The `repl.predict` event handles command prediction (ghost text suggestions).

### Trigger Types

| Trigger     | When               | Debounce | Use Case                   |
| ----------- | ------------------ | -------- | -------------------------- |
| `instant`   | Every input change | 0ms      | Fast predictions (history) |
| `debounced` | After typing pause | 200ms    | Slow predictions (LLM)     |

The Go layer calls `repl.predict` twice per input change:

1. **Instant call** (`trigger: "instant"`) - for fast, synchronous predictions
2. **Debounced call** (`trigger: "debounced"`) - for slow, async predictions (only if instant returned no result)

### Context Object

```gsh
ctx = {
    input: "git commit -m",     // Current input text
    trigger: "instant",         // "instant" | "debounced"
}
```

### Result Object

```gsh
{
    prediction: "git commit -m \"fix: description\"",  // Full predicted command, or null
}
```

### Default Implementation

The default prediction middleware in `cmd/gsh/defaults/middleware/prediction.gsh`:

```gsh
tool __onPredict(ctx, next) {
    input = ctx.input
    trigger = ctx.trigger

    # Skip agent chat messages
    if (input != null && input.startsWith("#")) {
        return next(ctx)
    }

    # For instant trigger, only check history (must be fast!)
    if (trigger == "instant") {
        if (input != null && input != "") {
            match = gsh.history.findPrefix(input, 10)
            if (match != null) {
                return { prediction: match }
            }
        }
        return next(ctx)
    }

    # For debounced trigger, try history first, then LLM
    if (input != null && input != "") {
        match = gsh.history.findPrefix(input, 10)
        if (match != null) {
            return { prediction: match }
        }
    }

    # Fall back to LLM prediction...
    # (LLM logic here)

    return next(ctx)
}

gsh.use("repl.predict", __onPredict)
```

### SDK: gsh.history.findPrefix()

```gsh
// Find the most recent command that starts with the given prefix
// Returns the full command string, or null if no match
match = gsh.history.findPrefix(prefix, limit)

// Parameters:
//   prefix (string): The prefix to search for
//   limit (number): Maximum number of history entries to search (default: 10)
//
// Returns:
//   string | null: The most recent matching command, or null
```

## Event: repl.highlight (Not Yet Implemented)

The `repl.highlight` event handles syntax highlighting.

### Context Object

```gsh
ctx = {
    input: "git commit -m \"fix bug\"",
    cursorPos: 14,
}
```

### Result Object

```gsh
{
    // Option A: Pre-rendered string with ANSI codes
    highlight: "\x1b[32mgit\x1b[0m commit -m \"fix bug\"",

    // Option B: Structured spans (easier to compose/merge)
    highlight: [
        { start: 0, end: 3, style: "command" },
        { start: 4, end: 10, style: "subcommand" },
        { start: 11, end: 13, style: "flag" },
        { start: 14, end: 23, style: "string" },
    ],
}
```

## Event: repl.completion (Not Yet Implemented)

The `repl.completion` event handles tab completion.

### Context Object

```gsh
ctx = {
    input: "git comm",
    cursorPos: 8,
    word: {
        text: "comm",
        start: 4,
        end: 8,
    },
    hints: {
        firstWord: "git",
        commandExists: true,
    },
}
```

### Result Object

```gsh
{
    // Option A: Rich objects
    completions: [
        { value: "commit", description: "Record changes" },
        { value: "checkout", description: "Switch branches" },
    ],
    // Option B: Simple strings
    completions: ["commit", "checkout", "cherry-pick"],
}
```

## Example: Custom Middleware

### Custom Prediction Source

```gsh
tool historyPredictor(ctx, next) {
    # Check history first for any trigger type
    if (ctx.input != null && ctx.input != "") {
        match = gsh.history.findPrefix(ctx.input, 10)
        if (match != null) {
            return { prediction: match }
        }
    }

    # Fall through to default (LLM) prediction
    return next(ctx)
}

gsh.use("repl.predict", historyPredictor)
```

### Custom Completions for kubectl (Future)

```gsh
tool k8sCompleter(ctx, next) {
    result = next(ctx)

    if (ctx.hints.firstWord == "kubectl") {
        # Add kubernetes-specific completions
        pods = exec("kubectl get pods -o name").split("\n")
        result.completions = pods.map(p => p.replace("pod/", ""))
    }

    return result
}

gsh.use("repl.completion", k8sCompleter)
```

### Custom Syntax Highlighting for a DSL (Future)

```gsh
tool queryHighlighter(ctx, next) {
    result = next(ctx)

    if (ctx.input.startsWith("@query")) {
        # Custom highlighting for query DSL
        result.highlight = highlightQueryDSL(ctx.input)
    }

    return result
}

gsh.use("repl.highlight", queryHighlighter)
```

## Open Design Questions

### 1. Highlight Format (for repl.highlight)

Should highlight return:

- **Option A: Styled string** (with ANSI codes)
- **Option B: Span array** (structured, easier to merge)
- **Option C: Both** - accept either format

**Leaning toward:** Option C with span array as the "preferred" format.

### 2. Debounce Configuration

Should users be able to configure debounce timing per event?

```gsh
gsh.config.predict = {
    instantEnabled: true,
    debouncedDelayMs: 200,
}
```

**Leaning toward:** Yes, but with sensible defaults.

### 3. Cancellation

If user keeps typing while middleware is running, should we:

- **Option A:** Let it finish but discard results (simpler)
- **Option B:** Actually cancel the execution (more complex)

**Decision:** Option A - the Go layer handles stale-check and discards outdated results.

### 4. Performance Budget

Target latencies:

- `repl.predict` (instant): < 10ms (history lookup)
- `repl.predict` (debounced): < 500ms (LLM call)
- `repl.highlight`: < 5ms
- `repl.completion`: < 50ms

### 5. Style Names for Spans (for repl.highlight)

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

### Phase 1: repl.predict Migration (Current)

1. ✅ Add `trigger` field to `repl.predict` context ("instant" | "debounced")
2. ✅ Add `gsh.history.findPrefix(prefix, limit)` SDK function
3. ✅ Update `prediction.gsh` to handle both triggers with history + LLM logic
4. ✅ Simplify Go prediction code to always delegate to gsh scripts

### Phase 2: repl.highlight (Future)

1. Add `repl.highlight` event
2. Create default highlighting middleware
3. Migrate Go highlighter to fallback

### Phase 3: repl.completion (Future)

1. Add `repl.completion` event
2. Create default completion middleware
3. Migrate Go completer to fallback

## Related Files

- `cmd/gsh/defaults/middleware/prediction.gsh` - Default prediction middleware
- `internal/repl/input/prediction.go` - Go prediction coordinator
- `internal/repl/predict/event_provider.go` - Event provider for repl.predict
- `internal/script/interpreter/repl_events.go` - Event context creation
- `internal/script/interpreter/builtin_sdk.go` - SDK functions including gsh.history
- `docs/sdk/05-events.md` - Event documentation
