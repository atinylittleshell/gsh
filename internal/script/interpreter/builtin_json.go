package interpreter

import (
	"encoding/json"
	"fmt"
)

// builtinJSONParse implements JSON.parse()
func builtinJSONParse(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("JSON.parse expects 1 argument, got %d", len(args))
	}

	str, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("JSON.parse expects a string argument, got %s", args[0].Type())
	}

	var result interface{}
	if err := json.Unmarshal([]byte(str.Value), &result); err != nil {
		return nil, fmt.Errorf("JSON.parse error: %v", err)
	}

	return jsonToValue(result), nil
}

// builtinJSONStringify implements JSON.stringify()
func builtinJSONStringify(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("JSON.stringify expects 1 argument, got %d", len(args))
	}

	jsonValue := valueToJSON(args[0])
	bytes, err := json.Marshal(jsonValue)
	if err != nil {
		return nil, fmt.Errorf("JSON.stringify error: %v", err)
	}

	return &StringValue{Value: string(bytes)}, nil
}

// jsonToValue converts a Go interface{} from json.Unmarshal to a Value
func jsonToValue(v interface{}) Value {
	if v == nil {
		return &NullValue{}
	}

	switch val := v.(type) {
	case bool:
		return &BoolValue{Value: val}
	case float64:
		return &NumberValue{Value: val}
	case string:
		return &StringValue{Value: val}
	case []interface{}:
		elements := make([]Value, len(val))
		for i, elem := range val {
			elements[i] = jsonToValue(elem)
		}
		return &ArrayValue{Elements: elements}
	case map[string]interface{}:
		properties := make(map[string]*PropertyDescriptor)
		for key, value := range val {
			properties[key] = &PropertyDescriptor{Value: jsonToValue(value)}
		}
		return &ObjectValue{Properties: properties}
	default:
		return &NullValue{}
	}
}

// valueToJSON converts a Value to a Go interface{} for json.Marshal
func valueToJSON(v Value) interface{} {
	switch val := v.(type) {
	case *NullValue:
		return nil
	case *BoolValue:
		return val.Value
	case *NumberValue:
		return val.Value
	case *StringValue:
		return val.Value
	case *ArrayValue:
		result := make([]interface{}, len(val.Elements))
		for i, elem := range val.Elements {
			result[i] = valueToJSON(elem)
		}
		return result
	case *ObjectValue:
		result := make(map[string]interface{})
		for key := range val.Properties {
			result[key] = valueToJSON(val.GetPropertyValue(key))
		}
		return result
	default:
		return nil
	}
}
