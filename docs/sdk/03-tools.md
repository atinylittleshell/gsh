# Tools

This chapter documents the built-in tools available through `gsh.tools`.

**Availability:** REPL + Script

## `gsh.tools`

The `gsh.tools` object provides built-in tools that agents can use to interact with the system.

### Available Tools

| Tool                  | Description                      |
| --------------------- | -------------------------------- |
| `gsh.tools.exec`      | Execute shell commands           |
| `gsh.tools.grep`      | Search files using grep patterns |
| `gsh.tools.view_file` | View file contents               |
| `gsh.tools.edit_file` | Edit file contents               |

## `gsh.tools.exec`

Executes shell commands and returns the output.

### Agent Usage

When an agent uses `exec`, it can run any shell command:

```gsh
agent shellAgent {
    model: gsh.models.workhorse,
    systemPrompt: "You are a helpful shell assistant.",
    tools: [gsh.tools.exec],
}

# Agent can now run commands like: ls -la, cat file.txt, etc.
```

### Tool Parameters

| Parameter | Type     | Description                  |
| --------- | -------- | ---------------------------- |
| `command` | `string` | The shell command to execute |

### Tool Output

Returns an object with:

- `stdout` - Standard output from the command
- `stderr` - Standard error output
- `exitCode` - Exit code (0 = success)

## `gsh.tools.grep`

Searches files for patterns using grep-like functionality.

### Agent Usage

```gsh
agent searchAgent {
    model: gsh.models.workhorse,
    systemPrompt: "You help users find things in their codebase.",
    tools: [gsh.tools.grep, gsh.tools.view_file],
}
```

### Tool Parameters

| Parameter   | Type      | Description                              |
| ----------- | --------- | ---------------------------------------- |
| `pattern`   | `string`  | The search pattern (regex)               |
| `path`      | `string`  | File or directory path to search         |
| `recursive` | `boolean` | Whether to search recursively (optional) |

## `gsh.tools.view_file`

Views the contents of a file.

### Agent Usage

```gsh
agent codeReader {
    model: gsh.models.workhorse,
    systemPrompt: "You read and explain code.",
    tools: [gsh.tools.view_file],
}
```

### Tool Parameters

| Parameter   | Type     | Description                     |
| ----------- | -------- | ------------------------------- |
| `path`      | `string` | Path to the file                |
| `startLine` | `number` | Starting line number (optional) |
| `endLine`   | `number` | Ending line number (optional)   |

## `gsh.tools.edit_file`

Edits file contents using find-and-replace.

### Agent Usage

```gsh
agent codeEditor {
    model: gsh.models.workhorse,
    systemPrompt: "You help users edit code files.",
    tools: [gsh.tools.view_file, gsh.tools.edit_file],
}
```

### Tool Parameters

| Parameter | Type     | Description      |
| --------- | -------- | ---------------- |
| `path`    | `string` | Path to the file |
| `find`    | `string` | Text to find     |
| `replace` | `string` | Replacement text |

## Combining Tools

Most agents benefit from multiple tools working together:

```gsh
agent codeAssistant {
    model: gsh.models.workhorse,
    systemPrompt: "You are an expert coding assistant.",
    tools: [
        gsh.tools.exec,
        gsh.tools.grep,
        gsh.tools.view_file,
        gsh.tools.edit_file,
    ],
}
```

This gives the agent full capabilities to:

1. Search for code patterns with `grep`
2. View file contents with `view_file`
3. Make edits with `edit_file`
4. Run commands with `exec` (tests, builds, etc.)

## Custom Tools

You can also define your own tools using the `tool` keyword and pass them to agents. See the [Script Guide](../script/11-tool-declarations.md) for details on tool declarations.

```gsh
tool myCustomTool(input: string): string {
    # Custom logic here
    return "result"
}

agent myAgent {
    model: gsh.models.workhorse,
    systemPrompt: "You are helpful.",
    tools: [gsh.tools.exec, myCustomTool],
}
```

---

**Next:** [Agents](04-agents.md) - Defining and using custom agents
