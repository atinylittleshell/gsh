package interpreter

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Helper function to parse and evaluate code
func parseAndEval(t *testing.T, input string) (*EvalResult, error) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	return interp.Eval(program)
}

func TestTryCatchBasic(t *testing.T) {
	input := `
x = 0
try {
	x = 1
	y = 1 / 0
	x = 2
} catch (error) {
	x = 10
}
`
	result, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// x should be 10 (set in catch block)
	xVal := result.Variables()["x"]
	if xVal.Type() != ValueTypeNumber {
		t.Errorf("x should be a number, got %s", xVal.Type())
	}

	numVal, ok := xVal.(*NumberValue)
	if !ok {
		t.Fatalf("x is not a NumberValue")
	}

	if numVal.Value != 10 {
		t.Errorf("x should be 10, got %v", numVal.Value)
	}
}

func TestTryCatchWithoutError(t *testing.T) {
	input := `
x = 0
try {
	x = 5
} catch (error) {
	x = 10
}
`
	result, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// x should be 5 (catch block not executed)
	xVal := result.Variables()["x"]
	numVal, ok := xVal.(*NumberValue)
	if !ok {
		t.Fatalf("x is not a NumberValue")
	}

	if numVal.Value != 5 {
		t.Errorf("x should be 5, got %v", numVal.Value)
	}
}

func TestTryCatchErrorObject(t *testing.T) {
	input := `
caughtError = false
try {
	x = undefinedVariable
} catch (error) {
	caughtError = true
}
`
	result, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// caughtError should be true
	caughtVal := result.Variables()["caughtError"]
	boolVal, ok := caughtVal.(*BoolValue)
	if !ok {
		t.Fatalf("caughtError is not a BoolValue, got %T", caughtVal)
	}

	if !boolVal.Value {
		t.Errorf("caughtError should be true")
	}
}

func TestTryFinally(t *testing.T) {
	input := `
x = 0
y = 0
try {
	x = 5
} finally {
	y = 10
}
`
	result, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// x should be 5, y should be 10
	xVal := result.Variables()["x"]
	xNum, ok := xVal.(*NumberValue)
	if !ok || xNum.Value != 5 {
		t.Errorf("x should be 5, got %v", xVal)
	}

	yVal := result.Variables()["y"]
	yNum, ok := yVal.(*NumberValue)
	if !ok || yNum.Value != 10 {
		t.Errorf("y should be 10, got %v", yVal)
	}
}

func TestTryFinallyWithError(t *testing.T) {
	input := `
x = 0
y = 0
try {
	x = 5
	z = undefinedVariable
} finally {
	y = 10
}
`
	_, err := parseAndEval(t, input)

	// Should have an error (not caught)
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	// The finally block should still have executed, but we can't check y
	// since the error propagates and we don't have the final environment
}

func TestTryCatchFinally(t *testing.T) {
	input := `
x = 0
y = 0
z = 0
try {
	x = 5
	w = undefinedVariable
} catch (error) {
	y = 20
} finally {
	z = 30
}
`
	result, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// x should be 5, y should be 20 (catch), z should be 30 (finally)
	xVal := result.Variables()["x"]
	xNum, ok := xVal.(*NumberValue)
	if !ok || xNum.Value != 5 {
		t.Errorf("x should be 5, got %v", xVal)
	}

	yVal := result.Variables()["y"]
	yNum, ok := yVal.(*NumberValue)
	if !ok || yNum.Value != 20 {
		t.Errorf("y should be 20, got %v", yVal)
	}

	zVal := result.Variables()["z"]
	zNum, ok := zVal.(*NumberValue)
	if !ok || zNum.Value != 30 {
		t.Errorf("z should be 30, got %v", zVal)
	}
}

func TestTryCatchNested(t *testing.T) {
	input := `
result = ""
try {
	try {
		x = undefinedVariable
		result = "no error"
	} catch (innerError) {
		result = "inner"
	}
} catch (outerError) {
	result = "outer"
}
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// result should be "inner" (inner catch handles it)
	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue")
	}

	if strVal.Value != "inner" {
		t.Errorf("result should be 'inner', got %s", strVal.Value)
	}
}

func TestTryCatchPropagation(t *testing.T) {
	input := `
result = ""
try {
	try {
		x = undefinedVariable
	} finally {
		result = "finally ran"
	}
} catch (outerError) {
	result = result + " caught"
}
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// result should be "finally ran caught"
	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue")
	}

	if strVal.Value != "finally ran caught" {
		t.Errorf("result should be 'finally ran caught', got %s", strVal.Value)
	}
}

