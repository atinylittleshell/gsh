# Starship Integration
# Automatically detect and integrate with Starship prompt if available.
# To disable: Add `gsh.removeAll("repl.prompt")` in ~/.gsh/repl.gsh and register your own handler.

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
tool onReplPrompt(ctx, next) {
    # Get starship prompt directly without intermediate variable to avoid race condition
    # with prediction system that may call this handler concurrently
    if (__starship_available) {
        __starship_exitCode = gsh.lastCommand.exitCode
        __starship_durationMs = gsh.lastCommand.durationMs
        __starship_result = exec(`starship prompt --status=${__starship_exitCode} --cmd-duration=${__starship_durationMs}`)
        if (__starship_result.exitCode == 0 && __starship_result.stdout != "") {
            if (gsh.version == "dev") {
                gsh.prompt = `[dev] ${__starship_result.stdout}`
            } else {
                gsh.prompt = __starship_result.stdout
            }
            return next(ctx)
        }
    }

    # Fallback to simple prompt
    if (gsh.version == "dev") {
        gsh.prompt = "[dev] gsh> "
    } else {
        gsh.prompt = "gsh> "
    }
    return next(ctx)
}
gsh.use("repl.prompt", onReplPrompt)
