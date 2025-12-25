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
