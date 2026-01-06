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

// PredictionHistoryItem represents a single history entry passed to repl.predict middleware.
type PredictionHistoryItem struct {
	Command   string
	Directory string
	ExitCode  *int32
}

// PredictionEventResult represents the result returned by repl.predict middleware.
type PredictionEventResult struct {
	Prediction string
	Source     string
}

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

// CreateReplPredictContext creates the context object for repl.predict event
// ctx: { input: string, history: Array<PredictionHistoryItem>, source?: string }
func CreateReplPredictContext(input string, history []PredictionHistoryItem, source string) Value {
	historyValues := make([]Value, 0, len(history))
	for _, entry := range history {
		props := map[string]*PropertyDescriptor{
			"command": &PropertyDescriptor{Value: &StringValue{Value: entry.Command}},
		}

		if entry.Directory != "" {
			props["directory"] = &PropertyDescriptor{Value: &StringValue{Value: entry.Directory}}
		} else {
			props["directory"] = &PropertyDescriptor{Value: &NullValue{}}
		}

		if entry.ExitCode != nil {
			props["exitCode"] = &PropertyDescriptor{Value: &NumberValue{Value: float64(*entry.ExitCode)}}
		} else {
			props["exitCode"] = &PropertyDescriptor{Value: &NullValue{}}
		}

		historyValues = append(historyValues, &ObjectValue{
			Properties: props,
		})
	}

	ctxObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"input":   &PropertyDescriptor{Value: &StringValue{Value: input}},
			"history": &PropertyDescriptor{Value: &ArrayValue{Elements: historyValues}},
		},
	}

	if source != "" {
		ctxObj.Properties["source"] = &PropertyDescriptor{Value: &StringValue{Value: source}}
	} else {
		ctxObj.Properties["source"] = &PropertyDescriptor{Value: &NullValue{}}
	}

	return ctxObj
}

// ExtractPredictionResult extracts a prediction result from an event handler return value.
// Supported return formats:
//   - string: treated as the prediction
//   - object: { prediction: string, error?: string }
//
// Returns prediction, error (if provided), and a boolean indicating whether a value was handled.
func ExtractPredictionResult(val Value) (PredictionEventResult, error, bool) {
	result := PredictionEventResult{}

	if val == nil || val.Type() == ValueTypeNull {
		return result, nil, false
	}

	// String return - treat as prediction
	if str, ok := val.(*StringValue); ok {
		result.Prediction = str.Value
		return result, nil, true
	}

	// Object return - look for prediction/error fields
	obj, ok := val.(*ObjectValue)
	if !ok {
		return result, fmt.Errorf("unexpected prediction return type: %s", val.Type()), true
	}

	if errVal := obj.GetPropertyValue("error"); errVal != nil && errVal.Type() != ValueTypeNull {
		if errStr, ok := errVal.(*StringValue); ok {
			return result, fmt.Errorf(errStr.Value), true
		}
		return result, fmt.Errorf("error must be a string, got %s", errVal.Type()), true
	}

	predVal := obj.GetPropertyValue("prediction")
	if predVal == nil || predVal.Type() == ValueTypeNull {
		return result, nil, true
	}

	predStr, ok := predVal.(*StringValue)
	if !ok {
		return result, fmt.Errorf("prediction must be a string, got %s", predVal.Type()), true
	}

	result.Prediction = predStr.Value

	if sourceVal := obj.GetPropertyValue("source"); sourceVal != nil && sourceVal.Type() != ValueTypeNull {
		if sourceStr, ok := sourceVal.(*StringValue); ok {
			result.Source = sourceStr.Value
		} else {
			return result, fmt.Errorf("source must be a string, got %s", sourceVal.Type()), true
		}
	}

	return result, nil, true
}
