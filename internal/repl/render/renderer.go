package render

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
)

// Renderer handles all agent-related output rendering in the REPL.
// It delegates formatting decisions to customizable hook tools defined in .gshrc.gsh
type Renderer struct {
	interp    *interpreter.Interpreter // For calling custom hooks
	writer    io.Writer
	termWidth func() int // Function to get current terminal width
}

// New creates a new Renderer instance
func New(interp *interpreter.Interpreter, writer io.Writer, termWidth func() int) *Renderer {
	return &Renderer{
		interp:    interp,
		writer:    writer,
		termWidth: termWidth,
	}
}

// RenderAgentHeader renders the agent header line using the GSH_AGENT_HEADER hook
func (r *Renderer) RenderAgentHeader(agentName string) {
	width := r.getTerminalWidth()

	header := r.callStringHook("GSH_AGENT_HEADER", map[string]interface{}{
		"agentName":     agentName,
		"terminalWidth": float64(width),
	})

	if header == "" {
		// Fallback if hook fails
		header = fmt.Sprintf("── agent: %s ───", agentName)
	}

	fmt.Fprintln(r.writer, HeaderStyle.Render(header))
}

// RenderAgentFooter renders the agent footer line using the GSH_AGENT_FOOTER hook
func (r *Renderer) RenderAgentFooter(inputTokens, outputTokens int, duration time.Duration) {
	width := r.getTerminalWidth()
	durationMs := duration.Milliseconds()

	footer := r.callStringHook("GSH_AGENT_FOOTER", map[string]interface{}{
		"inputTokens":   float64(inputTokens),
		"outputTokens":  float64(outputTokens),
		"durationMs":    float64(durationMs),
		"terminalWidth": float64(width),
	})

	if footer == "" {
		// Fallback if hook fails
		footer = fmt.Sprintf("── %d in · %d out · %.1fs ───", inputTokens, outputTokens, duration.Seconds())
	}

	fmt.Fprintln(r.writer)
	fmt.Fprintln(r.writer, HeaderStyle.Render(footer))
}

// StartThinkingSpinner starts a "Thinking..." spinner and returns a stop function.
// The stop function blocks until the spinner has fully stopped and cleared the line.
func (r *Renderer) StartThinkingSpinner(ctx context.Context) func() {
	spinner := NewSpinner(r.writer)
	spinner.SetMessage("Thinking...")
	return spinner.Start(ctx)
}

// RenderAgentText renders agent response text (no special formatting)
func (r *Renderer) RenderAgentText(text string) {
	fmt.Fprint(r.writer, text)
}

// RenderExecStart renders the start of an exec tool call using GSH_EXEC_START hook
func (r *Renderer) RenderExecStart(command string) {
	output := r.callStringHook("GSH_EXEC_START", map[string]interface{}{
		"command": command,
	})

	if output == "" {
		// Fallback if hook fails
		output = fmt.Sprintf("%s %s", SymbolExec, command)
	}

	// Apply styling to the symbol portion
	if strings.HasPrefix(output, SymbolExec) {
		styled := ExecStartStyle.Render(SymbolExec) + output[len(SymbolExec):]
		fmt.Fprintln(r.writer, styled)
	} else {
		fmt.Fprintln(r.writer, output)
	}
}

// RenderExecEnd renders the completion of an exec tool call using GSH_EXEC_END hook
func (r *Renderer) RenderExecEnd(command string, duration time.Duration, exitCode int) {
	// Extract first word of command
	commandFirstWord := command
	if idx := strings.Index(command, " "); idx > 0 {
		commandFirstWord = command[:idx]
	}

	durationMs := duration.Milliseconds()

	output := r.callStringHook("GSH_EXEC_END", map[string]interface{}{
		"commandFirstWord": commandFirstWord,
		"durationMs":       float64(durationMs),
		"exitCode":         float64(exitCode),
	})

	if output == "" {
		// Fallback if hook fails
		if exitCode == 0 {
			output = fmt.Sprintf("%s %s (%.1fs)", SymbolSuccess, commandFirstWord, duration.Seconds())
		} else {
			output = fmt.Sprintf("%s %s (%.1fs) exit code %d", SymbolError, commandFirstWord, duration.Seconds(), exitCode)
		}
	}

	// Apply styling based on success/failure
	if strings.HasPrefix(output, SymbolSuccess) {
		styled := SuccessStyle.Render(SymbolSuccess) + output[len(SymbolSuccess):]
		fmt.Fprintln(r.writer, styled)
	} else if strings.HasPrefix(output, SymbolError) {
		styled := ErrorStyle.Render(SymbolError) + output[len(SymbolError):]
		fmt.Fprintln(r.writer, styled)
	} else {
		fmt.Fprintln(r.writer, output)
	}
}

