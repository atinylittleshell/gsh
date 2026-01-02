# Models

This chapter documents the model tier system and how to configure AI models.

**Availability:** REPL + Script

## `gsh.models`

The `gsh.models` object provides a tiered model system for different tasks.

### Model Tiers

| Tier                   | Description                   | Typical Use                                            |
| ---------------------- | ----------------------------- | ------------------------------------------------------ |
| `gsh.models.lite`      | Fast, lightweight model       | Command predictions, quick completions                 |
| `gsh.models.workhorse` | Capable general-purpose model | Agent tasks, code generation                           |
| `gsh.models.premium`   | Most capable model            | Complex reasoning (falls back to workhorse if not set) |

### Example

```gsh
# Assign models to tiers
gsh.models.lite = fastModel
gsh.models.workhorse = capableModel
gsh.models.premium = bestModel
```

### Dynamic Resolution

Model tiers support dynamic resolution—when you reference `gsh.models.lite` in an agent definition, gsh looks up the current value each time it's needed:

```gsh
# Change models at runtime
gsh.on("repl.ready", tool handler(ctx) {
    if (env.USE_LOCAL_MODELS == "true") {
        gsh.models.lite = localModel
        gsh.models.workhorse = localAgentModel
    }
})
```

Changes take effect immediately for subsequent predictions and agent calls.

## Model Declaration

Declare models using the `model` keyword:

```gsh
model myModel {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-5.2",
}
```

### Required Fields

| Field      | Type     | Description                                                     |
| ---------- | -------- | --------------------------------------------------------------- |
| `provider` | `string` | Provider type (currently `"openai"` for OpenAI-compatible APIs) |
| `apiKey`   | `string` | API key for authentication                                      |
| `model`    | `string` | Model identifier                                                |

### Optional Fields

| Field     | Type     | Description                                 |
| --------- | -------- | ------------------------------------------- |
| `baseURL` | `string` | API endpoint URL (defaults to OpenAI's API) |

## Provider Examples

### OpenAI

```gsh
model gpt4 {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-5.2",
}
```

### Ollama (Local Models)

[Ollama](https://ollama.com) allows you to run models locally with an OpenAI-compatible API.

```gsh
model gemma {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

gsh.models.lite = gemma
```

**Key points:**

- Use `provider: "openai"` (Ollama is OpenAI-compatible)
- Set `apiKey: "ollama"` (required placeholder)
- Use `baseURL: "http://localhost:11434/v1"`
- Model name should match output from `ollama list`

### OpenRouter

[OpenRouter](https://openrouter.ai) provides access to multiple models through a single API.

```gsh
model openrouterModel {
    provider: "openai",
    apiKey: env.OPENROUTER_API_KEY,
    baseURL: "https://openrouter.ai/api/v1",
    model: "anthropic/claude-opus-4.5",
}
```

**Key points:**

- Use `provider: "openai"` (OpenRouter is OpenAI-compatible)
- Get your API key from https://openrouter.ai
- Model names use format `{provider}/{model-name}`

## Environment Variables

Store API keys in environment variables rather than in config files:

```bash
# ~/.gshenv or ~/.bashrc
export OPENAI_API_KEY="sk-..."
export OPENROUTER_API_KEY="sk-or-..."
```

Reference them in your config:

```gsh
model gpt4 {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-5.2",
}
```

## Troubleshooting

### Model not responding

1. Check your API key: `echo $OPENAI_API_KEY`
2. Verify the model name is correct
3. Enable debug logging: `gsh.logging.level = "debug"`
4. Check logs: `tail -f ~/.gsh/gsh.log`

### Ollama connection refused

1. Ensure Ollama is running: `ollama serve`
2. Check it's listening: `curl http://localhost:11434/api/tags`
3. Verify model is pulled: `ollama list`

---

**Next:** [Tools](03-tools.md) - Built-in tools for agents
