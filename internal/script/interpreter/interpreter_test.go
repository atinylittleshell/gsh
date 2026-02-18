package interpreter

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Helper function to parse and evaluate code
func testEval(t *testing.T, input string) Value {
	t.Helper()
	result := testEvalFull(t, input)
	return result.FinalResult
}

// Helper function to parse and evaluate code, returning full result
func testEvalFull(t *testing.T, input string) *EvalResult {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	return result
}

// Helper function to parse and evaluate code expecting an error
func testEvalError(t *testing.T, input string) error {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	_, err := interp.Eval(program)
	return err
}

func TestVariableDeclaration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x = 5", "5"},
		{"name = \"Alice\"", "Alice"},
		{"isActive = true", "true"},
		{"value = 42.5", "42.5"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		if result.String() != tt.expected {
			t.Errorf("for input %q: expected %q, got %q", tt.input, tt.expected, result.String())
		}
	}
}

func TestVariableAssignment(t *testing.T) {
	input := `
x = 5
x = 10
x
`
	result := testEval(t, input)
	if result.String() != "10" {
		t.Errorf("expected 10, got %s", result.String())
	}
}

func TestVariableReassignment(t *testing.T) {
	input := `
count = 0
count = count + 1
count = count + 1
count
`
	result := testEval(t, input)
	if result.String() != "2" {
		t.Errorf("expected 2, got %s", result.String())
	}
}

func TestUndefinedVariable(t *testing.T) {
	input := "undefinedVar"
	err := testEvalError(t, input)
	if err == nil {
		t.Fatal("expected error for undefined variable, got nil")
	}
}

func TestNumberLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"x = 5", 5},
		{"x = 0", 0},
		{"x = 42", 42},
		{"x = 3.14", 3.14},
		{"x = 99.99", 99.99},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		numVal, ok := result.(*NumberValue)
		if !ok {
			t.Errorf("expected NumberValue, got %T", result)
			continue
		}
		if numVal.Value != tt.expected {
			t.Errorf("for input %q: expected %f, got %f", tt.input, tt.expected, numVal.Value)
		}
	}
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`x = "hello"`, "hello"},
		{`x = ""`, ""},
		{`x = "Hello, world!"`, "Hello, world!"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		strVal, ok := result.(*StringValue)
		if !ok {
			t.Errorf("expected StringValue, got %T", result)
			continue
		}
		if strVal.Value != tt.expected {
			t.Errorf("for input %q: expected %q, got %q", tt.input, tt.expected, strVal.Value)
		}
	}
}

func TestBooleanLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"x = true", true},
		{"x = false", false},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		boolVal, ok := result.(*BoolValue)
		if !ok {
			t.Errorf("expected BoolValue, got %T", result)
			continue
		}
		if boolVal.Value != tt.expected {
			t.Errorf("for input %q: expected %v, got %v", tt.input, tt.expected, boolVal.Value)
		}
	}
}

func TestNullLiterals(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"x = null"},
		{"null"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		nullVal, ok := result.(*NullValue)
		if !ok {
			t.Errorf("expected NullValue, got %T", result)
			continue
		}
		if nullVal.String() != "null" {
			t.Errorf("for input %q: expected 'null', got %q", tt.input, nullVal.String())
		}
		if nullVal.Type() != ValueTypeNull {
			t.Errorf("for input %q: expected ValueTypeNull, got %v", tt.input, nullVal.Type())
		}
		if nullVal.IsTruthy() {
			t.Errorf("for input %q: null should be falsy", tt.input)
		}
	}
}

func TestArithmeticOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"x = 5 + 3", 8},
		{"x = 10 - 4", 6},
		{"x = 6 * 7", 42},
		{"x = 20 / 4", 5},
		{"x = 10 % 3", 1},
		{"x = 2 + 3 * 4", 14},   // precedence: 3*4 first, then +2
		{"x = (2 + 3) * 4", 20}, // parentheses override precedence
		{"x = 10 / 2 + 3", 8},   // 10/2=5, then 5+3=8
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		numVal, ok := result.(*NumberValue)
		if !ok {
			t.Errorf("expected NumberValue, got %T", result)
			continue
		}
		if numVal.Value != tt.expected {
			t.Errorf("for input %q: expected %f, got %f", tt.input, tt.expected, numVal.Value)
		}
	}
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"x = 5 < 10", true},
		{"x = 10 < 5", false},
		{"x = 5 <= 5", true},
		{"x = 5 <= 4", false},
		{"x = 10 > 5", true},
		{"x = 5 > 10", false},
		{"x = 5 >= 5", true},
		{"x = 4 >= 5", false},
		{"x = 5 == 5", true},
		{"x = 5 == 6", false},
		{"x = 5 != 6", true},
		{"x = 5 != 5", false},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		boolVal, ok := result.(*BoolValue)
		if !ok {
			t.Errorf("expected BoolValue, got %T", result)
			continue
		}
		if boolVal.Value != tt.expected {
			t.Errorf("for input %q: expected %v, got %v", tt.input, tt.expected, boolVal.Value)
		}
	}
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"x = true && true", true},
		{"x = true && false", false},
		{"x = false && true", false},
		{"x = false && false", false},
		{"x = true || true", true},
		{"x = true || false", true},
		{"x = false || true", true},
		{"x = false || false", false},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		boolVal, ok := result.(*BoolValue)
		if !ok {
			t.Errorf("expected BoolValue, got %T", result)
			continue
		}
		if boolVal.Value != tt.expected {
			t.Errorf("for input %q: expected %v, got %v", tt.input, tt.expected, boolVal.Value)
		}
	}
}

func TestUnaryOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x = -5", "-5"},
		{"x = !true", "false"},
		{"x = !false", "true"},
		{"x = -(-5)", "5"},
		{"x = !!true", "true"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		if result.String() != tt.expected {
			t.Errorf("for input %q: expected %q, got %q", tt.input, tt.expected, result.String())
		}
	}
}

func TestStringConcatenation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`x = "Hello" + " " + "World"`, "Hello World"},
		{`x = "Value: " + 42`, "Value: 42"},
		{`x = 10 + " items"`, "10 items"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		if result.String() != tt.expected {
			t.Errorf("for input %q: expected %q, got %q", tt.input, tt.expected, result.String())
		}
	}
}

func TestArrayLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`x = [1, 2, 3]`, `[1, 2, 3]`},
		{`x = ["a", "b", "c"]`, `["a", "b", "c"]`},
		{`x = []`, `[]`},
		{`x = [1, "two", true]`, `[1, "two", true]`},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		if result.String() != tt.expected {
			t.Errorf("for input %q: expected %q, got %q", tt.input, tt.expected, result.String())
		}
	}
}

func TestObjectLiterals(t *testing.T) {
	input := `x = {name: "Alice", age: 30}`
	result := testEval(t, input)

	objVal, ok := result.(*ObjectValue)
	if !ok {
		t.Fatalf("expected ObjectValue, got %T", result)
	}

	// Check name property
	nameVal := objVal.GetPropertyValue("name")
	if nameVal.String() != "Alice" {
		t.Errorf("expected name to be 'Alice', got %q", nameVal.String())
	}

	// Check age property
	ageVal := objVal.GetPropertyValue("age")
	if ageVal.String() != "30" {
		t.Errorf("expected age to be '30', got %q", ageVal.String())
	}
}

func TestComplexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`x = 2 + 3 * 4`, "14"},
		{`x = (2 + 3) * 4`, "20"},
		{`x = 10 - 5 - 2`, "3"},
		{`x = 10 / 2 / 5`, "1"},
		{`x = 5 > 3 && 10 < 20`, "true"},
		{`x = 5 < 3 || 10 < 20`, "true"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		if result.String() != tt.expected {
			t.Errorf("for input %q: expected %q, got %q", tt.input, tt.expected, result.String())
		}
	}
}

func TestMultipleStatements(t *testing.T) {
	input := `
a = 5
b = 10
c = a + b
c
`
	result := testEval(t, input)
	if result.String() != "15" {
		t.Errorf("expected 15, got %s", result.String())
	}
}

func TestDivisionByZero(t *testing.T) {
	input := "x = 10 / 0"
	err := testEvalError(t, input)
	if err == nil {
		t.Fatal("expected error for division by zero, got nil")
	}
}

func TestModuloByZero(t *testing.T) {
	input := "x = 10 % 0"
	err := testEvalError(t, input)
	if err == nil {
		t.Fatal("expected error for modulo by zero, got nil")
	}
}

func TestTypeErrors(t *testing.T) {
	tests := []string{
		`x = -"hello"`, // unary minus on string
	}

	for _, input := range tests {
		err := testEvalError(t, input)
		if err == nil {
			t.Errorf("expected error for input %q, got nil", input)
		}
	}
}

func TestVariableScoping(t *testing.T) {
	input := `
outer = 10
inner = 20
result = outer + inner
result
`
	result := testEval(t, input)
	if result.String() != "30" {
		t.Errorf("expected 30, got %s", result.String())
	}
}

