package interpreter

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestMapConstruction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "empty map",
			input: `m = Map()
print(m)`,
			expected: "Map({})\n",
		},
		{
			name: "map from array of pairs",
			input: `m = Map([["name", "Alice"], ["age", 30]])
print(m.size)`,
			expected: "2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			interp := New(nil)
			var err error
			output := captureOutput(func() {
				_, err = interp.Eval(program)
			})

			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if output != tt.expected {
				t.Errorf("expected output %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestMapMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name: "map set and get",
			input: `m = Map()
m.set("key", "value")
result = m.get("key")
print(result)`,
			expected: "value\n",
		},
		{
			name: "map get non-existent key returns null",
			input: `m = Map()
result = m.get("missing")
print(result)`,
			expected: "null\n",
		},
		{
			name: "map has method",
			input: `m = Map()
m.set("key", "value")
print(m.has("key"))
print(m.has("missing"))`,
			expected: "true\nfalse\n",
		},
		{
			name: "map delete method",
			input: `m = Map()
m.set("key", "value")
deleted = m.delete("key")
print(deleted)
print(m.has("key"))`,
			expected: "true\nfalse\n",
		},
		{
			name: "map delete non-existent key",
			input: `m = Map()
deleted = m.delete("missing")
print(deleted)`,
			expected: "false\n",
		},
		{
			name: "map size property",
			input: `m = Map()
print(m.size)
m.set("a", 1)
m.set("b", 2)
print(m.size)`,
			expected: "0\n2\n",
		},
		{
			name: "map keys method",
			input: `m = Map([["name", "Alice"], ["age", 30]])
keys = m.keys()
print(keys.length)`,
			expected: "2\n",
		},
		{
			name: "map values method",
			input: `m = Map([["name", "Alice"], ["age", 30]])
values = m.values()
print(values.length)`,
			expected: "2\n",
		},
		{
			name: "map entries method",
			input: `m = Map([["name", "Alice"]])
entries = m.entries()
print(entries.length)
print(entries[0][0])`,
			expected: "1\nname\n",
		},
		{
			name: "map chaining",
			input: `m = Map()
m.set("a", 1).set("b", 2).set("c", 3)
print(m.size)`,
			expected: "3\n",
		},
		{
			name: "map with different value types",
			input: `m = Map()
m.set("string", "hello")
m.set("number", 42)
m.set("bool", true)
m.set("array", [1, 2, 3])
print(m.size)
print(m.get("number"))`,
			expected: "4\n42\n",
		},
		{
			name: "map overwrite existing key",
			input: `m = Map()
m.set("key", "old")
m.set("key", "new")
print(m.size)
print(m.get("key"))`,
			expected: "1\nnew\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			interp := New(nil)
			var err error
			output := captureOutput(func() {
				_, err = interp.Eval(program)
			})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if tt.expected != "" && output != tt.expected {
				t.Errorf("expected output %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestSetConstruction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "empty set",
			input: `s = Set()
print(s)`,
			expected: "Set({})\n",
		},
		{
			name: "set from array with duplicates",
			input: `s = Set([1, 2, 3, 2, 1])
print(s.size)`,
			expected: "3\n",
		},
		{
			name: "set from array of strings",
			input: `s = Set(["apple", "banana", "cherry"])
print(s.size)`,
			expected: "3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			interp := New(nil)
			var err error
			output := captureOutput(func() {
				_, err = interp.Eval(program)
			})

			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if tt.expected != "" && output != tt.expected {
				t.Errorf("expected output %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestSetMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name: "set add method",
			input: `s = Set()
s.add(1)
s.add(2)
print(s.size)`,
			expected: "2\n",
		},
		{
			name: "set add duplicate",
			input: `s = Set()
s.add(1)
s.add(1)
print(s.size)`,
			expected: "1\n",
		},
		{
			name: "set has method",
			input: `s = Set([1, 2, 3])
print(s.has(2))
print(s.has(5))`,
			expected: "true\nfalse\n",
		},
		{
			name: "set delete method",
			input: `s = Set([1, 2, 3])
deleted = s.delete(2)
print(deleted)
print(s.has(2))`,
			expected: "true\nfalse\n",
		},
		{
			name: "set delete non-existent",
			input: `s = Set([1, 2, 3])
deleted = s.delete(5)
print(deleted)`,
			expected: "false\n",
		},
		{
			name: "set size property",
			input: `s = Set()
print(s.size)
s.add(1)
s.add(2)
print(s.size)`,
			expected: "0\n2\n",
		},
		{
			name: "set chaining",
			input: `s = Set()
s.add(1).add(2).add(3)
print(s.size)`,
			expected: "3\n",
		},
		{
			name: "set with strings",
			input: `s = Set(["apple", "banana", "apple"])
print(s.size)
print(s.has("apple"))`,
			expected: "2\ntrue\n",
		},
		{
			name: "set with mixed types",
			input: `s = Set([1, "two", true, 4])
print(s.size)
print(s.has("two"))
print(s.has(1))`,
			expected: "4\ntrue\ntrue\n",
		},
		{
			name: "set add and delete operations",
			input: `s = Set([1, 2, 3])
s.add(4)
s.delete(2)
print(s.size)
print(s.has(2))
print(s.has(4))`,
			expected: "3\nfalse\ntrue\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			interp := New(nil)
			var err error
			output := captureOutput(func() {
				_, err = interp.Eval(program)
			})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if tt.expected != "" && output != tt.expected {
				t.Errorf("expected output %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestMapSetIntegration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "map with set values",
			input: `m = Map()
s = Set([1, 2, 3])
m.set("numbers", s)
retrieved = m.get("numbers")
print(retrieved.size)`,
			expected: "3\n",
		},
		{
			name: "set with map in tool",
			input: `tool process() {
  m = Map()
  m.set("count", 42)
  return m.get("count")
}
result = process()
print(result)`,
			expected: "42\n",
		},
		{
			name: "nested maps",
			input: `outer = Map()
inner = Map()
inner.set("key", "value")
outer.set("inner", inner)
result = outer.get("inner")
print(result.get("key"))`,
			expected: "value\n",
		},
		{
			name: "map and set in conditionals",
			input: `m = Map()
m.set("exists", true)
if (m.has("exists")) {
  print("found")
}
s = Set([1, 2, 3])
if (s.has(2)) {
  print("contains 2")
}`,
			expected: "found\ncontains 2\n",
		},
		{
			name: "map iteration with entries",
			input: `m = Map([["a", 1], ["b", 2]])
entries = m.entries()
print(entries.length)
print(m.get("a"))
print(m.get("b"))`,
			expected: "2\n1\n2\n",
		},
		{
			name: "set uniqueness based on string representation",
			input: `s = Set()
s.add(1)
s.add("1")
s.add(1)
print(s.size)`,
			expected: "1\n", // "1" and 1 have the same string representation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			interp := New(nil)
			var err error
			output := captureOutput(func() {
				_, err = interp.Eval(program)
			})

			if err != nil {
				t.Fatalf("eval error: %v", err)
			}

			if output != tt.expected {
				t.Errorf("expected output %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestMapSetEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "map with empty string key",
			input: `m = Map()
m.set("", "empty key")
print(m.get(""))`,
			wantErr: false,
		},
		{
			name:    "map constructor with invalid array",
			input:   `m = Map([1, 2, 3])`,
			wantErr: true,
		},
		{
			name:    "map constructor with wrong pair length",
			input:   `m = Map([["key"]])`,
			wantErr: true,
		},
		{
			name:    "map constructor with non-string key",
			input:   `m = Map([[1, "value"]])`,
			wantErr: true,
		},
		{
			name:    "set constructor with non-array",
			input:   `s = Set("not an array")`,
			wantErr: true,
		},
		{
			name: "map size is read-only property",
			input: `m = Map()
m.set("a", 1)
print(m.size)`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				if !tt.wantErr {
					t.Fatalf("unexpected parse errors: %v", p.Errors())
				}
				return
			}

			interp := New(nil)
			var err error
			captureOutput(func() {
				_, err = interp.Eval(program)
			})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
