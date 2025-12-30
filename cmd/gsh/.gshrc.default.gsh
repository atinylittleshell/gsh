# Default gsh configuration file
# This file is loaded before ~/.gshrc.gsh and provides sensible defaults

# Default model for predictions (lightweight, fast)
model GSH_PREDICT_MODEL {
    provider: "openai",
    apiKey: "ollama",
    model: "gemma3:1b",
    baseURL: "http://localhost:11434/v1",
}

# Default model for agent interactions (more capable)
model GSH_AGENT_MODEL {
    provider: "openai",
    apiKey: "ollama",
    model: "devstral-small-2",
    baseURL: "http://localhost:11434/v1",
}

# =============================================================================
# SDK Configuration
# =============================================================================

# Configure integrations
gsh.integrations.starship = true

# Set log level: "debug", "info", "warn", "error"
if (gsh.version == "dev") {
    gsh.logging.level = "debug"
} else {
    gsh.logging.level = "info"
}

# Configure REPL models
if (gsh.repl != null) {
    # Fast, lightweight for simple operations such as command prediction
    gsh.repl.models.lite = GSH_PREDICT_MODEL
    # Capable model for agentic work
    gsh.repl.models.workhorse = GSH_AGENT_MODEL
    # gsh.repl.models.premium not set - reserved for future high-value tasks
}

# =============================================================================
# Event Handlers
# =============================================================================
# These event handlers customize how agent interactions are displayed in the REPL.
# Override any of these in your ~/.gshrc.gsh to customize the appearance.

# Renders the header line when an agent starts responding
# Example output: "── gsh ─────────────────────────────"
tool onAgentStart(ctx) {
    width = gsh.terminal.width
    if (width > 80) {
        width = 80
    }
    name = gsh.repl.currentAgent.name
    if (name == "default") {
        name = "gsh"
    }
    padding = width - 4 - name.length  # "── " prefix (3) + " " before padding (1)
    if (padding < 3) {
        padding = 3
    }
    header = "── " + name + " " + "─".repeat(padding)
    print(gsh.ui.styles.primary(header))
}
gsh.on("agent.start", onAgentStart)

# Renders the footer line when an agent finishes responding
# Example output: "── 523 in · 324 out · 1.2s ────────────────────"
# Example with cache: "── 523 in (80% cached) · 324 out · 1.2s ─────"
# Tokens are formatted with K/M suffix for large numbers
# Duration uses appropriate units (ms, s, m, h, d)
tool onAgentEnd(ctx) {
    width = gsh.terminal.width
    if (width > 80) {
        width = 80
    }

    # Format tokens with K/M suffix
    tool formatTokens (count: int) {
        if (count >= 1000000) {
            return (count / 1000000).toFixed(2) + "M"
        }
        if (count >= 1000) {
            return (count / 1000).toFixed(1) + "K"
        }
        return "" + count
    }

    # Format duration with appropriate units
    tool formatDuration (durationMs: int) {
        if (durationMs < 1000) {
            return "" + durationMs + "ms"
        }
        durationSec = durationMs / 1000
        if (durationSec < 60) {
            return durationSec.toFixed(1) + "s"
        }
        durationMin = durationSec / 60
        if (durationMin < 60) {
            return durationMin.toFixed(1) + "m"
        }
        durationHour = durationMin / 60
        if (durationHour < 24) {
            return durationHour.toFixed(1) + "h"
        }
        return (durationHour / 24).toFixed(1) + "d"
    }

    # Build the text, including cache ratio next to input tokens if there are cached tokens
    inputStr = formatTokens(ctx.query.inputTokens)
    text = inputStr + " in"
    if (ctx.query.cachedTokens > 0) {
        cacheRatio = (ctx.query.cachedTokens / ctx.query.inputTokens * 100).toFixed(0)
        text = text + " (" + cacheRatio + "% cached)"
    }
    text = text + " · " + formatTokens(ctx.query.outputTokens) + " out · " + formatDuration(ctx.query.durationMs)

    padding = width - 4 - text.length
    if (padding < 3) {
        padding = 3
    }
    footer = "── " + text + " " + "─".repeat(padding)
    print("")
    print(gsh.ui.styles.primary(footer))
    print("")
}
gsh.on("agent.end", onAgentEnd)

