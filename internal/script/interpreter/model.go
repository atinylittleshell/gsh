package interpreter

import (
	"fmt"

	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// evalModelDeclaration evaluates a model declaration
func (i *Interpreter) evalModelDeclaration(node *parser.ModelDeclaration) (Value, error) {
	modelName := node.Name.Value

	// Evaluate each config field and store as Value
	config := make(map[string]Value)

	for key, expr := range node.Config {
		value, err := i.evalExpression(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate model config field '%s': %w", key, err)
		}

		// Validate common config fields
		switch key {
		case "provider":
			if _, ok := value.(*StringValue); !ok {
				return nil, fmt.Errorf("model config 'provider' must be a string, got %s", value.Type())
			}
		case "apiKey":
			// apiKey can be any type (string, null for missing env vars, etc.)
			// No validation needed - it will be used as-is
		case "model":
			if _, ok := value.(*StringValue); !ok {
				return nil, fmt.Errorf("model config 'model' must be a string, got %s", value.Type())
			}
		case "baseURL":
			if _, ok := value.(*StringValue); !ok {
				return nil, fmt.Errorf("model config 'baseURL' must be a string, got %s", value.Type())
			}
		case "temperature":
			if _, ok := value.(*NumberValue); !ok {
				return nil, fmt.Errorf("model config 'temperature' must be a number, got %s", value.Type())
			}
		case "maxTokens":
			if _, ok := value.(*NumberValue); !ok {
				return nil, fmt.Errorf("model config 'maxTokens' must be a number, got %s", value.Type())
			}
		case "headers":
			// headers must be an object with string values
			obj, ok := value.(*ObjectValue)
			if !ok {
				return nil, fmt.Errorf("model config 'headers' must be an object, got %s", value.Type())
			}
			// Validate that all header values are strings
			for headerKey := range obj.Properties {
				headerVal := obj.GetPropertyValue(headerKey)
				if _, ok := headerVal.(*StringValue); !ok {
					return nil, fmt.Errorf("model config 'headers.%s' must be a string, got %s", headerKey, headerVal.Type())
				}
			}
			// Allow other fields without validation for extensibility
		}

		config[key] = value
	}

	// Resolve provider from registry
	var provider ModelProvider
	if providerVal, ok := config["provider"]; ok {
		if providerStr, ok := providerVal.(*StringValue); ok {
			var found bool
			provider, found = i.providerRegistry.Get(providerStr.Value)
			if !found {
				return nil, fmt.Errorf("unknown model provider: %s", providerStr.Value)
			}
		}
	}

	// Create the model value with resolved provider
	model := &ModelValue{
		Name:     modelName,
		Config:   config,
		Provider: provider,
	}

	// Register the model in the environment
	i.env.Set(modelName, model)

	return model, nil
}
