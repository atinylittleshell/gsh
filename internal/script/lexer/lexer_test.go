package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

model claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
}

agent DataAnalyst {
    model: claude,
    systemPrompt: """
        You are a data analyst.
        Analyze data carefully.
    """,
}

tool analyze(data: string): string {
    result = data | DataAnalyst
    return result
}

x = 10
y = 20.5
z = x + y * 2
name = "Alice"
greeting = 'Hello'
template = ` + "`Hello ${name}`" + `

if (x > 5) {
    print("greater")
} else {
    print("smaller")
}

for (item of items) {
    print(item)
}

while (x < 100) {
    x = x + 1
}

try {
    doSomething()
} catch (error) {
    print(error)
}

# This is a comment

result = a == b
result = a != b
result = a <= b
result = a >= b
result = a && b
result = a || b
result = a ?? b
result = !a
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		// mcp filesystem {
		{KW_MCP, "mcp"},
		{IDENT, "filesystem"},
		{LBRACE, "{"},

		// command: "npx",
		{IDENT, "command"},
		{COLON, ":"},
		{STRING, "npx"},
		{COMMA, ","},

		// args: ["-y", "@modelcontextprotocol/server-filesystem"],
		{IDENT, "args"},
		{COLON, ":"},
		{LBRACKET, "["},
		{STRING, "-y"},
		{COMMA, ","},
		{STRING, "@modelcontextprotocol/server-filesystem"},
		{RBRACKET, "]"},
		{COMMA, ","},

		// }
		{RBRACE, "}"},

		// model claude {
		{KW_MODEL, "model"},
		{IDENT, "claude"},
		{LBRACE, "{"},

		// provider: "anthropic",
		{IDENT, "provider"},
		{COLON, ":"},
		{STRING, "anthropic"},
		{COMMA, ","},

		// apiKey: env.ANTHROPIC_API_KEY,
		{IDENT, "apiKey"},
		{COLON, ":"},
		{IDENT, "env"},
		{DOT, "."},
		{IDENT, "ANTHROPIC_API_KEY"},
		{COMMA, ","},

		// }
		{RBRACE, "}"},

		// agent DataAnalyst {
		{KW_AGENT, "agent"},
		{IDENT, "DataAnalyst"},
		{LBRACE, "{"},

		// model: claude,
		{KW_MODEL, "model"},
		{COLON, ":"},
		{IDENT, "claude"},
		{COMMA, ","},

		// systemPrompt: """...""",
		{IDENT, "systemPrompt"},
		{COLON, ":"},
		{STRING, "You are a data analyst.\nAnalyze data carefully."},
		{COMMA, ","},

		// }
		{RBRACE, "}"},

		// tool analyze(data: string): string {
		{KW_TOOL, "tool"},
		{IDENT, "analyze"},
		{LPAREN, "("},
		{IDENT, "data"},
		{COLON, ":"},
		{IDENT, "string"},
		{RPAREN, ")"},
		{COLON, ":"},
		{IDENT, "string"},
		{LBRACE, "{"},

		// result = data | DataAnalyst
		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "data"},
		{OP_PIPE, "|"},
		{IDENT, "DataAnalyst"},

		// return result
		{KW_RETURN, "return"},
		{IDENT, "result"},

		// }
		{RBRACE, "}"},

		// x = 10
		{IDENT, "x"},
		{OP_ASSIGN, "="},
		{NUMBER, "10"},

		// y = 20.5
		{IDENT, "y"},
		{OP_ASSIGN, "="},
		{NUMBER, "20.5"},

		// z = x + y * 2
		{IDENT, "z"},
		{OP_ASSIGN, "="},
		{IDENT, "x"},
		{OP_PLUS, "+"},
		{IDENT, "y"},
		{OP_ASTERISK, "*"},
		{NUMBER, "2"},

		// name = "Alice"
		{IDENT, "name"},
		{OP_ASSIGN, "="},
		{STRING, "Alice"},

		// greeting = 'Hello'
		{IDENT, "greeting"},
		{OP_ASSIGN, "="},
		{STRING, "Hello"},

		// template = `Hello ${name}`
		{IDENT, "template"},
		{OP_ASSIGN, "="},
		{TEMPLATE_LITERAL, "Hello ${name}"},

		// if (x > 5) {
		{KW_IF, "if"},
		{LPAREN, "("},
		{IDENT, "x"},
		{OP_GT, ">"},
		{NUMBER, "5"},
		{RPAREN, ")"},
		{LBRACE, "{"},

		// print("greater")
		{IDENT, "print"},
		{LPAREN, "("},
		{STRING, "greater"},
		{RPAREN, ")"},

		// } else {
		{RBRACE, "}"},
		{KW_ELSE, "else"},
		{LBRACE, "{"},

		// print("smaller")
		{IDENT, "print"},
		{LPAREN, "("},
		{STRING, "smaller"},
		{RPAREN, ")"},

		// }
		{RBRACE, "}"},

		// for (item of items) {
		{KW_FOR, "for"},
		{LPAREN, "("},
		{IDENT, "item"},
		{KW_OF, "of"},
		{IDENT, "items"},
		{RPAREN, ")"},
		{LBRACE, "{"},

		// print(item)
		{IDENT, "print"},
		{LPAREN, "("},
		{IDENT, "item"},
		{RPAREN, ")"},

		// }
		{RBRACE, "}"},

		// while (x < 100) {
		{KW_WHILE, "while"},
		{LPAREN, "("},
		{IDENT, "x"},
		{OP_LT, "<"},
		{NUMBER, "100"},
		{RPAREN, ")"},
		{LBRACE, "{"},

		// x = x + 1
		{IDENT, "x"},
		{OP_ASSIGN, "="},
		{IDENT, "x"},
		{OP_PLUS, "+"},
		{NUMBER, "1"},

		// }
		{RBRACE, "}"},

		// try {
		{KW_TRY, "try"},
		{LBRACE, "{"},

		// doSomething()
		{IDENT, "doSomething"},
		{LPAREN, "("},
		{RPAREN, ")"},

		// } catch (error) {
		{RBRACE, "}"},
		{KW_CATCH, "catch"},
		{LPAREN, "("},
		{IDENT, "error"},
		{RPAREN, ")"},
		{LBRACE, "{"},

		// print(error)
		{IDENT, "print"},
		{LPAREN, "("},
		{IDENT, "error"},
		{RPAREN, ")"},

		// }
		{RBRACE, "}"},

		// Comments are skipped, so next token should be result
		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "a"},
		{OP_EQ, "=="},
		{IDENT, "b"},

		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "a"},
		{OP_NEQ, "!="},
		{IDENT, "b"},

		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "a"},
		{OP_LTE, "<="},
		{IDENT, "b"},

		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "a"},
		{OP_GTE, ">="},
		{IDENT, "b"},

		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "a"},
		{OP_AND, "&&"},
		{IDENT, "b"},

		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "a"},
		{OP_OR, "||"},
		{IDENT, "b"},

		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{IDENT, "a"},
		{OP_NULLCOAL, "??"},
		{IDENT, "b"},

		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{OP_BANG, "!"},
		{IDENT, "a"},

		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expected   string
		isTemplate bool
	}{
		{
			name:     "double quotes",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "single quotes",
			input:    `'hello world'`,
			expected: "hello world",
		},
		{
			name:       "template literal",
			input:      "`hello ${name}`",
			expected:   "hello ${name}",
			isTemplate: true,
		},
		{
			name:     "escaped characters in double quotes",
			input:    `"hello\nworld\ttab"`,
			expected: "hello\nworld\ttab",
		},
		{
			name:     "escaped quotes",
			input:    `"say \"hello\""`,
			expected: `say "hello"`,
		},
		{
			name:     "triple quoted string",
			input:    `"""hello world"""`,
			expected: "hello world",
		},
		{
			name: "triple quoted multiline",
			input: `"""
    Line 1
    Line 2
    Line 3
"""`,
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name: "triple quoted with varying indentation",
			input: `"""
        You are a data analyst.
        Analyze data carefully.
    """`,
			expected: "You are a data analyst.\nAnalyze data carefully.",
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: "",
		},
		{
			name:     "string with special characters",
			input:    `"@modelcontextprotocol/server-filesystem"`,
			expected: "@modelcontextprotocol/server-filesystem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			expectedType := STRING
			if tt.isTemplate {
				expectedType = TEMPLATE_LITERAL
			}

			if tok.Type != expectedType {
				t.Fatalf("token type wrong. expected=%v, got=%q", expectedType, tok.Type)
			}

			if tok.Literal != tt.expected {
				t.Fatalf("literal wrong. expected=%q, got=%q", tt.expected, tok.Literal)
			}
		})
	}
}

func TestNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "integer", input: "42", expected: "42"},
		{name: "float", input: "3.14", expected: "3.14"},
		{name: "float with leading zero", input: "0.5", expected: "0.5"},
		{name: "large number", input: "123456789", expected: "123456789"},
		{name: "zero", input: "0", expected: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			if tok.Type != NUMBER {
				t.Fatalf("token type wrong. expected=NUMBER, got=%q", tok.Type)
			}

			if tok.Literal != tt.expected {
				t.Fatalf("literal wrong. expected=%q, got=%q", tt.expected, tok.Literal)
			}
		})
	}
}

func TestComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "hash comment", input: "# This is a comment\nx = 1"},
		{name: "hash comment without newline", input: "# This is a comment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			// Comments should be skipped, so first token should be either x (IDENT) or EOF
			if tok.Type != IDENT && tok.Type != EOF {
				t.Fatalf("comments should be skipped. expected IDENT or EOF, got=%q", tok.Type)
			}
		})
	}
}

func TestLineAndColumnTracking(t *testing.T) {
	input := `x = 10
y = 20
z = 30`

	expectedPositions := []struct {
		line   int
		column int
	}{
		{1, 1}, // x
		{1, 3}, // =
		{1, 5}, // 10
		{2, 1}, // y
		{2, 3}, // =
		{2, 5}, // 20
		{3, 1}, // z
		{3, 3}, // =
		{3, 5}, // 30
	}

	l := New(input)

	for i, expected := range expectedPositions {
		tok := l.NextToken()

		if tok.Line != expected.line {
			t.Errorf("token[%d] - line wrong. expected=%d, got=%d (literal=%q)",
				i, expected.line, tok.Line, tok.Literal)
		}

		if tok.Column != expected.column {
			t.Errorf("token[%d] - column wrong. expected=%d, got=%d (literal=%q)",
				i, expected.column, tok.Column, tok.Literal)
		}
	}
}

