package interpreter

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Helper function to parse and evaluate code for throw tests
func parseAndEvalThrow(t *testing.T, input string) (*EvalResult, error) {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	interp := New(nil)
	return interp.Eval(program)
}

func TestThrowStringCaughtByCatch(t *testing.T) {
	input := `
result = ""
try {
	throw "something went wrong"
	result = "not reached"
} catch (error) {
	result = error.message
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue, got %T", resultVal)
	}

	if strVal.Value != "something went wrong" {
		t.Errorf("result should be 'something went wrong', got %s", strVal.Value)
	}
}

func TestThrowObjectWithMessagePassedThrough(t *testing.T) {
	input := `
msg = ""
code = 0
try {
	throw {message: "not found", code: 404}
} catch (error) {
	msg = error.message
	code = error.code
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgVal := res.Variables()["msg"]
	strVal, ok := msgVal.(*StringValue)
	if !ok {
		t.Fatalf("msg is not a StringValue, got %T", msgVal)
	}
	if strVal.Value != "not found" {
		t.Errorf("msg should be 'not found', got %s", strVal.Value)
	}

	codeVal := res.Variables()["code"]
	numVal, ok := codeVal.(*NumberValue)
	if !ok {
		t.Fatalf("code is not a NumberValue, got %T", codeVal)
	}
	if numVal.Value != 404 {
		t.Errorf("code should be 404, got %v", numVal.Value)
	}
}

func TestThrowNonStringPrimitiveWrapped(t *testing.T) {
	input := `
result = ""
try {
	throw 42
} catch (error) {
	result = error.message
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue, got %T", resultVal)
	}

	if strVal.Value != "42" {
		t.Errorf("result should be '42', got %s", strVal.Value)
	}
}

func TestThrowInsideFunctionPropagatesToCallerCatch(t *testing.T) {
	input := `
tool failing() {
	throw "function error"
	return "not reached"
}

result = ""
try {
	failing()
} catch (error) {
	result = error.message
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue, got %T", resultVal)
	}

	if strVal.Value != "function error" {
		t.Errorf("result should be 'function error', got %s", strVal.Value)
	}
}

func TestThrowWithoutTryCatchIsUncaughtError(t *testing.T) {
	input := `
throw "uncaught error"
`
	_, err := parseAndEvalThrow(t, input)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	if !strings.Contains(err.Error(), "uncaught error") {
		t.Errorf("error should contain 'uncaught error', got: %v", err)
	}
}

func TestThrowInNestedTryCatch(t *testing.T) {
	input := `
result = ""
try {
	try {
		throw "inner error"
	} catch (innerErr) {
		result = "inner: " + innerErr.message
	}
} catch (outerErr) {
	result = "outer: " + outerErr.message
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue, got %T", resultVal)
	}

	if strVal.Value != "inner: inner error" {
		t.Errorf("result should be 'inner: inner error', got %s", strVal.Value)
	}
}

func TestThrowInsideCatchBlockRethrow(t *testing.T) {
	input := `
result = ""
try {
	try {
		throw "original error"
	} catch (e) {
		throw "rethrown: " + e.message
	}
} catch (outerErr) {
	result = outerErr.message
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue, got %T", resultVal)
	}

	if strVal.Value != "rethrown: original error" {
		t.Errorf("result should be 'rethrown: original error', got %s", strVal.Value)
	}
}

func TestThrowWithFinallyStillRuns(t *testing.T) {
	input := `
result = ""
finallyRan = false
try {
	throw "error with finally"
} catch (e) {
	result = e.message
} finally {
	finallyRan = true
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultVal := res.Variables()["result"]
	strVal, ok := resultVal.(*StringValue)
	if !ok {
		t.Fatalf("result is not a StringValue, got %T", resultVal)
	}
	if strVal.Value != "error with finally" {
		t.Errorf("result should be 'error with finally', got %s", strVal.Value)
	}

	finallyVal := res.Variables()["finallyRan"]
	boolVal, ok := finallyVal.(*BoolValue)
	if !ok {
		t.Fatalf("finallyRan is not a BoolValue, got %T", finallyVal)
	}
	if !boolVal.Value {
		t.Errorf("finallyRan should be true")
	}
}

func TestThrowInsideLoopDoesNotBehaveAsBreak(t *testing.T) {
	input := `
count = 0
errors = 0
for (i of [1, 2, 3]) {
	try {
		if (i == 2) {
			throw "error on 2"
		}
		count = count + 1
	} catch (e) {
		errors = errors + 1
	}
}
`
	res, err := parseAndEvalThrow(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// count should be 2 (iterations 1 and 3 succeeded)
	countVal := res.Variables()["count"]
	countNum, ok := countVal.(*NumberValue)
	if !ok {
		t.Fatalf("count is not a NumberValue, got %T", countVal)
	}
	if countNum.Value != 2 {
		t.Errorf("count should be 2, got %v", countNum.Value)
	}

	// errors should be 1 (iteration 2 threw)
	errorsVal := res.Variables()["errors"]
	errorsNum, ok := errorsVal.(*NumberValue)
	if !ok {
		t.Fatalf("errors is not a NumberValue, got %T", errorsVal)
	}
	if errorsNum.Value != 1 {
		t.Errorf("errors should be 1, got %v", errorsNum.Value)
	}
}
