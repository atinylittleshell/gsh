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
	interp       *interpreter.Interpreter // For calling custom hooks
	writer       io.Writer
	termWidth    func() int // Function to get current terminal width
	termHeight   func() int // Function to get current terminal height
	currentAgent string     // Name of the current agent

	// Track lines printed by RenderToolExecuting so RenderToolComplete can replace them
	lastToolExecutingLines int
}

// New creates a new Renderer instance
func New(interp *interpreter.Interpreter, writer io.Writer, termWidth func() int) *Renderer {
	return &Renderer{
		interp:    interp,
		writer:    writer,
		termWidth: termWidth,
	}
}

// SetTermHeight sets the function to get terminal height
func (r *Renderer) SetTermHeight(termHeight func() int) {
	r.termHeight = termHeight
}

// SetCurrentAgent sets the current agent name for the render context
func (r *Renderer) SetCurrentAgent(agentName string) {
	r.currentAgent = agentName
}

// RenderContext represents the context passed to all render hooks.
// All fields except terminal may be null depending on the hook being called.
type RenderContext struct {
	Terminal *TerminalContext `json:"terminal"`
	Agent    *AgentContext    `json:"agent"`
	Repl     *ReplContext     `json:"repl"`
	Query    *QueryContext    `json:"query"`
	Exec     *ExecContext     `json:"exec"`
	ToolCall *ToolCallContext `json:"toolCall"`
}

// ReplContext contains REPL state information (e.g., for prompt rendering)
type ReplContext struct {
	LastExitCode   int   `json:"lastExitCode"`
	LastDurationMs int64 `json:"lastDurationMs"`
}

// TerminalContext contains terminal information
type TerminalContext struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// AgentContext contains agent information
type AgentContext struct {
	Name string `json:"name"`
}

// QueryContext contains query/turn statistics
type QueryContext struct {
	DurationMs   int64 `json:"durationMs"`
	InputTokens  int   `json:"inputTokens"`
	OutputTokens int   `json:"outputTokens"`
	CachedTokens int   `json:"cachedTokens"` // Number of input tokens that were cache hits
}

// ExecContext contains exec tool information
type ExecContext struct {
	Command          string `json:"command"`
	CommandFirstWord string `json:"commandFirstWord"`
	DurationMs       int64  `json:"durationMs"`
	ExitCode         int    `json:"exitCode"`
}

// ToolCallContext contains non-exec tool call information
type ToolCallContext struct {
	Name       string                 `json:"name"`
	Status     string                 `json:"status"` // "pending", "executing", "success", "error"
	Args       map[string]interface{} `json:"args"`
	DurationMs int64                  `json:"durationMs"`
	Output     string                 `json:"output"`
}

// newBaseContext creates a RenderContext with terminal and agent info populated
func (r *Renderer) newBaseContext() *RenderContext {
	ctx := &RenderContext{
		Terminal: &TerminalContext{
			Width:  r.getTerminalWidth(),
			Height: r.getTerminalHeight(),
		},
	}

	if r.currentAgent != "" {
		ctx.Agent = &AgentContext{Name: r.currentAgent}
	}

	return ctx
}

// NewPromptContext creates a RenderContext for GSH_PROMPT with last command info
func (r *Renderer) NewPromptContext(lastExitCode int, lastDurationMs int64) *RenderContext {
	ctx := r.newBaseContext()
	ctx.Repl = &ReplContext{
		LastExitCode:   lastExitCode,
		LastDurationMs: lastDurationMs,
	}
	return ctx
}

// CallPromptHook calls the GSH_PROMPT hook with the given context and returns the prompt string
func (r *Renderer) CallPromptHook(ctx *RenderContext) string {
	return r.callHookWithContext("GSH_PROMPT", ctx)
}

