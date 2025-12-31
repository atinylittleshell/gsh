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
			// Convert map[string]Value to map[string]*PropertyDescriptor
			props := make(map[string]*PropertyDescriptor)
			for k, v := range tt.properties {
				props[k] = &PropertyDescriptor{Value: v}
			}
			obj := &ObjectValue{Properties: props}

			if obj.Type() != ValueTypeObject {
				t.Errorf("ObjectValue.Type() = %v, want %v", obj.Type(), ValueTypeObject)
			}

			if obj.IsTruthy() != tt.truthy {
				t.Errorf("ObjectValue.IsTruthy() = %v, want %v", obj.IsTruthy(), tt.truthy)
			}

			// Test equality
			sameProps := make(map[string]*PropertyDescriptor)
			for k, v := range tt.properties {
				sameProps[k] = &PropertyDescriptor{Value: v}
			}
			same := &ObjectValue{Properties: sameProps}
			if !obj.Equals(same) {
				t.Error("ObjectValue should equal another ObjectValue with same properties")
			}

			different := &ObjectValue{Properties: map[string]*PropertyDescriptor{"x": {Value: &NumberValue{Value: 99}}}}
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
		Properties: map[string]*PropertyDescriptor{
			"name": {Value: &StringValue{Value: "Alice"}},
			"age":  {Value: &NumberValue{Value: 30}},
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

func TestObjectValue_DeepMerge(t *testing.T) {
	t.Run("nil override returns shallow copy", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"a": {Value: &StringValue{Value: "base_a"}},
				"b": {Value: &NumberValue{Value: 1}},
			},
		}

		result := base.DeepMerge(nil)

		// Should have same properties
		if len(result.Properties) != 2 {
			t.Errorf("expected 2 properties, got %d", len(result.Properties))
		}
		if result.GetPropertyValue("a").String() != "base_a" {
			t.Errorf("expected a='base_a', got %s", result.GetPropertyValue("a").String())
		}

		// Should be a copy, not the same object
		if result == base {
			t.Error("DeepMerge should return a new object, not the same reference")
		}
	})

	t.Run("empty base with override", func(t *testing.T) {
		base := &ObjectValue{Properties: map[string]*PropertyDescriptor{}}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"x": {Value: &StringValue{Value: "override_x"}},
			},
		}

		result := base.DeepMerge(override)

		if len(result.Properties) != 1 {
			t.Errorf("expected 1 property, got %d", len(result.Properties))
		}
		if result.GetPropertyValue("x").String() != "override_x" {
			t.Errorf("expected x='override_x', got %s", result.GetPropertyValue("x").String())
		}
	})

	t.Run("base with empty override", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"a": {Value: &StringValue{Value: "base_a"}},
			},
		}
		override := &ObjectValue{Properties: map[string]*PropertyDescriptor{}}

		result := base.DeepMerge(override)

		if len(result.Properties) != 1 {
			t.Errorf("expected 1 property, got %d", len(result.Properties))
		}
		if result.GetPropertyValue("a").String() != "base_a" {
			t.Errorf("expected a='base_a', got %s", result.GetPropertyValue("a").String())
		}
	})

	t.Run("simple merge with no overlapping keys", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"a": {Value: &StringValue{Value: "base_a"}},
				"b": {Value: &NumberValue{Value: 1}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"c": {Value: &StringValue{Value: "override_c"}},
				"d": {Value: &NumberValue{Value: 2}},
			},
		}

		result := base.DeepMerge(override)

		if len(result.Properties) != 4 {
			t.Errorf("expected 4 properties, got %d", len(result.Properties))
		}
		if result.GetPropertyValue("a").String() != "base_a" {
			t.Errorf("expected a='base_a', got %s", result.GetPropertyValue("a").String())
		}
		if result.GetPropertyValue("c").String() != "override_c" {
			t.Errorf("expected c='override_c', got %s", result.GetPropertyValue("c").String())
		}
	})

	t.Run("override replaces non-object values", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"name":  {Value: &StringValue{Value: "base_name"}},
				"count": {Value: &NumberValue{Value: 10}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"name": {Value: &StringValue{Value: "override_name"}},
			},
		}

		result := base.DeepMerge(override)

		if result.GetPropertyValue("name").String() != "override_name" {
			t.Errorf("expected name='override_name', got %s", result.GetPropertyValue("name").String())
		}
		// count should be preserved
		if result.GetPropertyValue("count").(*NumberValue).Value != 10 {
			t.Errorf("expected count=10, got %v", result.GetPropertyValue("count"))
		}
	})

	t.Run("deep merge nested objects", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"config": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"host": {Value: &StringValue{Value: "localhost"}},
						"port": {Value: &NumberValue{Value: 8080}},
					},
				}},
				"name": {Value: &StringValue{Value: "app"}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"config": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"port": {Value: &NumberValue{Value: 9090}},
					},
				}},
			},
		}

		result := base.DeepMerge(override)

		// name should be preserved
		if result.GetPropertyValue("name").String() != "app" {
			t.Errorf("expected name='app', got %s", result.GetPropertyValue("name").String())
		}

		// config should be merged
		config, ok := result.GetPropertyValue("config").(*ObjectValue)
		if !ok {
			t.Fatal("expected config to be an ObjectValue")
		}

		// host should be preserved from base
		if config.GetPropertyValue("host").String() != "localhost" {
			t.Errorf("expected host='localhost', got %s", config.GetPropertyValue("host").String())
		}

		// port should be overridden
		if config.GetPropertyValue("port").(*NumberValue).Value != 9090 {
			t.Errorf("expected port=9090, got %v", config.GetPropertyValue("port"))
		}
	})

	t.Run("deeply nested merge (3 levels)", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"level1": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"level2": {Value: &ObjectValue{
							Properties: map[string]*PropertyDescriptor{
								"a": {Value: &StringValue{Value: "base_a"}},
								"b": {Value: &StringValue{Value: "base_b"}},
							},
						}},
						"other": {Value: &StringValue{Value: "base_other"}},
					},
				}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"level1": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"level2": {Value: &ObjectValue{
							Properties: map[string]*PropertyDescriptor{
								"b": {Value: &StringValue{Value: "override_b"}},
								"c": {Value: &StringValue{Value: "override_c"}},
							},
						}},
					},
				}},
			},
		}

		result := base.DeepMerge(override)

		level1, ok := result.GetPropertyValue("level1").(*ObjectValue)
		if !ok {
			t.Fatal("expected level1 to be an ObjectValue")
		}

		// other should be preserved
		if level1.GetPropertyValue("other").String() != "base_other" {
			t.Errorf("expected other='base_other', got %s", level1.GetPropertyValue("other").String())
		}

		level2, ok := level1.GetPropertyValue("level2").(*ObjectValue)
		if !ok {
			t.Fatal("expected level2 to be an ObjectValue")
		}

		// a should be preserved from base
		if level2.GetPropertyValue("a").String() != "base_a" {
			t.Errorf("expected a='base_a', got %s", level2.GetPropertyValue("a").String())
		}

		// b should be overridden
		if level2.GetPropertyValue("b").String() != "override_b" {
			t.Errorf("expected b='override_b', got %s", level2.GetPropertyValue("b").String())
		}

		// c should be added from override
		if level2.GetPropertyValue("c").String() != "override_c" {
			t.Errorf("expected c='override_c', got %s", level2.GetPropertyValue("c").String())
		}
	})

	t.Run("override object replaces non-object", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"config": {Value: &StringValue{Value: "string_value"}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"config": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"host": {Value: &StringValue{Value: "localhost"}},
					},
				}},
			},
		}

		result := base.DeepMerge(override)

		config, ok := result.GetPropertyValue("config").(*ObjectValue)
		if !ok {
			t.Fatal("expected config to be an ObjectValue after override")
		}
		if config.GetPropertyValue("host").String() != "localhost" {
			t.Errorf("expected host='localhost', got %s", config.GetPropertyValue("host").String())
		}
	})

	t.Run("override non-object replaces object", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"config": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"host": {Value: &StringValue{Value: "localhost"}},
					},
				}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"config": {Value: &StringValue{Value: "simple_string"}},
			},
		}

		result := base.DeepMerge(override)

		if result.GetPropertyValue("config").String() != "simple_string" {
			t.Errorf("expected config='simple_string', got %s", result.GetPropertyValue("config").String())
		}
	})

	t.Run("does not modify original objects", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"a": {Value: &StringValue{Value: "base_a"}},
				"nested": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"x": {Value: &StringValue{Value: "base_x"}},
					},
				}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"a": {Value: &StringValue{Value: "override_a"}},
				"nested": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"y": {Value: &StringValue{Value: "override_y"}},
					},
				}},
			},
		}

		// Save original values
		originalBaseA := base.GetPropertyValue("a").String()
		originalBaseNestedX := base.GetPropertyValue("nested").(*ObjectValue).GetPropertyValue("x").String()
		originalOverrideA := override.GetPropertyValue("a").String()

		_ = base.DeepMerge(override)

		// Check base wasn't modified
		if base.GetPropertyValue("a").String() != originalBaseA {
			t.Error("base object was modified")
		}
		if base.GetPropertyValue("nested").(*ObjectValue).GetPropertyValue("x").String() != originalBaseNestedX {
			t.Error("base nested object was modified")
		}
		// Base nested should not have y
		if _, exists := base.GetPropertyValue("nested").(*ObjectValue).Properties["y"]; exists {
			t.Error("base nested object should not have 'y' property")
		}

		// Check override wasn't modified
		if override.GetPropertyValue("a").String() != originalOverrideA {
			t.Error("override object was modified")
		}
	})

	t.Run("returned object is fully independent - modifying it does not affect originals", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"preserved": {Value: &StringValue{Value: "base_preserved"}},
				"nested": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"baseOnly": {Value: &StringValue{Value: "base_nested_value"}},
					},
				}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"added": {Value: &StringValue{Value: "override_added"}},
				"nested": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"overrideOnly": {Value: &StringValue{Value: "override_nested_value"}},
					},
				}},
			},
		}

		result := base.DeepMerge(override)

		// Modify the result extensively
		result.Properties["preserved"] = &PropertyDescriptor{Value: &StringValue{Value: "MODIFIED"}}
		result.Properties["added"] = &PropertyDescriptor{Value: &StringValue{Value: "MODIFIED"}}
		result.Properties["newKey"] = &PropertyDescriptor{Value: &StringValue{Value: "NEW"}}

		resultNested := result.GetPropertyValue("nested").(*ObjectValue)
		resultNested.Properties["baseOnly"] = &PropertyDescriptor{Value: &StringValue{Value: "MODIFIED"}}
		resultNested.Properties["overrideOnly"] = &PropertyDescriptor{Value: &StringValue{Value: "MODIFIED"}}
		resultNested.Properties["newNestedKey"] = &PropertyDescriptor{Value: &StringValue{Value: "NEW"}}

		// Base should be completely unaffected
		if base.GetPropertyValue("preserved").String() != "base_preserved" {
			t.Error("modifying result affected base.preserved")
		}
		baseNested := base.GetPropertyValue("nested").(*ObjectValue)
		if baseNested.GetPropertyValue("baseOnly").String() != "base_nested_value" {
			t.Error("modifying result affected base.nested.baseOnly")
		}
		if _, exists := baseNested.Properties["overrideOnly"]; exists {
			t.Error("base.nested should not have overrideOnly")
		}
		if _, exists := baseNested.Properties["newNestedKey"]; exists {
			t.Error("base.nested should not have newNestedKey")
		}

		// Override should be completely unaffected
		if override.GetPropertyValue("added").String() != "override_added" {
			t.Error("modifying result affected override.added")
		}
		overrideNested := override.GetPropertyValue("nested").(*ObjectValue)
		if overrideNested.GetPropertyValue("overrideOnly").String() != "override_nested_value" {
			t.Error("modifying result affected override.nested.overrideOnly")
		}
		if _, exists := overrideNested.Properties["baseOnly"]; exists {
			t.Error("override.nested should not have baseOnly")
		}
		if _, exists := overrideNested.Properties["newNestedKey"]; exists {
			t.Error("override.nested should not have newNestedKey")
		}
	})

	t.Run("handles arrays (no deep merge, just replace)", func(t *testing.T) {
		base := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"items": {Value: &ArrayValue{
					Elements: []Value{
						&StringValue{Value: "a"},
						&StringValue{Value: "b"},
					},
				}},
			},
		}
		override := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"items": {Value: &ArrayValue{
					Elements: []Value{
						&StringValue{Value: "c"},
					},
				}},
			},
		}

		result := base.DeepMerge(override)

		arr, ok := result.GetPropertyValue("items").(*ArrayValue)
		if !ok {
			t.Fatal("expected items to be an ArrayValue")
		}
		if len(arr.Elements) != 1 {
			t.Errorf("expected 1 element, got %d", len(arr.Elements))
		}
		if arr.Elements[0].String() != "c" {
			t.Errorf("expected first element to be 'c', got %s", arr.Elements[0].String())
		}
	})

	t.Run("GSH_CONFIG-like merge scenario", func(t *testing.T) {
		// Simulates default GSH_CONFIG
		defaultConfig := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"prompt":              {Value: &StringValue{Value: "gsh> "}},
				"logLevel":            {Value: &StringValue{Value: "info"}},
				"starshipIntegration": {Value: &BoolValue{Value: true}},
				"showWelcome":         {Value: &BoolValue{Value: true}},
				"experimental": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"featureA": {Value: &BoolValue{Value: false}},
						"featureB": {Value: &BoolValue{Value: false}},
					},
				}},
			},
		}

		// User only wants to change logLevel and enable featureA
		userConfig := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"logLevel": {Value: &StringValue{Value: "debug"}},
				"experimental": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"featureA": {Value: &BoolValue{Value: true}},
					},
				}},
			},
		}

		result := defaultConfig.DeepMerge(userConfig)

		// Check preserved defaults
		if result.GetPropertyValue("prompt").String() != "gsh> " {
			t.Errorf("expected prompt='gsh> ', got %s", result.GetPropertyValue("prompt").String())
		}
		if !result.GetPropertyValue("starshipIntegration").(*BoolValue).Value {
			t.Error("expected starshipIntegration=true")
		}
		if !result.GetPropertyValue("showWelcome").(*BoolValue).Value {
			t.Error("expected showWelcome=true")
		}

		// Check user override
		if result.GetPropertyValue("logLevel").String() != "debug" {
			t.Errorf("expected logLevel='debug', got %s", result.GetPropertyValue("logLevel").String())
		}

		// Check nested experimental merge
		exp, ok := result.GetPropertyValue("experimental").(*ObjectValue)
		if !ok {
			t.Fatal("expected experimental to be an ObjectValue")
		}
		if !exp.GetPropertyValue("featureA").(*BoolValue).Value {
			t.Error("expected featureA=true (user override)")
		}
		if exp.GetPropertyValue("featureB").(*BoolValue).Value {
			t.Error("expected featureB=false (preserved default)")
		}
	})
}

