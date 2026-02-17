package interpreter

import (
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// evalExpression evaluates an expression.
// The returned value is always unwrapped (DynamicValue is resolved to its underlying value).
// This ensures consumers don't need to manually unwrap DynamicValue.
func (i *Interpreter) evalExpression(expr parser.Expression) (Value, error) {
	var result Value
	var err error

	switch node := expr.(type) {
	case *parser.NumberLiteral:
		result, err = i.evalNumberLiteral(node)
	case *parser.StringLiteral:
		result, err = i.evalStringLiteral(node)
	case *parser.BooleanLiteral:
		result, err = i.evalBooleanLiteral(node)
	case *parser.NullLiteral:
		result, err = i.evalNullLiteral(node)
	case *parser.Identifier:
		result, err = i.evalIdentifier(node)
	case *parser.BinaryExpression:
		result, err = i.evalBinaryExpression(node)
	case *parser.UnaryExpression:
		result, err = i.evalUnaryExpression(node)
	case *parser.ArrayLiteral:
		result, err = i.evalArrayLiteral(node)
	case *parser.ObjectLiteral:
		result, err = i.evalObjectLiteral(node)
	case *parser.CallExpression:
		result, err = i.evalCallExpression(node)
	case *parser.MemberExpression:
		result, err = i.evalMemberExpression(node)
	case *parser.IndexExpression:
		result, err = i.evalIndexExpression(node)
	case *parser.PipeExpression:
		result, err = i.evalPipeExpression(node)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}

	if err != nil {
		return nil, err
	}

	// Always unwrap DynamicValue at the exit point.
	// This ensures all consumers of evalExpression get the underlying value
	// without needing to remember to call UnwrapValue manually.
	return UnwrapValue(result), nil
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
	// Only process interpolation for template literals (backtick strings)
	if !node.IsTemplate {
		return &StringValue{Value: node.Value}, nil
	}

	// Check if the template string contains interpolations
	str := node.Value

	// First, process interpolations if present
	var result string
	var err error
	if containsInterpolation(str) {
		result, err = i.interpolateString(str)
		if err != nil {
			return nil, err
		}
	} else {
		result = str
	}

	// Then, convert escaped dollar placeholders back to literal dollars
	result = strings.ReplaceAll(result, "\x00ESCAPED_DOLLAR\x00", "$")

	return &StringValue{Value: result}, nil
}

// containsInterpolation checks if a string contains ${...} expressions
func containsInterpolation(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '$' && s[i+1] == '{' {
			return true
		}
	}
	return false
}

// interpolateString processes template literal interpolations
func (i *Interpreter) interpolateString(template string) (string, error) {
	var result []rune
	runes := []rune(template)
	pos := 0

	for pos < len(runes) {
		// Look for ${
		if pos < len(runes)-1 && runes[pos] == '$' && runes[pos+1] == '{' {
			// Find the matching closing brace
			braceCount := 1
			start := pos + 2
			end := start

			for end < len(runes) && braceCount > 0 {
				switch runes[end] {
				case '{':
					braceCount++
				case '}':
					braceCount--
				}
				if braceCount > 0 {
					end++
				}
			}

			if braceCount != 0 {
				return "", fmt.Errorf("unclosed template literal interpolation")
			}

			// Extract the expression string
			exprStr := string(runes[start:end])

			// Parse and evaluate the expression
			value, err := i.evalInterpolationExpression(exprStr)
			if err != nil {
				return "", fmt.Errorf("error in template literal interpolation: %w", err)
			}

			// Convert value to string and append
			result = append(result, []rune(value.String())...)

			// Move past the closing brace
			pos = end + 1
		} else {
			// Regular character
			result = append(result, runes[pos])
			pos++
		}
	}

	return string(result), nil
}

// evalInterpolationExpression parses and evaluates an expression from a template literal
func (i *Interpreter) evalInterpolationExpression(exprStr string) (Value, error) {
	// Use the existing lexer and parser to parse the expression
	// We wrap it in a simple expression statement to use the parser
	l := lexer.New(exprStr)
	p := parser.New(l)

	// Parse as a program (which will parse the expression as an expression statement)
	program := p.ParseProgram()

	// Check for parse errors
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("error parsing template literal expression: %s", p.Errors()[0])
	}

	// The program should have exactly one expression statement
	if len(program.Statements) != 1 {
		return nil, fmt.Errorf("template literal expression must be a single expression")
	}

	// Extract the expression from the expression statement
	exprStmt, ok := program.Statements[0].(*parser.ExpressionStatement)
	if !ok {
		return nil, fmt.Errorf("template literal must contain an expression, not a statement")
	}

	// Evaluate the expression using existing evaluation logic
	return i.evalExpression(exprStmt.Expression)
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
		return nil, NewRuntimeError("undefined variable: %s (line %d, column %d)",
			node.Value, node.Token.Line, node.Token.Column)
	}
	return value, nil
}