// RenderToolPending renders a tool in pending state (streaming args from LLM)
func (r *Renderer) RenderToolPending(toolName string) {
	output := r.callToolStatusHook(toolName, "pending", nil, 0)

	if output == "" {
		output = fmt.Sprintf("%s %s", SymbolToolPending, toolName)
	}

	// Apply styling
	if strings.HasPrefix(output, SymbolToolPending) {
		styled := ToolPendingStyle.Render(SymbolToolPending) + output[len(SymbolToolPending):]
		fmt.Fprint(r.writer, styled)
	} else {
		fmt.Fprint(r.writer, output)
	}
}

// StartToolSpinner starts a spinner for a tool and returns a stop function.
// The stop function blocks until the spinner has fully stopped.
// The spinner will display: "○ toolName ⠋"
func (r *Renderer) StartToolSpinner(ctx context.Context, toolName string) func() {
	spinner := NewInlineSpinner(r.writer)
	prefix := ToolPendingStyle.Render(SymbolToolPending) + " " + toolName
	spinner.SetPrefix(prefix)
	return spinner.Start(ctx)
}

// RenderToolExecuting renders a tool in executing state (args complete, running)
func (r *Renderer) RenderToolExecuting(toolName string, args map[string]interface{}) {
	output := r.callToolStatusHook(toolName, "executing", args, 0)

	if output == "" {
		output = r.formatToolStatus(toolName, "executing", args, 0)
	}

	r.renderToolOutput(output, true)
}

// RenderToolComplete renders a tool in complete state (success or error)
func (r *Renderer) RenderToolComplete(toolName string, args map[string]interface{}, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	durationMs := duration.Milliseconds()
	output := r.callToolStatusHook(toolName, status, args, durationMs)

	if output == "" {
		output = r.formatToolStatus(toolName, status, args, durationMs)
	}

	r.renderToolOutput(output, success)
}

// RenderToolOutput renders tool output using the GSH_TOOL_OUTPUT hook
func (r *Renderer) RenderToolOutput(toolName string, output string) {
	width := r.getTerminalWidth()

	rendered := r.callStringHook("GSH_TOOL_OUTPUT", map[string]interface{}{
		"toolName":      toolName,
		"output":        output,
		"terminalWidth": float64(width),
	})

	// Only print if hook returns non-empty
	if rendered != "" {
		fmt.Fprintln(r.writer, DimStyle.Render(rendered))
	}
}

// RenderSystemMessage renders a system/status message with → prefix
func (r *Renderer) RenderSystemMessage(message string) {
	fmt.Fprintln(r.writer, SystemMessageStyle.Render(fmt.Sprintf("%s %s", SymbolSystemMessage, message)))
}

// callStringHook calls a hook tool and returns its string result
func (r *Renderer) callStringHook(hookName string, args map[string]interface{}) string {
	if r.interp == nil {
		return ""
	}

	// Build the tool call expression
	result, err := r.callTool(hookName, args)
	if err != nil {
		return ""
	}

	if strVal, ok := result.(*interpreter.StringValue); ok {
		return strVal.Value
	}

	return ""
}

// callToolStatusHook calls the GSH_TOOL_STATUS hook
func (r *Renderer) callToolStatusHook(toolName, status string, args map[string]interface{}, durationMs int64) string {
	if r.interp == nil {
		return ""
	}

	// Convert args to an object value
	argsObj := make(map[string]interface{})
	for k, v := range args {
		argsObj[k] = v
	}

	hookArgs := map[string]interface{}{
		"toolName":   toolName,
		"status":     status,
		"args":       argsObj,
		"durationMs": float64(durationMs),
	}

	result, err := r.callTool("GSH_TOOL_STATUS", hookArgs)
	if err != nil {
		return ""
	}

	if strVal, ok := result.(*interpreter.StringValue); ok {
		return strVal.Value
	}

	return ""
}

// callTool invokes a tool defined in the interpreter
func (r *Renderer) callTool(toolName string, args map[string]interface{}) (interpreter.Value, error) {
	// Look up the tool in the interpreter's environment
	vars := r.interp.GetVariables()
	toolVal, exists := vars[toolName]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	tool, ok := toolVal.(*interpreter.ToolValue)
	if !ok {
		return nil, fmt.Errorf("%s is not a tool", toolName)
	}

	// Convert args map to interpreter values in parameter order
	interpArgs := make([]interpreter.Value, len(tool.Parameters))
	for i, paramName := range tool.Parameters {
		if val, exists := args[paramName]; exists {
			interpArgs[i] = toInterpreterValue(val)
		} else {
			interpArgs[i] = &interpreter.NullValue{}
		}
	}

	// Call the tool
	return r.interp.CallTool(tool, interpArgs)
}

