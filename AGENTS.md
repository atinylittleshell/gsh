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
