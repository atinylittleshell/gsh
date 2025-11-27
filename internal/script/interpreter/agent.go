package interpreter

import (
	"fmt"

	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// evalAgentDeclaration evaluates an agent declaration
func (i *Interpreter) evalAgentDeclaration(node *parser.AgentDeclaration) (Value, error) {
	agentName := node.Name.Value

	// Evaluate each config field and store as Value
	config := make(map[string]Value)

	for key, expr := range node.Config {
		value, err := i.evalExpression(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate agent config field '%s': %w", key, err)
		}

		// Validate common config fields
		switch key {
		case "model":
			// model must be a reference to a ModelValue (not a string)
			if _, ok := value.(*ModelValue); !ok {
				return nil, fmt.Errorf("agent config 'model' must be a model reference, got %s", value.Type())
			}
		case "systemPrompt":
			if _, ok := value.(*StringValue); !ok {
				return nil, fmt.Errorf("agent config 'systemPrompt' must be a string, got %s", value.Type())
			}
		case "tools":
			if _, ok := value.(*ArrayValue); !ok {
				return nil, fmt.Errorf("agent config 'tools' must be an array, got %s", value.Type())
			}
			// Allow other fields without validation for extensibility
		}

		config[key] = value
	}

	// Validate required fields
	if _, ok := config["model"]; !ok {
		return nil, fmt.Errorf("agent '%s' must have a 'model' field", agentName)
	}

	// Create the agent value
	agent := &AgentValue{
		Name:   agentName,
		Config: config,
	}

	// Register the agent in the environment
	err := i.env.Define(agentName, agent)
	if err != nil {
		return nil, err
	}

	return agent, nil
}