func TestOperators(t *testing.T) {
	input := `= + - * / % ! == != < > <= >= && || | ? ??`

	expectedTypes := []TokenType{
		OP_ASSIGN, OP_PLUS, OP_MINUS, OP_ASTERISK, OP_SLASH, OP_PERCENT,
		OP_BANG, OP_EQ, OP_NEQ, OP_LT, OP_GT, OP_LTE, OP_GTE,
		OP_AND, OP_OR, OP_PIPE, OP_QUESTION, OP_NULLCOAL,
	}

	l := New(input)

	for i, expectedType := range expectedTypes {
		tok := l.NextToken()

		if tok.Type != expectedType {
			t.Fatalf("token[%d] - type wrong. expected=%q, got=%q",
				i, expectedType, tok.Type)
		}
	}
}

func TestDelimiters(t *testing.T) {
	input := `, : ; . ( ) { } [ ]`

	expectedTypes := []TokenType{
		COMMA, COLON, SEMICOLON, DOT,
		LPAREN, RPAREN, LBRACE, RBRACE, LBRACKET, RBRACKET,
	}

	l := New(input)

	for i, expectedType := range expectedTypes {
		tok := l.NextToken()

		if tok.Type != expectedType {
			t.Fatalf("token[%d] - type wrong. expected=%q, got=%q",
				i, expectedType, tok.Type)
		}
	}
}

func TestKeywords(t *testing.T) {
	input := `mcp model agent tool if else for of while break continue try catch return import export from`

	expectedTypes := []TokenType{
		KW_MCP, KW_MODEL, KW_AGENT, KW_TOOL, KW_IF, KW_ELSE,
		KW_FOR, KW_OF, KW_WHILE, KW_BREAK, KW_CONTINUE, KW_TRY, KW_CATCH, KW_RETURN,
		KW_IMPORT, KW_EXPORT, KW_FROM,
	}

	l := New(input)

	for i, expectedType := range expectedTypes {
		tok := l.NextToken()

		if tok.Type != expectedType {
			t.Fatalf("token[%d] - type wrong. expected=%q, got=%q",
				i, expectedType, tok.Type)
		}
	}
}

