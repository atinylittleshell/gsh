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

// Helper functions to create event context objects

// createAgentStartContext creates the context object for agent.start event
// ctx: { agent: { name }, message: string }
func createAgentStartContext(agentName, message string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"name": {Value: &StringValue{Value: agentName}},
				},
			}},
			"message": {Value: &StringValue{Value: message}},
		},
	}
}

// createAgentEndContext creates the context object for agent.end event
// ctx: { agent: { name }, query: { inputTokens, outputTokens, cachedTokens, durationMs }, error }
func createAgentEndContext(agentName string, stopReason string, durationMs int64, inputTokens, outputTokens, cachedTokens int, err error) Value {
	var errorVal Value = &NullValue{}
	if err != nil {
		errorVal = &StringValue{Value: err.Error()}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"agent": {Value: &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"name": {Value: &StringValue{Value: agentName}},
				},
			}},
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
// ctx: { iteration: number }
func createIterationStartContext(iteration int) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"iteration": {Value: &NumberValue{Value: float64(iteration + 1)}}, // 1-based for users
		},
	}
}

// createIterationEndContext creates the context object for agent.iteration.end event
// ctx: { iteration: number, usage: { inputTokens, outputTokens, cachedTokens } }
func createIterationEndContext(iteration int, inputTokens, outputTokens, cachedTokens int) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
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
// ctx: { content: string }
func createChunkContext(content string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"content": {Value: &StringValue{Value: content}},
		},
	}
}

// createToolPendingContext creates the context object for agent.tool.pending event
// This fires when a tool call starts streaming from the LLM (status: pending, before args are complete)
// ctx: { toolCall: { id, name, status } }
func createToolPendingContext(id, name string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
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
// ctx: { toolCall: { id, name, args, durationMs?, output?, error? } }
func createToolCallContext(id, name string, args map[string]interface{}, durationMs *int64, output *string, err error) Value {
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
