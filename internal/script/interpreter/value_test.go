package interpreter

import (
	"testing"
)

func TestValueType_String(t *testing.T) {
	tests := []struct {
		vt       ValueType
		expected string
	}{
		{ValueTypeNull, "null"},
		{ValueTypeNumber, "number"},
		{ValueTypeString, "string"},
		{ValueTypeBool, "boolean"},
		{ValueTypeArray, "array"},
		{ValueTypeObject, "object"},
		{ValueTypeTool, "tool"},
		{ValueTypeError, "error"},
	}

	for _, tt := range tests {
		if got := tt.vt.String(); got != tt.expected {
			t.Errorf("ValueType.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestNullValue(t *testing.T) {
	null := &NullValue{}

	if null.Type() != ValueTypeNull {
		t.Errorf("NullValue.Type() = %v, want %v", null.Type(), ValueTypeNull)
	}

	if null.String() != "null" {
		t.Errorf("NullValue.String() = %v, want null", null.String())
	}

	if null.IsTruthy() {
		t.Error("NullValue.IsTruthy() = true, want false")
	}

	// Test equality with another null
	other := &NullValue{}
	if !null.Equals(other) {
		t.Error("NullValue should equal another NullValue")
	}

	// Test inequality with non-null
	if null.Equals(&NumberValue{Value: 0}) {
		t.Error("NullValue should not equal NumberValue")
	}
}

func TestNumberValue(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		str    string
		truthy bool
	}{
		{"zero", 0, "0", false},
		{"positive integer", 42, "42", true},
		{"negative integer", -10, "-10", true},
		{"positive float", 3.14, "3.14", true},
		{"negative float", -2.5, "-2.5", true},
		{"large number", 1000000, "1000000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num := &NumberValue{Value: tt.value}

			if num.Type() != ValueTypeNumber {
				t.Errorf("NumberValue.Type() = %v, want %v", num.Type(), ValueTypeNumber)
			}

			if num.String() != tt.str {
				t.Errorf("NumberValue.String() = %v, want %v", num.String(), tt.str)
			}

			if num.IsTruthy() != tt.truthy {
				t.Errorf("NumberValue.IsTruthy() = %v, want %v", num.IsTruthy(), tt.truthy)
			}

			// Test equality
			same := &NumberValue{Value: tt.value}
			if !num.Equals(same) {
				t.Error("NumberValue should equal another NumberValue with same value")
			}

			different := &NumberValue{Value: tt.value + 1}
			if num.Equals(different) {
				t.Error("NumberValue should not equal NumberValue with different value")
			}

			if num.Equals(&StringValue{Value: tt.str}) {
				t.Error("NumberValue should not equal StringValue")
			}
		})
	}
}

func TestStringValue(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		truthy bool
	}{
		{"empty string", "", false},
		{"non-empty string", "hello", true},
		{"space", " ", true},
		{"long string", "this is a longer string", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := &StringValue{Value: tt.value}

			if str.Type() != ValueTypeString {
				t.Errorf("StringValue.Type() = %v, want %v", str.Type(), ValueTypeString)
			}

			if str.String() != tt.value {
				t.Errorf("StringValue.String() = %v, want %v", str.String(), tt.value)
			}

			if str.IsTruthy() != tt.truthy {
				t.Errorf("StringValue.IsTruthy() = %v, want %v", str.IsTruthy(), tt.truthy)
			}

			// Test equality
			same := &StringValue{Value: tt.value}
			if !str.Equals(same) {
				t.Error("StringValue should equal another StringValue with same value")
			}

			different := &StringValue{Value: tt.value + "x"}
			if str.Equals(different) {
				t.Error("StringValue should not equal StringValue with different value")
			}

			if str.Equals(&NumberValue{Value: 42}) {
				t.Error("StringValue should not equal NumberValue")
			}
		})
	}
}

