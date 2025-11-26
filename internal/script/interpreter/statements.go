package interpreter

import (
	"fmt"

	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// ControlFlowSignal represents a control flow interruption (break, continue, return)
type ControlFlowSignal int

const (
	// SignalNone indicates normal execution
	SignalNone ControlFlowSignal = iota
	// SignalBreak indicates a break statement
	SignalBreak
	// SignalContinue indicates a continue statement
	SignalContinue
	// SignalReturn indicates a return statement
	SignalReturn
)

// ControlFlowError represents a control flow interruption
type ControlFlowError struct {
	Signal ControlFlowSignal
	Value  Value // For return statements
}

func (c *ControlFlowError) Error() string {
	switch c.Signal {
	case SignalBreak:
		return "break statement outside of loop"
	case SignalContinue:
		return "continue statement outside of loop"
	case SignalReturn:
		return "return statement outside of tool"
	default:
		return "unknown control flow signal"
	}
}

// evalStatement evaluates a statement
func (i *Interpreter) evalStatement(stmt parser.Statement) (Value, error) {
	switch node := stmt.(type) {
	case *parser.AssignmentStatement:
		return i.evalAssignmentStatement(node)
	case *parser.ExpressionStatement:
		return i.evalExpression(node.Expression)
	case *parser.IfStatement:
		return i.evalIfStatement(node)
	case *parser.WhileStatement:
		return i.evalWhileStatement(node)
	case *parser.ForOfStatement:
		return i.evalForOfStatement(node)
	case *parser.BreakStatement:
		return nil, &ControlFlowError{Signal: SignalBreak}
	case *parser.ContinueStatement:
		return nil, &ControlFlowError{Signal: SignalContinue}
	case *parser.ReturnStatement:
		return i.evalReturnStatement(node)
	case *parser.ToolDeclaration:
		return i.evalToolDeclaration(node)
	case *parser.McpDeclaration:
		return i.evalMcpDeclaration(node)
	case *parser.BlockStatement:
		return i.evalBlockStatement(node)
	case *parser.TryStatement:
		return i.evalTryStatement(node)
	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// evalAssignmentStatement evaluates an assignment statement
func (i *Interpreter) evalAssignmentStatement(stmt *parser.AssignmentStatement) (Value, error) {
	// Evaluate the right-hand side
	value, err := i.evalExpression(stmt.Value)
	if err != nil {
		return nil, err
	}

	// Check if variable already exists
	varName := stmt.Name.Value
	if i.env.Has(varName) {
		// Variable exists, update it
		err := i.env.Update(varName, value)
		if err != nil {
			return nil, err
		}
	} else {
		// Variable doesn't exist, define it
		err := i.env.Define(varName, value)
		if err != nil {
			return nil, err
		}
	}

	return value, nil
}

// evalIfStatement evaluates an if statement
func (i *Interpreter) evalIfStatement(node *parser.IfStatement) (Value, error) {
	// Evaluate the condition
	condition, err := i.evalExpression(node.Condition)
	if err != nil {
		return nil, err
	}

	// Check if condition is truthy
	if condition.IsTruthy() {
		// Execute consequence block
		return i.evalBlockStatement(node.Consequence)
	} else if node.Alternative != nil {
		// Execute alternative (else or else if)
		return i.evalStatement(node.Alternative)
	}

	return &NullValue{}, nil
}

// evalWhileStatement evaluates a while statement
func (i *Interpreter) evalWhileStatement(node *parser.WhileStatement) (Value, error) {
	var result Value = &NullValue{}

	for {
		// Evaluate the condition
		condition, err := i.evalExpression(node.Condition)
		if err != nil {
			return nil, err
		}

		// If condition is false, exit loop
		if !condition.IsTruthy() {
			break
		}

		// Execute the body
		result, err = i.evalBlockStatement(node.Body)
		if err != nil {
			// Check for control flow signals
			if cfErr, ok := err.(*ControlFlowError); ok {
				switch cfErr.Signal {
				case SignalBreak:
					// Break out of the loop
					return &NullValue{}, nil
				case SignalContinue:
					// Continue to next iteration
					continue
				default:
					// Other signals (like return) propagate up
					return nil, err
				}
			}
			return nil, err
		}
	}

	return result, nil
}

// evalForOfStatement evaluates a for-of statement
func (i *Interpreter) evalForOfStatement(node *parser.ForOfStatement) (Value, error) {
	// Evaluate the iterable expression
	iterable, err := i.evalExpression(node.Iterable)
	if err != nil {
		return nil, err
	}

	// Get the elements to iterate over
	var elements []Value
	switch iter := iterable.(type) {
	case *ArrayValue:
		elements = iter.Elements
	case *StringValue:
		// Iterate over characters in the string
		runes := []rune(iter.Value)
		elements = make([]Value, len(runes))
		for i, r := range runes {
			elements[i] = &StringValue{Value: string(r)}
		}
	default:
		return nil, fmt.Errorf("for-of requires an iterable (array or string), got %s", iterable.Type())
	}

	var result Value = &NullValue{}

	// Iterate over elements
	for _, elem := range elements {
		// Set the loop variable
		i.env.Set(node.Variable.Value, elem)

		// Execute the body
		result, err = i.evalBlockStatement(node.Body)
		if err != nil {
			// Check for control flow signals
			if cfErr, ok := err.(*ControlFlowError); ok {
				switch cfErr.Signal {
				case SignalBreak:
					// Break out of the loop
					return &NullValue{}, nil
				case SignalContinue:
					// Continue to next iteration
					continue
				default:
					// Other signals (like return) propagate up
					return nil, err
				}
			}
			return nil, err
		}
	}

	return result, nil
}

// evalBlockStatement evaluates a block statement
func (i *Interpreter) evalBlockStatement(node *parser.BlockStatement) (Value, error) {
	// Create a new enclosed environment for the block scope
	prevEnv := i.env
	i.env = NewEnclosedEnvironment(prevEnv)
	defer func() {
		i.env = prevEnv
	}()

	var result Value = &NullValue{}

	for _, stmt := range node.Statements {
		val, err := i.evalStatement(stmt)
		if err != nil {
			return nil, err
		}
		result = val
	}

	return result, nil
}

// evalReturnStatement evaluates a return statement
func (i *Interpreter) evalReturnStatement(node *parser.ReturnStatement) (Value, error) {
	var returnValue Value = &NullValue{}

	if node.ReturnValue != nil {
		val, err := i.evalExpression(node.ReturnValue)
		if err != nil {
			return nil, err
		}
		returnValue = val
	}

	return nil, &ControlFlowError{
		Signal: SignalReturn,
		Value:  returnValue,
	}
}

// evalToolDeclaration evaluates a tool declaration
func (i *Interpreter) evalToolDeclaration(node *parser.ToolDeclaration) (Value, error) {
	// Extract parameter names and types
	params := make([]string, len(node.Parameters))
	paramTypes := make(map[string]string)

	for idx, param := range node.Parameters {
		params[idx] = param.Name.Value
		if param.Type != nil {
			paramTypes[param.Name.Value] = param.Type.Value
		}
	}

	// Create the tool value
	tool := &ToolValue{
		Name:       node.Name.Value,
		Parameters: params,
		ParamTypes: paramTypes,
		Body:       node.Body,
		Env:        i.env, // Capture current environment for closure
	}

	if node.ReturnType != nil {
		tool.ReturnType = node.ReturnType.Value
	}

	// Register the tool in the environment
	err := i.env.Define(node.Name.Value, tool)
	if err != nil {
		return nil, err
	}

	return tool, nil
}

// evalTryStatement evaluates a try/catch/finally statement
func (i *Interpreter) evalTryStatement(node *parser.TryStatement) (Value, error) {
	var result Value
	var tryError error

	// Execute the try block
	result, tryError = i.evalBlockStatement(node.Block)

	// If there was an error and we have a catch clause, handle it
	if tryError != nil && node.CatchClause != nil {
		// Don't catch control flow signals (break, continue, return)
		if _, isControlFlow := tryError.(*ControlFlowError); !isControlFlow {
			// Create an error object with a message property
			errorObj := &ObjectValue{
				Properties: map[string]Value{
					"message": &StringValue{Value: tryError.Error()},
				},
			}

			// Bind the error parameter to the current scope temporarily
			var savedErrorValue Value
			var hadErrorParam bool
			if node.CatchClause.Parameter != nil {
				paramName := node.CatchClause.Parameter.Value
				// Save existing value if the parameter name is already defined
				savedErrorValue, hadErrorParam = i.env.Get(paramName)
				// Set the error parameter in current scope
				i.env.Set(paramName, errorObj)
			}

			// Execute the catch block (it will have its own scope via BlockStatement)
			catchResult, catchErr := i.evalBlockStatement(node.CatchClause.Block)

			// Restore the error parameter if it was shadowed
			if node.CatchClause.Parameter != nil {
				paramName := node.CatchClause.Parameter.Value
				if hadErrorParam {
					i.env.Set(paramName, savedErrorValue)
				} else {
					i.env.Delete(paramName)
				}
			}

			// If the catch block executed successfully, clear the error
			if catchErr == nil {
				tryError = nil
				result = catchResult
			} else {
				// If catch block had an error, that becomes the new error
				tryError = catchErr
			}
		}
	}

	// Execute the finally block if present
	if node.FinallyClause != nil {
		_, finallyErr := i.evalBlockStatement(node.FinallyClause.Block)
		// Finally errors override previous errors
		if finallyErr != nil {
			return nil, finallyErr
		}
	}

	// If there's still an error after catch (or no catch clause), propagate it
	if tryError != nil {
		return nil, tryError
	}

	return result, nil
}