// toInterpreterObject converts a RenderContext to an interpreter ObjectValue
func (r *Renderer) contextToInterpreterObject(ctx *RenderContext) *interpreter.ObjectValue {
	props := make(map[string]interpreter.Value)

	// Terminal is always present
	if ctx.Terminal != nil {
		props["terminal"] = &interpreter.ObjectValue{
			Properties: map[string]interpreter.Value{
				"width":  &interpreter.NumberValue{Value: float64(ctx.Terminal.Width)},
				"height": &interpreter.NumberValue{Value: float64(ctx.Terminal.Height)},
			},
		}
	}

	// Agent may be null
	if ctx.Agent != nil {
		props["agent"] = &interpreter.ObjectValue{
			Properties: map[string]interpreter.Value{
				"name": &interpreter.StringValue{Value: ctx.Agent.Name},
			},
		}
	} else {
		props["agent"] = &interpreter.NullValue{}
	}

	// Repl may be null
	if ctx.Repl != nil {
		props["repl"] = &interpreter.ObjectValue{
			Properties: map[string]interpreter.Value{
				"lastExitCode":   &interpreter.NumberValue{Value: float64(ctx.Repl.LastExitCode)},
				"lastDurationMs": &interpreter.NumberValue{Value: float64(ctx.Repl.LastDurationMs)},
			},
		}
	} else {
		props["repl"] = &interpreter.NullValue{}
	}

	// Query may be null
	if ctx.Query != nil {
		props["query"] = &interpreter.ObjectValue{
			Properties: map[string]interpreter.Value{
				"durationMs":   &interpreter.NumberValue{Value: float64(ctx.Query.DurationMs)},
				"inputTokens":  &interpreter.NumberValue{Value: float64(ctx.Query.InputTokens)},
				"outputTokens": &interpreter.NumberValue{Value: float64(ctx.Query.OutputTokens)},
				"cachedTokens": &interpreter.NumberValue{Value: float64(ctx.Query.CachedTokens)},
			},
		}
	} else {
		props["query"] = &interpreter.NullValue{}
	}

	// Exec may be null
	if ctx.Exec != nil {
		props["exec"] = &interpreter.ObjectValue{
			Properties: map[string]interpreter.Value{
				"command":          &interpreter.StringValue{Value: ctx.Exec.Command},
				"commandFirstWord": &interpreter.StringValue{Value: ctx.Exec.CommandFirstWord},
				"durationMs":       &interpreter.NumberValue{Value: float64(ctx.Exec.DurationMs)},
				"exitCode":         &interpreter.NumberValue{Value: float64(ctx.Exec.ExitCode)},
			},
		}
	} else {
		props["exec"] = &interpreter.NullValue{}
	}

	// ToolCall may be null
	if ctx.ToolCall != nil {
		argsObj := make(map[string]interpreter.Value)
		for k, v := range ctx.ToolCall.Args {
			argsObj[k] = toInterpreterValue(v)
		}

		props["toolCall"] = &interpreter.ObjectValue{
			Properties: map[string]interpreter.Value{
				"name":       &interpreter.StringValue{Value: ctx.ToolCall.Name},
				"status":     &interpreter.StringValue{Value: ctx.ToolCall.Status},
				"args":       &interpreter.ObjectValue{Properties: argsObj},
				"durationMs": &interpreter.NumberValue{Value: float64(ctx.ToolCall.DurationMs)},
				"output":     &interpreter.StringValue{Value: ctx.ToolCall.Output},
			},
		}
	} else {
		props["toolCall"] = &interpreter.NullValue{}
	}

	return &interpreter.ObjectValue{Properties: props}
}

// RenderAgentHeader renders the agent header line using the GSH_AGENT_HEADER hook
func (r *Renderer) RenderAgentHeader(agentName string) {
	// Update current agent for context
	r.currentAgent = agentName

	ctx := r.newBaseContext()
	header := r.callHookWithContext("GSH_AGENT_HEADER", ctx)

	if header == "" {
		// Fallback if hook fails
		header = fmt.Sprintf("── agent: %s ───", agentName)
	}

	fmt.Fprintln(r.writer, HeaderStyle.Render(header))
}

