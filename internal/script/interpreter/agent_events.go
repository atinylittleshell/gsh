package interpreter

// Agent lifecycle event names
const (
	EventAgentStart          = "agent.start"
	EventAgentEnd            = "agent.end"
	EventAgentIterationStart = "agent.iteration.start"
	EventAgentIterationEnd   = "agent.iteration.end"
	EventAgentChunk          = "agent.chunk"
	EventAgentToolPending    = "agent.tool.pending"
	EventAgentToolStart      = "agent.tool.start"
	EventAgentToolEnd        = "agent.tool.end"
)

// ToolOverride represents an override returned by an event handler for tool events.
// Used by agent.tool.start and agent.tool.end event handlers to override tool execution.
type ToolOverride struct {
	Result string // The result to use instead of actual tool execution
	Error  string // Optional error message (if set, tool is considered failed)
}

// extractToolOverride extracts a ToolOverride from an event handler return value.
// Event handlers can return { result: "..." } or { result: "...", error: "..." }
// to override tool execution behavior.
// Returns nil if the value is not a valid override.
func extractToolOverride(val Value) *ToolOverride {
	if val == nil {
		return nil
	}

	obj, ok := val.(*ObjectValue)
	if !ok {
		return nil
	}

	// Check for "result" property - required for an override
	resultProp := obj.GetPropertyValue("result")
	if resultProp == nil || resultProp.Type() == ValueTypeNull {
		return nil
	}

	override := &ToolOverride{}

	// Extract result (required)
	if str, ok := resultProp.(*StringValue); ok {
		override.Result = str.Value
	} else {
		// Result must be a string
		return nil
	}

	// Extract error (optional)
	if errorProp := obj.GetPropertyValue("error"); errorProp != nil && errorProp.Type() == ValueTypeString {
		if str, ok := errorProp.(*StringValue); ok {
			override.Error = str.Value
		}
	}

	return override
}

// Helper functions to create event context objects

// agentValueToContextObject converts an AgentValue to an ObjectValue for event contexts.
// This provides access to agent.name, agent.metadata, and other config properties.
func agentValueToContextObject(agent *AgentValue) Value {
	if agent == nil {
		return &NullValue{}
	}

	props := map[string]*PropertyDescriptor{
		"name": {Value: &StringValue{Value: agent.Name}},
	}

	// Add metadata if present (as an object for nested property access)
	if metadataVal, ok := agent.Config["metadata"]; ok {
		props["metadata"] = &PropertyDescriptor{Value: metadataVal}
	} else {
		// Provide empty object so ctx.agent.metadata.* doesn't error
		props["metadata"] = &PropertyDescriptor{Value: &ObjectValue{Properties: map[string]*PropertyDescriptor{}}}
	}

	// Add other config fields that might be useful
	if systemPrompt, ok := agent.Config["systemPrompt"]; ok {
		props["systemPrompt"] = &PropertyDescriptor{Value: systemPrompt}
	}

	return &ObjectValue{Properties: props}
}

// createAgentStartContext creates the context object for agent.start event
// ctx: { agent: { name, metadata, ... }, message: string }
func createAgentStartContext(agent *AgentValue, message string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent":   {Value: agentValueToContextObject(agent)},
			"message": {Value: &StringValue{Value: message}},
		},
	}
}

// createAgentEndContext creates the context object for agent.end event
// ctx: { agent: { name, metadata, ... }, query: { inputTokens, outputTokens, cachedTokens, durationMs }, error }
func createAgentEndContext(agent *AgentValue, stopReason string, durationMs int64, inputTokens, outputTokens, cachedTokens int, err error) Value {
	var errorVal Value = &NullValue{}
	if err != nil {
		errorVal = &StringValue{Value: err.Error()}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent": {Value: agentValueToContextObject(agent)},
			"query": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"inputTokens":  {Value: &NumberValue{Value: float64(inputTokens)}},
					"outputTokens": {Value: &NumberValue{Value: float64(outputTokens)}},
					"cachedTokens": {Value: &NumberValue{Value: float64(cachedTokens)}},
					"durationMs":   {Value: &NumberValue{Value: float64(durationMs)}},
				},
			}},
			"error": {Value: errorVal},
		},
	}
}

// createIterationStartContext creates the context object for agent.iteration.start event
// ctx: { agent: { name, metadata, ... }, iteration: number }
func createIterationStartContext(agent *AgentValue, iteration int) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent":     {Value: agentValueToContextObject(agent)},
			"iteration": {Value: &NumberValue{Value: float64(iteration + 1)}}, // 1-based for users
		},
	}
}

