# Default gsh configuration
# This file is loaded before ~/.gsh/repl.gsh to provide sensible defaults.

# Configure models
import "./models.gsh"

# Register event handlers
import "./events/agent.gsh"
import "./events/ready.gsh"

# Enable starship prompt integration
import "./starship.gsh"

# Import default middleware
import "./middleware/agent.gsh"
import "./middleware/prediction.gsh"

# Set log level: "debug", "info", "warn", "error"
if (gsh.version == "dev") {
    gsh.logging.level = "debug"
} else {
    gsh.logging.level = "info"
}