// RenderAgentFooter renders the agent footer line using the GSH_AGENT_FOOTER hook
func (r *Renderer) RenderAgentFooter(inputTokens, outputTokens, cachedTokens int, duration time.Duration) {
	ctx := r.newBaseContext()
	ctx.Query = &QueryContext{
		DurationMs:   duration.Milliseconds(),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CachedTokens: cachedTokens,
	}

	footer := r.callHookWithContext("GSH_AGENT_FOOTER", ctx)

	if footer == "" {
		// Fallback if hook fails - include cache ratio next to input tokens if there are cached tokens
		if cachedTokens > 0 && inputTokens > 0 {
			cacheRatio := float64(cachedTokens) / float64(inputTokens) * 100
			footer = fmt.Sprintf("── %d in (%.0f%% cached) · %d out · %.1fs ───", inputTokens, cacheRatio, outputTokens, duration.Seconds())
		} else {
			footer = fmt.Sprintf("── %d in · %d out · %.1fs ───", inputTokens, outputTokens, duration.Seconds())
		}
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
	// Extract first word of command
	commandFirstWord := command
	if idx := strings.Index(command, " "); idx > 0 {
		commandFirstWord = command[:idx]
	}

	ctx := r.newBaseContext()
	ctx.Exec = &ExecContext{
		Command:          command,
		CommandFirstWord: commandFirstWord,
		DurationMs:       0,
		ExitCode:         0,
	}

	output := r.callHookWithContext("GSH_EXEC_START", ctx)

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

	ctx := r.newBaseContext()
	ctx.Exec = &ExecContext{
		Command:          command,
		CommandFirstWord: commandFirstWord,
		DurationMs:       duration.Milliseconds(),
		ExitCode:         exitCode,
	}

	output := r.callHookWithContext("GSH_EXEC_END", ctx)

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
	ctx := r.newBaseContext()
	ctx.ToolCall = &ToolCallContext{
		Name:       toolName,
		Status:     "pending",
		Args:       make(map[string]interface{}),
		DurationMs: 0,
		Output:     "",
	}

	output := r.callHookWithContext("GSH_TOOL_STATUS", ctx)

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
	ctx := r.newBaseContext()
	ctx.ToolCall = &ToolCallContext{
		Name:       toolName,
		Status:     "executing",
		Args:       args,
		DurationMs: 0,
		Output:     "",
	}

	output := r.callHookWithContext("GSH_TOOL_STATUS", ctx)

	if output == "" {
		output = r.formatToolStatus(toolName, "executing", args, 0)
	}

	// Count how many lines this output will produce (for later replacement)
	r.lastToolExecutingLines = strings.Count(output, "\n") + 1

	r.renderToolOutput(output, true)
}

// RenderToolComplete renders a tool in complete state (success or error)
// It replaces the previously rendered "executing" lines with the completion status.
func (r *Renderer) RenderToolComplete(toolName string, args map[string]interface{}, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	ctx := r.newBaseContext()
	ctx.ToolCall = &ToolCallContext{
		Name:       toolName,
		Status:     status,
		Args:       args,
		DurationMs: duration.Milliseconds(),
		Output:     "",
	}

	output := r.callHookWithContext("GSH_TOOL_STATUS", ctx)

	if output == "" {
		output = r.formatToolStatus(toolName, status, args, duration.Milliseconds())
	}

	// Move cursor up and clear the lines printed by RenderToolExecuting
	if r.lastToolExecutingLines > 0 {
		// Move cursor up N lines and clear each line
		for i := 0; i < r.lastToolExecutingLines; i++ {
			fmt.Fprintf(r.writer, "\033[A\033[K") // Move up one line and clear it
		}
		r.lastToolExecutingLines = 0
	}

	r.renderToolOutput(output, success)
}

// RenderToolOutput renders tool output using the GSH_TOOL_OUTPUT hook
func (r *Renderer) RenderToolOutput(toolName string, output string) {
	ctx := r.newBaseContext()
	ctx.ToolCall = &ToolCallContext{
		Name:       toolName,
		Status:     "",
		Args:       make(map[string]interface{}),
		DurationMs: 0,
		Output:     output,
	}

	rendered := r.callHookWithContext("GSH_TOOL_OUTPUT", ctx)

	// Only print if hook returns non-empty
	if rendered != "" {
		fmt.Fprintln(r.writer, DimStyle.Render(rendered))
	}
}

// RenderSystemMessage renders a system/status message with → prefix
func (r *Renderer) RenderSystemMessage(message string) {
	fmt.Fprintln(r.writer, SystemMessageStyle.Render(fmt.Sprintf("%s %s", SymbolSystemMessage, message)))
}

// callHookWithContext calls a hook tool with the RenderContext and returns its string result
func (r *Renderer) callHookWithContext(hookName string, ctx *RenderContext) string {
	if r.interp == nil {
		return ""
	}

	// Look up the tool in the interpreter's environment
	vars := r.interp.GetVariables()
	toolVal, exists := vars[hookName]
	if !exists {
		return ""
	}

	tool, ok := toolVal.(*interpreter.ToolValue)
	if !ok {
		return ""
	}

	// Convert context to interpreter object and pass as single argument
	ctxObj := r.contextToInterpreterObject(ctx)
	interpArgs := []interpreter.Value{ctxObj}

	// Call the tool
	result, err := r.interp.CallTool(tool, interpArgs)
	if err != nil {
		return ""
	}

	if strVal, ok := result.(*interpreter.StringValue); ok {
		return strVal.Value
	}

	return ""
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

// getTerminalHeight returns the current terminal height, with a sensible default
func (r *Renderer) getTerminalHeight() int {
	if r.termHeight != nil {
		height := r.termHeight()
		if height > 0 {
			return height
		}
	}
	return 24 // Default fallback
}

// GetVariable retrieves a variable from the interpreter's environment
func (r *Renderer) GetVariable(name string) interpreter.Value {
	if r.interp == nil {
		return nil
	}
	vars := r.interp.GetVariables()
	return vars[name]
}