# Renders the start line for exec (shell command) tool calls
# Example output: "▶ ls -la"
tool onExecStart(ctx) {
    print("")
    print(gsh.ui.styles.primary("▶") + " " + ctx.exec.command)
}
gsh.on("agent.exec.start", onExecStart)


# Renders the completion line for exec (shell command) tool calls
# Example output (success): "● ls ✓ (0.1s)"
# Example output (failure): "● cat ✗ (0.1s) exit code 1"
tool onExecEnd(ctx) {
    durationSec = (ctx.exec.durationMs / 1000).toFixed(1)
    if (ctx.exec.exitCode == 0) {
        line = "● " + ctx.exec.commandFirstWord + " ✓ " + gsh.ui.styles.dim("(" + durationSec + "s)")
        print(gsh.ui.styles.success(line))
    } else {
        line = "● " + ctx.exec.commandFirstWord + " ✗ " + gsh.ui.styles.dim("(" + durationSec + "s) exit code " + ctx.exec.exitCode)
        print(gsh.ui.styles.error(line))
    }
}
gsh.on("agent.exec.end", onExecEnd)


# Map to track spinner IDs by tool call ID
__toolSpinnerMap = {}

# Track the thinking spinner ID
__thinkingSpinnerId = null
__thinkingSpinnerStopped = false

# Track if we've printed any real (non-whitespace) text content in this iteration.
# This helps us skip leading whitespace before tool calls.
__printedRealText = false

# Renders the thinking spinner when agent iteration starts
tool onIterationStart(ctx) {
    __thinkingSpinnerId = gsh.ui.spinner.start("Thinking...")
    __thinkingSpinnerStopped = false
    __printedRealText = false
}
gsh.on("agent.iteration.start", onIterationStart)

# Handles each chunk of agent output - stops spinner and prints content
tool onChunk(ctx) {
    content = ctx.content

    # Check if this is real content (not just whitespace)
    isRealContent = content.trim() != ""

    # Stop spinner on first content chunk
    if (__thinkingSpinnerId != null && !__thinkingSpinnerStopped) {
        gsh.ui.spinner.stop(__thinkingSpinnerId)
        __thinkingSpinnerId = null
        __thinkingSpinnerStopped = true
        # Print a newline to separate spinner from content
        print("")
    }

    # Track if we've printed real text
    if (isRealContent) {
        __printedRealText = true
    }

    # Skip rendering whitespace-only chunks if we haven't printed real text yet.
    # This prevents empty lines from appearing before tool calls.
    if (!isRealContent && !__printedRealText) {
        return ""
    }

    # Print the content (without trailing newline - content already includes formatting)
    gsh.ui.write(content)
}
gsh.on("agent.chunk", onChunk)

# Renders the status line for non-exec tool calls (start)
# Example output: "○ read_file"
tool onToolStart(ctx) {
    id = gsh.ui.spinner.start(ctx.toolCall.name)
    __toolSpinnerMap[ctx.toolCall.id] = id
}
gsh.on("agent.tool.start", onToolStart)

# Renders the status line for non-exec tool calls (end)
# Example output (success): "● read_file ✓ (0.02s)"
# Example output (error):   "● read_file ✗ (0.01s)"
tool onToolEnd(ctx) {
    spinnerId = __toolSpinnerMap[ctx.toolCall.id]
    if (spinnerId != null) {
        gsh.ui.spinner.stop(spinnerId)
    }
    durationSec = (ctx.toolCall.durationMs / 1000).toFixed(2)
    
    if (ctx.toolCall.error != null) {
        line = "● " + ctx.toolCall.name + " " + gsh.ui.styles.error("✗") + " " + gsh.ui.styles.dim("(" + durationSec + "s)")
        print(line)
    } else {
        line = "● " + ctx.toolCall.name + " " + gsh.ui.styles.success("✓") + " " + gsh.ui.styles.dim("(" + durationSec + "s)")
        print(line)
    }
}
gsh.on("agent.tool.end", onToolEnd)

# =============================================================================
# REPL Events
# =============================================================================