func TestIdentifiers(t *testing.T) {
	input := `foo bar _underscore camelCase PascalCase snake_case CONSTANT`

	expectedLiterals := []string{
		"foo", "bar", "_underscore", "camelCase", "PascalCase", "snake_case", "CONSTANT",
	}

	l := New(input)

	for i, expectedLiteral := range expectedLiterals {
		tok := l.NextToken()

		if tok.Type != IDENT {
			t.Fatalf("token[%d] - type wrong. expected=IDENT, got=%q",
				i, tok.Type)
		}

		if tok.Literal != expectedLiteral {
			t.Fatalf("token[%d] - literal wrong. expected=%q, got=%q",
				i, expectedLiteral, tok.Literal)
		}
	}
}

func TestIllegalCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "single ampersand", input: "&"},
		{name: "at symbol", input: "@"},
		{name: "dollar sign", input: "$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			// Some characters might be ILLEGAL, others might be parsed differently
			// Just check that we don't panic
			_ = tok
		})
	}
}

func TestEmptyInput(t *testing.T) {
	l := New("")
	tok := l.NextToken()

	if tok.Type != EOF {
		t.Fatalf("empty input should return EOF, got=%q", tok.Type)
	}
}

func TestWhitespaceHandling(t *testing.T) {
	input := `  x   =   10  `

	expectedTokens := []struct {
		typ     TokenType
		literal string
	}{
		{IDENT, "x"},
		{OP_ASSIGN, "="},
		{NUMBER, "10"},
		{EOF, ""},
	}

	l := New(input)

	for i, expected := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != expected.typ {
			t.Fatalf("token[%d] - type wrong. expected=%q, got=%q",
				i, expected.typ, tok.Type)
		}

		if tok.Literal != expected.literal {
			t.Fatalf("token[%d] - literal wrong. expected=%q, got=%q",
				i, expected.literal, tok.Literal)
		}
	}
}

func TestComplexExpression(t *testing.T) {
	input := `result = (x + y) * z / 2 - foo.bar()`

	expectedTokens := []struct {
		typ     TokenType
		literal string
	}{
		{IDENT, "result"},
		{OP_ASSIGN, "="},
		{LPAREN, "("},
		{IDENT, "x"},
		{OP_PLUS, "+"},
		{IDENT, "y"},
		{RPAREN, ")"},
		{OP_ASTERISK, "*"},
		{IDENT, "z"},
		{OP_SLASH, "/"},
		{NUMBER, "2"},
		{OP_MINUS, "-"},
		{IDENT, "foo"},
		{DOT, "."},
		{IDENT, "bar"},
		{LPAREN, "("},
		{RPAREN, ")"},
		{EOF, ""},
	}

	l := New(input)

	for i, expected := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != expected.typ {
			t.Fatalf("token[%d] - type wrong. expected=%q, got=%q",
				i, expected.typ, tok.Type)
		}

		if tok.Literal != expected.literal {
			t.Fatalf("token[%d] - literal wrong. expected=%q, got=%q",
				i, expected.literal, tok.Literal)
		}
	}
}

func TestMemberAccess(t *testing.T) {
	input := `filesystem.read_file env.HOME github.create_issue`

	expectedTokens := []struct {
		typ     TokenType
		literal string
	}{
		{IDENT, "filesystem"},
		{DOT, "."},
		{IDENT, "read_file"},
		{IDENT, "env"},
		{DOT, "."},
		{IDENT, "HOME"},
		{IDENT, "github"},
		{DOT, "."},
		{IDENT, "create_issue"},
		{EOF, ""},
	}

	l := New(input)

	for i, expected := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != expected.typ {
			t.Fatalf("token[%d] - type wrong. expected=%q, got=%q",
				i, expected.typ, tok.Type)
		}

		if tok.Literal != expected.literal {
			t.Fatalf("token[%d] - literal wrong. expected=%q, got=%q",
				i, expected.literal, tok.Literal)
		}
	}
}

