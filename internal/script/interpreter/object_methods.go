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
	for key := range obj.Properties {
		values = append(values, obj.GetPropertyValue(key))
	}
	return &ArrayValue{Elements: values}, nil
}

// objectEntriesImpl implements the entries method
func objectEntriesImpl(obj *ObjectValue, args []Value) (Value, error) {
	entries := make([]Value, 0, len(obj.Properties))
	for key := range obj.Properties {
		entry := &ArrayValue{
			Elements: []Value{
				&StringValue{Value: key},
				obj.GetPropertyValue(key),
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

// Map method implementations

// mapMethodImpl is a function type for map method implementations
type mapMethodImpl func(m *MapValue, args []Value) (Value, error)

// MapMethodValue wraps a map method that needs to be bound to an instance at call time
type MapMethodValue struct {
	Name string
	Impl mapMethodImpl
	Map  *MapValue // The map instance this method is bound to
}

func (m *MapMethodValue) Type() ValueType         { return ValueTypeTool }
func (m *MapMethodValue) String() string          { return fmt.Sprintf("<map method: %s>", m.Name) }
func (m *MapMethodValue) IsTruthy() bool          { return true }
func (m *MapMethodValue) Equals(other Value) bool { return false }

// mapGetImpl implements the get method
func mapGetImpl(m *MapValue, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Map.get() expects exactly 1 argument, got %d", len(args))
	}
	if args[0].Type() != ValueTypeString {
		return nil, fmt.Errorf("Map.get() expects a string key, got %s", args[0].Type())
	}

	key := args[0].(*StringValue).Value
	value, exists := m.Entries[key]
	if !exists {
		return &NullValue{}, nil
	}
	return value, nil
}

// mapSetImpl implements the set method
func mapSetImpl(m *MapValue, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Map.set() expects exactly 2 arguments, got %d", len(args))
	}
	if args[0].Type() != ValueTypeString {
		return nil, fmt.Errorf("Map.set() expects a string key, got %s", args[0].Type())
	}

	key := args[0].(*StringValue).Value
	m.Entries[key] = args[1]
	return m, nil // Return the map for chaining
}

// mapHasImpl implements the has method
func mapHasImpl(m *MapValue, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Map.has() expects exactly 1 argument, got %d", len(args))
	}
	if args[0].Type() != ValueTypeString {
		return nil, fmt.Errorf("Map.has() expects a string key, got %s", args[0].Type())
	}

	key := args[0].(*StringValue).Value
	_, exists := m.Entries[key]
	return &BoolValue{Value: exists}, nil
}

// mapDeleteImpl implements the delete method
func mapDeleteImpl(m *MapValue, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Map.delete() expects exactly 1 argument, got %d", len(args))
	}
	if args[0].Type() != ValueTypeString {
		return nil, fmt.Errorf("Map.delete() expects a string key, got %s", args[0].Type())
	}

	key := args[0].(*StringValue).Value
	_, exists := m.Entries[key]
	if exists {
		delete(m.Entries, key)
	}
	return &BoolValue{Value: exists}, nil
}

// mapKeysImpl implements the keys method
func mapKeysImpl(m *MapValue, args []Value) (Value, error) {
	keys := make([]Value, 0, len(m.Entries))
	for key := range m.Entries {
		keys = append(keys, &StringValue{Value: key})
	}
	return &ArrayValue{Elements: keys}, nil
}

// mapValuesImpl implements the values method
func mapValuesImpl(m *MapValue, args []Value) (Value, error) {
	values := make([]Value, 0, len(m.Entries))
	for _, value := range m.Entries {
		values = append(values, value)
	}
	return &ArrayValue{Elements: values}, nil
}

// mapEntriesImpl implements the entries method
func mapEntriesImpl(m *MapValue, args []Value) (Value, error) {
	entries := make([]Value, 0, len(m.Entries))
	for key, value := range m.Entries {
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

// mapSizeImpl implements the size property getter
func mapSizeImpl(m *MapValue, args []Value) (Value, error) {
	return &NumberValue{Value: float64(len(m.Entries))}, nil
}

// Set method implementations

// setMethodImpl is a function type for set method implementations
type setMethodImpl func(s *SetValue, args []Value) (Value, error)

// SetMethodValue wraps a set method that needs to be bound to an instance at call time
type SetMethodValue struct {
	Name string
	Impl setMethodImpl
	Set  *SetValue // The set instance this method is bound to
}

func (s *SetMethodValue) Type() ValueType         { return ValueTypeTool }
func (s *SetMethodValue) String() string          { return fmt.Sprintf("<set method: %s>", s.Name) }
func (s *SetMethodValue) IsTruthy() bool          { return true }
func (s *SetMethodValue) Equals(other Value) bool { return false }

// setAddImpl implements the add method
func setAddImpl(s *SetValue, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Set.add() expects exactly 1 argument, got %d", len(args))
	}

	key := args[0].String()
	s.Elements[key] = args[0]
	return s, nil // Return the set for chaining
}

// setHasImpl implements the has method
func setHasImpl(s *SetValue, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Set.has() expects exactly 1 argument, got %d", len(args))
	}

	key := args[0].String()
	_, exists := s.Elements[key]
	return &BoolValue{Value: exists}, nil
}

// setDeleteImpl implements the delete method
func setDeleteImpl(s *SetValue, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Set.delete() expects exactly 1 argument, got %d", len(args))
	}

	key := args[0].String()
	_, exists := s.Elements[key]
	if exists {
		delete(s.Elements, key)
	}
	return &BoolValue{Value: exists}, nil
}

// setSizeImpl implements the size property getter
func setSizeImpl(s *SetValue, args []Value) (Value, error) {
	return &NumberValue{Value: float64(len(s.Elements))}, nil
}
