package interpreter

import (
	"encoding/json"
	"fmt"
)

// InterfaceToValue converts a Go interface{} to a Value.
// This is the canonical conversion function used throughout the interpreter.
func InterfaceToValue(val interface{}) Value {
	switch v := val.(type) {
	case nil:
		return &NullValue{}
	case bool:
		return &BoolValue{Value: v}
	case float64:
		return &NumberValue{Value: v}
	case int:
		return &NumberValue{Value: float64(v)}
	case int64:
		return &NumberValue{Value: float64(v)}
	case string:
		return &StringValue{Value: v}
	case []interface{}:
		elements := make([]Value, len(v))
		for idx, elem := range v {
			elements[idx] = InterfaceToValue(elem)
		}
		return &ArrayValue{Elements: elements}
	case map[string]interface{}:
		properties := make(map[string]*PropertyDescriptor)
		for key, val := range v {
			properties[key] = &PropertyDescriptor{Value: InterfaceToValue(val)}
		}
		return &ObjectValue{Properties: properties}
	default:
		return &StringValue{Value: fmt.Sprintf("%v", v)}
	}
}

// ValueToInterface converts a Value to interface{}.
// This is the canonical conversion function used throughout the interpreter.
func ValueToInterface(val Value) interface{} {
	switch v := val.(type) {
	case *NullValue:
		return nil
	case *BoolValue:
		return v.Value
	case *NumberValue:
		return v.Value
	case *StringValue:
		return v.Value
	case *ArrayValue:
		arr := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			arr[i] = ValueToInterface(elem)
		}
		return arr
	case *ObjectValue:
		result := make(map[string]interface{})
		for key := range v.Properties {
			result[key] = ValueToInterface(v.GetPropertyValue(key))
		}
		return result
	default:
		return val.String()
	}
}

// interfaceToValue converts a Go interface{} to a Value.
// This is a method wrapper around InterfaceToValue for backward compatibility.
func (i *Interpreter) interfaceToValue(val interface{}) Value {
	return InterfaceToValue(val)
}

// valueToInterface converts a Value to interface{}.
// This is a method wrapper around ValueToInterface for backward compatibility.
func (i *Interpreter) valueToInterface(val Value) interface{} {
	return ValueToInterface(val)
}

// jsonToValue converts a JSON value to a GSH Value
func (i *Interpreter) jsonToValue(jsonVal interface{}) (Value, error) {
	switch v := jsonVal.(type) {
	case nil:
		return &NullValue{}, nil
	case bool:
		return &BoolValue{Value: v}, nil
	case float64:
		return &NumberValue{Value: v}, nil
	case string:
		return &StringValue{Value: v}, nil
	case []interface{}:
		elements := make([]Value, len(v))
		for idx, elem := range v {
			val, err := i.jsonToValue(elem)
			if err != nil {
				return nil, err
			}
			elements[idx] = val
		}
		return &ArrayValue{Elements: elements}, nil
	case map[string]interface{}:
		properties := make(map[string]*PropertyDescriptor)
		for key, val := range v {
			gshVal, err := i.jsonToValue(val)
			if err != nil {
				return nil, err
			}
			properties[key] = &PropertyDescriptor{Value: gshVal}
		}
		return &ObjectValue{Properties: properties}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON type: %T", jsonVal)
	}
}

// valueToJSON converts a GSH Value to a JSON string.
// It uses json.Marshal for proper escaping of special characters.
func (i *Interpreter) valueToJSON(val Value) (string, error) {
	// Convert Value to interface{} and use json.Marshal for proper escaping
	jsonBytes, err := json.Marshal(i.valueToInterface(val))
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
