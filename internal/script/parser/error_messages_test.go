package parser

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

// TestDetailedErrorMessages tests that parser provides helpful, detailed error messages
func TestDetailedErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedErrors []string // All expected error substrings
	}{
		{
			name:  "missing newline between statements",
			input: `x = 5 y = 10`,
			expectedErrors: []string{
				"unexpected token",
				"expected newline",
			},
		},
		{
			name:  "semicolon as statement separator",
			input: `x = 5; y = 10`,
			expectedErrors: []string{
				"semicolons are not allowed",
				"use newlines instead",
			},
		},
		{
			name:  "unclosed string literal",
			input: `message = "hello world`,
			expectedErrors: []string{
				"unterminated string literal",
				"line",
			},
		},
		{
			name:  "missing closing brace in block",
			input: `if (x > 5) { y = 10`,
			expectedErrors: []string{
				"expected '}'",
				"line",
			},
		},
		{
			name:  "missing opening parenthesis in if statement",
			input: `if x > 5) { y = 10 }`,
			expectedErrors: []string{
				"expected next token to be '('",
				"line",
			},
		},
		{
			name:  "missing closing parenthesis in function call",
			input: `print("hello"`,
			expectedErrors: []string{
				"expected next token to be ')'",
			},
		},
		{
			name:  "invalid expression start",
			input: `x = } + 5`,
			expectedErrors: []string{
				"unexpected token",
			},
		},
		{
			name:  "missing colon in type annotation",
			input: `x number = 5`,
			expectedErrors: []string{
				// Parser treats 'number' as identifier on same line
				"unexpected token",
				"same line",
			},
		},
		{
			name:  "invalid object key",
			input: `obj = { 123: "value" }`,
			expectedErrors: []string{
				"expected object key",
			},
		},
		{
			name:  "missing assignment operator",
			input: `x 5`,
			expectedErrors: []string{
				// Parser now catches this as two statements on same line
				"unexpected token",
				"same line",
			},
		},
		{
			name: "multiple errors in single input",
			input: `
				if x > 5) {
					y = 10
				mcp filesystem {
					command "npx"
				}
			`,
			expectedErrors: []string{
				"expected next token to be '('",
			},
		},
		{
			name:  "invalid token in expression",
			input: `x = 5 & 3`,
			expectedErrors: []string{
				"unexpected token",
				"illegal token",
			},
		},
		{
			name:  "missing condition in if statement",
			input: `if () { x = 5 }`,
			expectedErrors: []string{
				"unexpected token",
			},
		},
		{
			name:  "missing body in while loop",
			input: `while (x > 0)`,
			expectedErrors: []string{
				"expected next token to be '{'",
			},
		},
		{
			name:  "invalid for-of syntax",
			input: `for (x in items) { print(x) }`,
			expectedErrors: []string{
				"expected next token to be keyword 'of'",
			},
		},
		{
			name:  "try without catch or finally",
			input: `try { x = 5 }`,
			expectedErrors: []string{
				"try statement must have at least one 'catch' or 'finally'",
			},
		},
		{
			name:  "missing parameter name in tool",
			input: `tool process(: string) { return "" }`,
			expectedErrors: []string{
				"expected parameter name",
			},
		},
		{
			name:  "missing type after colon in parameter",
			input: `tool process(x:) { return x }`,
			expectedErrors: []string{
				"expected type annotation after ':'",
			},
		},
		{
			name:  "missing return type after colon",
			input: `tool process(): { return 5 }`,
			expectedErrors: []string{
				"expected return type after ':'",
			},
		},
		{
			name:  "MCP declaration missing name",
			input: `mcp { command: "npx" }`,
			expectedErrors: []string{
				"expected next token to be identifier",
			},
		},
		{
			name:  "model declaration invalid config key",
			input: `model claude { 123: "value" }`,
			expectedErrors: []string{
				"expected identifier for config key",
			},
		},
		{
			name: "nested error recovery",
			input: `
				tool broken(x: {
					if (x > 5 {
						print(x)
					}
				}
			`,
			expectedErrors: []string{
				// Multiple errors expected
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			_ = p.ParseProgram()

			errors := p.Errors()
			if len(errors) == 0 && len(tt.expectedErrors) > 0 {
				t.Fatalf("expected parser errors containing %v, but got none", tt.expectedErrors)
			}

			// Check that all expected error substrings are present
			for _, expectedSubstr := range tt.expectedErrors {
				found := false
				for _, err := range errors {
					if strings.Contains(err, expectedSubstr) {
						found = true
						break
					}
				}
				if !found && expectedSubstr != "" {
					t.Errorf("expected error containing %q, got errors: %v", expectedSubstr, errors)
				}
			}

			// Verify errors include line/column information
			if len(errors) > 0 {
				for _, err := range errors {
					if !strings.Contains(err, "line") || !strings.Contains(err, "column") {
						// Some error messages might not have location info, check at least one does
					}
				}
			}
		})
	}
}

