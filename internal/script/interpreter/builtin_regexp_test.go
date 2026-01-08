package interpreter

import (
	"regexp"
	"testing"
)

func TestRegexpMatch(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{
			name:     "simple match",
			code:     `Regexp.match("hello world", "world")`,
			expected: []string{"world"},
		},
		{
			name:     "match with capture groups",
			code:     `Regexp.match("hello world", "(\\w+) (\\w+)")`,
			expected: []string{"hello world", "hello", "world"},
		},
		{
			name:     "no match returns null",
			code:     `Regexp.match("hello", "xyz")`,
			expected: nil,
		},
		{
			name:     "match email pattern",
			code:     `Regexp.match("contact: test@example.com", "(\\w+)@(\\w+\\.\\w+)")`,
			expected: []string{"test@example.com", "test", "example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.EvalString(tt.code, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expected == nil {
				if _, ok := result.FinalResult.(*NullValue); !ok {
					t.Errorf("expected null, got %v", result.FinalResult)
				}
				return
			}

			arr, ok := result.FinalResult.(*ArrayValue)
			if !ok {
				t.Fatalf("expected array, got %s", result.FinalResult.Type())
			}

			expected := tt.expected.([]string)
			if len(arr.Elements) != len(expected) {
				t.Errorf("expected %d elements, got %d", len(expected), len(arr.Elements))
				return
			}

			for i, elem := range arr.Elements {
				str, ok := elem.(*StringValue)
				if !ok {
					t.Errorf("element %d: expected string, got %s", i, elem.Type())
					continue
				}
				if str.Value != expected[i] {
					t.Errorf("element %d: expected %q, got %q", i, expected[i], str.Value)
				}
			}
		})
	}
}

func TestRegexpTest(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "simple match",
			code:     `Regexp.test("hello world", "world")`,
			expected: true,
		},
		{
			name:     "no match",
			code:     `Regexp.test("hello", "xyz")`,
			expected: false,
		},
		{
			name:     "pattern at start",
			code:     `Regexp.test("hello world", "^hello")`,
			expected: true,
		},
		{
			name:     "pattern at end",
			code:     `Regexp.test("hello world", "world$")`,
			expected: true,
		},
		{
			name:     "digit pattern",
			code:     `Regexp.test("abc123def", "\\d+")`,
			expected: true,
		},
		{
			name:     "no digit",
			code:     `Regexp.test("abcdef", "\\d+")`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.EvalString(tt.code, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			boolVal, ok := result.FinalResult.(*BoolValue)
			if !ok {
				t.Fatalf("expected bool, got %s", result.FinalResult.Type())
			}

			if boolVal.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolVal.Value)
			}
		})
	}
}

func TestRegexpReplace(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple replace",
			code:     `Regexp.replace("hello world", "world", "gsh")`,
			expected: "hello gsh",
		},
		{
			name:     "replace only first",
			code:     `Regexp.replace("hello hello hello", "hello", "hi")`,
			expected: "hi hello hello",
		},
		{
			name:     "replace with capture group",
			code:     `Regexp.replace("hello world", "(\\w+) (\\w+)", "$2 $1")`,
			expected: "world hello",
		},
		{
			name:     "no match",
			code:     `Regexp.replace("hello", "xyz", "abc")`,
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.EvalString(tt.code, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			strVal, ok := result.FinalResult.(*StringValue)
			if !ok {
				t.Fatalf("expected string, got %s", result.FinalResult.Type())
			}

			if strVal.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, strVal.Value)
			}
		})
	}
}

func TestRegexpReplaceAll(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "replace all occurrences",
			code:     `Regexp.replaceAll("hello hello hello", "hello", "hi")`,
			expected: "hi hi hi",
		},
		{
			name:     "replace digits",
			code:     `Regexp.replaceAll("a1b2c3", "\\d", "X")`,
			expected: "aXbXcX",
		},
		{
			name:     "replace with capture group",
			code:     `Regexp.replaceAll("foo bar baz", "(\\w+)", "[$1]")`,
			expected: "[foo] [bar] [baz]",
		},
		{
			name:     "no match",
			code:     `Regexp.replaceAll("hello", "xyz", "abc")`,
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.EvalString(tt.code, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			strVal, ok := result.FinalResult.(*StringValue)
			if !ok {
				t.Fatalf("expected string, got %s", result.FinalResult.Type())
			}

			if strVal.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, strVal.Value)
			}
		})
	}
}

