package interpreter

import "fmt"

// builtinMap implements the Map() constructor
// Map() creates an empty map
// Map([[key1, val1], [key2, val2]]) creates a map from array of key-value pairs
func builtinMap(args []Value) (Value, error) {
	if len(args) == 0 {
		// Empty map
		return &MapValue{Entries: make(map[string]Value)}, nil
	}

	if len(args) != 1 {
		return nil, fmt.Errorf("Map() takes 0 or 1 arguments, got %d", len(args))
	}

	// Expect an array of [key, value] pairs
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("Map() argument must be an array of [key, value] pairs")
	}

	entries := make(map[string]Value)
	for i, elem := range arr.Elements {
		pair, ok := elem.(*ArrayValue)
		if !ok || len(pair.Elements) != 2 {
			return nil, fmt.Errorf("Map() entry %d must be a [key, value] pair", i)
		}

		// Key must be a string
		key, ok := pair.Elements[0].(*StringValue)
		if !ok {
			return nil, fmt.Errorf("Map() entry %d key must be a string", i)
		}

		entries[key.Value] = pair.Elements[1]
	}

	return &MapValue{Entries: entries}, nil
}

// builtinSet implements the Set() constructor
// Set() creates an empty set
// Set([val1, val2, val3]) creates a set from array of values
func builtinSet(args []Value) (Value, error) {
	if len(args) == 0 {
		// Empty set
		return &SetValue{Elements: make(map[string]Value)}, nil
	}

	if len(args) != 1 {
		return nil, fmt.Errorf("Set() takes 0 or 1 arguments, got %d", len(args))
	}

	// Expect an array of values
	arr, ok := args[0].(*ArrayValue)
	if !ok {
		return nil, fmt.Errorf("Set() argument must be an array of values")
	}

	elements := make(map[string]Value)
	for _, elem := range arr.Elements {
		// Use string representation as key for uniqueness
		key := elem.String()
		elements[key] = elem
	}

	return &SetValue{Elements: elements}, nil
}