func TestPipeOperator(t *testing.T) {
	input := `"prompt" | Agent | "message" | Agent2`

	expectedTokens := []struct {
		typ     TokenType
		literal string
	}{
		{STRING, "prompt"},
		{OP_PIPE, "|"},
		{IDENT, "Agent"},
		{OP_PIPE, "|"},
		{STRING, "message"},
		{OP_PIPE, "|"},
		{IDENT, "Agent2"},
		{EOF, ""},
	}

	l := New(input)

	for i, expected := range expectedTokens {
		tok := l.NextToken()

		if tok.Type != expected.typ {
			t.Fatalf("token[%d] - type wrong. expected=%q, got=%q",
				i, expected.typ, tok.Type)
		}

		if tok.Literal != expected.literal {
			t.Fatalf("token[%d] - literal wrong. expected=%q, got=%q",
				i, expected.literal, tok.Literal)
		}
	}
}

func TestDedent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "uniform indentation",
			input: `    Line 1
    Line 2
    Line 3`,
			expected: `Line 1
Line 2
Line 3`,
		},
		{
			name: "mixed indentation levels",
			input: `    Line 1
        Line 2 (indented more)
    Line 3`,
			expected: `Line 1
    Line 2 (indented more)
Line 3`,
		},
		{
			name: "empty lines",
			input: `    Line 1

    Line 3`,
			expected: `Line 1

Line 3`,
		},
		{
			name:     "no indentation",
			input:    "Line 1\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "single line",
			input:    "    Single line",
			expected: "Single line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedent(tt.input)
			if result != tt.expected {
				t.Errorf("dedent failed.\nexpected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "newline", input: `"line1\nline2"`, expected: "line1\nline2"},
		{name: "tab", input: `"col1\tcol2"`, expected: "col1\tcol2"},
		{name: "carriage return", input: `"text\rmore"`, expected: "text\rmore"},
		{name: "backslash", input: `"path\\to\\file"`, expected: "path\\to\\file"},
		{name: "double quote", input: `"say \"hi\""`, expected: `say "hi"`},
		{name: "single quote in double", input: `"it's"`, expected: "it's"},
		{name: "single quote escaped", input: `'it\'s'`, expected: "it's"},
		{name: "unicode escape ESC", input: `"\u001b[31m"`, expected: "\x1b[31m"},
		{name: "unicode escape multiple", input: `"\u001b[38;5;11mtext\u001b[0m"`, expected: "\x1b[38;5;11mtext\x1b[0m"},
		{name: "unicode escape emoji", input: `"\u263A"`, expected: "\u263A"},
		{name: "unicode escape lowercase", input: `"\u00e9"`, expected: "é"},
		{name: "unicode escape uppercase hex", input: `"\u00E9"`, expected: "é"},
		{name: "unicode incomplete fallback", input: `"\u00"`, expected: "\\u00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			tok := l.NextToken()

			if tok.Type != STRING {
				t.Fatalf("token type wrong. expected=STRING, got=%q", tok.Type)
			}

			if tok.Literal != tt.expected {
				t.Fatalf("literal wrong. expected=%q, got=%q", tt.expected, tok.Literal)
			}
		})
	}
}

func TestRealWorldScript(t *testing.T) {
	input := `#!/usr/bin/env gsh

mcp filesystem {
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-filesystem"],
}

model claude {
    provider: "anthropic",
    apiKey: env.ANTHROPIC_API_KEY,
    model: "claude-3-5-sonnet-20241022",
}

agent Analyzer {
    model: claude,
    systemPrompt: """
        You are a helpful assistant.
    """,
}

tool process(input: string): string {
    result = input | Analyzer
    return result
}

x = 42
print(x)
`

	l := New(input)
	tokenCount := 0

	for {
		tok := l.NextToken()
		tokenCount++

		if tok.Type == EOF {
			break
		}

		// Ensure no token has illegal type (except intentional cases)
		if tok.Type == ILLEGAL {
			t.Errorf("found ILLEGAL token at line %d, column %d: %q",
				tok.Line, tok.Column, tok.Literal)
		}

		// Ensure line and column are reasonable
		if tok.Line < 1 {
			t.Errorf("token has invalid line number: %d", tok.Line)
		}
	}

	// Should have parsed a reasonable number of tokens
	if tokenCount < 50 {
		t.Errorf("expected at least 50 tokens, got %d", tokenCount)
	}
}