// createIterationEndContext creates the context object for agent.iteration.end event
// ctx: { agent: { name, metadata, ... }, iteration: number, usage: { inputTokens, outputTokens, cachedTokens } }
func createIterationEndContext(agent *AgentValue, iteration int, inputTokens, outputTokens, cachedTokens int) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent":     {Value: agentValueToContextObject(agent)},
			"iteration": {Value: &NumberValue{Value: float64(iteration + 1)}}, // 1-based for users
			"usage": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"inputTokens":  {Value: &NumberValue{Value: float64(inputTokens)}},
					"outputTokens": {Value: &NumberValue{Value: float64(outputTokens)}},
					"cachedTokens": {Value: &NumberValue{Value: float64(cachedTokens)}},
				},
			}},
		},
	}
}

// createChunkContext creates the context object for agent.chunk event
// ctx: { agent: { name, metadata, ... }, content: string }
func createChunkContext(agent *AgentValue, content string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent":   {Value: agentValueToContextObject(agent)},
			"content": {Value: &StringValue{Value: content}},
		},
	}
}

// createToolPendingContext creates the context object for agent.tool.pending event
// This fires when a tool call starts streaming from the LLM (status: pending, before args are complete)
// ctx: { agent: { name, metadata, ... }, toolCall: { id, name, status } }
func createToolPendingContext(agent *AgentValue, id, name string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent": {Value: agentValueToContextObject(agent)},
			"toolCall": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"id":     {Value: &StringValue{Value: id}},
					"name":   {Value: &StringValue{Value: name}},
					"status": {Value: &StringValue{Value: "pending"}},
				},
			}},
		},
	}
}

// createToolCallContext creates the context object for agent.tool.start/end events
// ctx: { agent: { name, metadata, ... }, toolCall: { id, name, args, durationMs?, output?, error? } }
func createToolCallContext(agent *AgentValue, id, name string, args map[string]interface{}, durationMs *int64, output *string, err error) Value {
	// Convert args to ObjectValue
	argsProps := make(map[string]*PropertyDescriptor)
	for k, v := range args {
		argsProps[k] = &PropertyDescriptor{Value: InterfaceToValue(v)}
	}

	toolCallProps := map[string]*PropertyDescriptor{
		"id":   {Value: &StringValue{Value: id}},
		"name": {Value: &StringValue{Value: name}},
		"args": {Value: &ObjectValue{Properties: argsProps}},
	}

	if durationMs != nil {
		toolCallProps["durationMs"] = &PropertyDescriptor{Value: &NumberValue{Value: float64(*durationMs)}}
	}
	if output != nil {
		toolCallProps["output"] = &PropertyDescriptor{Value: &StringValue{Value: *output}}
	}
	if err != nil {
		toolCallProps["error"] = &PropertyDescriptor{Value: &StringValue{Value: err.Error()}}
	} else if durationMs != nil {
		// Only set error to null on end events (when durationMs is present)
		toolCallProps["error"] = &PropertyDescriptor{Value: &NullValue{}}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent":    {Value: agentValueToContextObject(agent)},
			"toolCall": {Value: &ObjectValue{Properties: toolCallProps}},
		},
	}
}

// CreateToolStartContext creates the context object for agent.tool.start event (public version)
func CreateToolStartContext(name string, args map[string]interface{}) Value {
	// Convert args to ObjectValue
	argsProps := make(map[string]*PropertyDescriptor)
	for k, v := range args {
		argsProps[k] = &PropertyDescriptor{Value: InterfaceToValue(v)}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"toolCall": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"name":   {Value: &StringValue{Value: name}},
					"status": {Value: &StringValue{Value: "executing"}},
					"args":   {Value: &ObjectValue{Properties: argsProps}},
				},
			}},
		},
	}
}

// CreateToolEndContext creates the context object for agent.tool.end event (public version)
func CreateToolEndContext(name string, args map[string]interface{}, durationMs int64, success bool, err error) Value {
	// Convert args to ObjectValue
	argsProps := make(map[string]*PropertyDescriptor)
	for k, v := range args {
		argsProps[k] = &PropertyDescriptor{Value: InterfaceToValue(v)}
	}

	status := "success"
	if !success {
		status = "error"
	}

	var errorVal Value = &NullValue{}
	if err != nil {
		errorVal = &StringValue{Value: err.Error()}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"toolCall": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"name":       {Value: &StringValue{Value: name}},
					"status":     {Value: &StringValue{Value: status}},
					"args":       {Value: &ObjectValue{Properties: argsProps}},
					"durationMs": {Value: &NumberValue{Value: float64(durationMs)}},
					"error":      {Value: errorVal},
				},
			}},
		},
	}
}