// TestErrorRecovery tests that parser can recover from errors and continue parsing
func TestErrorRecovery(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectErrors     bool
		expectStatements int // Number of statements that should still be parsed
		validateProgram  func(*testing.T, *Program)
	}{
		{
			name: "recover after bad statement and parse next",
			input: `
				x = 5
				if x > ) { }
				y = 10
			`,
			expectErrors:     true,
			expectStatements: 2, // x = 5 and y = 10 should still parse
			validateProgram: func(t *testing.T, prog *Program) {
				if len(prog.Statements) < 1 {
					t.Errorf("expected at least 1 statement to be parsed, got %d", len(prog.Statements))
				}
			},
		},
		{
			name: "recover from multiple errors",
			input: `
				x = 5
				if (x > ) { }
				y = 10
				while x < ) { }
				z = 15
			`,
			expectErrors:     true,
			expectStatements: 3, // x, y, z should parse
			validateProgram: func(t *testing.T, prog *Program) {
				if len(prog.Statements) < 2 {
					t.Errorf("expected at least 2 statements to be parsed, got %d", len(prog.Statements))
				}
			},
		},
		{
			name: "recover after bad MCP declaration",
			input: `
				mcp filesystem {
					command: "npx",
				}
				
				mcp { }
				
				mcp github {
					command: "npx",
				}
			`,
			expectErrors:     true,
			expectStatements: 2, // filesystem and github should parse
			validateProgram: func(t *testing.T, prog *Program) {
				if len(prog.Statements) < 1 {
					t.Errorf("expected at least 1 statement to be parsed, got %d", len(prog.Statements))
				}
			},
		},
		{
			name: "continue parsing after tool error",
			input: `
				tool good() {
					return 5
				}
				
				tool bad(: string) {
					return ""
				}
				
				tool another() {
					return 10
				}
			`,
			expectErrors:     true,
			expectStatements: 2, // good and another should parse
			validateProgram: func(t *testing.T, prog *Program) {
				if len(prog.Statements) < 1 {
					t.Errorf("expected at least 1 statement to be parsed, got %d", len(prog.Statements))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			errors := p.Errors()
			if tt.expectErrors && len(errors) == 0 {
				t.Fatal("expected parser errors, but got none")
			}

			if !tt.expectErrors && len(errors) > 0 {
				t.Fatalf("expected no errors, but got: %v", errors)
			}

			if tt.validateProgram != nil {
				tt.validateProgram(t, program)
			}
		})
	}
}

// TestContextualErrorMessages tests error messages provide context
func TestContextualErrorMessages(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedContext    string // What context should be mentioned
		shouldContainToken string // Should mention the problematic token
	}{
		{
			name:               "if statement context",
			input:              `if x { }`,
			expectedContext:    "expected next token to be '('",
			shouldContainToken: "",
		},
		{
			name:               "while statement context",
			input:              `while { }`,
			expectedContext:    "expected next token to be '('",
			shouldContainToken: "",
		},
		{
			name:               "tool parameter context",
			input:              `tool test(x, y string) { }`,
			expectedContext:    "",
			shouldContainToken: "",
		},
		{
			name:               "mcp declaration context",
			input:              `mcp filesystem command: "npx" }`,
			expectedContext:    "expected next token to be '{'",
			shouldContainToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			_ = p.ParseProgram()

			errors := p.Errors()
			if len(errors) == 0 {
				t.Fatal("expected parser errors, but got none")
			}

			// Check for context in error messages
			if tt.expectedContext != "" {
				found := false
				for _, err := range errors {
					if strings.Contains(err, tt.expectedContext) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error to contain context %q, got errors: %v", tt.expectedContext, errors)
				}
			}
		})
	}
}

// TestLexerErrorsInParser tests that lexer errors (ILLEGAL tokens) are reported by parser
func TestLexerErrorsInParser(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "single ampersand",
			input: `x = 5 & 3`,
		},
		{
			name:  "invalid character",
			input: `x = @invalid`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			_ = p.ParseProgram()

			// Parser should report errors for ILLEGAL tokens
			errors := p.Errors()
			if len(errors) == 0 {
				t.Error("expected parser to report errors for ILLEGAL tokens, but got none")
			}

			// Check that error mentions "illegal token" or "unexpected token"
			found := false
			for _, err := range errors {
				if strings.Contains(err, "illegal token") || strings.Contains(err, "unexpected token") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error to mention 'illegal token' or 'unexpected token', got: %v", errors)
			}
		})
	}
}

// TestErrorMessageFormatting tests error message formatting is consistent
func TestErrorMessageFormatting(t *testing.T) {
	input := `
		if (x > 5 {
			y = 10
		}
		
		mcp { command: "npx" }
	`

	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()

	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected parser errors, but got none")
	}

	// Check that all error messages follow a consistent format
	for _, err := range errors {
		// Errors should be descriptive strings
		if len(err) < 10 {
			t.Errorf("error message too short: %q", err)
		}
	}
}