// toInterpreterValue converts a Go value to an interpreter Value
func toInterpreterValue(v interface{}) interpreter.Value {
	if v == nil {
		return &interpreter.NullValue{}
	}

	switch val := v.(type) {
	case string:
		return &interpreter.StringValue{Value: val}
	case float64:
		return &interpreter.NumberValue{Value: val}
	case int:
		return &interpreter.NumberValue{Value: float64(val)}
	case int64:
		return &interpreter.NumberValue{Value: float64(val)}
	case bool:
		return &interpreter.BoolValue{Value: val}
	case map[string]interface{}:
		obj := make(map[string]interpreter.Value)
		for k, v := range val {
			obj[k] = toInterpreterValue(v)
		}
		return &interpreter.ObjectValue{Properties: obj}
	case []interface{}:
		arr := make([]interpreter.Value, len(val))
		for i, v := range val {
			arr[i] = toInterpreterValue(v)
		}
		return &interpreter.ArrayValue{Elements: arr}
	default:
		// Fallback: convert to string
		return &interpreter.StringValue{Value: fmt.Sprintf("%v", val)}
	}
}

// formatToolStatus provides fallback formatting for tool status
func (r *Renderer) formatToolStatus(toolName, status string, args map[string]interface{}, durationMs int64) string {
	var sb strings.Builder

	durationSec := float64(durationMs) / 1000.0

	switch status {
	case "pending":
		sb.WriteString(fmt.Sprintf("%s %s", SymbolToolPending, toolName))
	case "executing":
		sb.WriteString(fmt.Sprintf("%s %s", SymbolToolPending, toolName))
		if len(args) > 0 {
			sb.WriteString("\n")
			sb.WriteString(r.formatArgs(args))
		}
	case "success":
		sb.WriteString(fmt.Sprintf("%s %s %s (%.1fs)", SymbolToolComplete, toolName, SymbolSuccess, durationSec))
		if len(args) > 0 {
			sb.WriteString("\n")
			sb.WriteString(r.formatArgs(args))
		}
	case "error":
		sb.WriteString(fmt.Sprintf("%s %s %s (%.1fs)", SymbolToolComplete, toolName, SymbolError, durationSec))
		if len(args) > 0 {
			sb.WriteString("\n")
			sb.WriteString(r.formatArgs(args))
		}
	}

	return sb.String()
}

// formatArgs formats tool arguments for display
func (r *Renderer) formatArgs(args map[string]interface{}) string {
	var sb strings.Builder
	for k, v := range args {
		valueStr := fmt.Sprintf("%v", v)
		if len(valueStr) > 60 {
			valueStr = valueStr[:57] + "..."
		}
		sb.WriteString(fmt.Sprintf("   %s: %s\n", k, valueStr))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// renderToolOutput renders tool status output with appropriate styling
func (r *Renderer) renderToolOutput(output string, success bool) {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i == 0 {
			// First line: style the symbol
			if strings.HasPrefix(line, SymbolToolPending) {
				styled := ToolPendingStyle.Render(SymbolToolPending) + line[len(SymbolToolPending):]
				fmt.Fprintln(r.writer, styled)
			} else if strings.HasPrefix(line, SymbolToolComplete) {
				var styled string
				if success {
					styled = SuccessStyle.Render(SymbolToolComplete) + line[len(SymbolToolComplete):]
				} else {
					styled = ErrorStyle.Render(SymbolToolComplete) + line[len(SymbolToolComplete):]
				}
				fmt.Fprintln(r.writer, styled)
			} else {
				fmt.Fprintln(r.writer, line)
			}
		} else {
			// Subsequent lines (args): print with dim style
			fmt.Fprintln(r.writer, DimStyle.Render(line))
		}
	}
}

// getTerminalWidth returns the current terminal width, with a sensible default
func (r *Renderer) getTerminalWidth() int {
	if r.termWidth != nil {
		width := r.termWidth()
		if width > 0 {
			return width
		}
	}
	return 80 // Default fallback
}

// GetVariable retrieves a variable from the interpreter's environment
func (r *Renderer) GetVariable(name string) interpreter.Value {
	if r.interp == nil {
		return nil
	}
	vars := r.interp.GetVariables()
	return vars[name]
}
