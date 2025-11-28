package interpreter

import (
	"fmt"
)

// Object method implementations

// objectMethodImpl is a function type for object method implementations
type objectMethodImpl func(obj *ObjectValue, args []Value) (Value, error)

// ObjectMethodValue wraps an object method that needs to be bound to an instance at call time
type ObjectMethodValue struct {
	Name string
	Impl objectMethodImpl
	Obj  *ObjectValue // The object instance this method is bound to
}

func (o *ObjectMethodValue) Type() ValueType         { return ValueTypeTool }
func (o *ObjectMethodValue) String() string          { return fmt.Sprintf("<object method: %s>", o.Name) }
func (o *ObjectMethodValue) IsTruthy() bool          { return true }
func (o *ObjectMethodValue) Equals(other Value) bool { return false }

// objectKeysImpl implements the keys method
func objectKeysImpl(obj *ObjectValue, args []Value) (Value, error) {
	keys := make([]Value, 0, len(obj.Properties))
	for key := range obj.Properties {
		keys = append(keys, &StringValue{Value: key})
	}
	return &ArrayValue{Elements: keys}, nil
}

// objectValuesImpl implements the values method
func objectValuesImpl(obj *ObjectValue, args []Value) (Value, error) {
	values := make([]Value, 0, len(obj.Properties))
	for _, value := range obj.Properties {
		values = append(values, value)
	}
	return &ArrayValue{Elements: values}, nil
}

// objectEntriesImpl implements the entries method
func objectEntriesImpl(obj *ObjectValue, args []Value) (Value, error) {
	entries := make([]Value, 0, len(obj.Properties))
	for key, value := range obj.Properties {
		entry := &ArrayValue{
			Elements: []Value{
				&StringValue{Value: key},
				value,
			},
		}
		entries = append(entries, entry)
	}
	return &ArrayValue{Elements: entries}, nil
}

// objectHasOwnPropertyImpl implements the hasOwnProperty method
func objectHasOwnPropertyImpl(obj *ObjectValue, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("hasOwnProperty() expects exactly 1 argument, got %d", len(args))
	}
	if args[0].Type() != ValueTypeString {
		return nil, fmt.Errorf("hasOwnProperty() expects a string argument, got %s", args[0].Type())
	}

	key := args[0].(*StringValue).Value
	_, exists := obj.Properties[key]
	return &BoolValue{Value: exists}, nil
}