func TestTryCatchWithToolCall(t *testing.T) {
	input := `
tool dangerous() {
	x = undefinedVariable
	return x
}

result = 0
try {
	result = dangerous()
} catch (error) {
	result = 99
}
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// result should be 99 (error was caught)
	resultVal := res.Variables()["result"]
	numVal, ok := resultVal.(*NumberValue)
	if !ok {
		t.Fatalf("result is not a NumberValue")
	}

	if numVal.Value != 99 {
		t.Errorf("result should be 99, got %v", numVal.Value)
	}
}

func TestTryCatchDoesNotCatchBreak(t *testing.T) {
	input := `
x = 0
for (i of [1, 2, 3]) {
	try {
		if (i == 2) {
			break
		}
		x = x + i
	} catch (error) {
		x = 999
	}
}
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// x should be 1 (loop breaks on i=2, catch doesn't intercept break)
	xVal := res.Variables()["x"]
	numVal, ok := xVal.(*NumberValue)
	if !ok {
		t.Fatalf("x is not a NumberValue")
	}

	if numVal.Value != 1 {
		t.Errorf("x should be 1, got %v", numVal.Value)
	}
}

func TestTryCatchDoesNotCatchContinue(t *testing.T) {
	input := `
x = 0
for (i of [1, 2, 3]) {
	try {
		if (i == 2) {
			continue
		}
		x = x + i
	} catch (error) {
		x = 999
	}
}
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// x should be 4 (1 + 3, skips 2, catch doesn't intercept continue)
	xVal := res.Variables()["x"]
	numVal, ok := xVal.(*NumberValue)
	if !ok {
		t.Fatalf("x is not a NumberValue")
	}

	if numVal.Value != 4 {
		t.Errorf("x should be 4, got %v", numVal.Value)
	}
}

func TestTryCatchDoesNotCatchReturn(t *testing.T) {
	input := `
tool testFunc() {
	try {
		return 42
	} catch (error) {
		return 999
	}
	return 0
}

result = testFunc()
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// result should be 42 (return is not caught)
	resultVal := res.Variables()["result"]
	numVal, ok := resultVal.(*NumberValue)
	if !ok {
		t.Fatalf("result is not a NumberValue")
	}

	if numVal.Value != 42 {
		t.Errorf("result should be 42, got %v", numVal.Value)
	}
}

func TestTryCatchScope(t *testing.T) {
	input := `
error = "outer"
innerError = "before"
try {
	x = undefinedVariable
} catch (error) {
	innerError = "caught"
}
outerError = error
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// outerError should still be "outer" (catch parameter doesn't leak)
	outerErrorVal := res.Variables()["outerError"]
	strVal, ok := outerErrorVal.(*StringValue)
	if !ok {
		t.Fatalf("outerError is not a StringValue")
	}

	if strVal.Value != "outer" {
		t.Errorf("outerError should be 'outer', got %s", strVal.Value)
	}

	// innerError should be "caught" (was updated from catch block)
	innerErrorVal := res.Variables()["innerError"]
	if innerErrorVal == nil {
		t.Fatalf("innerError is nil")
	}
	innerStr, ok := innerErrorVal.(*StringValue)
	if !ok {
		t.Fatalf("innerError is not a StringValue, got %T: %v", innerErrorVal, innerErrorVal)
	}

	if innerStr.Value != "caught" {
		t.Errorf("innerError should be 'caught', got %s", innerStr.Value)
	}
}

func TestFinallyOverridesError(t *testing.T) {
	input := `
x = 0
try {
	y = undefinedVariable
} catch (error) {
	x = 5
} finally {
	z = undefinedVariable2
}
`
	_, err := parseAndEval(t, input)

	// Should have an error from finally block
	if err == nil {
		t.Fatalf("expected error from finally block but got none")
	}

	// The error should be about undefinedVariable2
	if !strings.Contains(err.Error(), "undefinedVariable2") {
		t.Errorf("error should mention undefinedVariable2, got: %v", err)
	}
}

func TestTryCatchReturnValue(t *testing.T) {
	input := `
tool testFunc() {
	try {
		return 10
	} catch (error) {
		return 20
	}
}

result = testFunc()
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultVal := res.Variables()["result"]
	numVal, ok := resultVal.(*NumberValue)
	if !ok {
		t.Fatalf("result is not a NumberValue")
	}

	if numVal.Value != 10 {
		t.Errorf("result should be 10, got %v", numVal.Value)
	}
}

func TestTryCatchWithinLoop(t *testing.T) {
	input := `
count = 0
errors = 0
for (i of [1, 2, 3]) {
	try {
		if (i == 2) {
			x = undefinedVariable
		}
		count = count + 1
	} catch (error) {
		errors = errors + 1
	}
}
`
	res, err := parseAndEval(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// count should be 2 (iterations 1 and 3 succeeded)
	countVal := res.Variables()["count"]
	countNum, ok := countVal.(*NumberValue)
	if !ok {
		t.Fatalf("count is not a NumberValue")
	}

	if countNum.Value != 2 {
		t.Errorf("count should be 2, got %v", countNum.Value)
	}

	// errors should be 1 (iteration 2 failed)
	errorsVal := res.Variables()["errors"]
	errorsNum, ok := errorsVal.(*NumberValue)
	if !ok {
		t.Fatalf("errors is not a NumberValue")
	}

	if errorsNum.Value != 1 {
		t.Errorf("errors should be 1, got %v", errorsNum.Value)
	}
}
