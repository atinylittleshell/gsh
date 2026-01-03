# Agent event handlers
# These customize how agent interactions are displayed in the REPL.
# Override any of these in your ~/.gsh/repl.gsh to customize the appearance.

# Track if we've printed any real (non-whitespace) text content in the latest agent iteration.
# This helps us skip leading whitespace before tool calls.
__printedRealText = false

# Well-known spinner ID for the "Thinking..." spinner
__THINKING_SPINNER_ID = "__thinking__"

# Renders the header line when an agent starts responding
# Example output: "── gsh ─────────────────────────────"
# For non-default agents: "── MyAgent ─────────────────────────"
tool onAgentStart(ctx, next) {
    width = gsh.terminal.width
    if (width > 80) {
        width = 80
    }
    name = ctx.agent.name
    if (name == null || name == "" || name == "__defaultAgent") {
        name = "gsh"
    }
    padding = width - 4 - name.length  # "── " prefix (3) + " " before padding (1)
    if (padding < 3) {
        padding = 3
    }
    header = `── ${name} ${"─".repeat(padding)}`
    print(gsh.ui.styles.primary(header))
    return next(ctx)
}
gsh.use("agent.start", onAgentStart)

# Renders the footer line when an agent finishes responding
# Example output: "── 523 in · 324 out · 1.2s ────────────────────"
# Example with cache: "── 523 in (80% cached) · 324 out · 1.2s ─────"
# Example with error: "── error ──────────────────────────────────────"
# Tokens are formatted with K/M suffix for large numbers
# Duration uses appropriate units (ms, s, m, h, d)
tool onAgentEnd(ctx, next) {
    # Always stop the thinking spinner (in case error occurred before any content)
    gsh.ui.spinner.stop(__THINKING_SPINNER_ID)

    width = gsh.terminal.width
    if (width > 80) {
        width = 80
    }

    # Check if there was an error
    if (ctx.error != null) {
        # Print the error message first
        print(`${gsh.ui.styles.error("error: ")}${ctx.error}`)
        
        # Show error footer
        text = "error"
        padding = width - 4 - text.length
        if (padding < 3) {
            padding = 3
        }
        footer = `── ${text} ${"─".repeat(padding)}`
        print(gsh.ui.styles.error(footer))
        return next(ctx)
    }

    # Format tokens with K/M suffix
    tool formatTokens (count: number) {
        if (count >= 1000000) {
            return `${(count / 1000000).toFixed(2)}M`
        }
        if (count >= 1000) {
            return `${(count / 1000).toFixed(1)}K`
        }
        return `${count}`
    }

    # Format duration with appropriate units
    tool formatDuration (durationMs: number) {
        if (durationMs < 1000) {
            return `${durationMs}ms`
        }
        durationSec = durationMs / 1000
        if (durationSec < 60) {
            return `${durationSec.toFixed(1)}s`
        }
        durationMin = durationSec / 60
        if (durationMin < 60) {
            return `${durationMin.toFixed(1)}m`
        }
        durationHour = durationMin / 60
        if (durationHour < 24) {
            return `${durationHour.toFixed(1)}h`
        }
        return `${(durationHour / 24).toFixed(1)}d`
    }

    # Build the text, including cache ratio next to input tokens if there are cached tokens
    # For ACP agents (where token counts are 0), only show duration
    text = ""
    if (ctx.query.inputTokens > 0 || ctx.query.outputTokens > 0) {
        inputStr = formatTokens(ctx.query.inputTokens)
        cacheStr = ""
        if (ctx.query.cachedTokens > 0) {
            cacheRatio = (ctx.query.cachedTokens / ctx.query.inputTokens * 100).toFixed(0)
            cacheStr = ` (${cacheRatio}% cached)`
        }
        text = `${inputStr} in${cacheStr} · ${formatTokens(ctx.query.outputTokens)} out · ${formatDuration(ctx.query.durationMs)}`
    } else {
        text = formatDuration(ctx.query.durationMs)
    }

    padding = width - 4 - text.length
    if (padding < 3) {
        padding = 3
    }
    footer = `── ${text} ${"─".repeat(padding)}`
    print("")
    print(gsh.ui.styles.primary(footer))
    print("")
    return next(ctx)
}
gsh.use("agent.end", onAgentEnd)

# Renders the thinking spinner when agent iteration starts
tool onIterationStart(ctx, next) {
    gsh.ui.spinner.start("Thinking...", __THINKING_SPINNER_ID)
    __printedRealText = false
    return next(ctx)
}
gsh.use("agent.iteration.start", onIterationStart)

# Handles each chunk of agent output - stops thinking spinner and prints content
tool onChunk(ctx, next) {
    content = ctx.content

    # Check if this is real content (not just whitespace)
    isRealContent = content.trim() != ""

    # Stop thinking spinner on first content chunk
    gsh.ui.spinner.stop(__THINKING_SPINNER_ID)

    # Track if we've printed real text
    if (isRealContent) {
        __printedRealText = true
    }

    # Skip rendering whitespace-only chunks if we haven't printed real text yet.
    # This prevents empty lines from appearing before tool calls.
    if (!isRealContent && !__printedRealText) {
        return next(ctx)
    }

    # Print the content (without trailing newline - content already includes formatting)
    gsh.ui.write(content)
    return next(ctx)
}
gsh.use("agent.chunk", onChunk)

