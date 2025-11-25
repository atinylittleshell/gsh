package interpreter

import (
	"testing"
)

func TestNewEnvironment(t *testing.T) {
	env := NewEnvironment()

	if env == nil {
		t.Fatal("NewEnvironment() returned nil")
	}

	if env.store == nil {
		t.Error("NewEnvironment() store is nil")
	}

	if env.outer != nil {
		t.Error("NewEnvironment() outer should be nil")
	}

	if len(env.store) != 0 {
		t.Errorf("NewEnvironment() store should be empty, got %d items", len(env.store))
	}
}

func TestNewEnclosedEnvironment(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &NumberValue{Value: 42})

	inner := NewEnclosedEnvironment(outer)

	if inner == nil {
		t.Fatal("NewEnclosedEnvironment() returned nil")
	}

	if inner.outer != outer {
		t.Error("NewEnclosedEnvironment() outer is not set correctly")
	}

	// Inner environment should be able to access outer variables
	value, ok := inner.Get("x")
	if !ok {
		t.Error("Inner environment should be able to access outer variables")
	}

	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 42 {
		t.Errorf("Expected x=42, got %v", value)
	}
}

func TestEnvironment_Set_and_Get(t *testing.T) {
	env := NewEnvironment()

	// Set a variable
	env.Set("name", &StringValue{Value: "Alice"})

	// Get the variable
	value, ok := env.Get("name")
	if !ok {
		t.Error("Get() should return true for existing variable")
	}

	if strVal, ok := value.(*StringValue); !ok || strVal.Value != "Alice" {
		t.Errorf("Expected name='Alice', got %v", value)
	}

	// Get non-existent variable
	_, ok = env.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent variable")
	}
}

func TestEnvironment_Set_Multiple(t *testing.T) {
	env := NewEnvironment()

	env.Set("x", &NumberValue{Value: 1})
	env.Set("y", &NumberValue{Value: 2})
	env.Set("z", &NumberValue{Value: 3})

	x, ok := env.Get("x")
	if !ok || x.(*NumberValue).Value != 1 {
		t.Errorf("Expected x=1, got %v", x)
	}

	y, ok := env.Get("y")
	if !ok || y.(*NumberValue).Value != 2 {
		t.Errorf("Expected y=2, got %v", y)
	}

	z, ok := env.Get("z")
	if !ok || z.(*NumberValue).Value != 3 {
		t.Errorf("Expected z=3, got %v", z)
	}
}

func TestEnvironment_Set_Update(t *testing.T) {
	env := NewEnvironment()

	// Set initial value
	env.Set("counter", &NumberValue{Value: 0})

	// Update value
	env.Set("counter", &NumberValue{Value: 1})

	value, ok := env.Get("counter")
	if !ok {
		t.Error("Get() should return true for existing variable")
	}

	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 1 {
		t.Errorf("Expected counter=1, got %v", value)
	}
}

func TestEnvironment_Nested_Scopes(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("global", &StringValue{Value: "outer"})

	inner := NewEnclosedEnvironment(outer)
	inner.Set("local", &StringValue{Value: "inner"})

	// Inner can access outer variables
	value, ok := inner.Get("global")
	if !ok {
		t.Error("Inner scope should access outer variables")
	}
	if strVal, ok := value.(*StringValue); !ok || strVal.Value != "outer" {
		t.Errorf("Expected global='outer', got %v", value)
	}

	// Inner can access its own variables
	value, ok = inner.Get("local")
	if !ok {
		t.Error("Inner scope should access its own variables")
	}
	if strVal, ok := value.(*StringValue); !ok || strVal.Value != "inner" {
		t.Errorf("Expected local='inner', got %v", value)
	}

	// Outer cannot access inner variables
	_, ok = outer.Get("local")
	if ok {
		t.Error("Outer scope should not access inner variables")
	}
}

func TestEnvironment_Shadowing(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &NumberValue{Value: 10})

	inner := NewEnclosedEnvironment(outer)
	inner.Set("x", &NumberValue{Value: 20})

	// Inner scope should see its own value
	value, ok := inner.Get("x")
	if !ok {
		t.Error("Get() should return true")
	}
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 20 {
		t.Errorf("Expected x=20 in inner scope, got %v", value)
	}

	// Outer scope should still have its original value
	value, ok = outer.Get("x")
	if !ok {
		t.Error("Get() should return true")
	}
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 10 {
		t.Errorf("Expected x=10 in outer scope, got %v", value)
	}
}

func TestEnvironment_Define(t *testing.T) {
	env := NewEnvironment()

	// Define a new variable
	err := env.Define("x", &NumberValue{Value: 42})
	if err != nil {
		t.Errorf("Define() should not return error, got %v", err)
	}

	// Get the defined variable
	value, ok := env.Get("x")
	if !ok {
		t.Error("Get() should return true for defined variable")
	}
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 42 {
		t.Errorf("Expected x=42, got %v", value)
	}

	// Try to define the same variable again
	err = env.Define("x", &NumberValue{Value: 100})
	if err == nil {
		t.Error("Define() should return error when variable already exists")
	}

	// Original value should remain unchanged
	value, _ = env.Get("x")
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 42 {
		t.Errorf("Expected x=42 (unchanged), got %v", value)
	}
}