func TestExpressionStatements(t *testing.T) {
	// Expression statements should evaluate to their value
	input := `
x = 5
x + 10
`
	result := testEval(t, input)
	if result.String() != "15" {
		t.Errorf("expected 15, got %s", result.String())
	}
}

func TestEqualityWithDifferentTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`x = 5 == "5"`, false},
		{`x = true == 1`, false},
		{`x = "hello" == "hello"`, true},
		{`x = null == null`, true},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		boolVal, ok := result.(*BoolValue)
		if !ok {
			t.Errorf("expected BoolValue, got %T", result)
			continue
		}
		if boolVal.Value != tt.expected {
			t.Errorf("for input %q: expected %v, got %v", tt.input, tt.expected, boolVal.Value)
		}
	}
}

func TestNestedArraysAndObjects(t *testing.T) {
	input := `x = {items: [1, 2, 3], nested: {value: 42}}`
	result := testEval(t, input)

	objVal, ok := result.(*ObjectValue)
	if !ok {
		t.Fatalf("expected ObjectValue, got %T", result)
	}

	// Check items array
	itemsVal := objVal.GetPropertyValue("items")
	if itemsVal.Type() != ValueTypeArray {
		t.Errorf("expected items to be array, got %s", itemsVal.Type())
	}

	// Check nested object
	nestedVal := objVal.GetPropertyValue("nested")
	if nestedVal.Type() != ValueTypeObject {
		t.Errorf("expected nested to be object, got %s", nestedVal.Type())
	}
}

func TestEvalResult_FinalResult(t *testing.T) {
	input := `
x = 5
y = 10
x + y
`
	result := testEvalFull(t, input)

	// Check that FinalResult contains the last expression value
	if result.FinalResult.String() != "15" {
		t.Errorf("expected FinalResult to be 15, got %s", result.FinalResult.String())
	}

	// Check that Value() method works
	if result.Value().String() != "15" {
		t.Errorf("expected Value() to return 15, got %s", result.Value().String())
	}
}

func TestEvalResult_Variables(t *testing.T) {
	input := `
x = 5
y = 10
name = "Alice"
result = x + y
`
	evalResult := testEvalFull(t, input)

	vars := evalResult.Variables()

	// Check that all variables are present
	if len(vars) != 4 {
		t.Errorf("expected 4 variables, got %d", len(vars))
	}

	// Check x
	if xVal, exists := vars["x"]; !exists {
		t.Error("expected variable 'x' to exist")
	} else if xVal.String() != "5" {
		t.Errorf("expected x=5, got %s", xVal.String())
	}

	// Check y
	if yVal, exists := vars["y"]; !exists {
		t.Error("expected variable 'y' to exist")
	} else if yVal.String() != "10" {
		t.Errorf("expected y=10, got %s", yVal.String())
	}

	// Check name
	if nameVal, exists := vars["name"]; !exists {
		t.Error("expected variable 'name' to exist")
	} else if nameVal.String() != "Alice" {
		t.Errorf("expected name='Alice', got %s", nameVal.String())
	}

	// Check result
	if resultVal, exists := vars["result"]; !exists {
		t.Error("expected variable 'result' to exist")
	} else if resultVal.String() != "15" {
		t.Errorf("expected result=15, got %s", resultVal.String())
	}
}

func TestEvalResult_FinalResultWithoutAssignment(t *testing.T) {
	input := `
x = 5
x * 2
`
	result := testEvalFull(t, input)

	// The final result should be the expression value (10)
	if result.FinalResult.String() != "10" {
		t.Errorf("expected FinalResult to be 10, got %s", result.FinalResult.String())
	}

	// But x should still be in variables
	vars := result.Variables()
	if xVal, exists := vars["x"]; !exists {
		t.Error("expected variable 'x' to exist")
	} else if xVal.String() != "5" {
		t.Errorf("expected x=5, got %s", xVal.String())
	}
}

func TestEvalResult_EmptyProgram(t *testing.T) {
	input := ``
	result := testEvalFull(t, input)

	// Empty program should return null
	if result.FinalResult.Type() != ValueTypeNull {
		t.Errorf("expected FinalResult to be null, got %s", result.FinalResult.Type())
	}

	// No variables should exist
	vars := result.Variables()
	if len(vars) != 0 {
		t.Errorf("expected 0 variables, got %d", len(vars))
	}
}

