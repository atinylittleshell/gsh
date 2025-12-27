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

# Default GSH configuration
GSH_CONFIG = {
    # Simple prompt (fallback if starship or GSH_PROMPT fails)
    prompt: "gsh> ",

    # Enable automatic starship integration
    starshipIntegration: true,
    
    # Show welcome screen on startup
    showWelcome: true,
    
    # Log level: "debug", "info", "warn", "error"
    logLevel: "info",
    
    # Model to use for predictions (reference to model defined above)
    predictModel: GSH_PREDICT_MODEL,
    
    # Model to use for the built-in default agent
    defaultAgentModel: GSH_AGENT_MODEL,
}

# =============================================================================
# Agent Rendering Hooks
# =============================================================================
# These tools customize how agent interactions are displayed in the REPL.
# Override any of these in your ~/.gshrc.gsh to customize the appearance.
#
# All hooks receive a single `ctx` parameter - a RenderContext object with:
#   ctx.terminal: { width, height }
#   ctx.agent: { name } or null
#   ctx.query: { durationMs, inputTokens, outputTokens } or null
#   ctx.exec: { command, commandFirstWord, durationMs, exitCode } or null
#   ctx.toolCall: { name, status, args, durationMs, output } or null

# Renders the header line when an agent starts responding
# Example output: "── agent: default ─────────────────────────────"
# Note: "agent" is a keyword in gsh, so we use bracket notation ctx["agent"]
tool GSH_AGENT_HEADER(ctx: object): string {
    width = ctx.terminal.width
    if (width > 80) {
        width = 80
    }
    text = ctx["agent"].name
    if (text == "default") {
        text = "gsh"
    }
    padding = width - 4 - text.length  # "── " prefix (3) + " " before padding (1)
    if (padding < 3) {
        padding = 3
    }
    return "── " + text + " " + "─".repeat(padding)
}

# Renders the footer line when an agent finishes responding
# Example output: "── 523 in · 324 out · 1.2s ────────────────────"
# Example with cache: "── 523 in (80% cached) · 324 out · 1.2s ─────"
tool GSH_AGENT_FOOTER(ctx: object): string {
    width = ctx.terminal.width
    if (width > 80) {
        width = 80
    }
    # Format duration: convert ms to seconds with 1 decimal place
    durationSec = (ctx.query.durationMs / 1000).toFixed(1)
    
    # Build the text, including cache ratio next to input tokens if there are cached tokens
    text = "" + ctx.query.inputTokens + " in"
    if (ctx.query.cachedTokens > 0 && ctx.query.inputTokens > 0) {
        cacheRatio = (ctx.query.cachedTokens / ctx.query.inputTokens) * 100
        text = text + " (" + cacheRatio.toFixed(0) + "% cached)"
    }
    text = text + " · " + ctx.query.outputTokens + " out · " + durationSec + "s"
    
    padding = width - 4 - text.length
    if (padding < 3) {
        padding = 3
    }
    return "── " + text + " " + "─".repeat(padding)
}

# Renders the start line for exec (shell command) tool calls
# Example output: "▶ ls -la"
tool GSH_EXEC_START(ctx: object): string {
    return "▶ " + ctx.exec.command
}

# Renders the completion line for exec (shell command) tool calls
# Example output (success): "✓ ls (0.1s)"
# Example output (failure): "✗ cat (0.1s) exit code 1"
tool GSH_EXEC_END(ctx: object): string {
    durationSec = (ctx.exec.durationMs / 1000).toFixed(1)
    if (ctx.exec.exitCode == 0) {
        return "✓ " + ctx.exec.commandFirstWord + " (" + durationSec + "s)"
    }
    return "✗ " + ctx.exec.commandFirstWord + " (" + durationSec + "s) exit code " + ctx.exec.exitCode
}

# Renders the status line for non-exec tool calls
# ctx.toolCall.status is one of: "pending", "executing", "success", "error"
# Example output (pending):   "○ read_file"
# Example output (executing): "○ read_file\n   path: \"/config.json\""
# Example output (success):   "● read_file ✓ (0.02s)\n   path: \"/config.json\""
# Example output (error):     "● read_file ✗ (0.01s)\n   path: \"/missing.txt\""
tool GSH_TOOL_STATUS(ctx: object): string {
    # Format arguments - one per line, indented
    argsStr = ""
    keys = ctx.toolCall.args.keys()
    for (key of keys) {
        value = ctx.toolCall.args[key]
        # Truncate long values
        valueStr = "" + value
        if (valueStr.length > 60) {
            valueStr = valueStr.substring(0, 57) + "..."
        }
        argsStr = argsStr + "   " + key + ": " + valueStr + "\n"
    }
    
    durationSec = (ctx.toolCall.durationMs / 1000).toFixed(2)
    
    if (ctx.toolCall.status == "pending") {
        return "○ " + ctx.toolCall.name
    }
    if (ctx.toolCall.status == "executing") {
        if (argsStr == "") {
            return "○ " + ctx.toolCall.name
        }
        return "○ " + ctx.toolCall.name + "\n" + argsStr
    }
    if (ctx.toolCall.status == "success") {
        if (argsStr == "") {
            return "● " + ctx.toolCall.name + " ✓ (" + durationSec + "s)"
        }
        return "● " + ctx.toolCall.name + " ✓ (" + durationSec + "s)\n" + argsStr
    }
    # error status
    if (argsStr == "") {
        return "● " + ctx.toolCall.name + " ✗ (" + durationSec + "s)"
    }
    return "● " + ctx.toolCall.name + " ✗ (" + durationSec + "s)\n" + argsStr
}

# Renders the output of non-exec tool calls
# Default returns empty string (no output shown)
# Override to display tool output if desired
tool GSH_TOOL_OUTPUT(ctx: object): string {
    return ""  # Default: show nothing
}
