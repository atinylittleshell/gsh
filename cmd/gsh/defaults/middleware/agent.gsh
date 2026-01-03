# Default Agent Middleware
# This middleware handles agent chat commands (prefixed with '#').

# Default agent for REPL chat interactions
agent __defaultAgent {
    model: gsh.models.workhorse,
    systemPrompt: "You are gsh, the generative shell. You help users with tasks in the shell. Use concise plain text (no markdown) for response.",
    tools: [gsh.tools.exec, gsh.tools.grep, gsh.tools.view_file, gsh.tools.edit_file],
}

# Conversation state (null means no active conversation)
__conversation = null

# Default input middleware - handles # prefix for agent chat
tool __defaultAgentMiddleware(ctx, next) {
    input = ctx.input.trim()
    
    # Skip empty input
    if (input == "") {
        return { handled: true }
    }
    
    # Handle # prefix for agent chat
    if (input.startsWith("#")) {
        message = input.substring(1).trim()
        
        # Empty message shows help
        if (message == "") {
            print("Agent mode: type your message after # to chat with the agent.")
            print("Commands:")
            print("  # /clear - clear conversation history")
            return { handled: true }
        }
        
        # Handle /clear command
        if (message == "/clear") {
            __conversation = null
            print("Conversation cleared")
            return { handled: true }
        }
        
        # Chat with the default agent using pipe expressions
        if (__conversation == null) {
            __conversation = message | __defaultAgent
        } else {
            __conversation = __conversation | message | __defaultAgent
        }
        return { handled: true }
    }
    
    # Not an agent command - pass to next middleware (fall through to shell)
    return next(ctx)
}

# Register the default middleware for command.input event
gsh.use("command.input", __defaultAgentMiddleware)
