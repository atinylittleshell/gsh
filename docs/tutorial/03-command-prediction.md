# Chapter 03: Command Prediction with LLMs

gsh can use AI models to predict and suggest shell commands as you type.
This chapter shows you how to configure and use this feature.

## What is Command Prediction?

As you type a command, gsh can:

- **Understand your intent** using AI models
- **Suggest completions** based on your command history and context
- **Learn your patterns** from previous commands you've run

For example, if you usually run `git status` after editing files, gsh might predict it before you even start typing `git st...`

## How Prediction Works

The REPL uses a two-stage prediction system by default:

1. **Prefix-Based Prediction** (fast, no LLM)

   - Looks through your command history
   - Finds commands starting with what you've typed
   - Returns the most recent match

2. **LLM-Based Prediction** (intelligent, requires model)
   - Takes your current input and context (directory, git status, etc.)
   - Sends to a configured AI model
   - Returns a predicted command

The REPL by default tries the fast approach first, then falls back to LLM prediction if needed.

## Setting Up Prediction

### Step 1: Choose a Model

For prediction, you want a **small, fast model** since predictions must be instant.

We recommend using `gemma3:1b` from Ollama locally. It's free, fast, and works offline.

### Step 2: Add to `repl.gsh`

Create or update `~/.gsh/repl.gsh`:

```gsh
# ~/.gsh/repl.gsh

# Define a model for predictions (small, fast)
model myPredictModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

# Tell gsh to use this model for predictions
gsh.models.lite = myPredictModel
```

> **Note:** `gsh.models.lite` uses dynamic resolution - if you change it later (e.g., in an event handler), predictions will automatically use the new model. See [Configuring Models](../repl/02-configuring-models.md#dynamic-model-resolution) for details.

### Step 3: Test It

Start a new gsh session:

```bash
gsh
```

Start typing a command you've run before:

```bash
gsh> git st
```

After a moment, you should see a prediction appear. Press **Tab** or **Right Arrow** to accept it, or keep typing to ignore it.

## Using Predictions in the REPL

### Accepting a Prediction

When you see a prediction:

```bash
gsh> git st[atus]
```

Accept it with:

- **Tab** - Accept the prediction
- **Right Arrow** - Accept the prediction
- **Ctrl+F** - Accept the prediction
- **Enter** - Run without the prediction
- **Backspace** - Reject and edit

### Viewing Without Accepting

You can view a prediction without running it. The predicted text appears as a hint after your cursor.

## Understanding Prediction Context

The prediction system considers several factors:

### 1. Current Working Directory

If you're in a git repository:

```bash
gsh> cd my-project
gsh> git  # Prediction might suggest "git status"
```

If you're in a data directory:

```bash
gsh> cd data
gsh> ls  # Prediction might suggest "ls -la" or "head"
```

### 2. Command History

The model learns from your habits:

```bash
gsh> npm run build    # You often follow this with npm test
gsh> npm run         # Prediction suggests "npm run test"
```

### 3. Git Status

If you're in a dirty repository:

```bash
gsh> git             # Prediction might suggest "git status" or "git add"
```

### 4. Time Context

The model might suggest time-appropriate commands:

```bash
gsh> ./build-and-   # Might complete to "build-and-test" or "build-and-deploy"
```

## Configuring Prediction Models

### Local Model with Ollama

First, install Ollama from [Ollama.com](https://ollama.com), then:

```bash
# Pull a model
ollama pull gemma3:1b
```

Then in `~/.gsh/repl.gsh`:

```gsh
model myPredictModel {
    provider: "openai",
    apiKey: "ollama",  # Magic value for local Ollama
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

gsh.models.lite = myPredictModel
```

**Advantages:**

- Free and private
- Works offline
- No API rate limits
- Instant predictions

**Disadvantages:**

- Requires local hardware
- Smaller models have lower quality
- Setup complexity

### Use a Hosted Model (e.g. OpenAI)

Create an account at [platform.openai.com](https://platform.openai.com) and get an API key.

In `~/.gsh/repl.gsh`:

```gsh
model OpenAIPredictor {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-5-mini",
}

gsh.models.lite = OpenAIPredictor
```

Set your API key:

```bash
export OPENAI_API_KEY="sk-..."
```

**Advantages:**

- Higher quality predictions
- No local setup required
- Works anywhere

**Disadvantages:**

- Costs money (though gpt-5-mini is relatively inexpensive)
- Higher latency
- Sends commands to external service
- Requires internet connection

## Troubleshooting

### Predictions Not Appearing

1. Check that `gsh.models.lite` is set:

   ```gsh
   print(gsh.models.lite)
   ```

2. Verify the model is reachable:

   ```bash
   # For example
   curl http://localhost:11434/api/generate -d '{"model":"gemma3:1b","prompt":"hello","stream":false}'
   ```

3. Check logs:
   ```bash
   tail -f ~/.gsh/gsh.log
   ```

### Predictions Are Slow

Try switching to a smaller model:

```gsh
model: "gemma3:270m",  # Smaller than gemma3:1b
```

### Predictions Are Wrong

Try using a more capable model:

```gsh
model: "gpt-5-mini",
```

### API Errors

**"Connection refused" for Ollama:**

```bash
# Start Ollama server
ollama serve
```

**"Invalid API key" for hosted models:**

```bash
# Verify key is correct and not expired
echo $OPENAI_API_KEY
```

**"Model not found":**

```bash
# For Ollama, pull the model first
ollama pull gemma3:1b

# For other providers, check model name spelling
```

## Privacy Considerations

- **Local models (Ollama)** - Everything stays on your machine
- **Cloud models** - Commands are sent to the provider's servers and may be logged
- **Check provider policies** - Most treat this data securely, but verify

## What's Next?

Predictions make you faster! Chapter 04 covers **Agents in the REPL**—how to use AI agents directly from your shell to perform complex tasks.

---

**Previous Chapter:** [Chapter 02: Configuration](02-configuration.md)

**Next Chapter:** [Chapter 04: Agents in the REPL](04-agents-in-the-repl.md)
