# Chapter 04: Command Prediction with LLMs

gsh can use AI models to predict and suggest shell commands as you type. This chapter shows you how to configure and use this feature.

## What is Command Prediction?

As you type a command, gsh can:

- **Suggest completions** based on your command history and context
- **Learn your patterns** from previous commands you've run
- **Understand your intent** using AI models

For example, if you usually run `git status` after editing files, gsh might predict it as you start typing `git st...`

## How Prediction Works

The REPL uses a two-stage prediction system:

1. **Prefix-Based Prediction** (fast, no LLM)

   - Looks through your command history
   - Finds commands starting with what you've typed
   - Returns the most recent match

2. **LLM-Based Prediction** (intelligent, requires model)
   - Takes your current input and context (directory, git status, etc.)
   - Sends to a configured AI model
   - Returns a predicted command

The REPL tries the fast approach first, then falls back to LLM prediction if needed.

## Setting Up Prediction

### Step 1: Choose a Model

For prediction, you want a **small, fast model** since predictions must be instant. We recommend:

- **Ollama models** (free, local):

  - `gemma3:1b` - Small, fast, good quality
  - `phi3` - Optimized for instruction following
  - `orca-mini` - Good balance of speed and capability

- **OpenAI models** (requires API key):

  - `gpt-4o-mini` - Fast and capable

- **Anthropic models** (requires API key):
  - `claude-opus-4-mini` - Excellent quality but slower

### Step 2: Add to `.gshrc.gsh`

Create or update `~/.gshrc.gsh`:

```gsh
# ~/.gshrc.gsh

# Define a model for predictions (small, fast)
model PredictModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

# Tell gsh to use this model for predictions
GSH_CONFIG = {
    predictModel: PredictModel,
}
```

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
gsh> git st[PREDICTION: git status]
```

Accept it with:

- **Tab** - Accept the prediction
- **Right Arrow** (End key) - Accept the prediction
- **Enter** - Run with the prediction
- **Backspace** - Reject and edit

### Viewing Without Accepting

You can view a prediction without running it. The predicted text appears as a hint after your cursor.

### Disabling Predictions

To disable predictions temporarily:

```bash
gsh> # Just type normally, predictions won't interfere
```

To disable permanently, update `GSH_CONFIG`:

```gsh
# ~/.gshrc.gsh
model PredictModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

GSH_CONFIG = {
    predictModel: null,  # Disable predictions
}
```

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

First, install Ollama from [ollama.ai](https://ollama.ai), then:

```bash
# Pull a model
ollama pull gemma3:1b

# Start the Ollama server (usually runs on port 11434)
ollama serve
```

Then in `~/.gshrc.gsh`:

```gsh
model LocalPredictor {
    provider: "openai",
    apiKey: "ollama",  # Magic value for local Ollama
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

GSH_CONFIG = {
    predictModel: LocalPredictor,
}
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

### OpenAI

Create an account at [platform.openai.com](https://platform.openai.com) and get an API key.

In `~/.gshrc.gsh`:

```gsh
model OpenAIPredictor {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-4o-mini",
}

GSH_CONFIG = {
    predictModel: OpenAIPredictor,
}
```

Set your API key:

```bash
export OPENAI_API_KEY="sk-..."
```

**Advantages:**

- Very high quality predictions
- No setup required
- Works anywhere

**Disadvantages:**

- Costs money (though gpt-4o-mini is inexpensive)
- Sends commands to external service
- Requires internet connection

### Anthropic (Claude)

Get an API key from [console.anthropic.com](https://console.anthropic.com).

In `~/.gshrc.gsh`:

```gsh
model ClaudePredictor {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-opus-4-mini",
}

GSH_CONFIG = {
    predictModel: ClaudePredictor,
}
```

Set your API key:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

**Advantages:**

- Excellent quality
- Privacy-conscious company
- Good for private repositories

**Disadvantages:**

- Costs money
- API calls required
- Requires internet

## Advanced Configuration

### Custom Prediction Prompt

The prediction system sends context to the model. To customize what context is sent, modify your configuration:

```gsh
# ~/.gshrc.gsh

model PredictModel {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

# Store configuration
GSH_CONFIG = {
    predictModel: PredictModel,

    # Optional: prediction settings
    // These would be actual config options if supported
}
```

### Hybrid Configuration

Use a fast local model for most predictions, but fall back to a powerful cloud model for complex commands:

```gsh
# ~/.gshrc.gsh

# Fast local model
model FastPredictor {
    provider: "openai",
    apiKey: "ollama",
    baseURL: "http://localhost:11434/v1",
    model: "gemma3:1b",
}

# Powerful cloud model (for scripting, not used by default)
model PowerfulPredictor {
    provider: "openai",
    apiKey: env.OPENAI_API_KEY,
    model: "gpt-4o",
}

# Use fast model by default
GSH_CONFIG = {
    predictModel: FastPredictor,
}
```

## Troubleshooting

### Predictions Not Appearing

1. Check that `predictModel` is set in `GSH_CONFIG`:

   ```gsh
   print(GSH_CONFIG)
   ```

2. Verify the model is reachable:

   ```bash
   # For Ollama
   curl http://localhost:11434/api/generate -d '{"model":"gemma3:1b","prompt":"hello","stream":false}'

   # For OpenAI
   curl -H "Authorization: Bearer $OPENAI_API_KEY" https://api.openai.com/v1/models
   ```

3. Check logs:
   ```bash
   tail -f ~/.gsh.log
   ```

### Predictions Are Slow

1. Switch to a smaller model:

   ```gsh
   model: "phi3",  # Smaller than gemma3:1b
   ```

2. Disable GPU (sometimes slower):

   ```bash
   # For Ollama, see their documentation
   ```

3. Reduce context being sent to model (this is automatic)

### Predictions Are Wrong

The model is learning from your history. Solutions:

1. Clear and rebuild history by running commands more often
2. Use a more capable model:

   ```gsh
   model: "gpt-4o-mini",
   ```

3. Add more context clues by being explicit in commands

### API Errors

**"Connection refused" for Ollama:**

```bash
# Start Ollama server
ollama serve
```

**"Invalid API key" for OpenAI:**

```bash
# Verify key is correct and not expired
echo $OPENAI_API_KEY

# Get a new key from platform.openai.com
```

**"Model not found":**

```bash
# For Ollama, pull the model first
ollama pull gemma3:1b

# For other providers, check model name spelling
```

## Performance Tips

1. **Use a small model** - Prediction must be nearly instant
2. **Keep history clean** - Don't have gigabytes of history
3. **Consider local-first** - Ollama is faster than cloud APIs
4. **Monitor costs** - Set up billing alerts if using cloud APIs
5. **Cache results** - gsh caches predictions automatically

## Privacy Considerations

- **Local models (Ollama)** - Everything stays on your machine
- **Cloud models** - Commands are sent to the provider's servers
- **Check provider policies** - Most treat this data securely, but verify

If privacy is critical, use Ollama locally.

## What's Next?

Predictions make you faster! Chapter 05 covers **Agents in the REPL**—how to use AI agents directly from your shell to perform complex tasks, not just predict commands.

---

**Previous Chapter:** [Chapter 03: Custom Prompts with Starship](03-custom-prompts-with-starship.md)

**Next Chapter:** [Chapter 05: Agents in the REPL](05-agents-in-the-repl.md)
