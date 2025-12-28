package interpreter

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Helper to evaluate script for object method tests
func evalObjectTest(t *testing.T, script string) (Value, error) {
	t.Helper()
	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	result, err := interp.Eval(program)
	if err != nil {
		return nil, err
	}
	return result.FinalResult, nil
}

func TestObjectKeys(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		wantType ValueType
		checkFn  func(Value) bool
	}{
		{
			name: "keys of simple object",
			script: `obj = {name: "John", age: 30}
obj.keys()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				if !ok || len(arr.Elements) != 2 {
					return false
				}
				// Check that we have both keys (order doesn't matter in maps)
				keys := make(map[string]bool)
				for _, elem := range arr.Elements {
					if str, ok := elem.(*StringValue); ok {
						keys[str.Value] = true
					}
				}
				return keys["name"] && keys["age"]
			},
		},
		{
			name: "keys of empty object",
			script: `obj = {}
obj.keys()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				return ok && len(arr.Elements) == 0
			},
		},
		{
			name: "keys of nested object",
			script: `obj = {x: 1, y: {a: 2}}
obj.keys()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				if !ok || len(arr.Elements) != 2 {
					return false
				}
				keys := make(map[string]bool)
				for _, elem := range arr.Elements {
					if str, ok := elem.(*StringValue); ok {
						keys[str.Value] = true
					}
				}
				return keys["x"] && keys["y"]
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, err := evalObjectTest(t, tt.script)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Type() != tt.wantType {
				t.Errorf("got type %v, want %v", result.Type(), tt.wantType)
			}
			if tt.checkFn != nil && !tt.checkFn(result) {
				t.Errorf("checkFn failed for result: %v", result)
			}
		})
	}
}

func TestObjectValues(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		wantType ValueType
		checkFn  func(Value) bool
	}{
		{
			name: "values of simple object",
			script: `obj = {name: "John", age: 30}
obj.values()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				if !ok || len(arr.Elements) != 2 {
					return false
				}
				// Check that we have the values (order doesn't matter)
				hasName := false
				hasAge := false
				for _, elem := range arr.Elements {
					if str, ok := elem.(*StringValue); ok && str.Value == "John" {
						hasName = true
					}
					if num, ok := elem.(*NumberValue); ok && num.Value == 30 {
						hasAge = true
					}
				}
				return hasName && hasAge
			},
		},
		{
			name: "values of empty object",
			script: `obj = {}
obj.values()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				return ok && len(arr.Elements) == 0
			},
		},
		{
			name: "values with different types",
			script: `obj = {a: 1, b: "hello", c: true, d: null}
obj.values()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				if !ok || len(arr.Elements) != 4 {
					return false
				}
				// Check that we have all types
				types := make(map[ValueType]bool)
				for _, elem := range arr.Elements {
					types[elem.Type()] = true
				}
				return types[ValueTypeNumber] && types[ValueTypeString] && types[ValueTypeBool] && types[ValueTypeNull]
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, err := evalObjectTest(t, tt.script)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Type() != tt.wantType {
				t.Errorf("got type %v, want %v", result.Type(), tt.wantType)
			}
			if tt.checkFn != nil && !tt.checkFn(result) {
				t.Errorf("checkFn failed for result: %v", result)
			}
		})
	}
}

