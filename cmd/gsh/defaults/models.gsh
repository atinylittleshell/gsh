# Model declarations for gsh
# These provide default model configurations that users can customize in ~/.gsh/repl.gsh

# Use for predictions (lightweight, fast)
model lite {
    provider: "openai",
    apiKey: "ollama",
    model: "gemma3:1b",
    baseURL: "http://localhost:11434/v1",
}

# Use for agent interactions (more capable)
model workhorse {
    provider: "openai",
    apiKey: "ollama",
    model: "gpt-oss:20b",
    baseURL: "http://localhost:11434/v1",
}

# Configure models
gsh.models.lite = lite
gsh.models.workhorse = workhorse
# gsh.models.premium not used yet - reserved for future high-value tasks