func TestEnvironment_Update(t *testing.T) {
	env := NewEnvironment()

	// Try to update non-existent variable
	err := env.Update("x", &NumberValue{Value: 42})
	if err == nil {
		t.Error("Update() should return error for non-existent variable")
	}

	// Define a variable
	env.Set("x", &NumberValue{Value: 10})

	// Update the variable
	err = env.Update("x", &NumberValue{Value: 20})
	if err != nil {
		t.Errorf("Update() should not return error, got %v", err)
	}

	// Check the updated value
	value, _ := env.Get("x")
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 20 {
		t.Errorf("Expected x=20, got %v", value)
	}
}

func TestEnvironment_Update_Nested(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &NumberValue{Value: 10})

	inner := NewEnclosedEnvironment(outer)

	// Update variable from inner scope
	err := inner.Update("x", &NumberValue{Value: 20})
	if err != nil {
		t.Errorf("Update() should not return error, got %v", err)
	}

	// Check that outer scope was updated
	value, _ := outer.Get("x")
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 20 {
		t.Errorf("Expected x=20 in outer scope, got %v", value)
	}

	// Inner scope should also see the updated value
	value, _ = inner.Get("x")
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 20 {
		t.Errorf("Expected x=20 in inner scope, got %v", value)
	}
}

func TestEnvironment_Has(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("global", &StringValue{Value: "outer"})

	inner := NewEnclosedEnvironment(outer)
	inner.Set("local", &StringValue{Value: "inner"})

	// Inner should see both variables
	if !inner.Has("global") {
		t.Error("Has() should return true for outer variable")
	}
	if !inner.Has("local") {
		t.Error("Has() should return true for inner variable")
	}
	if inner.Has("nonexistent") {
		t.Error("Has() should return false for non-existent variable")
	}

	// Outer should only see its own variable
	if !outer.Has("global") {
		t.Error("Has() should return true for own variable")
	}
	if outer.Has("local") {
		t.Error("Has() should return false for inner variable")
	}
}

func TestEnvironment_Delete(t *testing.T) {
	env := NewEnvironment()
	env.Set("x", &NumberValue{Value: 42})
	env.Set("y", &NumberValue{Value: 100})

	// Delete existing variable
	if !env.Delete("x") {
		t.Error("Delete() should return true for existing variable")
	}

	// Variable should no longer exist
	if env.Has("x") {
		t.Error("Variable should not exist after deletion")
	}

	// Other variables should remain
	if !env.Has("y") {
		t.Error("Other variables should remain after deletion")
	}

	// Delete non-existent variable
	if env.Delete("nonexistent") {
		t.Error("Delete() should return false for non-existent variable")
	}
}

func TestEnvironment_Delete_Nested(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &NumberValue{Value: 10})

	inner := NewEnclosedEnvironment(outer)
	inner.Set("x", &NumberValue{Value: 20})

	// Delete from inner scope
	if !inner.Delete("x") {
		t.Error("Delete() should return true for existing variable in current scope")
	}

	// Inner scope should now see outer's value
	value, ok := inner.Get("x")
	if !ok {
		t.Error("Should be able to access outer variable after inner deletion")
	}
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 10 {
		t.Errorf("Expected x=10 from outer scope, got %v", value)
	}

	// Outer scope should still have its value
	value, _ = outer.Get("x")
	if numVal, ok := value.(*NumberValue); !ok || numVal.Value != 10 {
		t.Errorf("Expected x=10 in outer scope, got %v", value)
	}
}

func TestEnvironment_Keys(t *testing.T) {
	env := NewEnvironment()
	env.Set("x", &NumberValue{Value: 1})
	env.Set("y", &NumberValue{Value: 2})
	env.Set("z", &NumberValue{Value: 3})

	keys := env.Keys()

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check that all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	if !keyMap["x"] || !keyMap["y"] || !keyMap["z"] {
		t.Errorf("Expected keys [x, y, z], got %v", keys)
	}
}

func TestEnvironment_Keys_Empty(t *testing.T) {
	env := NewEnvironment()
	keys := env.Keys()

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for empty environment, got %d", len(keys))
	}
}

func TestEnvironment_AllKeys(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("a", &NumberValue{Value: 1})
	outer.Set("b", &NumberValue{Value: 2})

	inner := NewEnclosedEnvironment(outer)
	inner.Set("c", &NumberValue{Value: 3})
	inner.Set("d", &NumberValue{Value: 4})

	allKeys := inner.AllKeys()

	if len(allKeys) != 4 {
		t.Errorf("Expected 4 keys, got %d", len(allKeys))
	}

	// Check that all keys are present
	keyMap := make(map[string]bool)
	for _, k := range allKeys {
		keyMap[k] = true
	}

	if !keyMap["a"] || !keyMap["b"] || !keyMap["c"] || !keyMap["d"] {
		t.Errorf("Expected keys [a, b, c, d], got %v", allKeys)
	}
}

