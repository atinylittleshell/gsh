# Starship prompt integration for gsh
# This file is automatically loaded when starship is detected in your PATH.
# To disable this integration, add to your ~/.gshrc.gsh:
#   GSH_CONFIG.starshipIntegration = false

# Set up environment variables for Starship
env.STARSHIP_SHELL = "gsh"

# Initialize starship session (for transient prompt support)
_starship_session = exec("starship session 2>/dev/null || echo ''")
if (_starship_session.exitCode == 0) {
    env.STARSHIP_SESSION_KEY = _starship_session.stdout
}

# GSH_PROMPT tool using Starship
# This overrides the default simple prompt with starship's dynamic prompt
# ctx.repl contains: { lastExitCode, lastDurationMs }
tool GSH_PROMPT(ctx: object): string {
    exitCode = ctx.repl.lastExitCode
    durationMs = ctx.repl.lastDurationMs
    result = exec(`starship prompt --status=${exitCode} --cmd-duration=${durationMs} 2>/dev/null`)
    if (result.exitCode == 0) {
        return result.stdout
    }
    # Fallback to simple prompt if starship fails
    return "gsh> "
}
