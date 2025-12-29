## Keeping code and docs in sync

When you made changes to gsh behavior, remember to update documentation under the `docs/` folder as needed.

## Writing gsh script examples

When writing a test case or code example that requires a custom model, always use a model from ollama like this:

```gsh
model exampleModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "devstral-small-2",
}
```

## Testing the binary

If you need to build the gsh binary for testing, you can just run `make build` or a custom build command outputting to `./bin/gsh`.

It's okay to overwrite the existing binary during testing.

## UI Colors and Styling

**Yellow is the primary UI color for gsh.** All UI elements that need highlighting or emphasis should use yellow (ANSI color 11).

The centralized color definitions are in `internal/repl/render/styles.go`:

- `ColorYellow` (ANSI 11) - Primary UI color for headers, success indicators, tool status, exec start, system messages, spinners
- `ColorRed` (ANSI 9) - Error indicators only
- `ColorGray` (ANSI 8) - Dim/secondary information like timing

When adding new UI elements:

1. **Always import and use the color constants** from `internal/repl/render/styles.go`
2. **Never hardcode color values** like `lipgloss.Color("12")` - use the centralized constants
3. For new styles, consider adding them to `styles.go` if they'll be reused
4. **Only style the symbol, not the message text** - When rendering messages with symbols (like `→`, `▶`, `✓`), apply color only to the symbol itself. Example: `SystemMessageStyle.Render(SymbolSystemMessage) + " " + message`

## Rendering Hooks

When modifying agent rendering (tool status, exec output, headers/footers), there are **two places** that need to be updated:

1. **Go fallback code** in `internal/repl/render/renderer.go` - used when hooks fail or return empty
2. **Default hook implementations** in `cmd/gsh/.gshrc.default.gsh` - the actual default behavior users see

Both should produce the same output format to maintain consistency.

## Code Organization

### Shared Utility Functions

The interpreter package (`internal/script/interpreter/`) contains canonical utility functions that should be reused rather than duplicated:

- `ValueToInterface(val Value) interface{}` - converts gsh Value to Go interface{}
- `InterfaceToValue(val interface{}) Value` - converts Go interface{} to gsh Value
- `CreateExecStartContext()` / `CreateExecEndContext()` - create event context objects

When implementing features that span multiple packages (e.g., interpreter and REPL), prefer:

1. Implementing canonical functions in the interpreter package
2. Exporting them (capitalize first letter) if needed by other packages
3. Having other packages import and call the canonical functions

### Avoiding Duplication

Before creating new helper functions for type conversion, context creation, or similar utilities:

1. Search for existing implementations. E.g. `grep -r "func.*ToInterface\|func.*ToValue" internal/`
2. Check `internal/script/interpreter/conversation.go` and `internal/script/interpreter/value.go` for common existing utilities
