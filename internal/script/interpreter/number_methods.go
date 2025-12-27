package interpreter

import (
	"fmt"
	"strconv"
)

// Number method implementations

// numberMethodImpl is a function type for number method implementations
type numberMethodImpl func(num *NumberValue, args []Value) (Value, error)

// NumberMethodValue wraps a number method that needs to be bound to an instance at call time
type NumberMethodValue struct {
	Name string
	Impl numberMethodImpl
	Num  *NumberValue // The number instance this method is bound to
}

func (n *NumberMethodValue) Type() ValueType         { return ValueTypeTool }
func (n *NumberMethodValue) String() string          { return fmt.Sprintf("<number method: %s>", n.Name) }
func (n *NumberMethodValue) IsTruthy() bool          { return true }
func (n *NumberMethodValue) Equals(other Value) bool { return false }

// numberToFixedImpl implements the toFixed method
// Returns a string representation of the number with the specified number of decimal places
func numberToFixedImpl(num *NumberValue, args []Value) (Value, error) {
	// Default to 0 decimal places if no argument provided
	decimals := 0
	if len(args) > 0 {
		if args[0].Type() != ValueTypeNumber {
			return nil, fmt.Errorf("toFixed() argument must be a number")
		}
		decimals = int(args[0].(*NumberValue).Value)
		if decimals < 0 {
			return nil, fmt.Errorf("toFixed() argument must be non-negative")
		}
		if decimals > 100 {
			return nil, fmt.Errorf("toFixed() argument must be at most 100")
		}
	}

	formatted := strconv.FormatFloat(num.Value, 'f', decimals, 64)
	return &StringValue{Value: formatted}, nil
}