func TestNewWithEnvironment(t *testing.T) {
	// Create a pre-populated environment
	env := NewEnvironment()
	env.Set("preexisting", &NumberValue{Value: 42})

	// Create interpreter with this environment
	interp := New(&Options{Env: env})

	l := lexer.New(`result = preexisting + 8`)
	p := parser.New(l)
	program := p.ParseProgram()

	evalResult, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	// Check that it used the preexisting variable
	if evalResult.FinalResult.String() != "50" {
		t.Errorf("expected result to be 50, got %s", evalResult.FinalResult.String())
	}

	// Check variables
	vars := evalResult.Variables()
	if len(vars) != 2 {
		t.Errorf("expected 2 variables, got %d", len(vars))
	}
}

func TestEvalResult_VariablesImmutability(t *testing.T) {
	input := `x = 5`
	result := testEvalFull(t, input)

	// Get variables
	vars1 := result.Variables()
	vars1["x"] = &NumberValue{Value: 999} // Modify the returned map

	// Get variables again
	vars2 := result.Variables()

	// The original should be unchanged (we return a copy)
	if vars2["x"].String() != "5" {
		t.Errorf("expected x to still be 5, got %s", vars2["x"].String())
	}
}

func TestInterpreter_Context_DefaultIsBackground(t *testing.T) {
	interp := New(nil)
	ctx := interp.Context()

	if ctx == nil {
		t.Fatal("Context() returned nil, expected non-nil context")
	}

	// Verify the context is not cancelled
	select {
	case <-ctx.Done():
		t.Error("default context should not be cancelled")
	default:
		// Expected - context is not cancelled
	}
}

func TestInterpreter_SetContext_And_Context(t *testing.T) {
	interp := New(nil)

	// Create a cancellable context
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set the context
	interp.SetContext(cancelCtx)

	// Verify we get the same context back
	if interp.Context() != cancelCtx {
		t.Error("Context() did not return the context set via SetContext()")
	}
}

func TestInterpreter_Context_CancellationIsDetectable(t *testing.T) {
	interp := New(nil)

	// Create and set a cancellable context
	cancelCtx, cancel := context.WithCancel(context.Background())
	interp.SetContext(cancelCtx)

	// Cancel the context
	cancel()

	// Verify the cancellation is detectable
	select {
	case <-interp.Context().Done():
		// Expected - context is cancelled
	default:
		t.Error("cancelled context should be done")
	}

	// Verify the error is context.Canceled
	if interp.Context().Err() != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", interp.Context().Err())
	}
}

func TestInterpreter_SetContext_NilReturnsBackground(t *testing.T) {
	interp := New(nil)

	// First set a cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel it
	interp.SetContext(cancelCtx)

	// Verify it's cancelled
	if interp.Context().Err() == nil {
		t.Error("context should be cancelled")
	}

	// Now set nil
	interp.SetContext(nil)

	// Verify we get a non-cancelled context back
	ctx := interp.Context()
	if ctx == nil {
		t.Fatal("Context() returned nil after SetContext(nil)")
	}

	select {
	case <-ctx.Done():
		t.Error("context after SetContext(nil) should not be cancelled")
	default:
		// Expected - context is not cancelled
	}
}

func TestInterpreter_Context_ThreadSafety(t *testing.T) {
	interp := New(nil)

	// Create multiple contexts
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel1()
	defer cancel2()

	// Concurrently set and get contexts
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			interp.SetContext(ctx1)
			interp.Context()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			interp.SetContext(ctx2)
			interp.Context()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we get here without a race condition, the test passes
}

// TestConcurrentEmitEvent verifies that concurrent EmitEvent calls from multiple
// goroutines do not cause data races or scope corruption. This is the core
// correctness property of the env-as-parameter refactor: each EmitEvent call
// creates its own isolated scope chain from globalEnv, so concurrent calls
// cannot interfere with each other.
func TestConcurrentEmitEvent(t *testing.T) {
	interp := New(nil)

	// Register an event handler that reads from ctx, creates a local variable,
	// and returns an object with the local variable value. If scope isolation
	// is broken, goroutines would see each other's local variables.
	handlerScript := `
		tool handler(ctx, next) {
			localVar = ctx.id
			# Do some work to increase the chance of interleaving
			i = 0
			while (i < 10) {
				i = i + 1
			}
			# Verify localVar still has the same value (not corrupted by another goroutine)
			return { id: localVar }
		}
	`
	l := lexer.New(handlerScript)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	interp.Eval(prog)
	handler, _ := interp.globalEnv.Get("handler")
	interp.eventManager.Use("test.concurrent", handler.(*ToolValue))

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan string, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			expectedID := fmt.Sprintf("goroutine-%d", id)
			ctx := &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"id": {Value: &StringValue{Value: expectedID}},
				},
			}

			result := interp.EmitEvent("test.concurrent", ctx)
			if result == nil {
				errors <- fmt.Sprintf("goroutine %d: EmitEvent returned nil", id)
				return
			}

			obj, ok := result.(*ObjectValue)
			if !ok {
				errors <- fmt.Sprintf("goroutine %d: expected ObjectValue, got %T", id, result)
				return
			}

			idVal := obj.GetPropertyValue("id")
			if idVal == nil {
				errors <- fmt.Sprintf("goroutine %d: id property is nil", id)
				return
			}

			if idVal.String() != expectedID {
				errors <- fmt.Sprintf("goroutine %d: expected id=%q, got %q (scope corruption!)", id, expectedID, idVal.String())
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	for errMsg := range errors {
		t.Error(errMsg)
	}
}

