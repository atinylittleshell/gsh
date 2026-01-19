# Default command prediction middleware.
# This handler listens to repl.predict and returns a predicted command string.
# 
# The handler receives a context with:
#   - input: string - the current input text
#   - trigger: "instant" | "debounced" - the prediction trigger type
#   - existingPrediction: string | null - the current prediction (if any)
#
# For "instant" trigger: Only fast operations (like history lookup) should run.
# For "debounced" trigger: Slower operations (like LLM calls) can run.
#
# The middleware is responsible for deciding whether to keep an existing prediction
# or generate a new one. This allows special cases like VCS commit messages to
# always get fresh predictions even if there's an existing prefix match.

import { parseVcsCommitMessage, isVcsCommitMessage, commitMessageInstructions } from "./vcs_commit.gsh"

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

    # Include last 10 history commands with metadata (chronological order, most recent at bottom)
    recentHistory = gsh.history.getRecent(10)
    if (recentHistory != null && recentHistory.length > 0) {
        historyLines = []
        for (entry of recentHistory) {
            historyLines.push(`<cmd exit="${entry.exitCode}">${entry.command}</cmd>`)
        }
        contextParts.push(`<history description="chronological order, most recent command last">\n${historyLines.join("\n")}\n</history>`)
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
            # Use fast check (no diff execution) to skip VCS commit messages
            if (!isVcsCommitMessage(entry.command)) {
              return entry.command
            } else {
              return null
            }
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

    if (input == null) {
        input = ""
    }

    # Special case: VCS commit/describe message - include changes and specific instructions
    vcsInfo = parseVcsCommitMessage(input)
    changesContext = ""
    commitInstructions = ""
    
    if (vcsInfo != null && vcsInfo.changes != null) {
        changesContext = `
# Changes to be Committed
<diff>
${vcsInfo.changes}
</diff>`
        commitInstructions = commitMessageInstructions
    }

    userMessage = ""
    if (input == "") {
        userMessage = `You are asked to predict the next command I'm likely to want to run.

# Instructions
* Based on the context, analyze my potential intent
* Your prediction must be a valid, single-line, complete bash command

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
${commitInstructions}

# Latest Context
${context}
${changesContext}

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
        content = lastMessage.content
        # Extract JSON from surrounding text (LLM might add extra text around the JSON)
        firstBrace = content.indexOf("{")
        lastBrace = content.lastIndexOf("}")
        if (firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace) {
            content = content.substring(firstBrace, lastBrace + 1)
        }
        parsed = JSON.parse(content)
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
    existingPrediction = ctx.existingPrediction

    # Skip agent chat messages
    if (input != null && input.startsWith("#")) {
        return next(ctx)
    }

    # Check if this is a VCS commit/describe message command (fast check, no diff)
    isCommitMessage = isVcsCommitMessage(input)

    # Check if we should keep the existing prediction
    # For regular commands: keep if existing prediction starts with input (prefix match)
    # For VCS commit messages: always generate fresh (don't keep stale suggestions)
    if (existingPrediction != null && existingPrediction != "") {
        if (!isCommitMessage && existingPrediction.startsWith(input)) {
            # Keep the existing prediction - it's still valid
            return { prediction: existingPrediction }
        }
    }

    # Whether it's "instant" or "debounced", try history first (unless commit message)
    if (!isCommitMessage) {
        historyMatch = __historyPredict(input)
        if (historyMatch != null) {
            return { prediction: historyMatch }
        }
    }

    # For "instant" trigger: that's it - don't use LLM which is heavy. wait for debounced trigger
    if (trigger == "instant") {
        return next(ctx)
    }

    # For "debounced" trigger we use LLM prediction if no history based prediction done above
    llmPrediction = __llmPredict(input)
    if (llmPrediction != null) {
        return { prediction: llmPrediction }
    }

    return next(ctx)
}

gsh.use("repl.predict", __onPredict)
