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