func TestBoolValue(t *testing.T) {
	tests := []struct {
		name   string
		value  bool
		str    string
		truthy bool
	}{
		{"true", true, "true", true},
		{"false", false, "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BoolValue{Value: tt.value}

			if b.Type() != ValueTypeBool {
				t.Errorf("BoolValue.Type() = %v, want %v", b.Type(), ValueTypeBool)
			}

			if b.String() != tt.str {
				t.Errorf("BoolValue.String() = %v, want %v", b.String(), tt.str)
			}

			if b.IsTruthy() != tt.truthy {
				t.Errorf("BoolValue.IsTruthy() = %v, want %v", b.IsTruthy(), tt.truthy)
			}

			// Test equality
			same := &BoolValue{Value: tt.value}
			if !b.Equals(same) {
				t.Error("BoolValue should equal another BoolValue with same value")
			}

			different := &BoolValue{Value: !tt.value}
			if b.Equals(different) {
				t.Error("BoolValue should not equal BoolValue with different value")
			}

			if b.Equals(&NullValue{}) {
				t.Error("BoolValue should not equal NullValue")
			}
		})
	}
}

func TestArrayValue(t *testing.T) {
	tests := []struct {
		name     string
		elements []Value
		str      string
		truthy   bool
	}{
		{
			"empty array",
			[]Value{},
			"[]",
			false,
		},
		{
			"array with numbers",
			[]Value{
				&NumberValue{Value: 1},
				&NumberValue{Value: 2},
				&NumberValue{Value: 3},
			},
			"[1, 2, 3]",
			true,
		},
		{
			"array with strings",
			[]Value{
				&StringValue{Value: "hello"},
				&StringValue{Value: "world"},
			},
			`["hello", "world"]`,
			true,
		},
		{
			"mixed array",
			[]Value{
				&NumberValue{Value: 42},
				&StringValue{Value: "test"},
				&BoolValue{Value: true},
			},
			`[42, "test", true]`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arr := &ArrayValue{Elements: tt.elements}

			if arr.Type() != ValueTypeArray {
				t.Errorf("ArrayValue.Type() = %v, want %v", arr.Type(), ValueTypeArray)
			}

			if arr.String() != tt.str {
				t.Errorf("ArrayValue.String() = %v, want %v", arr.String(), tt.str)
			}

			if arr.IsTruthy() != tt.truthy {
				t.Errorf("ArrayValue.IsTruthy() = %v, want %v", arr.IsTruthy(), tt.truthy)
			}

			// Test equality
			same := &ArrayValue{Elements: tt.elements}
			if !arr.Equals(same) {
				t.Error("ArrayValue should equal another ArrayValue with same elements")
			}

			different := &ArrayValue{Elements: []Value{&NumberValue{Value: 99}}}
			if len(tt.elements) > 0 && arr.Equals(different) {
				t.Error("ArrayValue should not equal ArrayValue with different elements")
			}

			if arr.Equals(&NullValue{}) {
				t.Error("ArrayValue should not equal NullValue")
			}
		})
	}
}

func TestArrayValue_Equality(t *testing.T) {
	// Test nested arrays
	arr1 := &ArrayValue{
		Elements: []Value{
			&ArrayValue{
				Elements: []Value{
					&NumberValue{Value: 1},
					&NumberValue{Value: 2},
				},
			},
		},
	}

	arr2 := &ArrayValue{
		Elements: []Value{
			&ArrayValue{
				Elements: []Value{
					&NumberValue{Value: 1},
					&NumberValue{Value: 2},
				},
			},
		},
	}

	if !arr1.Equals(arr2) {
		t.Error("Nested arrays with same values should be equal")
	}

	arr3 := &ArrayValue{
		Elements: []Value{
			&ArrayValue{
				Elements: []Value{
					&NumberValue{Value: 1},
					&NumberValue{Value: 3},
				},
			},
		},
	}

	if arr1.Equals(arr3) {
		t.Error("Nested arrays with different values should not be equal")
	}
}