func TestRegexpSplit(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "split by whitespace",
			code:     `Regexp.split("hello world gsh", "\\s+")`,
			expected: []string{"hello", "world", "gsh"},
		},
		{
			name:     "split by comma and space",
			code:     `Regexp.split("a, b, c", ",\\s*")`,
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "split with limit",
			code:     `Regexp.split("a-b-c-d", "-", 2)`,
			expected: []string{"a", "b-c-d"},
		},
		{
			name:     "no match",
			code:     `Regexp.split("hello", "xyz")`,
			expected: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.EvalString(tt.code, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := result.FinalResult.(*ArrayValue)
			if !ok {
				t.Fatalf("expected array, got %s", result.FinalResult.Type())
			}

			if len(arr.Elements) != len(tt.expected) {
				t.Errorf("expected %d elements, got %d", len(tt.expected), len(arr.Elements))
				return
			}

			for i, elem := range arr.Elements {
				str, ok := elem.(*StringValue)
				if !ok {
					t.Errorf("element %d: expected string, got %s", i, elem.Type())
					continue
				}
				if str.Value != tt.expected[i] {
					t.Errorf("element %d: expected %q, got %q", i, tt.expected[i], str.Value)
				}
			}
		})
	}
}

func TestRegexpFindAll(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "find all words",
			code:     `Regexp.findAll("hello world gsh", "\\w+")`,
			expected: []string{"hello", "world", "gsh"},
		},
		{
			name:     "find all digits",
			code:     `Regexp.findAll("a1b2c3", "\\d")`,
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "find with limit",
			code:     `Regexp.findAll("a1b2c3d4", "\\d", 2)`,
			expected: []string{"1", "2"},
		},
		{
			name:     "no match returns empty array",
			code:     `Regexp.findAll("hello", "\\d")`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.EvalString(tt.code, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			arr, ok := result.FinalResult.(*ArrayValue)
			if !ok {
				t.Fatalf("expected array, got %s", result.FinalResult.Type())
			}

			if len(arr.Elements) != len(tt.expected) {
				t.Errorf("expected %d elements, got %d", len(tt.expected), len(arr.Elements))
				return
			}

			for i, elem := range arr.Elements {
				str, ok := elem.(*StringValue)
				if !ok {
					t.Errorf("element %d: expected string, got %s", i, elem.Type())
					continue
				}
				if str.Value != tt.expected[i] {
					t.Errorf("element %d: expected %q, got %q", i, tt.expected[i], str.Value)
				}
			}
		})
	}
}

func TestRegexpEscape(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "escape special chars",
			code:     `Regexp.escape("hello.world")`,
			expected: `hello\.world`,
		},
		{
			name:     "escape brackets",
			code:     `Regexp.escape("[test]")`,
			expected: `\[test\]`,
		},
		{
			name:     "escape complex pattern",
			code:     `Regexp.escape("a+b*c?d^e$f")`,
			expected: `a\+b\*c\?d\^e\$f`,
		},
		{
			name:     "no special chars",
			code:     `Regexp.escape("hello")`,
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interp.EvalString(tt.code, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			strVal, ok := result.FinalResult.(*StringValue)
			if !ok {
				t.Fatalf("expected string, got %s", result.FinalResult.Type())
			}

			if strVal.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, strVal.Value)
			}
		})
	}
}

func TestRegexpInvalidPattern(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name string
		code string
	}{
		{
			name: "invalid pattern in match",
			code: `Regexp.match("hello", "[")`,
		},
		{
			name: "invalid pattern in test",
			code: `Regexp.test("hello", "[")`,
		},
		{
			name: "invalid pattern in replace",
			code: `Regexp.replace("hello", "[", "x")`,
		},
		{
			name: "invalid pattern in replaceAll",
			code: `Regexp.replaceAll("hello", "[", "x")`,
		},
		{
			name: "invalid pattern in split",
			code: `Regexp.split("hello", "[")`,
		},
		{
			name: "invalid pattern in findAll",
			code: `Regexp.findAll("hello", "[")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interp.EvalString(tt.code, nil)
			if err == nil {
				t.Fatal("expected error for invalid pattern")
			}
		})
	}
}

func TestRegexpArgumentErrors(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	tests := []struct {
		name string
		code string
	}{
		{
			name: "match wrong arg count",
			code: `Regexp.match("hello")`,
		},
		{
			name: "match non-string first arg",
			code: `Regexp.match(123, "\\d")`,
		},
		{
			name: "match non-string second arg",
			code: `Regexp.match("hello", 123)`,
		},
		{
			name: "test wrong arg count",
			code: `Regexp.test("hello")`,
		},
		{
			name: "replace wrong arg count",
			code: `Regexp.replace("hello", "l")`,
		},
		{
			name: "escape wrong arg count",
			code: `Regexp.escape()`,
		},
		{
			name: "escape non-string arg",
			code: `Regexp.escape(123)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interp.EvalString(tt.code, nil)
			if err == nil {
				t.Fatal("expected error for invalid arguments")
			}
		})
	}
}

