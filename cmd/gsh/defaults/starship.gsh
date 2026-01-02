# Starship Integration
# Automatically detect and integrate with Starship prompt if available.
# To disable: unregister this handler with gsh.off("repl.prompt") and register your own.

# Check if starship is available in PATH
__starship_check = exec("which starship 2>/dev/null")
__starship_available = __starship_check.exitCode == 0

# Set up environment variables for Starship if available
if (__starship_available) {
    env.STARSHIP_SHELL = "gsh"
    
    # Initialize starship session (for transient prompt support)
    __starship_session = exec("starship session 2>/dev/null || echo ''")
    if (__starship_session.exitCode == 0) {
        env.STARSHIP_SESSION_KEY = __starship_session.stdout
    }
}

# Prompt handler - uses Starship if available, otherwise falls back to simple prompt
tool onReplPrompt() {
    # Default to simple prompt
    promptText = "gsh> "

    if (__starship_available) {
        exitCode = gsh.lastCommand.exitCode
        durationMs = gsh.lastCommand.durationMs
        result = exec(`starship prompt --status=${exitCode} --cmd-duration=${durationMs} 2>/dev/null`)
        if (result.exitCode == 0) {
            promptText = result.stdout
        }
    }
    
    if (gsh.version == "dev") {
        gsh.prompt = `[dev] ${promptText}`
    } else {
        gsh.prompt = promptText
    }
}
gsh.on("repl.prompt", onReplPrompt)