# Show welcome message when REPL starts
tool onReplReady() {
    # ASCII art logo
    logo = [
        "  __ _ ___| |__  ",
        " / _` / __| '_ \\ ",
        "| (_| \\__ \\ | | |",
        " \\__, |___/_| |_|",
        " |___/           "
    ]
    
    # Tips pool (randomly selected)
    tips = [
        "use # to chat with the agent",
        "use # /clear to reset the conversation",
        "use # /agents to list available agents",
        "use # /agent <name> to switch agents",
        "agents remember context across messages in a session",
        "press Tab to autocomplete commands and file paths",
        "press Up/Down to navigate command history",
        "press Ctrl+A to jump to start of line",
        "press Ctrl+E to jump to end of line",
        "you can customize event handlers in ~/.gshrc.gsh",
        "starship integration is automatic if starship is in PATH",
        "use gsh.logging.level for troubleshooting (\"debug\", \"info\", \"warn\", \"error\")",
        "you can define bash aliases in ~/.bashrc",
        "press Ctrl+F to accept a command prediction",
        "command predictions use your command history for context",
        "use a small fast model like gemma3:1b for predictions",
        "define custom agents with specialized system prompts and tools",
        "agents in the REPL can execute shell commands and access files",
        "connect to MCP servers to give agents more capabilities",
        "define custom agents, tools, and MCP servers in ~/.gshrc.gsh",
        "run gsh scripts with: gsh script.gsh",
        "use exec() in scripts to run bash commands",
        "press Ctrl+D on an empty line to exit"
    ]
    
    # Get a random tip using Math.random() and Math.floor()
    tipIndex = Math.floor(Math.random() * tips.length)
    tip = tips[tipIndex]
    
    # Style helpers using gsh.ui.styles
    styles = gsh.ui.styles
    
    # Get terminal width
    width = gsh.terminal.width
    logoWidth = 18
    minGap = 4
    maxInfoWidth = 40
    
    # Build info lines
    infoLines = []
    infoLines.push(styles.primary(styles.bold("The G Shell")))
    infoLines.push("")
    
    # Version
    if (gsh.version != "" && gsh.version != "dev") {
        infoLines.push(styles.dim("version: ") + styles.primary(gsh.version))
    } else if (gsh.version == "dev") {
        infoLines.push(styles.dim("version: ") + styles.dim(styles.italic("development")))
    }
    
    # Predict model
    if (gsh.repl != null && gsh.repl.models != null && gsh.repl.models.lite != null) {
        liteModel = gsh.repl.models.lite
        predictName = liteModel["model"]
        infoLines.push(styles.dim("predict: ") + styles.primary(predictName))
    } else {
        infoLines.push(styles.dim("predict: ") + styles.dim(styles.italic("not configured")))
    }
    
    # Agent model
    if (gsh.repl != null && gsh.repl.models != null && gsh.repl.models.workhorse != null) {
        workhorseModel = gsh.repl.models.workhorse
        agentName = workhorseModel["model"]
        infoLines.push(styles.dim("agent:   ") + styles.primary(agentName))
    } else {
        infoLines.push(styles.dim("agent:   ") + styles.dim(styles.italic("not configured")))
    }
    
    # Calculate layout
    infoWidth = width - logoWidth - minGap
    if (infoWidth > maxInfoWidth) {
        infoWidth = maxInfoWidth
    }
    
    # Check if terminal is too narrow
    if (infoWidth < 20) {
        # Just show info without logo
        for (line of infoLines) {
            print(line)
        }
        print("")
        if (tip != "") {
            print(styles.dim(styles.italic("tip: ")) + styles.dim(tip))
        }
        print("")
        return ""
    }
    
    # Two-column layout
    numLines = logo.length
    if (infoLines.length > numLines) {
        numLines = infoLines.length
    }
    
    print("")
    i = 0
    while (i < numLines) {
        # Logo line
        logoLine = ""
        if (i < logo.length) {
            logoLine = styles.primary(logo[i])
        } else {
            logoLine = " ".repeat(logoWidth)
        }
        
        # Info line
        infoLine = ""
        if (i < infoLines.length) {
            infoLine = infoLines[i]
        }
        
        # Combine
        gap = " ".repeat(minGap)
        print(logoLine + gap + infoLine)
        
        i = i + 1
    }
    
    # Tip at bottom
    print("")
    if (tip != "") {
        print(styles.dim(styles.italic("tip: ") + tip))
    }
    print("")
    
    return ""
}
gsh.on("repl.ready", onReplReady)

# Custom prompt handler - called before each prompt is displayed
# Add [dev] prefix for development builds
tool onReplPrompt() {
    if (gsh.version == "dev") {
        gsh.repl.prompt = "[dev] gsh> "
    } else {
        gsh.repl.prompt = "gsh> "
    }
}
gsh.on("repl.prompt", onReplPrompt)
