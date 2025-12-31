# Imports and Modules

gsh supports a module system that allows you to organize your scripts into reusable components. This enables better code organization, reusability, and maintainability.

## Basic Syntax

### Side-Effect Import

Import a file for its side effects only (e.g., registering event handlers):

```gsh
import "./path/to/file.gsh"
```

This executes the file but does not bring any symbols into the current scope.

### Selective Import

Import specific exported symbols from a file:

```gsh
import { symbolA, symbolB } from "./path/to/file.gsh"
```

Only symbols marked with `export` in the source file will be available for import.

### Export Declaration

Mark symbols as available for import by other files:

```gsh
# Export a variable
export myVariable = 42

# Export a tool (function)
export tool myHelper(x) {
    return x * 2
}

# Export a model
export model myModel {
    provider: "openai",
    model: "gpt-4o",
}

# Export an agent
export agent myAgent {
    model: myModel,
    tools: [myHelper],
}
```

Non-exported symbols remain private to the file.

## Path Resolution

Import paths are resolved relative to the current script's location:

- `./` - Relative to current script's directory
- `../` - Parent directory of current script
- Absolute paths - Use the exact path specified

### Examples

```gsh
# Import from same directory
import { helper } from "./helpers.gsh"

# Import from subdirectory
import { utils } from "./lib/utils.gsh"

# Import from parent directory
import { config } from "../config.gsh"

# Import using absolute path
import { tool } from "/home/user/scripts/tool.gsh"
```

## Module Scope

Each imported file has its own scope:

```gsh
# file: helpers.gsh
privateVar = "not visible outside"      # Private to this file

export publicVar = "visible to importers"  # Exported

export tool publicTool() {
    return privateVar  # Can access private vars internally
}
```

```gsh
# file: main.gsh
import { publicVar, publicTool } from "./helpers.gsh"

print(publicVar)      # Works: "visible to importers"
print(publicTool())   # Works: "not visible outside"
print(privateVar)     # Error: undefined variable
```

## Side-Effect Imports

When using `import "./file.gsh"` without `{ }`:

- The file executes in its own scope
- Side effects (like `gsh.on()` calls) take effect
- No symbols are imported into the current scope

This is useful for modules that self-register:

```gsh
# file: events/agent.gsh
tool onAgentStart(ctx) {
    print("Agent started")
}
gsh.on("agent.start", onAgentStart)  # Self-registers
```

```gsh
# file: init.gsh
import "./events/agent.gsh"  # Just runs the file, handler is registered
```

## Module Caching

Each unique file path is only executed once per interpreter session. Subsequent imports of the same file return the cached exports:

```gsh
# file: counter.gsh
export count = 0
count = count + 1
```

```gsh
# file: main.gsh
import { count } from "./counter.gsh"
print(count)  # 1

import { count } from "./counter.gsh"
print(count)  # Still 1 (module wasn't re-executed)
```

## Circular Import Prevention

Circular imports are detected and result in an error:

```gsh
# file: a.gsh
import "./b.gsh"  # OK
export aVar = 1

# file: b.gsh
import "./a.gsh"  # Error: circular import detected
export bVar = 2
```

## Best Practices

### Organize Related Code

Group related functionality into modules:

```
~/.gsh/
├── repl.gsh          # Main config, imports modules
├── models.gsh          # Model declarations
├── agents.gsh          # Agent declarations
├── events/
│   ├── agent.gsh       # Agent event handlers
│   └── repl.gsh        # REPL event handlers
└── tools/
    ├── git.gsh         # Git-related tools
    └── docker.gsh      # Docker-related tools
```

### Use Explicit Exports

Only export what you intend to be public:

```gsh
# Good: explicit exports
export tool publicAPI() { ... }
tool internalHelper() { ... }  # Private

# Avoid: exporting everything
```

### Prefer Selective Imports

Import only what you need:

```gsh
# Good: import specific symbols
import { specificTool } from "./tools.gsh"

# Less ideal for large modules: side-effect import
import "./tools.gsh"
```

## Example: Modular Configuration

```gsh
# ~/.gsh/repl.gsh - Main configuration

# Import model definitions
import "./models.gsh"

# Import event handlers (side-effect)
import "./events/agent.gsh"
import "./events/repl.gsh"

# Import and use custom tools
import { formatOutput } from "./tools/formatting.gsh"

# Use imported symbols
gsh.models.workhorse = workhorse
```

```gsh
# ~/.gsh/models.gsh
export model lite {
    provider: "openai",
    model: "gpt-4o-mini",
}

export model workhorse {
    provider: "openai",
    model: "gpt-4o",
}
```

```gsh
# ~/.gsh/events/agent.gsh
tool onAgentStart(ctx) {
    print("🤖 Agent starting...")
}
gsh.on("agent.start", onAgentStart)
```
