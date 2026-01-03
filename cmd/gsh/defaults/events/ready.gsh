# REPL event handlers
# These customize the REPL experience including the welcome message.
# Override any of these in your ~/.gsh/repl.gsh to customize.

# Show welcome message when REPL starts
tool onReplReady(ctx, next) {
    # ASCII art logo
    logo = [
        "  ░██████    ░██████   ░██     ░██ ",
        " ░██   ░██  ░██   ░██  ░██     ░██ ",
        "░██        ░██         ░██     ░██ ",
        "░██  █████  ░████████  ░██████████ ",
        "░██     ██         ░██ ░██     ░██ ",
        " ░██  ░███  ░██   ░██  ░██     ░██ ",
        "  ░█████░█   ░██████   ░██     ░██ "
    ]
    
    # Tips pool (randomly selected)
    tips = [
        "use # to chat with the agent",
        "use # /clear to reset the conversation",
        "the default agent remembers context across messages in a session",
        "press Tab to autocomplete commands and file paths",
        "press Up/Down to navigate command history",
        "press Ctrl+A to jump to start of line",
        "press Ctrl+E to jump to end of line",
        "press Ctrl+F to accept a command prediction",
        "command predictions use your command history for context",
        "use a small fast model like gemma3:1b for predictions",
        "you can customize event handlers in ~/.gsh/repl.gsh",
        "starship integration is automatic if starship is in PATH",
        "use gsh.logging.level for troubleshooting (\"debug\", \"info\", \"warn\", \"error\")",
        "you can define bash aliases in ~/.gshrc",
        "the agent can execute shell commands and access files",
        "connect to MCP servers to give agents more capabilities",
        "define custom tools and MCP servers in ~/.gsh/repl.gsh",
        "run gsh scripts with: gsh run script.gsh",
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
    logoWidth = 35
    minGap = 4
    maxInfoWidth = 40
    
    # Build info lines
    infoLines = []
    infoLines.push(styles.primary(styles.bold("The G Shell")))
    infoLines.push("")
    
    # Version
    if (gsh.version != "" && gsh.version != "dev") {
        infoLines.push(`${styles.dim("version:   ")}${styles.primary(gsh.version)}`)
    } else if (gsh.version == "dev") {
        infoLines.push(`${styles.dim("version:   ")}${styles.dim(styles.italic("development"))}`)
    }
    
    infoLines.push("")
    
    # Lite model tier
    if (gsh.models != null && gsh.models.lite != null) {
        liteModel = gsh.models.lite
        liteName = liteModel.model
        infoLines.push(`${styles.dim("lite:      ")}${styles.primary(liteName)}`)
    } else {
        infoLines.push(`${styles.dim("lite:      ")}${styles.dim(styles.italic("not configured"))}`)
    }
    
    # Workhorse model tier
    if (gsh.models != null && gsh.models.workhorse != null) {
        workhorseModel = gsh.models.workhorse
        workhorseName = workhorseModel.model
        infoLines.push(`${styles.dim("workhorse: ")}${styles.primary(workhorseName)}`)
    } else {
        infoLines.push(`${styles.dim("workhorse: ")}${styles.dim(styles.italic("not configured"))}`)
    }
    
    # Premium model tier
    if (gsh.models != null && gsh.models.premium != null) {
        premiumModel = gsh.models.premium
        premiumName = premiumModel.model
        infoLines.push(`${styles.dim("premium:   ")}${styles.primary(premiumName)}`)
    } else {
        infoLines.push(`${styles.dim("premium:   ")}${styles.dim(styles.italic("not configured"))}`)
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
        return
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
        print(styles.dim(styles.italic("tip: ") + styles.dim(tip)))
    }
    print("")
    return next(ctx)
}
gsh.use("repl.ready", onReplReady)
