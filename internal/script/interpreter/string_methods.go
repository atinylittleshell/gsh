package interpreter

import (
	"fmt"
	"strings"
)

// String method implementations

// stringMethodImpl is a function type for string method implementations
type stringMethodImpl func(str *StringValue, args []Value) (Value, error)

// StringMethodValue wraps a string method that needs to be bound to an instance at call time
type StringMethodValue struct {
	Name string
	Impl stringMethodImpl
	Str  *StringValue // The string instance this method is bound to
}

func (s *StringMethodValue) Type() ValueType         { return ValueTypeTool }
func (s *StringMethodValue) String() string          { return fmt.Sprintf("<string method: %s>", s.Name) }
func (s *StringMethodValue) IsTruthy() bool          { return true }
func (s *StringMethodValue) Equals(other Value) bool { return false }

// stringToUpperImpl implements the toUpperCase method
func stringToUpperImpl(str *StringValue, args []Value) (Value, error) {
	return &StringValue{Value: strings.ToUpper(str.Value)}, nil
}

// stringToLowerImpl implements the toLowerCase method
func stringToLowerImpl(str *StringValue, args []Value) (Value, error) {
	return &StringValue{Value: strings.ToLower(str.Value)}, nil
}

// stringSplitImpl implements the split method
func stringSplitImpl(str *StringValue, args []Value) (Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("split() requires a separator argument")
	}
	if args[0].Type() != ValueTypeString {
		return nil, fmt.Errorf("split() separator must be a string")
	}

	separator := args[0].(*StringValue).Value
	parts := strings.Split(str.Value, separator)

	// Convert to array of string values
	elements := make([]Value, len(parts))
	for i, part := range parts {
		elements[i] = &StringValue{Value: part}
	}

	return &ArrayValue{Elements: elements}, nil
}

// stringTrimImpl implements the trim method
func stringTrimImpl(str *StringValue, args []Value) (Value, error) {
	return &StringValue{Value: strings.TrimSpace(str.Value)}, nil
}
