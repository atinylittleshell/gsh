package bash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreprocessTypesetCommands_EmptyInput(t *testing.T) {
	result := PreprocessTypesetCommands("")
	assert.Equal(t, "", result, "empty input should return empty string")
}

func TestPreprocessTypesetCommands_NilLikeInput(t *testing.T) {
	// Test various edge cases that might be problematic
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"whitespace only", "   ", "   "},
		{"single space", " ", " "},
		{"newline only", "\n", "\n"},
		{"mixed whitespace", " \t\n ", " \t\n "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should return %q", tc.input, tc.expected)
		})
	}
}

func TestPreprocessTypesetCommands_NormalTransformation(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"typeset -f", "typeset -f", "gsh_typeset -f"},
		{"declare -f", "declare -f", "gsh_typeset -f"},
		{"typeset -F", "typeset -F", "gsh_typeset -F"},
		{"declare -F", "declare -F", "gsh_typeset -F"},
		{"typeset -p", "typeset -p", "gsh_typeset -p"},
		{"declare -p", "declare -p", "gsh_typeset -p"},
		{"extra spaces typeset -f", "typeset  -f", "gsh_typeset -f"},
		{"extra spaces declare -f", "declare  -f", "gsh_typeset -f"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should be transformed to %q", tc.input, tc.expected)
		})
	}
}

func TestPreprocessTypesetCommands_MixedContent(t *testing.T) {
	input := `#!/bin/bash
typeset -f myfunc
declare -F
echo "hello world"
typeset -p VAR1
declare -p VAR2`

	expected := `#!/bin/bash
gsh_typeset -f myfunc
gsh_typeset -F
echo "hello world"
gsh_typeset -p VAR1
gsh_typeset -p VAR2`

	result := PreprocessTypesetCommands(input)
	assert.Equal(t, expected, result, "mixed content should be transformed correctly")
}

func TestPreprocessTypesetCommands_NoTransformation(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"other command", "echo hello", "echo hello"},
		{"typeset without flag", "typeset VAR=value", "typeset VAR=value"},
		{"declare without flag", "declare VAR=value", "declare VAR=value"},
		{"different flag", "typeset -i x=1", "typeset -i x=1"},
		{"comment", "# typeset -f", "# typeset -f"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should remain unchanged", tc.input)
		})
	}
}

func TestPreprocessTypesetCommands_LargeInput(t *testing.T) {
	// Create a large input (but under the 10MB limit)
	largeInput := make([]byte, 1024*1024) // 1MB
	for i := range largeInput {
		largeInput[i] = 'a'
	}
	largeInput[0] = 't'
	largeInput[1] = 'y'
	largeInput[2] = 'p'
	largeInput[3] = 'e'
	largeInput[4] = 's'
	largeInput[5] = 'e'
	largeInput[6] = 't'
	largeInput[7] = ' '
	largeInput[8] = '-'
	largeInput[9] = 'f'

	result := PreprocessTypesetCommands(string(largeInput))
	assert.Contains(t, result, "gsh_typeset -f", "large input should still be processed correctly")
}

func TestPreprocessTypesetCommands_InputSizeLimit(t *testing.T) {
	// Test that inputs over 10MB are truncated
	oversizedInput := make([]byte, 11*1024*1024) // 11MB
	for i := range oversizedInput {
		oversizedInput[i] = 'x'
	}

	// Add a transformation target at the beginning
	oversizedInput[0] = 't'
	oversizedInput[1] = 'y'
	oversizedInput[2] = 'p'
	oversizedInput[3] = 'e'
	oversizedInput[4] = 's'
	oversizedInput[5] = 'e'
	oversizedInput[6] = 't'
	oversizedInput[7] = ' '
	oversizedInput[8] = '-'
	oversizedInput[9] = 'f'

	result := PreprocessTypesetCommands(string(oversizedInput))
	assert.Contains(t, result, "gsh_typeset -f", "oversized input should still process the first portion")
	assert.Less(t, len(result), 11*1024*1024, "result should be truncated to under 11MB")
}