func TestObjectValue_DeepCopy(t *testing.T) {
	t.Run("nil object returns nil", func(t *testing.T) {
		var obj *ObjectValue = nil
		result := obj.DeepCopy()
		if result != nil {
			t.Error("DeepCopy of nil should return nil")
		}
	})

	t.Run("creates independent copy of flat object", func(t *testing.T) {
		original := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"a": {Value: &StringValue{Value: "original"}},
			},
		}

		copied := original.DeepCopy()

		// Should have same values
		if copied.GetPropertyValue("a").String() != "original" {
			t.Errorf("expected a='original', got %s", copied.GetPropertyValue("a").String())
		}

		// Should be different object
		if copied == original {
			t.Error("DeepCopy should return a new object")
		}

		// Modifying copy's map shouldn't affect original
		copied.Properties["b"] = &PropertyDescriptor{Value: &StringValue{Value: "new"}}
		if _, exists := original.Properties["b"]; exists {
			t.Error("modifying copy should not affect original")
		}
	})

	t.Run("creates independent copy of nested objects", func(t *testing.T) {
		original := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"nested": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"x": {Value: &StringValue{Value: "original_x"}},
					},
				}},
			},
		}

		copied := original.DeepCopy()

		// Modify nested object through the copy
		copiedNested := copied.GetPropertyValue("nested").(*ObjectValue)
		copiedNested.Properties["x"] = &PropertyDescriptor{Value: &StringValue{Value: "modified_x"}}
		copiedNested.Properties["y"] = &PropertyDescriptor{Value: &StringValue{Value: "new_y"}}

		// Original should NOT be affected
		originalNested := original.GetPropertyValue("nested").(*ObjectValue)
		if originalNested.GetPropertyValue("x").String() != "original_x" {
			t.Error("modifying nested object in copy should not affect original")
		}
		if _, exists := originalNested.Properties["y"]; exists {
			t.Error("adding to nested object in copy should not affect original")
		}
	})

	t.Run("creates independent copy of deeply nested objects", func(t *testing.T) {
		original := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"level1": {Value: &ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"level2": {Value: &ObjectValue{
							Properties: map[string]*PropertyDescriptor{
								"value": {Value: &StringValue{Value: "deep_original"}},
							},
						}},
					},
				}},
			},
		}

		copied := original.DeepCopy()

		// Modify deeply nested object
		level1 := copied.GetPropertyValue("level1").(*ObjectValue)
		level2 := level1.GetPropertyValue("level2").(*ObjectValue)
		level2.Properties["value"] = &PropertyDescriptor{Value: &StringValue{Value: "deep_modified"}}

		// Original should NOT be affected
		origLevel1 := original.GetPropertyValue("level1").(*ObjectValue)
		origLevel2 := origLevel1.GetPropertyValue("level2").(*ObjectValue)
		if origLevel2.GetPropertyValue("value").String() != "deep_original" {
			t.Error("modifying deeply nested object in copy should not affect original")
		}
	})

	t.Run("creates independent copy of arrays", func(t *testing.T) {
		original := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"items": {Value: &ArrayValue{
					Elements: []Value{
						&StringValue{Value: "a"},
						&ObjectValue{
							Properties: map[string]*PropertyDescriptor{
								"nested": {Value: &StringValue{Value: "original_nested"}},
							},
						},
					},
				}},
			},
		}

		copied := original.DeepCopy()

		// Modify array through the copy
		copiedArray := copied.GetPropertyValue("items").(*ArrayValue)
		copiedArray.Elements[0] = &StringValue{Value: "modified_a"}

		// Modify nested object inside array
		copiedNestedObj := copiedArray.Elements[1].(*ObjectValue)
		copiedNestedObj.Properties["nested"] = &PropertyDescriptor{Value: &StringValue{Value: "modified_nested"}}

		// Original should NOT be affected
		originalArray := original.GetPropertyValue("items").(*ArrayValue)
		if originalArray.Elements[0].String() != "a" {
			t.Error("modifying array element in copy should not affect original")
		}
		originalNestedObj := originalArray.Elements[1].(*ObjectValue)
		if originalNestedObj.GetPropertyValue("nested").String() != "original_nested" {
			t.Error("modifying nested object in array in copy should not affect original")
		}
	})
}

