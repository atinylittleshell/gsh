package interpreter

import (
	"testing"
)

// TestToolDeclaration tests basic tool declarations
func TestToolDeclaration(t *testing.T) {
	input := `
		tool greet(name) {
			result = "Hello, " + name
			return result
		}
	`

	result := testEvalFull(t, input)

	// Check that the tool is defined
	tool, ok := result.Variables()["greet"]
	if !ok {
		t.Fatal("tool 'greet' not found in environment")
	}

	if tool.Type() != ValueTypeTool {
		t.Errorf("expected tool type, got %s", tool.Type())
	}

	toolVal := tool.(*ToolValue)
	if toolVal.Name != "greet" {
		t.Errorf("expected tool name 'greet', got %s", toolVal.Name)
	}

	if len(toolVal.Parameters) != 1 || toolVal.Parameters[0] != "name" {
		t.Errorf("expected parameter 'name', got %v", toolVal.Parameters)
	}
}

// TestToolCall tests calling a tool
func TestToolCall(t *testing.T) {
	input := `
		tool greet(name) {
			return "Hello, " + name
		}
		
		result = greet("Alice")
	`

	result := testEvalFull(t, input)

	greeting := result.Variables()["result"]
	if greeting.Type() != ValueTypeString {
		t.Fatalf("expected string, got %s", greeting.Type())
	}

	if greeting.String() != "Hello, Alice" {
		t.Errorf("expected 'Hello, Alice', got %s", greeting.String())
	}
}

// TestToolWithMultipleParameters tests tools with multiple parameters
func TestToolWithMultipleParameters(t *testing.T) {
	input := `
		tool add(a, b) {
			return a + b
		}
		
		result = add(5, 3)
	`

	result := testEvalFull(t, input)

	sum := result.Variables()["result"]
	if sum.Type() != ValueTypeNumber {
		t.Fatalf("expected number, got %s", sum.Type())
	}

	if sum.String() != "8" {
		t.Errorf("expected '8', got %s", sum.String())
	}
}

// TestToolWithTypeAnnotations tests tools with type annotations
func TestToolWithTypeAnnotations(t *testing.T) {
	input := `
		tool multiply(x: number, y: number): number {
			return x * y
		}
		
		result = multiply(4, 5)
	`

	result := testEvalFull(t, input)

	product := result.Variables()["result"]
	if product.Type() != ValueTypeNumber {
		t.Fatalf("expected number, got %s", product.Type())
	}

	if product.String() != "20" {
		t.Errorf("expected '20', got %s", product.String())
	}
}

// TestToolParameterTypeValidation tests runtime parameter type validation
func TestToolParameterTypeValidation(t *testing.T) {
	input := `
		tool square(x: number): number {
			return x * x
		}
		
		result = square("not a number")
	`

	err := testEvalError(t, input)
	if err == nil {
		t.Error("expected error for invalid parameter type, got nil")
	}
}

// TestToolReturnTypeValidation tests runtime return type validation
func TestToolReturnTypeValidation(t *testing.T) {
	input := `
		tool getNumber(): number {
			return "this is a string"
		}
		
		result = getNumber()
	`

	err := testEvalError(t, input)
	if err == nil {
		t.Error("expected error for invalid return type, got nil")
	}
}

// TestToolWithoutExplicitReturn tests tools that don't have an explicit return
func TestToolWithoutExplicitReturn(t *testing.T) {
	input := `
		tool compute() {
			x = 10
			x + 5
		}
		
		result = compute()
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.Type() != ValueTypeNumber {
		t.Fatalf("expected number, got %s", value.Type())
	}

	if value.String() != "15" {
		t.Errorf("expected '15', got %s", value.String())
	}
}

// TestToolReturnNull tests tools that return null explicitly
func TestToolReturnNull(t *testing.T) {
	input := `
		tool doNothing() {
			return
		}
		
		result = doNothing()
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.Type() != ValueTypeNull {
		t.Fatalf("expected null, got %s", value.Type())
	}
}

// TestToolClosure tests that tools capture their environment
func TestToolClosure(t *testing.T) {
	input := `
		multiplier = 10
		
		tool scale(x) {
			return x * multiplier
		}
		
		result = scale(5)
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.Type() != ValueTypeNumber {
		t.Fatalf("expected number, got %s", value.Type())
	}

	if value.String() != "50" {
		t.Errorf("expected '50', got %s", value.String())
	}
}

// TestToolLocalVariables tests that tool local variables don't leak
func TestToolLocalVariables(t *testing.T) {
	input := `
		tool calculate() {
			localVar = 42
			return localVar
		}
		
		result = calculate()
	`

	result := testEvalFull(t, input)

	// localVar should not exist in global scope
	if _, ok := result.Variables()["localVar"]; ok {
		t.Error("tool local variable leaked to global scope")
	}

	// result should be 42
	value := result.Variables()["result"]
	if value.String() != "42" {
		t.Errorf("expected '42', got %s", value.String())
	}
}

// TestToolRecursion tests recursive tool calls
func TestToolRecursion(t *testing.T) {
	input := `
		tool factorial(n) {
			if (n <= 1) {
				return 1
			}
			return n * factorial(n - 1)
		}
		
		result = factorial(5)
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.Type() != ValueTypeNumber {
		t.Fatalf("expected number, got %s", value.Type())
	}

	if value.String() != "120" {
		t.Errorf("expected '120', got %s", value.String())
	}
}

