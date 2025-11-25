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