// evalBinaryExpression evaluates a binary expression
func (i *Interpreter) evalBinaryExpression(node *parser.BinaryExpression) (Value, error) {
	// Evaluate left operand first
	left, err := i.evalExpression(node.Left)
	if err != nil {
		return nil, err
	}

	// Handle short-circuit evaluation for logical operators
	// For &&: if left is falsy, return false without evaluating right
	// For ||: if left is truthy, return true without evaluating right
	// Note: DynamicValue unwrapping is handled by evalExpression
	switch node.Operator {
	case "&&":
		if !left.IsTruthy() {
			return &BoolValue{Value: false}, nil
		}
		// Left is truthy, evaluate right and return its truthiness
		right, err := i.evalExpression(node.Right)
		if err != nil {
			return nil, err
		}
		return &BoolValue{Value: right.IsTruthy()}, nil

	case "||":
		if left.IsTruthy() {
			return &BoolValue{Value: true}, nil
		}
		// Left is falsy, evaluate right and return its truthiness
		right, err := i.evalExpression(node.Right)
		if err != nil {
			return nil, err
		}
		return &BoolValue{Value: right.IsTruthy()}, nil

	case "??":
		// Nullish coalescing: if left is null, return right; otherwise return left
		if left.Type() == ValueTypeNull {
			return i.evalExpression(node.Right)
		}
		return left, nil
	}

	// For all other operators, evaluate both operands
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
				return nil, NewRuntimeError("division by zero")
			}
			return &NumberValue{Value: leftNum / rightNum}, nil
		case "%":
			if rightNum == 0 {
				return nil, NewRuntimeError("modulo by zero")
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

	// Handle string comparison (lexicographic, like JavaScript)
	if left.Type() == ValueTypeString && right.Type() == ValueTypeString {
		leftStr := left.(*StringValue).Value
		rightStr := right.(*StringValue).Value

		switch op {
		case "<":
			return &BoolValue{Value: leftStr < rightStr}, nil
		case "<=":
			return &BoolValue{Value: leftStr <= rightStr}, nil
		case ">":
			return &BoolValue{Value: leftStr > rightStr}, nil
		case ">=":
			return &BoolValue{Value: leftStr >= rightStr}, nil
		}
	}

	// Handle equality comparisons for all types
	if op == "==" {
		return &BoolValue{Value: left.Equals(right)}, nil
	}
	if op == "!=" {
		return &BoolValue{Value: !left.Equals(right)}, nil
	}

	// Note: &&, ||, and ?? are handled in evalBinaryExpression for short-circuit evaluation

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
	properties := make(map[string]*PropertyDescriptor)

	for key, expr := range node.Pairs {
		val, err := i.evalExpression(expr)
		if err != nil {
			return nil, err
		}
		properties[key] = &PropertyDescriptor{Value: val}
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

	// Check if it's an array method
	if arrayMethod, ok := function.(*ArrayMethodValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		// Call the array method with the bound array instance
		return arrayMethod.Impl(arrayMethod.Arr, args)
	}

	// Check if it's a string method
	if stringMethod, ok := function.(*StringMethodValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		// Call the string method with the bound string instance
		return stringMethod.Impl(stringMethod.Str, args)
	}

	// Check if it's a number method
	if numberMethod, ok := function.(*NumberMethodValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		// Call the number method with the bound number instance
		return numberMethod.Impl(numberMethod.Num, args)
	}

	// Check if it's an object method
	if objectMethod, ok := function.(*ObjectMethodValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		// Call the object method with the bound object instance
		return objectMethod.Impl(objectMethod.Obj, args)
	}

	// Check if it's a map method
	if mapMethod, ok := function.(*MapMethodValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		// Call the map method with the bound map instance
		return mapMethod.Impl(mapMethod.Map, args)
	}

	// Check if it's a set method
	if setMethod, ok := function.(*SetMethodValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		// Call the set method with the bound set instance
		return setMethod.Impl(setMethod.Set, args)
	}

	// Check if it's an MCP tool
	if mcpTool, ok := function.(*MCPToolValue); ok {
		return i.callMCPTool(mcpTool, node.Arguments)
	}

	// Check if it's a native tool (gsh.tools.*)
	if nativeTool, ok := function.(*NativeToolValue); ok {
		return i.callNativeTool(nativeTool, node.Arguments)
	}

	// Check if it's an ACP session method
	if acpMethod, ok := function.(*ACPSessionMethodValue); ok {
		// Evaluate arguments
		args := make([]Value, len(node.Arguments))
		for idx, argExpr := range node.Arguments {
			val, err := i.evalExpression(argExpr)
			if err != nil {
				return nil, err
			}
			args[idx] = val
		}
		return i.callACPSessionMethod(acpMethod, args)
	}

	// Check if it's a user-defined tool
	tool, ok := function.(*ToolValue)
	if !ok {
		return nil, NewRuntimeError("cannot call non-tool value of type %s (line %d, column %d)",
			function.Type(), node.Token.Line, node.Token.Column)
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
		return nil, NewRuntimeError("tool %s expects %d arguments, got %d (line %d, column %d)",
			tool.Name, len(tool.Parameters), len(args), node.Token.Line, node.Token.Column)
	}

	// Validate parameter types (runtime type checking)
	for idx, paramName := range tool.Parameters {
		if expectedType, hasType := tool.ParamTypes[paramName]; hasType {
			actualType := args[idx].Type().String()
			if !i.typesMatch(expectedType, actualType) {
				return nil, NewRuntimeError("tool %s parameter %s expects type %s, got %s",
					tool.Name, paramName, expectedType, actualType)
			}
		}
	}

	// Call the tool
	result, err := i.CallTool(tool, args)
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
// CallTool calls a tool with the given arguments
func (i *Interpreter) CallTool(tool *ToolValue, args []Value) (Value, error) {
	// Get the body as a block statement
	body, ok := tool.Body.(*parser.BlockStatement)
	if !ok {
		return nil, fmt.Errorf("invalid tool body")
	}

	// Push stack frame for this tool call
	i.pushStackFrame(tool.Name, fmt.Sprintf("tool '%s'", tool.Name))

	// Ensure we handle errors properly before popping the stack frame
	var finalErr error
	defer func() {
		// Pop stack frame after error handling
		i.popStackFrame()
	}()

	// Create a new enclosed environment for the tool execution (closure)
	// Start with the tool's captured environment, allowing read/write access
	// to outer scope variables for consistency with if/for blocks
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
			// Wrap the error with current stack frame before returning
			finalErr = i.wrapError(err, stmt)
			return nil, finalErr
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

	// Handle array properties/methods
	if arrVal, ok := object.(*ArrayValue); ok {
		return i.getArrayProperty(arrVal, propertyName, node)
	}

	// Handle string properties/methods
	if strVal, ok := object.(*StringValue); ok {
		return i.getStringProperty(strVal, propertyName, node)
	}

	// Handle number properties/methods
	if numVal, ok := object.(*NumberValue); ok {
		return i.getNumberProperty(numVal, propertyName, node)
	}

	// Handle map properties/methods
	if mapVal, ok := object.(*MapValue); ok {
		return i.getMapProperty(mapVal, propertyName, node)
	}

	// Handle set properties/methods
	if setVal, ok := object.(*SetValue); ok {
		return i.getSetProperty(setVal, propertyName, node)
	}

	// Handle ACP session properties/methods
	if acpSession, ok := object.(*ACPSessionValue); ok {
		return i.getACPSessionProperty(acpSession, propertyName)
	}

	// Handle regular objects
	if objVal, ok := object.(*ObjectValue); ok {
		return i.getObjectProperty(objVal, propertyName, node)
	}

	// Handle objects with GetProperty method (like LoggingObjectValue)
	type PropertyGetter interface {
		GetProperty(name string) Value
	}
	if getter, ok := object.(PropertyGetter); ok {
		return getter.GetProperty(propertyName), nil
	}

	return nil, NewRuntimeError("cannot access property '%s' on type %s (line %d, column %d)",
		propertyName, object.Type(), node.Token.Line, node.Token.Column)
}

// getArrayProperty returns array properties and methods
func (i *Interpreter) getArrayProperty(arr *ArrayValue, property string, node *parser.MemberExpression) (Value, error) {
	switch property {
	case "length":
		return &NumberValue{Value: float64(len(arr.Elements))}, nil
	case "push":
		return &ArrayMethodValue{Name: "push", Impl: arrayPushImpl, Arr: arr}, nil
	case "pop":
		return &ArrayMethodValue{Name: "pop", Impl: arrayPopImpl, Arr: arr}, nil
	case "shift":
		return &ArrayMethodValue{Name: "shift", Impl: arrayShiftImpl, Arr: arr}, nil
	case "unshift":
		return &ArrayMethodValue{Name: "unshift", Impl: arrayUnshiftImpl, Arr: arr}, nil
	case "join":
		return &ArrayMethodValue{Name: "join", Impl: arrayJoinImpl, Arr: arr}, nil
	case "slice":
		return &ArrayMethodValue{Name: "slice", Impl: arraySliceImpl, Arr: arr}, nil
	case "reverse":
		return &ArrayMethodValue{Name: "reverse", Impl: arrayReverseImpl, Arr: arr}, nil
	default:
		return nil, NewRuntimeError("array property '%s' not found (line %d, column %d)",
			property, node.Token.Line, node.Token.Column)
	}
}

// getStringProperty returns string properties and methods
func (i *Interpreter) getStringProperty(str *StringValue, property string, node *parser.MemberExpression) (Value, error) {
	switch property {
	case "length":
		return &NumberValue{Value: float64(len([]rune(str.Value)))}, nil
	case "toUpperCase":
		return &StringMethodValue{Name: "toUpperCase", Impl: stringToUpperImpl, Str: str}, nil
	case "toLowerCase":
		return &StringMethodValue{Name: "toLowerCase", Impl: stringToLowerImpl, Str: str}, nil
	case "split":
		return &StringMethodValue{Name: "split", Impl: stringSplitImpl, Str: str}, nil
	case "trim":
		return &StringMethodValue{Name: "trim", Impl: stringTrimImpl, Str: str}, nil
	case "trimStart":
		return &StringMethodValue{Name: "trimStart", Impl: stringTrimStartImpl, Str: str}, nil
	case "trimEnd":
		return &StringMethodValue{Name: "trimEnd", Impl: stringTrimEndImpl, Str: str}, nil
	case "indexOf":
		return &StringMethodValue{Name: "indexOf", Impl: stringIndexOfImpl, Str: str}, nil
	case "lastIndexOf":
		return &StringMethodValue{Name: "lastIndexOf", Impl: stringLastIndexOfImpl, Str: str}, nil
	case "substring":
		return &StringMethodValue{Name: "substring", Impl: stringSubstringImpl, Str: str}, nil
	case "slice":
		return &StringMethodValue{Name: "slice", Impl: stringSliceImpl, Str: str}, nil
	case "startsWith":
		return &StringMethodValue{Name: "startsWith", Impl: stringStartsWithImpl, Str: str}, nil
	case "endsWith":
		return &StringMethodValue{Name: "endsWith", Impl: stringEndsWithImpl, Str: str}, nil
	case "includes":
		return &StringMethodValue{Name: "includes", Impl: stringIncludesImpl, Str: str}, nil
	case "replace":
		return &StringMethodValue{Name: "replace", Impl: stringReplaceImpl, Str: str}, nil
	case "replaceAll":
		return &StringMethodValue{Name: "replaceAll", Impl: stringReplaceAllImpl, Str: str}, nil
	case "repeat":
		return &StringMethodValue{Name: "repeat", Impl: stringRepeatImpl, Str: str}, nil
	case "padStart":
		return &StringMethodValue{Name: "padStart", Impl: stringPadStartImpl, Str: str}, nil
	case "padEnd":
		return &StringMethodValue{Name: "padEnd", Impl: stringPadEndImpl, Str: str}, nil
	case "charAt":
		return &StringMethodValue{Name: "charAt", Impl: stringCharAtImpl, Str: str}, nil
	default:
		return nil, NewRuntimeError("string property '%s' not found (line %d, column %d)",
			property, node.Token.Line, node.Token.Column)
	}
}

// getNumberProperty returns number properties and methods
func (i *Interpreter) getNumberProperty(num *NumberValue, property string, node *parser.MemberExpression) (Value, error) {
	switch property {
	case "toFixed":
		return &NumberMethodValue{Name: "toFixed", Impl: numberToFixedImpl, Num: num}, nil
	default:
		return nil, NewRuntimeError("number property '%s' not found (line %d, column %d)",
			property, node.Token.Line, node.Token.Column)
	}
}

// getObjectProperty returns object properties and methods
func (i *Interpreter) getObjectProperty(obj *ObjectValue, property string, node *parser.MemberExpression) (Value, error) {
	// Check for built-in methods first
	switch property {
	case "keys":
		return &ObjectMethodValue{Name: "keys", Impl: objectKeysImpl, Obj: obj}, nil
	case "values":
		return &ObjectMethodValue{Name: "values", Impl: objectValuesImpl, Obj: obj}, nil
	case "entries":
		return &ObjectMethodValue{Name: "entries", Impl: objectEntriesImpl, Obj: obj}, nil
	case "hasOwnProperty":
		return &ObjectMethodValue{Name: "hasOwnProperty", Impl: objectHasOwnPropertyImpl, Obj: obj}, nil
	}

	// Check for user-defined properties
	value := obj.GetPropertyValue(property)
	// Return the value (will be null if property doesn't exist)
	return value, nil
}

// getMapProperty returns map properties and methods
func (i *Interpreter) getMapProperty(m *MapValue, property string, node *parser.MemberExpression) (Value, error) {
	switch property {
	case "get":
		return &MapMethodValue{Name: "get", Impl: mapGetImpl, Map: m}, nil
	case "set":
		return &MapMethodValue{Name: "set", Impl: mapSetImpl, Map: m}, nil
	case "has":
		return &MapMethodValue{Name: "has", Impl: mapHasImpl, Map: m}, nil
	case "delete":
		return &MapMethodValue{Name: "delete", Impl: mapDeleteImpl, Map: m}, nil
	case "keys":
		return &MapMethodValue{Name: "keys", Impl: mapKeysImpl, Map: m}, nil
	case "values":
		return &MapMethodValue{Name: "values", Impl: mapValuesImpl, Map: m}, nil
	case "entries":
		return &MapMethodValue{Name: "entries", Impl: mapEntriesImpl, Map: m}, nil
	case "size":
		return mapSizeImpl(m, nil)
	default:
		return nil, NewRuntimeError("map property '%s' not found (line %d, column %d)",
			property, node.Token.Line, node.Token.Column)
	}
}

// getSetProperty returns set properties and methods
func (i *Interpreter) getSetProperty(s *SetValue, property string, node *parser.MemberExpression) (Value, error) {
	switch property {
	case "add":
		return &SetMethodValue{Name: "add", Impl: setAddImpl, Set: s}, nil
	case "has":
		return &SetMethodValue{Name: "has", Impl: setHasImpl, Set: s}, nil
	case "delete":
		return &SetMethodValue{Name: "delete", Impl: setDeleteImpl, Set: s}, nil
	case "size":
		return setSizeImpl(s, nil)
	default:
		return nil, NewRuntimeError("set property '%s' not found (line %d, column %d)",
			property, node.Token.Line, node.Token.Column)
	}
}

// evalIndexExpression evaluates an index expression (array[index] or object[key] or map[key])
func (i *Interpreter) evalIndexExpression(node *parser.IndexExpression) (Value, error) {
	left, err := i.evalExpression(node.Left)
	if err != nil {
		return nil, err
	}

	index, err := i.evalExpression(node.Index)
	if err != nil {
		return nil, err
	}

	// Handle array indexing
	if arrVal, ok := left.(*ArrayValue); ok {
		if index.Type() != ValueTypeNumber {
			return nil, NewRuntimeError("array index must be a number, got %s (line %d, column %d)",
				index.Type(), node.Token.Line, node.Token.Column)
		}
		idx := int(index.(*NumberValue).Value)
		if idx < 0 || idx >= len(arrVal.Elements) {
			return nil, NewRuntimeError("array index out of bounds: %d (length: %d) (line %d, column %d)",
				idx, len(arrVal.Elements), node.Token.Line, node.Token.Column)
		}
		return arrVal.Elements[idx], nil
	}

	// Handle object indexing with string keys
	if objVal, ok := left.(*ObjectValue); ok {
		if index.Type() != ValueTypeString {
			return nil, NewRuntimeError("object index must be a string, got %s (line %d, column %d)",
				index.Type(), node.Token.Line, node.Token.Column)
		}
		key := index.(*StringValue).Value
		return objVal.GetPropertyValue(key), nil
	}

	// Handle map indexing with string keys
	if mapVal, ok := left.(*MapValue); ok {
		if index.Type() != ValueTypeString {
			return nil, NewRuntimeError("map index must be a string, got %s (line %d, column %d)",
				index.Type(), node.Token.Line, node.Token.Column)
		}
		key := index.(*StringValue).Value
		if val, exists := mapVal.Entries[key]; exists {
			return val, nil
		}
		// Return null for missing keys (consistent with map.get() behavior)
		return &NullValue{}, nil
	}

	// Handle model indexing with string keys (access config properties)
	if modelVal, ok := left.(*ModelValue); ok {
		if index.Type() != ValueTypeString {
			return nil, NewRuntimeError("model index must be a string, got %s (line %d, column %d)",
				index.Type(), node.Token.Line, node.Token.Column)
		}
		key := index.(*StringValue).Value
		return modelVal.GetProperty(key), nil
	}

	// Handle custom indexable types (like REPLAgentsArrayValue)
	if indexable, ok := left.(Indexable); ok {
		if index.Type() != ValueTypeNumber {
			return nil, NewRuntimeError("index must be a number, got %s (line %d, column %d)",
				index.Type(), node.Token.Line, node.Token.Column)
		}
		idx := int(index.(*NumberValue).Value)
		return indexable.GetIndex(idx), nil
	}

	return nil, NewRuntimeError("cannot index type %s (line %d, column %d)",
		left.Type(), node.Token.Line, node.Token.Column)
}

// callNativeTool calls a native tool (gsh.tools.*) with the given arguments.
// Native tools accept a single object argument containing all parameters.
func (i *Interpreter) callNativeTool(tool *NativeToolValue, argExprs []parser.Expression) (Value, error) {
	// Build arguments map from expressions
	args := make(map[string]interface{})

	if len(argExprs) == 0 {
		// No arguments - call with empty args
		result, err := tool.Invoke(args)
		if err != nil {
			return nil, err
		}
		return i.nativeResultToValue(result)
	}

	// Evaluate first argument
	firstArg, err := i.evalExpression(argExprs[0])
	if err != nil {
		return nil, err
	}

	// If single object argument, use it as the arguments map
	if len(argExprs) == 1 {
		if objVal, ok := firstArg.(*ObjectValue); ok {
			// Convert object properties to map[string]interface{}
			for key := range objVal.Properties {
				args[key] = ValueToInterface(objVal.GetPropertyValue(key))
			}
			result, err := tool.Invoke(args)
			if err != nil {
				return nil, err
			}
			return i.nativeResultToValue(result)
		}
	}

	// For native tools, we require an object argument
	return nil, fmt.Errorf("native tool %s requires an object argument with named parameters", tool.Name)
}

// nativeResultToValue converts a native tool result to a Value.
// Native tools typically return strings (JSON formatted), but can return other types.
func (i *Interpreter) nativeResultToValue(result interface{}) (Value, error) {
	switch v := result.(type) {
	case nil:
		return &NullValue{}, nil
	case string:
		return &StringValue{Value: v}, nil
	case bool:
		return &BoolValue{Value: v}, nil
	case float64:
		return &NumberValue{Value: v}, nil
	case int:
		return &NumberValue{Value: float64(v)}, nil
	case int64:
		return &NumberValue{Value: float64(v)}, nil
	case []interface{}:
		elements := make([]Value, len(v))
		for idx, elem := range v {
			val, err := i.nativeResultToValue(elem)
			if err != nil {
				return nil, err
			}
			elements[idx] = val
		}
		return &ArrayValue{Elements: elements}, nil
	case map[string]interface{}:
		properties := make(map[string]*PropertyDescriptor)
		for key, val := range v {
			gshVal, err := i.nativeResultToValue(val)
			if err != nil {
				return nil, err
			}
			properties[key] = &PropertyDescriptor{Value: gshVal}
		}
		return &ObjectValue{Properties: properties}, nil
	default:
		return &StringValue{Value: fmt.Sprintf("%v", v)}, nil
	}
}