func TestArrayValue_DeepCopy(t *testing.T) {
	t.Run("nil array returns nil", func(t *testing.T) {
		var arr *ArrayValue = nil
		result := arr.DeepCopy()
		if result != nil {
			t.Error("DeepCopy of nil should return nil")
		}
	})

	t.Run("creates independent copy", func(t *testing.T) {
		original := &ArrayValue{
			Elements: []Value{
				&StringValue{Value: "a"},
				&NumberValue{Value: 42},
				&ObjectValue{
					Properties: map[string]*PropertyDescriptor{
						"key": {Value: &StringValue{Value: "value"}},
					},
				},
			},
		}

		copied := original.DeepCopy()

		// Should have same values
		if len(copied.Elements) != 3 {
			t.Errorf("expected 3 elements, got %d", len(copied.Elements))
		}

		// Modify the copy
		copied.Elements[0] = &StringValue{Value: "modified"}
		copiedObj := copied.Elements[2].(*ObjectValue)
		copiedObj.Properties["key"] = &PropertyDescriptor{Value: &StringValue{Value: "modified_value"}}

		// Original should NOT be affected
		if original.Elements[0].String() != "a" {
			t.Error("modifying copy element should not affect original")
		}
		originalObj := original.Elements[2].(*ObjectValue)
		if originalObj.GetPropertyValue("key").String() != "value" {
			t.Error("modifying nested object in copy should not affect original")
		}
	})
}

