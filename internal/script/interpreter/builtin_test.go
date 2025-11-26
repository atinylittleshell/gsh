package interpreter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// Helper function to capture stdout
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// Helper function to capture stderr
func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "print string",
			input:    `print("Hello, world!")`,
			expected: "Hello, world!\n",
		},
		{
			name:     "print number",
			input:    `print(42)`,
			expected: "42\n",
		},
		{
			name:     "print boolean",
			input:    `print(true)`,
			expected: "true\n",
		},
		{
			name:     "print multiple values",
			input:    `print("The answer is", 42)`,
			expected: "The answer is 42\n",
		},
		{
			name:     "print with variable",
			input:    "x = \"test\"\nprint(x)",
			expected: "test\n",
		},
		{
			name:     "print array",
			input:    `print([1, 2, 3])`,
			expected: "[1, 2, 3]\n",
		},
		{
			name:     "print object",
			input:    `print({name: "Alice", age: 30})`,
			expected: "{age: 30, name: \"Alice\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			interp := New()
			output := captureOutput(func() {
				_, err := interp.Eval(program)
				if err != nil {
					t.Fatalf("eval error: %v", err)
				}
			})

			// For objects, the order might vary, so we check if output contains expected parts
			if strings.Contains(tt.input, "{") && strings.Contains(tt.input, "}") {
				if !strings.Contains(output, "name: \"Alice\"") || !strings.Contains(output, "age: 30") {
					t.Errorf("output = %q, want output containing name and age", output)
				}
			} else if output != tt.expected {
				t.Errorf("output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestLogFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "log.debug",
			input:    `log.debug("Debug message")`,
			expected: "[DEBUG] Debug message\n",
		},
		{
			name:     "log.info",
			input:    `log.info("Info message")`,
			expected: "[INFO] Info message\n",
		},
		{
			name:     "log.warn",
			input:    `log.warn("Warning message")`,
			expected: "[WARN] Warning message\n",
		},
		{
			name:     "log.error",
			input:    `log.error("Error message")`,
			expected: "[ERROR] Error message\n",
		},
		{
			name:     "log with multiple values",
			input:    `log.info("Status:", 200, "OK")`,
			expected: "[INFO] Status: 200 OK\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			interp := New()
			output := captureStderr(func() {
				_, err := interp.Eval(program)
				if err != nil {
					t.Fatalf("eval error: %v", err)
				}
			})

			if output != tt.expected {
				t.Errorf("output = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestJSONParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		checkResult func(*testing.T, Value)
		expectError bool
	}{
		{
			name:  "parse simple object",
			input: `result = JSON.parse("{\"name\": \"Alice\", \"age\": 30}")`,
			checkResult: func(t *testing.T, result Value) {
				obj, ok := result.(*ObjectValue)
				if !ok {
					t.Fatalf("result is not ObjectValue, got %T", result)
				}
				if name, ok := obj.Properties["name"].(*StringValue); !ok || name.Value != "Alice" {
					t.Errorf("name = %v, want Alice", obj.Properties["name"])
				}
				if age, ok := obj.Properties["age"].(*NumberValue); !ok || age.Value != 30 {
					t.Errorf("age = %v, want 30", obj.Properties["age"])
				}
			},
		},
		{
			name:  "parse array",
			input: `result = JSON.parse("[1, 2, 3]")`,
			checkResult: func(t *testing.T, result Value) {
				arr, ok := result.(*ArrayValue)
				if !ok {
					t.Fatalf("result is not ArrayValue, got %T", result)
				}
				if len(arr.Elements) != 3 {
					t.Errorf("array length = %d, want 3", len(arr.Elements))
				}
			},
		},
		{
			name:  "parse nested structure",
			input: `result = JSON.parse("{\"user\": {\"name\": \"Bob\"}, \"items\": [1, 2]}")`,
			checkResult: func(t *testing.T, result Value) {
				obj, ok := result.(*ObjectValue)
				if !ok {
					t.Fatalf("result is not ObjectValue, got %T", result)
				}
				user, ok := obj.Properties["user"].(*ObjectValue)
				if !ok {
					t.Fatalf("user is not ObjectValue, got %T", obj.Properties["user"])
				}
				if name, ok := user.Properties["name"].(*StringValue); !ok || name.Value != "Bob" {
					t.Errorf("user.name = %v, want Bob", user.Properties["name"])
				}
			},
		},
		{
			name:        "parse invalid JSON",
			input:       `result = JSON.parse("invalid json")`,
			expectError: true,
		},
		{
			name:  "parse null",
			input: `result = JSON.parse("null")`,
			checkResult: func(t *testing.T, result Value) {
				if _, ok := result.(*NullValue); !ok {
					t.Fatalf("result is not NullValue, got %T", result)
				}
			},
		},
		{
			name:  "parse boolean",
			input: `result = JSON.parse("true")`,
			checkResult: func(t *testing.T, result Value) {
				if b, ok := result.(*BoolValue); !ok || !b.Value {
					t.Fatalf("result is not true boolean, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			interp := New()
			evalResult, err := interp.Eval(program)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			result, ok := evalResult.Variables()["result"]
			if !ok {
				t.Fatalf("result variable not found")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestJSONStringify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "stringify object",
			input:    `result = JSON.stringify({name: "Alice", age: 30})`,
			expected: `{"age":30,"name":"Alice"}`,
		},
		{
			name:     "stringify array",
			input:    `result = JSON.stringify([1, 2, 3])`,
			expected: `[1,2,3]`,
		},
		{
			name:     "stringify string",
			input:    `result = JSON.stringify("hello")`,
			expected: `"hello"`,
		},
		{
			name:     "stringify number",
			input:    `result = JSON.stringify(42)`,
			expected: `42`,
		},
		{
			name:     "stringify boolean",
			input:    `result = JSON.stringify(true)`,
			expected: `true`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			interp := New()
			evalResult, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			result, ok := evalResult.Variables()["result"]
			if !ok {
				t.Fatalf("result variable not found")
			}

			strVal, ok := result.(*StringValue)
			if !ok {
				t.Fatalf("result is not StringValue, got %T", result)
			}

			// Parse both to compare as JSON (order-independent for objects)
			var expectedJSON, actualJSON interface{}
			if err := json.Unmarshal([]byte(tt.expected), &expectedJSON); err != nil {
				t.Fatalf("failed to parse expected JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(strVal.Value), &actualJSON); err != nil {
				t.Fatalf("failed to parse actual JSON: %v", err)
			}

			expectedStr := fmt.Sprintf("%v", expectedJSON)
			actualStr := fmt.Sprintf("%v", actualJSON)
			if expectedStr != actualStr {
				t.Errorf("result = %q, want %q", strVal.Value, tt.expected)
			}
		})
	}
}

func TestEnvAccess(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("TEST_NUM", "123")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("TEST_NUM")

	tests := []struct {
		name        string
		input       string
		checkResult func(*testing.T, Value)
	}{
		{
			name:  "access existing env var",
			input: `result = env.TEST_VAR`,
			checkResult: func(t *testing.T, result Value) {
				strVal, ok := result.(*StringValue)
				if !ok {
					t.Fatalf("result is not StringValue, got %T", result)
				}
				if strVal.Value != "test_value" {
					t.Errorf("result = %q, want test_value", strVal.Value)
				}
			},
		},
		{
			name:  "access non-existent env var",
			input: `result = env.NON_EXISTENT`,
			checkResult: func(t *testing.T, result Value) {
				if _, ok := result.(*NullValue); !ok {
					t.Fatalf("result is not NullValue, got %T", result)
				}
			},
		},
		{
			name:  "use env var in string interpolation",
			input: `result = "Value: " + env.TEST_VAR`,
			checkResult: func(t *testing.T, result Value) {
				strVal, ok := result.(*StringValue)
				if !ok {
					t.Fatalf("result is not StringValue, got %T", result)
				}
				if strVal.Value != "Value: test_value" {
					t.Errorf("result = %q, want 'Value: test_value'", strVal.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			interp := New()
			evalResult, err := interp.Eval(program)
			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			result, ok := evalResult.Variables()["result"]
			if !ok {
				t.Fatalf("result variable not found")
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestBuiltinsRegistered(t *testing.T) {
	interp := New()

	// Check that print is registered
	printVal, ok := interp.env.Get("print")
	if !ok {
		t.Errorf("print function not registered")
	}
	if _, ok := printVal.(*BuiltinValue); !ok {
		t.Errorf("print is not a BuiltinValue, got %T", printVal)
	}

	// Check that JSON is registered
	jsonVal, ok := interp.env.Get("JSON")
	if !ok {
		t.Errorf("JSON object not registered")
	}
	jsonObj, ok := jsonVal.(*ObjectValue)
	if !ok {
		t.Errorf("JSON is not an ObjectValue, got %T", jsonVal)
	}
	if _, ok := jsonObj.Properties["parse"].(*BuiltinValue); !ok {
		t.Errorf("JSON.parse is not a BuiltinValue")
	}
	if _, ok := jsonObj.Properties["stringify"].(*BuiltinValue); !ok {
		t.Errorf("JSON.stringify is not a BuiltinValue")
	}

	// Check that log is registered
	logVal, ok := interp.env.Get("log")
	if !ok {
		t.Errorf("log object not registered")
	}
	logObj, ok := logVal.(*ObjectValue)
	if !ok {
		t.Errorf("log is not an ObjectValue, got %T", logVal)
	}
	for _, method := range []string{"debug", "info", "warn", "error"} {
		if _, ok := logObj.Properties[method].(*BuiltinValue); !ok {
			t.Errorf("log.%s is not a BuiltinValue", method)
		}
	}

	// Check that env is registered
	envVal, ok := interp.env.Get("env")
	if !ok {
		t.Errorf("env object not registered")
	}
	if _, ok := envVal.(*EnvValue); !ok {
		t.Errorf("env is not an EnvValue, got %T", envVal)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	input := `
		original = {name: "Alice", age: 30, items: [1, 2, 3]}
		jsonStr = JSON.stringify(original)
		parsed = JSON.parse(jsonStr)
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	interp := New()
	evalResult, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	vars := evalResult.Variables()
	original := vars["original"]
	parsed := vars["parsed"]

	// Check that they're equal
	if !original.Equals(parsed) {
		t.Errorf("round trip failed: original = %v, parsed = %v", original, parsed)
	}
}