// TestRegexpCache tests that the regex cache works correctly
func TestRegexpCache(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Test that repeated use of the same pattern works (uses cache)
	code := `
results = []
items = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
for (item of items) {
    if (Regexp.test("hello123", "\\d+")) {
        results.push(item)
    }
}
results.length
`
	result, err := interp.EvalString(code, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	numVal, ok := result.FinalResult.(*NumberValue)
	if !ok {
		t.Fatalf("expected number, got %s", result.FinalResult.Type())
	}

	if numVal.Value != 10 {
		t.Errorf("expected 10, got %v", numVal.Value)
	}
}

// TestRegexpCacheEviction tests that the cache evicts old entries correctly
func TestRegexpCacheEviction(t *testing.T) {
	// Create a small cache for testing
	cache := &regexpCache{
		cache:   make(map[string]*regexp.Regexp),
		order:   make([]string, 0, 4),
		maxSize: 4,
	}

	// Add patterns to fill the cache
	patterns := []string{"a", "b", "c", "d"}
	for _, p := range patterns {
		_, err := cache.getOrCompile(p)
		if err != nil {
			t.Fatalf("unexpected error compiling %q: %v", p, err)
		}
	}

	// Verify cache is full
	if len(cache.cache) != 4 {
		t.Errorf("expected cache size 4, got %d", len(cache.cache))
	}

	// Add another pattern, should evict "a"
	_, err := cache.getOrCompile("e")
	if err != nil {
		t.Fatalf("unexpected error compiling 'e': %v", err)
	}

	// Verify "a" was evicted
	if _, ok := cache.cache["a"]; ok {
		t.Error("expected 'a' to be evicted from cache")
	}

	// Verify "e" is in cache
	if _, ok := cache.cache["e"]; !ok {
		t.Error("expected 'e' to be in cache")
	}

	// Verify size is still 4
	if len(cache.cache) != 4 {
		t.Errorf("expected cache size 4, got %d", len(cache.cache))
	}
}

// TestRegexpCacheHit tests that cache hits return the same compiled regex
func TestRegexpCacheHit(t *testing.T) {
	cache := &regexpCache{
		cache:   make(map[string]*regexp.Regexp),
		order:   make([]string, 0, 4),
		maxSize: 4,
	}

	// Compile a pattern
	re1, err := cache.getOrCompile("\\d+")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get the same pattern again
	re2, err := cache.getOrCompile("\\d+")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be the same pointer
	if re1 != re2 {
		t.Error("expected cache hit to return same regex instance")
	}
}

// TestRegexpCacheInvalidPattern tests that invalid patterns return errors and are not cached
func TestRegexpCacheInvalidPattern(t *testing.T) {
	cache := &regexpCache{
		cache:   make(map[string]*regexp.Regexp),
		order:   make([]string, 0, 4),
		maxSize: 4,
	}

	// Try to compile an invalid pattern
	_, err := cache.getOrCompile("[")
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}

	// Verify it was not cached
	if _, ok := cache.cache["["]; ok {
		t.Error("expected invalid pattern to not be cached")
	}
}

// TestRegexpEscapeAndUse tests that escaped strings can be used in patterns
func TestRegexpEscapeAndUse(t *testing.T) {
	interp := New(&Options{})
	defer interp.Close()

	// Test that escaped pattern works correctly
	code := `
escaped = Regexp.escape("hello.world")
Regexp.test("hello.world", escaped)
`
	result, err := interp.EvalString(code, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal, ok := result.FinalResult.(*BoolValue)
	if !ok {
		t.Fatalf("expected bool, got %s", result.FinalResult.Type())
	}

	if !boolVal.Value {
		t.Error("expected escaped pattern to match literal string")
	}

	// Verify that unescaped pattern would match more broadly
	code2 := `Regexp.test("helloXworld", "hello.world")`
	result2, err := interp.EvalString(code2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	boolVal2, ok := result2.FinalResult.(*BoolValue)
	if !ok {
		t.Fatalf("expected bool, got %s", result2.FinalResult.Type())
	}

	if !boolVal2.Value {
		t.Error("expected unescaped pattern to match (. matches any char)")
	}
}