func TestModelResolver(t *testing.T) {
	t.Run("ModelValue implements ModelResolver and returns itself", func(t *testing.T) {
		model := &ModelValue{
			Name: "testModel",
		}

		// ModelValue should implement ModelResolver
		var resolver ModelResolver = model

		// GetModel should return the model itself
		result := resolver.GetModel()
		if result != model {
			t.Errorf("expected ModelValue.GetModel() to return itself, got %v", result)
		}
	})

	t.Run("SDKModelRef implements Value interface", func(t *testing.T) {
		ref := &SDKModelRef{Tier: "workhorse"}

		if ref.Type() != ValueTypeModel {
			t.Errorf("expected type %v, got %v", ValueTypeModel, ref.Type())
		}
		if ref.String() != "gsh.models.workhorse" {
			t.Errorf("expected String() 'gsh.models.workhorse', got %q", ref.String())
		}
		if !ref.IsTruthy() {
			t.Error("expected SDKModelRef to be truthy")
		}
	})

	t.Run("SDKModelRef.Equals", func(t *testing.T) {
		ref1 := &SDKModelRef{Tier: "lite"}
		ref2 := &SDKModelRef{Tier: "lite"}
		ref3 := &SDKModelRef{Tier: "workhorse"}

		if !ref1.Equals(ref2) {
			t.Error("expected SDKModelRef with same tier to be equal")
		}
		if ref1.Equals(ref3) {
			t.Error("expected SDKModelRef with different tier to not be equal")
		}
		if ref1.Equals(&StringValue{Value: "lite"}) {
			t.Error("expected SDKModelRef to not equal other value types")
		}
	})

	t.Run("SDKModelRef resolves to correct model tier", func(t *testing.T) {
		liteModel := &ModelValue{Name: "liteModel"}
		workhorseModel := &ModelValue{Name: "workhorseModel"}
		premiumModel := &ModelValue{Name: "premiumModel"}

		models := &Models{
			Lite:      liteModel,
			Workhorse: workhorseModel,
			Premium:   premiumModel,
		}

		tests := []struct {
			tier     string
			expected *ModelValue
		}{
			{"lite", liteModel},
			{"workhorse", workhorseModel},
			{"premium", premiumModel},
		}

		for _, tc := range tests {
			ref := &SDKModelRef{Tier: tc.tier, Models: models}
			result := ref.GetModel()
			if result != tc.expected {
				t.Errorf("SDKModelRef{Tier: %q}.GetModel() = %v, expected %v", tc.tier, result, tc.expected)
			}
		}
	})

	t.Run("SDKModelRef returns nil when Models is nil", func(t *testing.T) {
		ref := &SDKModelRef{Tier: "lite", Models: nil}
		result := ref.GetModel()
		if result != nil {
			t.Errorf("expected nil when Models is nil, got %v", result)
		}
	})

	t.Run("SDKModelRef returns nil for unknown tier", func(t *testing.T) {
		models := &Models{
			Lite: &ModelValue{Name: "liteModel"},
		}
		ref := &SDKModelRef{Tier: "unknown", Models: models}
		result := ref.GetModel()
		if result != nil {
			t.Errorf("expected nil for unknown tier, got %v", result)
		}
	})

	t.Run("SDKModelRef dynamically resolves when model tier changes", func(t *testing.T) {
		// Initial model
		initialModel := &ModelValue{Name: "initialModel"}
		models := &Models{
			Workhorse: initialModel,
		}

		ref := &SDKModelRef{Tier: "workhorse", Models: models}

		// First resolution
		result1 := ref.GetModel()
		if result1 != initialModel {
			t.Errorf("first resolution: expected %v, got %v", initialModel, result1)
		}

		// Change the model in the tier
		newModel := &ModelValue{Name: "newModel"}
		models.Workhorse = newModel

		// Second resolution should return the new model
		result2 := ref.GetModel()
		if result2 != newModel {
			t.Errorf("second resolution: expected %v, got %v", newModel, result2)
		}

		// Verify the first result is still the old model (static)
		if result1 == result2 {
			t.Error("expected different model instances after tier change")
		}
	})
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
