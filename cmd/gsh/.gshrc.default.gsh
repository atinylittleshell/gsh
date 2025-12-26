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

# Renders the header line when an agent starts responding
# Example output: "── agent: default ─────────────────────────────"
tool GSH_AGENT_HEADER(agentName: string, terminalWidth: number): string {
    width = terminalWidth
    if (width > 80) {
        width = 80
    }
    text = agentName
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
tool GSH_AGENT_FOOTER(inputTokens: number, outputTokens: number, durationMs: number, terminalWidth: number): string {
    width = terminalWidth
    if (width > 80) {
        width = 80
    }
    # Format duration: convert ms to seconds with 1 decimal place
    durationSec = durationMs / 1000
    text = "" + inputTokens + " in · " + outputTokens + " out · " + durationSec + "s"
    padding = width - 4 - text.length
    if (padding < 3) {
        padding = 3
    }
    return "── " + text + " " + "─".repeat(padding)
}

# Renders the start line for exec (shell command) tool calls
# Example output: "▶ ls -la"
tool GSH_EXEC_START(command: string): string {
    return "▶ " + command
}

# Renders the completion line for exec (shell command) tool calls
# Example output (success): "✓ ls (0.1s)"
# Example output (failure): "✗ cat (0.1s) exit code 1"
tool GSH_EXEC_END(commandFirstWord: string, durationMs: number, exitCode: number): string {
    durationSec = durationMs / 1000
    if (exitCode == 0) {
        return "✓ " + commandFirstWord + " (" + durationSec + "s)"
    }
    return "✗ " + commandFirstWord + " (" + durationSec + "s) exit code " + exitCode
}

# Renders the status line for non-exec tool calls
# status is one of: "pending", "executing", "success", "error"
# Example output (pending):   "○ read_file"
# Example output (executing): "○ read_file\n   path: \"/config.json\""
# Example output (success):   "● read_file ✓ (0.02s)\n   path: \"/config.json\""
# Example output (error):     "● read_file ✗ (0.01s)\n   path: \"/missing.txt\""
tool GSH_TOOL_STATUS(toolName: string, status: string, args: object, durationMs: number): string {
    # Format arguments - one per line, indented
    argsStr = ""
    keys = args.keys()
    for (key of keys) {
        value = args[key]
        # Truncate long values
        valueStr = "" + value
        if (valueStr.length > 60) {
            valueStr = valueStr.substring(0, 57) + "..."
        }
        argsStr = argsStr + "   " + key + ": " + valueStr + "\n"
    }
    
    durationSec = durationMs / 1000
    
    if (status == "pending") {
        return "○ " + toolName
    }
    if (status == "executing") {
        if (argsStr == "") {
            return "○ " + toolName
        }
        return "○ " + toolName + "\n" + argsStr
    }
    if (status == "success") {
        if (argsStr == "") {
            return "● " + toolName + " ✓ (" + durationSec + "s)"
        }
        return "● " + toolName + " ✓ (" + durationSec + "s)\n" + argsStr
    }
    # error status
    if (argsStr == "") {
        return "● " + toolName + " ✗ (" + durationSec + "s)"
    }
    return "● " + toolName + " ✗ (" + durationSec + "s)\n" + argsStr
}

# Renders the output of non-exec tool calls
# Default returns empty string (no output shown)
# Override to display tool output if desired
tool GSH_TOOL_OUTPUT(toolName: string, output: string, terminalWidth: number): string {
    return ""  # Default: show nothing
}
