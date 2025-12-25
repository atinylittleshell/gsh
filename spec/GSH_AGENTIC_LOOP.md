# GSH Agentic Loop Implementation Plan

This document outlines the plan for implementing an agentic loop with tool calling support for the GSH REPL agent, including a built-in "exec" tool for shell command execution.

## Current State Analysis

### Agent in REPL (`internal/repl/agent/agent.go`)

- Simple message sending via `SendMessage()` that calls `StreamingChatCompletion`
- **Does NOT handle tool calls** - just appends the response to conversation history
- No tool definitions or execution logic

### Script Interpreter (`internal/script/interpreter/conversation.go`)

- Has partial tool call handling in `executeAgent()`
- **Only does ONE iteration** - after executing tools, makes another call but doesn't loop if that call also returns tool calls
- Supports both user-defined tools (`ToolValue`) and MCP tools (`MCPToolValue`)

### OpenAI Provider (`internal/script/interpreter/provider_openai.go`)

- Already supports tools in requests via `openAITool` structs
- Parses tool calls from responses correctly, including streaming
- Accumulates streamed tool call arguments properly

### ChatMessage Structure (`internal/script/interpreter/provider.go`)

Current fields:

```go
type ChatMessage struct {
    Role    string // "system", "user", "assistant", "tool"
    Content string
    Name    string // Optional: name of the tool or user
}
```

**Missing fields needed for proper tool call support:**

- `ToolCallID` - Required by OpenAI API for tool result messages
- `ToolCalls` - Assistant messages with tool calls need to include the tool call objects

### PTY Support

- `creack/pty` is already in dependencies (via go.sum)
- Can be used for live output display while capturing

---

## Key Issues to Address

1. **ChatMessage needs `ToolCallID`** - Required by OpenAI API for tool result messages
2. **ChatMessage needs `ToolCalls`** - Assistant messages requesting tools must include tool call objects
3. **OpenAI provider needs to send `tool_call_id`** - When sending tool result messages back to the API
4. **REPL agent needs agentic loop** - Currently has no tool support at all
5. **exec tool implementation** - Need PTY for live output + capture simultaneously

---

## Implementation Plan

### Phase 1: Extend Message Types for Tool Calls

#### 1.1 Update `ChatMessage` struct

File: `internal/script/interpreter/provider.go`

```go
type ChatMessage struct {
    Role       string         // "system", "user", "assistant", "tool"
    Content    string
    Name       string         // Optional: name of the tool
    ToolCallID string         // For tool result messages (required by OpenAI API)
    ToolCalls  []ChatToolCall // For assistant messages that request tool calls
}
```

#### 1.2 Update OpenAI provider message serialization

File: `internal/script/interpreter/provider_openai.go`

Update `openAIMessage` struct and message conversion to:

- Include `tool_call_id` field in tool result messages
- Include `tool_calls` array in assistant messages when present
- Handle the new fields during serialization/deserialization

---

### Phase 2: Implement Proper Agentic Loop

#### 2.1 Fix Script Interpreter Loop

File: `internal/script/interpreter/conversation.go`

Update `executeAgent()` to:

- Loop until no tool calls are returned (currently only does one iteration)
- Add max iterations safeguard (e.g., 10 iterations) to prevent infinite loops
- Properly track tool calls in assistant messages

#### 2.2 Add Agentic Loop to REPL Agent

File: `internal/repl/agent/agent.go`

Update `SendMessage()` to:

- Accept tool definitions
- Implement full agentic loop with streaming support
- Execute tool calls between iterations
- Provide callbacks for tool execution status (so UI can show progress)

Pattern:

```go
func (m *Manager) SendMessage(ctx context.Context, message string, onChunk func(string)) error {
    // ... setup messages ...

    const maxIterations = 10
    for iteration := 0; iteration < maxIterations; iteration++ {
        response, err := state.Provider.StreamingChatCompletion(request, onChunk)
        if err != nil {
            return err
        }

        // Add assistant response to conversation
        state.Conversation = append(state.Conversation, ChatMessage{
            Role:      "assistant",
            Content:   response.Content,
            ToolCalls: response.ToolCalls,
        })

        // If no tool calls, we're done
        if len(response.ToolCalls) == 0 {
            break
        }

        // Execute tool calls and add results
        for _, toolCall := range response.ToolCalls {
            result := executeToolCall(toolCall)
            state.Conversation = append(state.Conversation, ChatMessage{
                Role:       "tool",
                Content:    result,
                ToolCallID: toolCall.ID,
                Name:       toolCall.Name,
            })
        }

        // Update request messages for next iteration
        request.Messages = buildMessages(state)
    }

    return nil
}
```

---

### Phase 3: Implement "exec" Tool

#### 3.1 Create PTY-based Command Executor

File: `internal/repl/agent/exec.go` (new file)

```go
package agent

import (
    "bytes"
    "context"
    "io"
    "os"
    "os/exec"

    "github.com/creack/pty"
)

// ExecResult contains the result of executing a shell command
type ExecResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
}

// ExecuteCommand runs a shell command with PTY support
// This allows live output display while also capturing the output
func ExecuteCommand(ctx context.Context, command string, stdout, stderr io.Writer) (*ExecResult, error) {
    cmd := exec.CommandContext(ctx, "bash", "-c", command)

    // Create PTY for the command
    ptmx, err := pty.Start(cmd)
    if err != nil {
        return nil, fmt.Errorf("failed to start pty: %w", err)
    }
    defer ptmx.Close()

    // Capture output while also writing to provided writers
    var outputBuf bytes.Buffer

    // Use MultiWriter to write to both the live output and capture buffer
    var writer io.Writer
    if stdout != nil {
        writer = io.MultiWriter(stdout, &outputBuf)
    } else {
        writer = &outputBuf
    }

    // Copy PTY output (this blocks until command completes)
    go io.Copy(writer, ptmx)

    // Wait for command to complete
    err = cmd.Wait()

    exitCode := 0
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        } else {
            return nil, fmt.Errorf("command execution failed: %w", err)
        }
    }

    return &ExecResult{
        Stdout:   outputBuf.String(),
        Stderr:   "", // PTY combines stdout/stderr
        ExitCode: exitCode,
    }, nil
}
```