func TestPreprocessTypesetCommands_InsideDoubleQuotes(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"typeset -f in double quotes", `echo "typeset -f myfunc"`, `echo "typeset -f myfunc"`},
		{"declare -f in double quotes", `echo "declare -f myfunc"`, `echo "declare -f myfunc"`},
		{"typeset -F in double quotes", `echo "typeset -F"`, `echo "typeset -F"`},
		{"declare -p in double quotes", `echo "declare -p VAR"`, `echo "declare -p VAR"`},
		{"mixed content with quotes", `echo "before"; typeset -f myfunc; echo "after"`, `echo "before"; gsh_typeset -f myfunc; echo "after"`},
		{"escaped quotes in command", `echo "He said \"typeset -f\" to me"`, `echo "He said \"typeset -f\" to me"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should remain unchanged inside double quotes", tc.input)
		})
	}
}

func TestPreprocessTypesetCommands_InsideSingleQuotes(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"typeset -f in single quotes", `echo 'typeset -f myfunc'`, `echo 'typeset -f myfunc'`},
		{"declare -f in single quotes", `echo 'declare -f myfunc'`, `echo 'declare -f myfunc'`},
		{"typeset -F in single quotes", `echo 'typeset -F'`, `echo 'typeset -F'`},
		{"declare -p in single quotes", `echo 'declare -p VAR'`, `echo 'declare -p VAR'`},
		{"mixed quotes", `echo "double"; echo 'single'; typeset -f`, `echo "double"; echo 'single'; gsh_typeset -f`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should remain unchanged inside single quotes", tc.input)
		})
	}
}

func TestPreprocessTypesetCommands_InsideHeredoc(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"typeset -f in heredoc",
			`cat << EOF
typeset -f myfunc
declare -f another
EOF`,
			`cat << EOF
typeset -f myfunc
declare -f another
EOF`,
		},
		{
			"typeset -F in heredoc",
			`cat << DELIM
typeset -F
declare -F
DELIM`,
			`cat << DELIM
typeset -F
declare -F
DELIM`,
		},
		{
			"typeset -p in heredoc",
			`cat << END
typeset -p VAR1
declare -p VAR2
END`,
			`cat << END
typeset -p VAR1
declare -p VAR2
END`,
		},
		{
			"indented heredoc with <<-",
			`cat <<- INDENT
	typeset -f myfunc
	declare -f another
INDENT`,
			`cat <<- INDENT
	typeset -f myfunc
	declare -f another
INDENT`,
		},
		{
			"mixed heredoc and normal",
			`cat << EOF
typeset -f inside
EOF
typeset -f outside`,
			`cat << EOF
typeset -f inside
EOF
gsh_typeset -f outside`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should not transform inside heredoc", tc.input)
		})
	}
}

func TestPreprocessTypesetCommands_InsideFunctionDefinition(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"typeset -f in function body",
			`myfunc() {
	typeset -f myfunc
	echo "hello"
}`,
			`myfunc() {
	gsh_typeset -f myfunc
	echo "hello"
}`,
		},
		{
			"declare -f in function body",
			`another_func() {
	declare -f another_func
	return 0
}`,
			`another_func() {
	gsh_typeset -f another_func
	return 0
}`,
		},
		{
			"function with mixed content",
			`test_func() {
	echo "start"
	typeset -F
	declare -p VAR
	echo "end"
}`,
			`test_func() {
	echo "start"
	gsh_typeset -F
	gsh_typeset -p VAR
	echo "end"
}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should transform inside function body", tc.input)
		})
	}
}

func TestPreprocessTypesetCommands_VariableAssignment(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"typeset -f in variable", `CMD="typeset -f"`, `CMD="typeset -f"`},
		{"declare -f in variable", `CMD='declare -f'`, `CMD='declare -f'`},
		{"typeset -F assignment", `VAR=typeset -F`, `VAR=typeset -F`},
		{"declare -p assignment", `VAR=declare -p`, `VAR=declare -p`},
		{"mixed assignment and command", `CMD="typeset -f"; typeset -f`, `CMD="typeset -f"; gsh_typeset -f`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should not transform in variable assignment", tc.input)
		})
	}
}

func TestPreprocessTypesetCommands_CommandSubstitution(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"typeset -f in backticks", "echo \\`typeset -f\\`", "echo \\`typeset -f\\`"},
		{"declare -f in backticks", "result=\\`declare -f\\`", "result=\\`declare -f\\`"},
		{"typeset -F in $()", "echo $(typeset -F)", "echo $(typeset -F)"},
		{"declare -p in $()", "VAR=$(declare -p)", "VAR=$(declare -p)"},
		{"nested command substitution", "echo $(echo $(typeset -f))", "echo $(echo $(typeset -f))"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should not transform in command substitution", tc.input)
		})
	}
}

func TestPreprocessTypesetCommands_ComplexEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"escaped quotes",
			`echo "He said \"typeset -f\" to me"`,
			`echo "He said \"typeset -f\" to me"`,
		},
		{
			"quotes within quotes",
			`echo "It's a 'typeset -f' command"`,
			`echo "It's a 'typeset -f' command"`,
		},
		{
			"multiline with quotes",
			`echo "line1
typeset -f
line3"`,
			`echo "line1
typeset -f
line3"`,
		},
		{
			"case statement",
			`case "$VAR" in
"typeset -f") echo "matched";;
esac`,
			`case "$VAR" in
"typeset -f") echo "matched";;
esac`,
		},
		{
			"array assignment",
			`ARR=(typeset -f "declare -F")`,
			`ARR=(typeset -f "declare -F")`,
		},
		{
			"herestring",
			`cat <<< "typeset -f"`,
			`cat <<< "typeset -f"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := PreprocessTypesetCommands(tc.input)
			assert.Equal(t, tc.expected, result, "input %q should handle complex edge case correctly", tc.input)
		})
	}
}
