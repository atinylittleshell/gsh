package interpreter

import "fmt"

// REPL event names
const (
	EventReplReady         = "repl.ready"
	EventReplExit          = "repl.exit"
	EventReplPrompt        = "repl.prompt"
	EventReplCommandBefore = "repl.command.before"
	EventReplCommandAfter  = "repl.command.after"
	EventReplPredict       = "repl.predict"
)

// CreateReplReadyContext creates the context object for repl.ready event
// ctx: null (no context needed)
func CreateReplReadyContext() Value {
	return &NullValue{}
}

// CreateReplExitContext creates the context object for repl.exit event
// ctx: null (no context needed)
func CreateReplExitContext() Value {
	return &NullValue{}
}

// CreateReplPromptContext creates the context object for repl.prompt event
// ctx: null (no context needed)
func CreateReplPromptContext() Value {
	return &NullValue{}
}

// CreateReplCommandBeforeContext creates the context object for repl.command.before event
// ctx: { command: string }
func CreateReplCommandBeforeContext(command string) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"command": {Value: &StringValue{Value: command}},
		},
	}
}

// CreateReplCommandAfterContext creates the context object for repl.command.after event
// ctx: { command: string, exitCode: number, durationMs: number }
func CreateReplCommandAfterContext(command string, exitCode int, durationMs int64) Value {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"command":    {Value: &StringValue{Value: command}},
			"exitCode":   {Value: &NumberValue{Value: float64(exitCode)}},
			"durationMs": {Value: &NumberValue{Value: float64(durationMs)}},
		},
	}
}

// PredictTrigger represents the trigger type for prediction events
type PredictTrigger string

const (
	// PredictTriggerInstant is used for instant predictions (must be fast, e.g., history lookup)
	PredictTriggerInstant PredictTrigger = "instant"
	// PredictTriggerDebounced is used for debounced predictions (can be slow, e.g., LLM)
	PredictTriggerDebounced PredictTrigger = "debounced"
)

// CreateReplPredictContext creates the context object for repl.predict event
// ctx: { input: string, trigger: "instant" | "debounced", existingPrediction: string | null }
func CreateReplPredictContext(input string, trigger PredictTrigger, existingPrediction string) Value {
	var existingPredictionValue Value
	if existingPrediction == "" {
		existingPredictionValue = &NullValue{}
	} else {
		existingPredictionValue = &StringValue{Value: existingPrediction}
	}

	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"input":              {Value: &StringValue{Value: input}},
			"trigger":            {Value: &StringValue{Value: string(trigger)}},
			"existingPrediction": {Value: existingPredictionValue},
		},
	}
}

// ExtractPredictionResult extracts a prediction result from an event handler return value.
// Supported return formats:
//   - string: treated as the prediction
//   - object: { prediction: string, error?: string }
//
// Returns prediction, error (if provided), and a boolean indicating whether a value was handled.
func ExtractPredictionResult(val Value) (string, error, bool) {
	if val == nil || val.Type() == ValueTypeNull {
		return "", nil, false
	}

	// String return - treat as prediction
	if str, ok := val.(*StringValue); ok {
		return str.Value, nil, true
	}

	// Object return - look for prediction/error fields
	obj, ok := val.(*ObjectValue)
	if !ok {
		return "", fmt.Errorf("unexpected prediction return type: %s", val.Type()), true
	}

	if errVal := obj.GetPropertyValue("error"); errVal != nil && errVal.Type() != ValueTypeNull {
		if errStr, ok := errVal.(*StringValue); ok {
			return "", fmt.Errorf(errStr.Value), true
		}
		return "", fmt.Errorf("error must be a string, got %s", errVal.Type()), true
	}

	predVal := obj.GetPropertyValue("prediction")
	if predVal == nil || predVal.Type() == ValueTypeNull {
		return "", nil, true
	}

	predStr, ok := predVal.(*StringValue)
	if !ok {
		return "", fmt.Errorf("prediction must be a string, got %s", predVal.Type()), true
	}

	return predStr.Value, nil, true
}