#### 3.2 Create Tool Definition

```go
// ExecToolDefinition returns the tool definition for the exec tool
func ExecToolDefinition() ChatTool {
    return ChatTool{
        Name:        "exec",
        Description: "Execute a shell command and return the output. Use this to run commands, scripts, or interact with the system.",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "command": map[string]interface{}{
                    "type":        "string",
                    "description": "The shell command to execute",
                },
            },
            "required": []string{"command"},
        },
    }
}
```

#### 3.3 Tool Execution Handler

```go
// ExecuteExecTool handles execution of the exec tool
func (m *Manager) ExecuteExecTool(ctx context.Context, args map[string]interface{}) (string, error) {
    command, ok := args["command"].(string)
    if !ok {
        return "", fmt.Errorf("exec tool requires 'command' argument as string")
    }

    // Execute with live output to current stdout
    result, err := ExecuteCommand(ctx, command, os.Stdout, os.Stderr)
    if err != nil {
        return fmt.Sprintf("Error executing command: %v", err), nil
    }

    // Return result as JSON for the agent
    return fmt.Sprintf(`{"stdout": %q, "exitCode": %d}`, result.Stdout, result.ExitCode), nil
}
```

---

### Phase 4: Wire Everything Together

#### 4.1 Update REPL Agent Initialization

File: `internal/repl/repl.go`

When creating the default agent, add the exec tool:

```go
defaultAgent := &interpreter.AgentValue{
    Name: "default",
    Config: map[string]interpreter.Value{
        "model": defaultAgentModel,
        "systemPrompt": &interpreter.StringValue{
            Value: "You are gsh, an AI-powered shell. You can execute commands using the exec tool.",
        },
        "tools": &interpreter.ArrayValue{
            Elements: []interpreter.Value{
                // Add exec tool here
            },
        },
    },
}
```

#### 4.2 Update State to Include Tools

File: `internal/repl/agent/agent.go`

Add tools to State struct:

```go
type State struct {
    Agent        *interpreter.AgentValue
    Provider     interpreter.ModelProvider
    Conversation []interpreter.ChatMessage
    Tools        []interpreter.ChatTool  // Available tools for this agent
}
```

#### 4.3 Streaming Callback Enhancement

Update the streaming callback to handle tool execution status:

```go
type StreamEvent struct {
    Type    string // "content", "tool_start", "tool_end"
    Content string
    Tool    *ToolExecutionInfo
}

type ToolExecutionInfo struct {
    Name      string
    Arguments map[string]interface{}
    Result    string
}
```

---

## Technical Details

### PTY Approach for Live Output + Capture

The key insight is using `io.MultiWriter` to write to multiple destinations:

```go
import (
    "github.com/creack/pty"
    "io"
    "os"
    "os/exec"
)

func executeWithPTY(command string) (string, int, error) {
    cmd := exec.Command("bash", "-c", command)

    // Create PTY - this gives us a pseudo-terminal
    ptmx, err := pty.Start(cmd)
    if err != nil {
        return "", 1, err
    }
    defer ptmx.Close()

    // Capture output while also writing to stdout
    var output bytes.Buffer
    writer := io.MultiWriter(os.Stdout, &output)

    // Copy PTY output to both destinations
    // PTY combines stdout and stderr into a single stream
    io.Copy(writer, ptmx)

    // Wait for command to complete
    err = cmd.Wait()
    exitCode := 0
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode = exitErr.ExitCode()
    }

    return output.String(), exitCode, nil
}
```

### OpenAI Tool Call Message Format

Per OpenAI API spec, the message sequence for tool calls is:

1. **User message**: `{"role": "user", "content": "..."}`
2. **Assistant message with tool calls**:
   ```json
   {
     "role": "assistant",
     "content": null,
     "tool_calls": [
       {
         "id": "call_abc123",
         "type": "function",
         "function": {
           "name": "exec",
           "arguments": "{\"command\": \"ls -la\"}"
         }
       }
     ]
   }
   ```
3. **Tool result message**:
   ```json
   {
     "role": "tool",
     "tool_call_id": "call_abc123",
     "content": "{\"stdout\": \"...\", \"exitCode\": 0}"
   }
   ```

---

## Testing Considerations

1. **Unit tests for exec tool** - Mock PTY or use simple commands
2. **Integration tests for agentic loop** - Use mock provider with predefined tool call scenarios
3. **Max iteration tests** - Ensure loop terminates properly
4. **Error handling tests** - Command failures, timeouts, context cancellation

---

## Future Enhancements

1. **Confirmation prompts** - Ask user before executing potentially dangerous commands
2. **Command sandboxing** - Restrict what commands can be executed
3. **Output streaming callbacks** - Real-time output events for UI updates
4. **Multiple tool support** - Add more built-in tools (read_file, write_file, etc.)
5. **Tool result truncation** - Handle very large outputs gracefully

---

## References

- OpenAI Function Calling: https://platform.openai.com/docs/guides/function-calling
- creack/pty: https://github.com/creack/pty
- Existing tool call handling: `internal/script/interpreter/conversation.go`
