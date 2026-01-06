# Default command prediction middleware.
# This handler listens to repl.predict and returns a predicted command string.

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
        lastExit = gsh.lastCommand.exitCode
        lastDuration = gsh.lastCommand.durationMs
        contextParts.push(`<last_command exit="${lastExit}" duration_ms="${lastDuration}"></last_command>`)
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
    },
}

tool __onPredict(ctx, next) {
    # Allow earlier middleware to override
    input = ctx.input

    # Skip agent chat messages
    if (input != null && input.startsWith("#")) {
        return next(ctx)
    }

    # Ensure prediction model is available
    if (gsh.models == null || gsh.models.lite == null) {
        return next(ctx)
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
        return next(ctx)
    }

    lastMessage = conv.lastMessage
    if (lastMessage == null || lastMessage.content == null) {
        return next(ctx)
    }

    prediction = ""
    try {
        parsed = JSON.parse(lastMessage.content)
        prediction = parsed.predicted_command
    } catch (e) {
        return next(ctx)
    }

    if (prediction == null || prediction == "") {
        return next(ctx)
    }

    if (input != "" && !prediction.startsWith(input)) {
        return next(ctx)
    }

    return { prediction: prediction }
}

gsh.use("repl.predict", __onPredict)