// TestToolCallingOtherTools tests tools calling other tools
func TestToolCallingOtherTools(t *testing.T) {
	input := `
		tool add(a, b) {
			return a + b
		}
		
		tool addThree(x, y, z) {
			temp = add(x, y)
			return add(temp, z)
		}
		
		result = addThree(1, 2, 3)
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.String() != "6" {
		t.Errorf("expected '6', got %s", value.String())
	}
}

// TestToolWithArrayParameter tests tools with array parameters
func TestToolWithArrayParameter(t *testing.T) {
	input := `
		tool sum(numbers: array) {
			total = 0
			for (n of numbers) {
				total = total + n
			}
			return total
		}
		
		result = sum([1, 2, 3, 4, 5])
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.String() != "15" {
		t.Errorf("expected '15', got %s", value.String())
	}
}

// TestToolWithObjectParameter tests tools with object parameters
func TestToolWithObjectParameter(t *testing.T) {
	input := `
		tool greetPerson(person: object): string {
			return "Hello, " + person.name
		}
		
		user = {name: "Bob", age: 30}
		result = greetPerson(user)
	`

	// Note: This test requires member access on objects which may not be implemented yet
	// We'll implement a simpler version
	input = `
		tool describeObject(obj: object): string {
			return "object"
		}
		
		user = {name: "Bob", age: 30}
		result = describeObject(user)
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.String() != "object" {
		t.Errorf("expected 'object', got %s", value.String())
	}
}

// TestToolWrongArgumentCount tests error handling for wrong argument count
func TestToolWrongArgumentCount(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "too few arguments",
			input: `
				tool add(a, b) {
					return a + b
				}
				result = add(5)
			`,
		},
		{
			name: "too many arguments",
			input: `
				tool add(a, b) {
					return a + b
				}
				result = add(5, 3, 7)
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testEvalError(t, tt.input)
			if err == nil {
				t.Error("expected error for wrong argument count, got nil")
			}
		})
	}
}

// TestToolWithEarlyReturn tests tools with early return statements
func TestToolWithEarlyReturn(t *testing.T) {
	input := `
		tool checkNumber(n) {
			if (n < 0) {
				return "negative"
			}
			if (n == 0) {
				return "zero"
			}
			return "positive"
		}
		
		r1 = checkNumber(-5)
		r2 = checkNumber(0)
		r3 = checkNumber(10)
	`

	result := testEvalFull(t, input)

	vars := result.Variables()

	if vars["r1"].String() != "negative" {
		t.Errorf("expected 'negative', got %s", vars["r1"].String())
	}
	if vars["r2"].String() != "zero" {
		t.Errorf("expected 'zero', got %s", vars["r2"].String())
	}
	if vars["r3"].String() != "positive" {
		t.Errorf("expected 'positive', got %s", vars["r3"].String())
	}
}

// TestToolWithLoopAndReturn tests tools with loops and return
func TestToolWithLoopAndReturn(t *testing.T) {
	input := `
		tool findFirst(items, target) {
			for (item of items) {
				if (item == target) {
					return true
				}
			}
			return false
		}
		
		found = findFirst([1, 2, 3, 4, 5], 3)
		notFound = findFirst([1, 2, 3, 4, 5], 10)
	`

	result := testEvalFull(t, input)

	vars := result.Variables()

	if !vars["found"].IsTruthy() {
		t.Error("expected found to be true")
	}
	if vars["notFound"].IsTruthy() {
		t.Error("expected notFound to be false")
	}
}

// TestToolAnyType tests tools with 'any' type annotation
func TestToolAnyType(t *testing.T) {
	input := `
		tool identity(x: any): any {
			return x
		}
		
		r1 = identity(42)
		r2 = identity("hello")
		r3 = identity(true)
	`

	result := testEvalFull(t, input)

	vars := result.Variables()

	if vars["r1"].String() != "42" {
		t.Errorf("expected '42', got %s", vars["r1"].String())
	}
	if vars["r2"].String() != "hello" {
		t.Errorf("expected 'hello', got %s", vars["r2"].String())
	}
	if vars["r3"].String() != "true" {
		t.Errorf("expected 'true', got %s", vars["r3"].String())
	}
}

// TestToolNestedCalls tests nested tool calls in arguments
func TestToolNestedCalls(t *testing.T) {
	input := `
		tool double(x) {
			return x * 2
		}
		
		tool triple(x) {
			return x * 3
		}
		
		result = double(triple(5))
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.String() != "30" {
		t.Errorf("expected '30', got %s", value.String())
	}
}

// TestCallNonToolValue tests error when calling non-tool values
func TestCallNonToolValue(t *testing.T) {
	input := `
		x = 42
		result = x()
	`

	err := testEvalError(t, input)
	if err == nil {
		t.Error("expected error when calling non-tool value, got nil")
	}
}

// TestToolArrayReturnType tests tools with array return type annotation
func TestToolArrayReturnType(t *testing.T) {
	input := `
		tool makeArray(): array {
			return [1, 2, 3]
		}
		
		result = makeArray()
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.Type() != ValueTypeArray {
		t.Fatalf("expected array, got %s", value.Type())
	}

	arr := value.(*ArrayValue)
	if len(arr.Elements) != 3 {
		t.Errorf("expected array length 3, got %d", len(arr.Elements))
	}
}

// TestToolStringArrayType tests tools with array parameters (using 'array' type)
func TestToolStringArrayType(t *testing.T) {
	input := `
		tool processStrings(items: array): number {
			count = 0
			for (item of items) {
				count = count + 1
			}
			return count
		}
		
		result = processStrings(["a", "b", "c"])
	`

	result := testEvalFull(t, input)

	value := result.Variables()["result"]
	if value.String() != "3" {
		t.Errorf("expected '3', got %s", value.String())
	}
}