func TestObjectEntries(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		wantType ValueType
		checkFn  func(Value) bool
	}{
		{
			name: "entries of simple object",
			script: `obj = {name: "John", age: 30}
obj.entries()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				if !ok || len(arr.Elements) != 2 {
					return false
				}
				// Each element should be an array of [key, value]
				for _, elem := range arr.Elements {
					entry, ok := elem.(*ArrayValue)
					if !ok || len(entry.Elements) != 2 {
						return false
					}
					// First element should be a string (key)
					if entry.Elements[0].Type() != ValueTypeString {
						return false
					}
				}
				return true
			},
		},
		{
			name: "entries of empty object",
			script: `obj = {}
obj.entries()`,
			wantType: ValueTypeArray,
			checkFn: func(v Value) bool {
				arr, ok := v.(*ArrayValue)
				return ok && len(arr.Elements) == 0
			},
		},
		{
			name: "entries with specific values",
			script: `obj = {x: 10, y: 20}
entries = obj.entries()
entries[0][0] + entries[0][1]`,
			wantType: ValueTypeString,
			checkFn: func(v Value) bool {
				str, ok := v.(*StringValue)
				if !ok {
					return false
				}
				// Should be either "x10" or "y20" depending on map iteration order
				return str.Value == "x10" || str.Value == "y20"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, err := evalObjectTest(t, tt.script)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Type() != tt.wantType {
				t.Errorf("got type %v, want %v", result.Type(), tt.wantType)
			}
			if tt.checkFn != nil && !tt.checkFn(result) {
				t.Errorf("checkFn failed for result: %v", result)
			}
		})
	}
}

func TestObjectHasOwnProperty(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		wantType   ValueType
		wantResult bool
	}{
		{
			name: "has existing property",
			script: `obj = {name: "John", age: 30}
obj.hasOwnProperty("name")`,
			wantType:   ValueTypeBool,
			wantResult: true,
		},
		{
			name: "does not have property",
			script: `obj = {name: "John", age: 30}
obj.hasOwnProperty("email")`,
			wantType:   ValueTypeBool,
			wantResult: false,
		},
		{
			name: "empty object",
			script: `obj = {}
obj.hasOwnProperty("anything")`,
			wantType:   ValueTypeBool,
			wantResult: false,
		},
		{
			name: "check with null value",
			script: `obj = {x: null}
obj.hasOwnProperty("x")`,
			wantType:   ValueTypeBool,
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, err := evalObjectTest(t, tt.script)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Type() != tt.wantType {
				t.Errorf("got type %v, want %v", result.Type(), tt.wantType)
			}
			if boolVal, ok := result.(*BoolValue); ok {
				if boolVal.Value != tt.wantResult {
					t.Errorf("got %v, want %v", boolVal.Value, tt.wantResult)
				}
			}
		})
	}
}

func TestObjectHasOwnPropertyErrors(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{
			name: "no arguments",
			script: `obj = {x: 1}
obj.hasOwnProperty()`,
		},
		{
			name: "too many arguments",
			script: `obj = {x: 1}
obj.hasOwnProperty("x", "y")`,
		},
		{
			name: "non-string argument",
			script: `obj = {x: 1}
obj.hasOwnProperty(123)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := evalObjectTest(t, tt.script)
			if err == nil {
				t.Error("expected error but got none")
			}
		})
	}
}

func TestObjectMethodsChaining(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		wantType ValueType
		checkFn  func(Value) bool
	}{
		{
			name: "keys then join",
			script: `obj = {a: 1, b: 2}
obj.keys().join(",")`,
			wantType: ValueTypeString,
			checkFn: func(v Value) bool {
				str, ok := v.(*StringValue)
				if !ok {
					return false
				}
				// Should be either "a,b" or "b,a" depending on map iteration
				return str.Value == "a,b" || str.Value == "b,a"
			},
		},
		{
			name: "values then length",
			script: `obj = {x: 10, y: 20, z: 30}
obj.values().length`,
			wantType: ValueTypeNumber,
			checkFn: func(v Value) bool {
				num, ok := v.(*NumberValue)
				return ok && num.Value == 3
			},
		},
		{
			name: "entries iteration",
			script: `obj = {a: 1, b: 2}
count = 0
for (entry of obj.entries()) { count = count + 1
} count`,
			wantType: ValueTypeNumber,
			checkFn: func(v Value) bool {
				num, ok := v.(*NumberValue)
				return ok && num.Value == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, err := evalObjectTest(t, tt.script)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Type() != tt.wantType {
				t.Errorf("got type %v, want %v", result.Type(), tt.wantType)
			}
			if tt.checkFn != nil && !tt.checkFn(result) {
				t.Errorf("checkFn failed for result: %v", result)
			}
		})
	}
}

