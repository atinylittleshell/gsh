package interpreter

import (
	"fmt"

	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// evalExpression evaluates an expression
func (i *Interpreter) evalExpression(expr parser.Expression) (Value, error) {
	switch node := expr.(type) {
	case *parser.NumberLiteral:
		return i.evalNumberLiteral(node)
	case *parser.StringLiteral:
		return i.evalStringLiteral(node)
	case *parser.BooleanLiteral:
		return i.evalBooleanLiteral(node)
	case *parser.NullLiteral:
		return i.evalNullLiteral(node)
	case *parser.Identifier:
		return i.evalIdentifier(node)
	case *parser.BinaryExpression:
		return i.evalBinaryExpression(node)
	case *parser.UnaryExpression:
		return i.evalUnaryExpression(node)
	case *parser.ArrayLiteral:
		return i.evalArrayLiteral(node)
	case *parser.ObjectLiteral:
		return i.evalObjectLiteral(node)
	case *parser.CallExpression:
		return i.evalCallExpression(node)
	case *parser.MemberExpression:
		return i.evalMemberExpression(node)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// evalNumberLiteral evaluates a number literal
func (i *Interpreter) evalNumberLiteral(node *parser.NumberLiteral) (Value, error) {
	// Parse the number string
	var value float64
	_, err := fmt.Sscanf(node.Value, "%f", &value)
	if err != nil {
		return nil, fmt.Errorf("invalid number literal: %s", node.Value)
	}
	return &NumberValue{Value: value}, nil
}

// evalStringLiteral evaluates a string literal
func (i *Interpreter) evalStringLiteral(node *parser.StringLiteral) (Value, error) {
	return &StringValue{Value: node.Value}, nil
}

// evalBooleanLiteral evaluates a boolean literal
func (i *Interpreter) evalBooleanLiteral(node *parser.BooleanLiteral) (Value, error) {
	return &BoolValue{Value: node.Value}, nil
}

// evalNullLiteral evaluates a null literal
func (i *Interpreter) evalNullLiteral(node *parser.NullLiteral) (Value, error) {
	return &NullValue{}, nil
}

// evalIdentifier evaluates an identifier (variable lookup)
func (i *Interpreter) evalIdentifier(node *parser.Identifier) (Value, error) {
	value, ok := i.env.Get(node.Value)
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s", node.Value)
	}
	return value, nil
}

// evalBinaryExpression evaluates a binary expression
func (i *Interpreter) evalBinaryExpression(node *parser.BinaryExpression) (Value, error) {
	left, err := i.evalExpression(node.Left)
	if err != nil {
		return nil, err
	}

	right, err := i.evalExpression(node.Right)
	if err != nil {
		return nil, err
	}

	return i.applyBinaryOperator(node.Operator, left, right)
}

// applyBinaryOperator applies a binary operator to two values
func (i *Interpreter) applyBinaryOperator(op string, left, right Value) (Value, error) {
	// Handle numeric operations
	if left.Type() == ValueTypeNumber && right.Type() == ValueTypeNumber {
		leftNum := left.(*NumberValue).Value
		rightNum := right.(*NumberValue).Value

		switch op {
		case "+":
			return &NumberValue{Value: leftNum + rightNum}, nil
		case "-":
			return &NumberValue{Value: leftNum - rightNum}, nil
		case "*":
			return &NumberValue{Value: leftNum * rightNum}, nil
		case "/":
			if rightNum == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return &NumberValue{Value: leftNum / rightNum}, nil
		case "%":
			if rightNum == 0 {
				return nil, fmt.Errorf("modulo by zero")
			}
			return &NumberValue{Value: float64(int64(leftNum) % int64(rightNum))}, nil
		case "<":
			return &BoolValue{Value: leftNum < rightNum}, nil
		case "<=":
			return &BoolValue{Value: leftNum <= rightNum}, nil
		case ">":
			return &BoolValue{Value: leftNum > rightNum}, nil
		case ">=":
			return &BoolValue{Value: leftNum >= rightNum}, nil
		case "==":
			return &BoolValue{Value: leftNum == rightNum}, nil
		case "!=":
			return &BoolValue{Value: leftNum != rightNum}, nil
		}
	}

	// Handle string concatenation
	if op == "+" && (left.Type() == ValueTypeString || right.Type() == ValueTypeString) {
		return &StringValue{Value: left.String() + right.String()}, nil
	}

	// Handle equality comparisons for all types
	if op == "==" {
		return &BoolValue{Value: left.Equals(right)}, nil
	}
	if op == "!=" {
		return &BoolValue{Value: !left.Equals(right)}, nil
	}

	// Handle logical operations
	if op == "&&" {
		return &BoolValue{Value: left.IsTruthy() && right.IsTruthy()}, nil
	}
	if op == "||" {
		return &BoolValue{Value: left.IsTruthy() || right.IsTruthy()}, nil
	}

	return nil, fmt.Errorf("unsupported operator '%s' for types %s and %s", op, left.Type(), right.Type())
}

// evalUnaryExpression evaluates a unary expression
func (i *Interpreter) evalUnaryExpression(node *parser.UnaryExpression) (Value, error) {
	right, err := i.evalExpression(node.Right)
	if err != nil {
		return nil, err
	}

	switch node.Operator {
	case "!":
		return &BoolValue{Value: !right.IsTruthy()}, nil
	case "-":
		if right.Type() != ValueTypeNumber {
			return nil, fmt.Errorf("unary minus operator requires number, got %s", right.Type())
		}
		return &NumberValue{Value: -right.(*NumberValue).Value}, nil
	case "+":
		if right.Type() != ValueTypeNumber {
			return nil, fmt.Errorf("unary plus operator requires number, got %s", right.Type())
		}
		return right, nil
	default:
		return nil, fmt.Errorf("unsupported unary operator: %s", node.Operator)
	}
}

// evalArrayLiteral evaluates an array literal
func (i *Interpreter) evalArrayLiteral(node *parser.ArrayLiteral) (Value, error) {
	elements := make([]Value, len(node.Elements))

	for idx, elem := range node.Elements {
		val, err := i.evalExpression(elem)
		if err != nil {
			return nil, err
		}
		elements[idx] = val
	}

	return &ArrayValue{Elements: elements}, nil
}

// evalObjectLiteral evaluates an object literal
func (i *Interpreter) evalObjectLiteral(node *parser.ObjectLiteral) (Value, error) {
	properties := make(map[string]Value)

	for key, expr := range node.Pairs {
		val, err := i.evalExpression(expr)
		if err != nil {
			return nil, err
		}
		properties[key] = val
	}

	return &ObjectValue{Properties: properties}, nil
}

// evalCallExpression evaluates a function/tool call
func (i *Interpreter) evalCallExpression(node *parser.CallExpression) (Value, error) {
	// Evaluate the function expression
	function, err := i.evalExpression(node.Function)
	if err != nil {
		return nil, err
	}

	// Check if it's a built-in function
	if builtin, ok := function.(*BuiltinValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		// Call the built-in function
		return builtin.Fn(args)
	}

	// Check if it's an MCP tool
	if mcpTool, ok := function.(*MCPToolValue); ok {
		return i.callMCPTool(mcpTool, node.Arguments)
	}

	// Check if it's a user-defined tool
	tool, ok := function.(*ToolValue)
	if !ok {
		return nil, fmt.Errorf("cannot call non-tool value of type %s", function.Type())
	}

	// Evaluate arguments
	args := make([]Value, len(node.Arguments))
	for idx, argExpr := range node.Arguments {
		val, err := i.evalExpression(argExpr)
		if err != nil {
			return nil, err
		}
		args[idx] = val
	}

	// Check parameter count
	if len(args) != len(tool.Parameters) {
		return nil, fmt.Errorf("tool %s expects %d arguments, got %d", tool.Name, len(tool.Parameters), len(args))
	}

	// Validate parameter types (runtime type checking)
	for idx, paramName := range tool.Parameters {
		if expectedType, hasType := tool.ParamTypes[paramName]; hasType {
			actualType := args[idx].Type().String()
			if !i.typesMatch(expectedType, actualType) {
				return nil, fmt.Errorf("tool %s parameter %s expects type %s, got %s",
					tool.Name, paramName, expectedType, actualType)
			}
		}
	}

	// Call the tool
	result, err := i.callTool(tool, args)
	if err != nil {
		return nil, err
	}

	// Validate return type if specified
	if tool.ReturnType != "" {
		actualType := result.Type().String()
		if !i.typesMatch(tool.ReturnType, actualType) {
			return nil, fmt.Errorf("tool %s expected to return %s, got %s",
				tool.Name, tool.ReturnType, actualType)
		}
	}

	return result, nil
}

// callTool executes a tool with the given arguments
func (i *Interpreter) callTool(tool *ToolValue, args []Value) (Value, error) {
	// Get the body as a block statement
	body, ok := tool.Body.(*parser.BlockStatement)
	if !ok {
		return nil, fmt.Errorf("invalid tool body")
	}

	// Create a new environment for the tool execution (closure)
	// Start with the tool's captured environment
	toolEnv := NewEnclosedEnvironment(tool.Env)

	// Bind parameters to arguments
	for idx, paramName := range tool.Parameters {
		toolEnv.Set(paramName, args[idx])
	}

	// Save current environment and switch to tool environment
	prevEnv := i.env
	i.env = toolEnv
	defer func() {
		i.env = prevEnv
	}()

	// Execute the tool body
	var result Value = &NullValue{}
	for _, stmt := range body.Statements {
		val, err := i.evalStatement(stmt)
		if err != nil {
			// Check for return signal
			if cfErr, ok := err.(*ControlFlowError); ok && cfErr.Signal == SignalReturn {
				return cfErr.Value, nil
			}
			return nil, err
		}
		result = val
	}

	// If no explicit return, return the last expression value
	return result, nil
}

// typesMatch checks if an actual type matches an expected type annotation
func (i *Interpreter) typesMatch(expected, actual string) bool {
	// Handle basic types
	if expected == actual {
		return true
	}

	// Handle "any" type which matches anything
	if expected == "any" {
		return true
	}

	// Handle array types (e.g., "string[]", "number[]")
	if len(expected) > 2 && expected[len(expected)-2:] == "[]" {
		if actual == "array" {
			// For now, we accept any array for typed array annotations
			// Full array element type checking would require more complex logic
			return true
		}
	}

	return false
}

// evalMemberExpression evaluates a member access expression (e.g., obj.property, env.HOME)
func (i *Interpreter) evalMemberExpression(node *parser.MemberExpression) (Value, error) {
	// Evaluate the object expression
	object, err := i.evalExpression(node.Object)
	if err != nil {
		return nil, err
	}

	propertyName := node.Property.Value

	// Handle special env object
	if envVal, ok := object.(*EnvValue); ok {
		return envVal.GetProperty(propertyName), nil
	}

	// Handle MCP proxy objects
	if mcpProxy, ok := object.(*MCPProxyValue); ok {
		return mcpProxy.GetProperty(propertyName)
	}

	// Handle regular objects
	if objVal, ok := object.(*ObjectValue); ok {
		if prop, exists := objVal.Properties[propertyName]; exists {
			return prop, nil
		}
		return nil, fmt.Errorf("property '%s' not found on object", propertyName)
	}

	return nil, fmt.Errorf("cannot access property '%s' on type %s", propertyName, object.Type())
}