// TestUnterminatedStringErrors tests that lexer reports errors for unterminated strings
func TestUnterminatedStringErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "unterminated double quote string",
			input:         `message = "hello world`,
			expectedError: "unterminated string literal",
		},
		{
			name:          "unterminated single quote string",
			input:         `message = 'hello world`,
			expectedError: "unterminated string literal",
		},
		{
			name:          "unterminated template string",
			input:         "message = `hello world",
			expectedError: "unterminated template string",
		},
		{
			name:          "unterminated triple-quoted double quote string",
			input:         `message = """hello world`,
			expectedError: "unterminated triple-quoted string",
		},
		{
			name:          "unterminated triple-quoted single quote string",
			input:         `message = '''hello world`,
			expectedError: "unterminated triple-quoted string",
		},
		{
			name:          "unterminated string with escape sequences",
			input:         `message = "hello\nworld`,
			expectedError: "unterminated string literal",
		},
		{
			name:          "unterminated string at EOF",
			input:         `"incomplete`,
			expectedError: "unterminated string literal",
		},
		{
			name:          "unterminated template at EOF",
			input:         "`incomplete",
			expectedError: "unterminated template string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)

			// Consume all tokens
			for {
				tok := l.NextToken()
				if tok.Type == EOF {
					break
				}
			}

			// Check that lexer has errors
			errors := l.Errors()
			if len(errors) == 0 {
				t.Fatalf("expected lexer to report error, but got none")
			}

			// Check that error message contains expected substring
			found := false
			for _, err := range errors {
				if containsSubstring(err, tt.expectedError) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected error containing %q, got errors: %v", tt.expectedError, errors)
			}
		})
	}
}

// TestValidStringsNoErrors tests that valid strings don't produce errors
func TestValidStringsNoErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "valid double quote string",
			input: `message = "hello world"`,
		},
		{
			name:  "valid single quote string",
			input: `message = 'hello world'`,
		},
		{
			name:  "valid template string",
			input: "message = `hello world`",
		},
		{
			name:  "valid triple-quoted string",
			input: `message = """hello world"""`,
		},
		{
			name:  "valid string with escape sequences",
			input: `message = "hello\nworld\t!"`,
		},
		{
			name:  "empty string",
			input: `message = ""`,
		},
		{
			name:  "multiple valid strings",
			input: `a = "hello"\nb = 'world'\nc = ` + "`template`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)

			// Consume all tokens
			for {
				tok := l.NextToken()
				if tok.Type == EOF {
					break
				}
			}

			// Check that lexer has no errors
			errors := l.Errors()
			if len(errors) > 0 {
				t.Errorf("expected no lexer errors, but got: %v", errors)
			}
		})
	}
}

// TestLexerErrorsIncludeLocation tests that error messages include line and column info
func TestLexerErrorsIncludeLocation(t *testing.T) {
	input := `
x = "valid"
y = "unterminated
`

	l := New(input)

	// Consume all tokens
	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			break
		}
	}

	errors := l.Errors()
	if len(errors) == 0 {
		t.Fatal("expected lexer error, but got none")
	}

	// Check that error includes "line" and "column"
	err := errors[0]
	if !containsSubstring(err, "line") || !containsSubstring(err, "column") {
		t.Errorf("expected error to include line and column information, got: %q", err)
	}

	// Check that it mentions line 3 (where the unterminated string starts)
	if !containsSubstring(err, "line 3") {
		t.Errorf("expected error to mention line 3, got: %q", err)
	}
}

// TestMultipleLexerErrors tests that lexer can report multiple errors
func TestMultipleLexerErrors(t *testing.T) {
	// Multiple unterminated strings on separate lines (each hits EOF)
	inputs := []string{
		`x = "unterminated`,
		`y = 'another unterminated`,
	}

	totalErrors := 0
	for _, input := range inputs {
		l := New(input)

		// Consume all tokens
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}

		errors := l.Errors()
		totalErrors += len(errors)
	}

	if totalErrors < 2 {
		t.Errorf("expected at least 2 lexer errors total, got %d", totalErrors)
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