func TestObjectValue(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]Value
		truthy     bool
	}{
		{
			"empty object",
			map[string]Value{},
			false,
		},
		{
			"object with number",
			map[string]Value{
				"count": &NumberValue{Value: 42},
			},
			true,
		},
		{
			"object with multiple properties",
			map[string]Value{
				"name": &StringValue{Value: "Alice"},
				"age":  &NumberValue{Value: 30},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &ObjectValue{Properties: tt.properties}

			if obj.Type() != ValueTypeObject {
				t.Errorf("ObjectValue.Type() = %v, want %v", obj.Type(), ValueTypeObject)
			}

			if obj.IsTruthy() != tt.truthy {
				t.Errorf("ObjectValue.IsTruthy() = %v, want %v", obj.IsTruthy(), tt.truthy)
			}

			// Test equality
			same := &ObjectValue{Properties: tt.properties}
			if !obj.Equals(same) {
				t.Error("ObjectValue should equal another ObjectValue with same properties")
			}

			different := &ObjectValue{Properties: map[string]Value{"x": &NumberValue{Value: 99}}}
			if len(tt.properties) > 0 && obj.Equals(different) {
				t.Error("ObjectValue should not equal ObjectValue with different properties")
			}

			if obj.Equals(&NullValue{}) {
				t.Error("ObjectValue should not equal NullValue")
			}
		})
	}
}

func TestObjectValue_String(t *testing.T) {
	obj := &ObjectValue{
		Properties: map[string]Value{
			"name": &StringValue{Value: "Alice"},
			"age":  &NumberValue{Value: 30},
		},
	}

	str := obj.String()

	// The order of properties in map iteration is not guaranteed,
	// so we check that the string contains expected parts
	if str[0] != '{' || str[len(str)-1] != '}' {
		t.Errorf("ObjectValue.String() should be surrounded by braces, got %v", str)
	}

	// Check that both properties are present
	if !contains(str, `name: "Alice"`) {
		t.Errorf("ObjectValue.String() should contain name property, got %v", str)
	}

	if !contains(str, "age: 30") {
		t.Errorf("ObjectValue.String() should contain age property, got %v", str)
	}
}

func TestErrorValue(t *testing.T) {
	tests := []struct {
		name    string
		message string
		str     string
	}{
		{"simple error", "something went wrong", "Error: something went wrong"},
		{"empty error", "", "Error: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ErrorValue{Message: tt.message}

			if err.Type() != ValueTypeError {
				t.Errorf("ErrorValue.Type() = %v, want %v", err.Type(), ValueTypeError)
			}

			if err.String() != tt.str {
				t.Errorf("ErrorValue.String() = %v, want %v", err.String(), tt.str)
			}

			if err.IsTruthy() {
				t.Error("ErrorValue.IsTruthy() should always be false")
			}

			// Test equality
			same := &ErrorValue{Message: tt.message}
			if !err.Equals(same) {
				t.Error("ErrorValue should equal another ErrorValue with same message")
			}

			different := &ErrorValue{Message: "different"}
			if tt.message != "" && err.Equals(different) {
				t.Error("ErrorValue should not equal ErrorValue with different message")
			}

			if err.Equals(&NullValue{}) {
				t.Error("ErrorValue should not equal NullValue")
			}
		})
	}
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{"simple message", "error occurred", []interface{}{}, "Error: error occurred"},
		{"formatted message", "value is %d", []interface{}{42}, "Error: value is 42"},
		{"multiple args", "%s: %v", []interface{}{"test", 123}, "Error: test: 123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.format, tt.args...)

			if err.Type() != ValueTypeError {
				t.Errorf("NewError().Type() = %v, want %v", err.Type(), ValueTypeError)
			}

			if err.String() != tt.expected {
				t.Errorf("NewError().String() = %v, want %v", err.String(), tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
