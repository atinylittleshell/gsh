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
    # Simple prompt (can be overridden in ~/.gshrc.gsh)
    prompt: "gsh> ",
    
    # Log level: "debug", "info", "warn", "error"
    logLevel: "info",
    
    # Model to use for predictions (reference to model defined above)
    predictModel: GSH_PREDICT_MODEL,
    
    # Model to use for the built-in default agent
    defaultAgentModel: GSH_AGENT_MODEL,
}
