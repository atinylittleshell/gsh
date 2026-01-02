package interpreter

import (
	"fmt"
	"strings"
)

// Array method implementations

// arrayMethodImpl is a function type for array method implementations
type arrayMethodImpl func(arr *ArrayValue, args []Value) (Value, error)

// ArrayMethodValue wraps an array method that needs to be bound to an instance at call time
type ArrayMethodValue struct {
	Name string
	Impl arrayMethodImpl
	Arr  *ArrayValue // The array instance this method is bound to
}

func (a *ArrayMethodValue) Type() ValueType         { return ValueTypeTool }
func (a *ArrayMethodValue) String() string          { return fmt.Sprintf("<array method: %s>", a.Name) }
func (a *ArrayMethodValue) IsTruthy() bool          { return true }
func (a *ArrayMethodValue) Equals(other Value) bool { return false }

// arrayPushImpl implements the push method
func arrayPushImpl(arr *ArrayValue, args []Value) (Value, error) {
	// Push all arguments to the array
	arr.Elements = append(arr.Elements, args...)
	// Return the new length
	return &NumberValue{Value: float64(len(arr.Elements))}, nil
}

// arrayPopImpl implements the pop method
func arrayPopImpl(arr *ArrayValue, args []Value) (Value, error) {
	if len(arr.Elements) == 0 {
		return &NullValue{}, nil
	}
	// Get the last element
	lastElement := arr.Elements[len(arr.Elements)-1]
	// Remove it from the array
	arr.Elements = arr.Elements[:len(arr.Elements)-1]
	return lastElement, nil
}

// arrayShiftImpl implements the shift method
func arrayShiftImpl(arr *ArrayValue, args []Value) (Value, error) {
	if len(arr.Elements) == 0 {
		return &NullValue{}, nil
	}
	// Get the first element
	firstElement := arr.Elements[0]
	// Remove it from the array
	arr.Elements = arr.Elements[1:]
	return firstElement, nil
}

// arrayUnshiftImpl implements the unshift method
func arrayUnshiftImpl(arr *ArrayValue, args []Value) (Value, error) {
	// Prepend all arguments to the array
	arr.Elements = append(args, arr.Elements...)
	// Return the new length
	return &NumberValue{Value: float64(len(arr.Elements))}, nil
}

// arrayJoinImpl implements the join method
func arrayJoinImpl(arr *ArrayValue, args []Value) (Value, error) {
	separator := ", "
	if len(args) > 0 {
		if args[0].Type() != ValueTypeString {
			return nil, fmt.Errorf("join() expects a string separator")
		}
		separator = args[0].(*StringValue).Value
	}

	// Convert all elements to strings and join
	var parts []string
	for _, elem := range arr.Elements {
		parts = append(parts, elem.String())
	}

	return &StringValue{Value: strings.Join(parts, separator)}, nil
}

// arraySliceImpl implements the slice method
func arraySliceImpl(arr *ArrayValue, args []Value) (Value, error) {
	start := 0
	end := len(arr.Elements)

	if len(args) > 0 {
		if args[0].Type() != ValueTypeNumber {
			return nil, fmt.Errorf("slice() start index must be a number")
		}
		start = int(args[0].(*NumberValue).Value)
		if start < 0 {
			start = len(arr.Elements) + start
			if start < 0 {
				start = 0
			}
		}
	}

	if len(args) > 1 {
		if args[1].Type() != ValueTypeNumber {
			return nil, fmt.Errorf("slice() end index must be a number")
		}
		end = int(args[1].(*NumberValue).Value)
		if end < 0 {
			end = len(arr.Elements) + end
		}
	}

	// Bounds checking
	if start > len(arr.Elements) {
		start = len(arr.Elements)
	}
	if end > len(arr.Elements) {
		end = len(arr.Elements)
	}
	if start > end {
		start = end
	}

	// Create a new array with the sliced elements
	newElements := make([]Value, end-start)
	copy(newElements, arr.Elements[start:end])

	return &ArrayValue{Elements: newElements}, nil
}

// arrayReverseImpl implements the reverse method
func arrayReverseImpl(arr *ArrayValue, args []Value) (Value, error) {
	// Reverse the array in place
	for i, j := 0, len(arr.Elements)-1; i < j; i, j = i+1, j-1 {
		arr.Elements[i], arr.Elements[j] = arr.Elements[j], arr.Elements[i]
	}
	return arr, nil
}