func TestObjectMethodValue(t *testing.T) {
	obj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"x": {Value: &NumberValue{Value: 1}},
			"y": {Value: &NumberValue{Value: 2}},
		},
	}

	method := &ObjectMethodValue{
		Name: "keys",
		Impl: objectKeysImpl,
		Obj:  obj,
	}

	if method.Type() != ValueTypeTool {
		t.Errorf("Type() = %v, want %v", method.Type(), ValueTypeTool)
	}

	if !method.IsTruthy() {
		t.Error("IsTruthy() should return true for method values")
	}

	if method.Equals(&NullValue{}) {
		t.Error("Equals() should return false for different types")
	}

	expectedStr := "<object method: keys>"
	if method.String() != expectedStr {
		t.Errorf("String() = %v, want %v", method.String(), expectedStr)
	}
}

func TestObjectKeysImpl(t *testing.T) {
	obj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"a": {Value: &NumberValue{Value: 1}},
			"b": {Value: &StringValue{Value: "hello"}},
		},
	}

	result, err := objectKeysImpl(obj, []Value{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	arr, ok := result.(*ArrayValue)
	if !ok {
		t.Fatalf("result is not ArrayValue, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("got %d elements, want 2", len(arr.Elements))
	}

	// Verify all elements are strings
	for _, elem := range arr.Elements {
		if elem.Type() != ValueTypeString {
			t.Errorf("element type is %v, want string", elem.Type())
		}
	}
}

func TestObjectValuesImpl(t *testing.T) {
	obj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"a": {Value: &NumberValue{Value: 1}},
			"b": {Value: &StringValue{Value: "hello"}},
		},
	}

	result, err := objectValuesImpl(obj, []Value{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	arr, ok := result.(*ArrayValue)
	if !ok {
		t.Fatalf("result is not ArrayValue, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("got %d elements, want 2", len(arr.Elements))
	}
}

func TestObjectEntriesImpl(t *testing.T) {
	obj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"x": {Value: &NumberValue{Value: 10}},
			"y": {Value: &NumberValue{Value: 20}},
		},
	}

	result, err := objectEntriesImpl(obj, []Value{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	arr, ok := result.(*ArrayValue)
	if !ok {
		t.Fatalf("result is not ArrayValue, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("got %d elements, want 2", len(arr.Elements))
	}

	// Each element should be a 2-element array
	for _, elem := range arr.Elements {
		entry, ok := elem.(*ArrayValue)
		if !ok {
			t.Errorf("entry is not ArrayValue, got %T", elem)
			continue
		}
		if len(entry.Elements) != 2 {
			t.Errorf("entry has %d elements, want 2", len(entry.Elements))
		}
		if entry.Elements[0].Type() != ValueTypeString {
			t.Errorf("key type is %v, want string", entry.Elements[0].Type())
		}
	}
}

func TestObjectHasOwnPropertyImpl(t *testing.T) {
	obj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"name": {Value: &StringValue{Value: "John"}},
		},
	}

	// Test existing property
	result, err := objectHasOwnPropertyImpl(obj, []Value{&StringValue{Value: "name"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	boolVal, ok := result.(*BoolValue)
	if !ok {
		t.Fatalf("result is not BoolValue, got %T", result)
	}
	if !boolVal.Value {
		t.Error("expected true for existing property")
	}

	// Test non-existing property
	result, err = objectHasOwnPropertyImpl(obj, []Value{&StringValue{Value: "age"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	boolVal, ok = result.(*BoolValue)
	if !ok {
		t.Fatalf("result is not BoolValue, got %T", result)
	}
	if boolVal.Value {
		t.Error("expected false for non-existing property")
	}

	// Test error cases
	_, err = objectHasOwnPropertyImpl(obj, []Value{})
	if err == nil {
		t.Error("expected error for no arguments")
	}

	_, err = objectHasOwnPropertyImpl(obj, []Value{&NumberValue{Value: 1}})
	if err == nil {
		t.Error("expected error for non-string argument")
	}
}