func TestEnvironment_AllKeys_Shadowing(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &NumberValue{Value: 1})
	outer.Set("y", &NumberValue{Value: 2})

	inner := NewEnclosedEnvironment(outer)
	inner.Set("x", &NumberValue{Value: 10}) // Shadow x
	inner.Set("z", &NumberValue{Value: 3})

	allKeys := inner.AllKeys()

	// Should have 3 unique keys (x is shadowed, not duplicated)
	if len(allKeys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(allKeys))
	}

	// Check that all keys are present
	keyMap := make(map[string]bool)
	for _, k := range allKeys {
		keyMap[k] = true
	}

	if !keyMap["x"] || !keyMap["y"] || !keyMap["z"] {
		t.Errorf("Expected keys [x, y, z], got %v", allKeys)
	}
}

func TestEnvironment_Clone(t *testing.T) {
	original := NewEnvironment()
	original.Set("x", &NumberValue{Value: 42})
	original.Set("y", &StringValue{Value: "test"})

	cloned := original.Clone()

	// Cloned should have the same values
	value, ok := cloned.Get("x")
	if !ok || value.(*NumberValue).Value != 42 {
		t.Error("Cloned environment should have same values")
	}

	value, ok = cloned.Get("y")
	if !ok || value.(*StringValue).Value != "test" {
		t.Error("Cloned environment should have same values")
	}

	// Modifying cloned should not affect original
	cloned.Set("x", &NumberValue{Value: 100})

	value, _ = original.Get("x")
	if value.(*NumberValue).Value != 42 {
		t.Error("Original should not be affected by changes to clone")
	}

	value, _ = cloned.Get("x")
	if value.(*NumberValue).Value != 100 {
		t.Error("Clone should have updated value")
	}
}

func TestEnvironment_Clone_Nested(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("a", &NumberValue{Value: 1})

	inner := NewEnclosedEnvironment(outer)
	inner.Set("b", &NumberValue{Value: 2})

	cloned := inner.Clone()

	// Cloned should have same outer reference
	if cloned.outer != inner.outer {
		t.Error("Cloned environment should have same outer reference")
	}

	// Cloned should have its own variables
	value, ok := cloned.Get("b")
	if !ok || value.(*NumberValue).Value != 2 {
		t.Error("Cloned environment should have own variables")
	}

	// Cloned should still access outer variables
	value, ok = cloned.Get("a")
	if !ok || value.(*NumberValue).Value != 1 {
		t.Error("Cloned environment should access outer variables")
	}
}

func TestEnvironment_Complex_Nesting(t *testing.T) {
	// Level 0 (global)
	global := NewEnvironment()
	global.Set("level", &NumberValue{Value: 0})
	global.Set("global", &StringValue{Value: "global"})

	// Level 1
	level1 := NewEnclosedEnvironment(global)
	level1.Set("level", &NumberValue{Value: 1})
	level1.Set("local1", &StringValue{Value: "level1"})

	// Level 2
	level2 := NewEnclosedEnvironment(level1)
	level2.Set("level", &NumberValue{Value: 2})
	level2.Set("local2", &StringValue{Value: "level2"})

	// Level 3
	level3 := NewEnclosedEnvironment(level2)
	level3.Set("local3", &StringValue{Value: "level3"})

	// Test access from deepest level
	value, _ := level3.Get("level")
	if value.(*NumberValue).Value != 2 {
		t.Error("Should see nearest shadowed value")
	}

	value, _ = level3.Get("global")
	if value.(*StringValue).Value != "global" {
		t.Error("Should access global variable")
	}

	value, _ = level3.Get("local1")
	if value.(*StringValue).Value != "level1" {
		t.Error("Should access parent variables")
	}

	value, _ = level3.Get("local2")
	if value.(*StringValue).Value != "level2" {
		t.Error("Should access grandparent variables")
	}

	value, _ = level3.Get("local3")
	if value.(*StringValue).Value != "level3" {
		t.Error("Should access own variables")
	}

	// Test that lower levels cannot access higher level variables
	if level1.Has("local2") || level1.Has("local3") {
		t.Error("Parent should not access child variables")
	}
}

func TestEnvironment_Set_Creates_Shadow(t *testing.T) {
	outer := NewEnvironment()
	outer.Set("x", &NumberValue{Value: 10})

	inner := NewEnclosedEnvironment(outer)

	// Set should create a new variable in inner scope (shadowing)
	inner.Set("x", &NumberValue{Value: 20})

	// Inner should see its own value
	value, _ := inner.Get("x")
	if value.(*NumberValue).Value != 20 {
		t.Error("Inner scope should see its own shadowed value")
	}

	// Outer should keep its original value
	value, _ = outer.Get("x")
	if value.(*NumberValue).Value != 10 {
		t.Error("Outer scope should keep original value")
	}
}
