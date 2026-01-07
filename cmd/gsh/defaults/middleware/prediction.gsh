# Default command prediction middleware.
# This handler listens to repl.predict and returns a predicted command string.
# 
# The handler receives a context with:
#   - input: string - the current input text
#   - trigger: "instant" | "debounced" - the prediction trigger type
#
# For "instant" trigger: Only fast operations (like history lookup) should run.
# For "debounced" trigger: Slower operations (like LLM calls) can run.

# Build lightweight context for the LLM using built-in facilities.
tool __predictionContext() {
    contextParts = []

    cwdResult = exec("pwd")
    if (cwdResult.exitCode == 0 && cwdResult.stdout != null) {
        contextParts.push(`<cwd>${cwdResult.stdout.trim()}</cwd>`)
    }

    gitResult = exec("git status --short --branch 2>/dev/null")
    if (gitResult.exitCode == 0 && gitResult.stdout != null && gitResult.stdout.trim() != "") {
        contextParts.push(`<git>${gitResult.stdout.trim()}</git>`)
    }

    if (gsh.lastCommand != null) {
        lastCmd = gsh.lastCommand.command
        lastExit = gsh.lastCommand.exitCode
        lastDuration = gsh.lastCommand.durationMs
        contextParts.push(`<last_command exit="${lastExit}" duration_ms="${lastDuration}">${lastCmd}</last_command>`)
    }

    return contextParts.join("\n")
}

# Agent used for predictions (prefix + null-state)
agent __predictionAgent {
    model: gsh.models.lite,
    systemPrompt: "You are gsh, an intelligent shell program. Respond ONLY with JSON using the schema: {\"predicted_command\": \"...\"}",
    tools: [],
    metadata: {
      hidden: true,
      streaming: false,
    },
}

# Try to get a prediction from command history
tool __historyPredict(input) {
    if (input == null || input == "") {
        return null
    }
    
    # Use gsh.history.findPrefix to search command history
    # Returns an array of { command, exitCode, timestamp } objects
    entries = gsh.history.findPrefix(input, 30)
    
    # Find the first successful command (exitCode == 0)
    for (entry of entries) {
        if (entry.exitCode == 0) {
            return entry.command
        }
    }
    
    return null
}

# Get LLM-based prediction
tool __llmPredict(input) {
    # Ensure prediction model is available
    if (gsh.models == null || gsh.models.lite == null) {
        return null
    }

    context = __predictionContext()
    bestPractices = "* Git commit messages should follow conventional commit message format"

    if (input == null) {
        input = ""
    }

    userMessage = ""
    if (input == "") {
        userMessage = `You are asked to predict the next command I'm likely to want to run.

# Instructions
* Based on the context, analyze my potential intent
* Your prediction must be a valid, single-line, complete bash command

# Best Practices
${bestPractices}

# Latest Context
${context}

Respond with JSON in this format: {"predicted_command": "your prediction here"}

Now predict what my next command should be.`
    } else {
        userMessage = `You will be given a partial bash command prefix entered by me, enclosed in <prefix> tags.
You are asked to predict what the complete bash command is.

# Instructions
* Based on the prefix and other context, analyze my potential intent
* Your prediction must start with the partial command as a prefix
* Your prediction must be a valid, single-line, complete bash command

# Best Practices
${bestPractices}

# Latest Context
${context}

Respond with JSON in this format: {"predicted_command": "your prediction here"}

<prefix>${input}</prefix>`
    }

    conv = userMessage | __predictionAgent
    if (conv == null) {
        return null
    }

    lastMessage = conv.lastMessage
    if (lastMessage == null || lastMessage.content == null) {
        return null
    }

    prediction = ""
    try {
        parsed = JSON.parse(lastMessage.content)
        prediction = parsed.predicted_command
    } catch (e) {
        return null
    }

    if (prediction == null || prediction == "") {
        return null
    }

    if (input != "" && !prediction.startsWith(input)) {
        return null
    }

    return prediction
}

tool __onPredict(ctx, next) {
    input = ctx.input
    trigger = ctx.trigger

    # Skip agent chat messages
    if (input != null && input.startsWith("#")) {
        return next(ctx)
    }

    # For "instant" trigger: only check history (must be fast!)
    if (trigger == "instant") {
        historyMatch = __historyPredict(input)
        if (historyMatch != null) {
            return { prediction: historyMatch }
        }
        # No instant prediction available
        return next(ctx)
    }

    # For "debounced" trigger: try history first, then LLM
    historyMatch = __historyPredict(input)
    if (historyMatch != null) {
        return { prediction: historyMatch }
    }

    # Fall back to LLM prediction
    llmPrediction = __llmPredict(input)
    if (llmPrediction != null) {
        return { prediction: llmPrediction }
    }

    return next(ctx)
}

gsh.use("repl.predict", __onPredict)