// TestConcurrentEmitEventWithEval verifies that EmitEvent from a goroutine
// does not interfere with EmitEvent on the main goroutine. This simulates
// the real-world scenario where the prediction goroutine calls EmitEvent
// concurrently with the REPL thread also emitting events (e.g., during
// agent execution).
//
// Note: concurrent Eval + EmitEvent still has a map race on globalEnv.store
// because Environment is not synchronized. That is a separate concern from
// the i.env field race that this refactor eliminates. In production, Eval
// and EmitEvent from different goroutines don't happen simultaneously — the
// prediction goroutine only fires EmitEvent, not Eval.
func TestConcurrentEmitEventWithEval(t *testing.T) {
	interp := New(nil)

	// Register event handlers for two different events
	handlerScript := `
		tool predictHandler(ctx, next) {
			localVar = ctx.value
			i = 0
			while (i < 5) {
				i = i + 1
			}
			return { result: localVar }
		}

		tool mainHandler(ctx, next) {
			localVar = ctx.value
			i = 0
			while (i < 5) {
				i = i + 1
			}
			return { result: localVar }
		}
	`
	l := lexer.New(handlerScript)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	interp.Eval(prog)

	predictHandler, _ := interp.globalEnv.Get("predictHandler")
	interp.eventManager.Use("test.predict", predictHandler.(*ToolValue))

	mainHandler, _ := interp.globalEnv.Get("mainHandler")
	interp.eventManager.Use("test.main", mainHandler.(*ToolValue))

	var wg sync.WaitGroup
	const iterations = 20

	// Goroutine 1: repeatedly call EmitEvent (simulates prediction goroutine)
	predictErrors := make(chan string, iterations)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			expected := fmt.Sprintf("predict-%d", i)
			ctx := &ObjectValue{
				Properties: map[string]*PropertyDescriptor{
					"value": {Value: &StringValue{Value: expected}},
				},
			}
			result := interp.EmitEvent("test.predict", ctx)
			if result == nil {
				predictErrors <- fmt.Sprintf("predict iteration %d: nil result", i)
				continue
			}
			obj, ok := result.(*ObjectValue)
			if !ok {
				predictErrors <- fmt.Sprintf("predict iteration %d: expected ObjectValue, got %T", i, result)
				continue
			}
			resultVal := obj.GetPropertyValue("result")
			if resultVal == nil || resultVal.String() != expected {
				predictErrors <- fmt.Sprintf("predict iteration %d: expected %q, got %v", i, expected, resultVal)
			}
		}
	}()

	// Main goroutine: repeatedly call EmitEvent (simulates REPL thread emitting events)
	for i := 0; i < iterations; i++ {
		expected := fmt.Sprintf("main-%d", i)
		ctx := &ObjectValue{
			Properties: map[string]*PropertyDescriptor{
				"value": {Value: &StringValue{Value: expected}},
			},
		}
		result := interp.EmitEvent("test.main", ctx)
		if result == nil {
			t.Errorf("main iteration %d: nil result", i)
			continue
		}
		obj, ok := result.(*ObjectValue)
		if !ok {
			t.Errorf("main iteration %d: expected ObjectValue, got %T", i, result)
			continue
		}
		resultVal := obj.GetPropertyValue("result")
		if resultVal == nil || resultVal.String() != expected {
			t.Errorf("main iteration %d: expected %q, got %v", i, expected, resultVal)
		}
	}

	wg.Wait()
	close(predictErrors)

	for errMsg := range predictErrors {
		t.Error(errMsg)
	}
}