# Handles when a tool call enters pending state (streaming from LLM)
# This fires before args are complete - we show the tool name spinner
# The spinner manager ensures only the most recent spinner renders
tool onToolPending(ctx, next) {
    # Stop thinking spinner
    gsh.ui.spinner.stop(__THINKING_SPINNER_ID)
    
    # Start a spinner for this tool using the tool call ID as the spinner ID
    gsh.ui.spinner.start(ctx.toolCall.name, ctx.toolCall.id)
    return next(ctx)
}
gsh.use("agent.tool.pending", onToolPending)

# Renders the status line for tool calls (execution start)
# This fires when tool execution actually begins (after streaming is complete)
# For exec tool: shows the command being executed (e.g., "▶ ls -la")
# For other tools: shows the tool name with dimmed args on separate lines
tool onToolStart(ctx, next) {
    # Stop the pending spinner for this tool (uses same ID)
    gsh.ui.spinner.stop(ctx.toolCall.id)
    
    # Special handling for exec tool - show the command
    if (ctx.toolCall.name == "exec") {
        command = ctx.toolCall.args.command
        if (command != null) {
            print(`${gsh.ui.styles.primary("▶")} ${command}`)
        }
        return next(ctx)
    }
    
    # Build args lines for non-exec tools (dimmed, one per line, no indent)
    argsLines = ""
    args = ctx.toolCall.args
    if (args != null) {
        argKeys = args.keys()
        if (argKeys.length > 0) {
            argParts = []
            for (key of argKeys) {
                value = args[key]
                valueStr = `${value}`
                # Truncate long values
                if (valueStr.length > 60) {
                    valueStr = `${valueStr.substring(0, 57)}...`
                }
                argParts.push(gsh.ui.styles.dim(`${key}: ${valueStr}`))
            }
            argsLines = `\n${argParts.join("\n")}`
        }
    }
    
    # Print the tool start line (similar to exec)
    print(`${gsh.ui.styles.primary("▶")} ${ctx.toolCall.name}${argsLines}`)
    return next(ctx)
}
gsh.use("agent.tool.start", onToolStart)

# Renders the status line for tool calls (end)
# For exec tool:
#   Example output (success): "● ls ✓ (0.1s)"
#   Example output (failure): "● cat ✗ (0.1s) exit code 1"
# For other tools:
#   Example output (success): "● grep ✓ (0.02s)"
#   Example output (error):   "● grep ✗ (0.01s)"
tool onToolEnd(ctx, next) {
    durationSec = (ctx.toolCall.durationMs / 1000).toFixed(2)
    
    # Special handling for exec tool - parse exit code from output
    if (ctx.toolCall.name == "exec") {
        # Extract first word of command for display
        command = ctx.toolCall.args.command
        commandFirstWord = command
        if (command != null) {
            spaceIdx = command.indexOf(" ")
            if (spaceIdx > 0) {
                commandFirstWord = command.substring(0, spaceIdx)
            }
        }
        
        # Parse exit code from tool output (JSON format: {"output": "...", "exitCode": 0})
        # The exec tool returns JSON, so we parse it to get the exitCode
        exitCode = 0
        output = ctx.toolCall.output
        if (output != null) {
            try {
                parsed = JSON.parse(output)
                if (parsed.exitCode != null) {
                    exitCode = parsed.exitCode
                }
            } catch (e) {
                # If parsing fails, assume exit code 0
                exitCode = 0
            }
        }
        
        durationSec = (ctx.toolCall.durationMs / 1000).toFixed(1)
        if (exitCode == 0) {
            durationStr = gsh.ui.styles.dim(`(${durationSec}s)`)
            line = `${gsh.ui.styles.primary("●")} ${commandFirstWord} ${gsh.ui.styles.success("✓")} ${durationStr}`
            print(line)
        } else {
            durationStr = gsh.ui.styles.dim(`(${durationSec}s) exit code ${exitCode}`)
            line = `${gsh.ui.styles.primary("●")} ${commandFirstWord} ${gsh.ui.styles.error("✗")} ${durationStr}`
            print(line)
        }
        print("")
        return next(ctx)
    }
    
    # Simple completion line for non-exec tools (args already shown at start)
    durationStr = gsh.ui.styles.dim(`(${durationSec}s)`)
    if (ctx.toolCall.error != null) {
        line = `${gsh.ui.styles.primary("●")} ${ctx.toolCall.name} ${gsh.ui.styles.error("✗")} ${durationStr}`
        print(line)
    } else {
        line = `${gsh.ui.styles.primary("●")} ${ctx.toolCall.name} ${gsh.ui.styles.success("✓")} ${durationStr}`
        print(line)
    }
    print("")
    return next(ctx)
}
gsh.use("agent.tool.end", onToolEnd)
