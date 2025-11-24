package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestMcpDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, Statement)
	}{
		{
			name: "basic MCP declaration with command",
			input: `mcp filesystem {
				command: "npx",
			}`,
			expected: func(t *testing.T, stmt Statement) {
				mcpDecl, ok := stmt.(*McpDeclaration)
				if !ok {
					t.Fatalf("stmt is not *McpDeclaration. got=%T", stmt)
				}

				if mcpDecl.Name.Value != "filesystem" {
					t.Errorf("mcpDecl.Name.Value not 'filesystem'. got=%q", mcpDecl.Name.Value)
				}

				if len(mcpDecl.Config) != 1 {
					t.Fatalf("mcpDecl.Config should have 1 entry. got=%d", len(mcpDecl.Config))
				}

				commandVal, ok := mcpDecl.Config["command"]
				if !ok {
					t.Fatalf("mcpDecl.Config missing 'command' key")
				}

				strLit, ok := commandVal.(*StringLiteral)
				if !ok {
					t.Fatalf("command value is not *StringLiteral. got=%T", commandVal)
				}

				if strLit.Value != "npx" {
					t.Errorf("command value not 'npx'. got=%q", strLit.Value)
				}
			},
		},
		{
			name: "MCP declaration with command and args",
			input: `mcp filesystem {
				command: "npx",
				args: ["-y", "@modelcontextprotocol/server-filesystem"],
			}`,
			expected: func(t *testing.T, stmt Statement) {
				mcpDecl, ok := stmt.(*McpDeclaration)
				if !ok {
					t.Fatalf("stmt is not *McpDeclaration. got=%T", stmt)
				}

				if mcpDecl.Name.Value != "filesystem" {
					t.Errorf("mcpDecl.Name.Value not 'filesystem'. got=%q", mcpDecl.Name.Value)
				}

				if len(mcpDecl.Config) != 2 {
					t.Fatalf("mcpDecl.Config should have 2 entries. got=%d", len(mcpDecl.Config))
				}

				// Check command
				commandVal, ok := mcpDecl.Config["command"]
				if !ok {
					t.Fatalf("mcpDecl.Config missing 'command' key")
				}

				strLit, ok := commandVal.(*StringLiteral)
				if !ok {
					t.Fatalf("command value is not *StringLiteral. got=%T", commandVal)
				}

				if strLit.Value != "npx" {
					t.Errorf("command value not 'npx'. got=%q", strLit.Value)
				}

				// Check args
				argsVal, ok := mcpDecl.Config["args"]
				if !ok {
					t.Fatalf("mcpDecl.Config missing 'args' key")
				}

				arrayLit, ok := argsVal.(*ArrayLiteral)
				if !ok {
					t.Fatalf("args value is not *ArrayLiteral. got=%T", argsVal)
				}

				if len(arrayLit.Elements) != 2 {
					t.Fatalf("args array should have 2 elements. got=%d", len(arrayLit.Elements))
				}

				arg0, ok := arrayLit.Elements[0].(*StringLiteral)
				if !ok {
					t.Fatalf("args[0] is not *StringLiteral. got=%T", arrayLit.Elements[0])
				}
				if arg0.Value != "-y" {
					t.Errorf("args[0] not '-y'. got=%q", arg0.Value)
				}

				arg1, ok := arrayLit.Elements[1].(*StringLiteral)
				if !ok {
					t.Fatalf("args[1] is not *StringLiteral. got=%T", arrayLit.Elements[1])
				}
				if arg1.Value != "@modelcontextprotocol/server-filesystem" {
					t.Errorf("args[1] not '@modelcontextprotocol/server-filesystem'. got=%q", arg1.Value)
				}
			},
		},
		{
			name: "MCP declaration with environment variables",
			input: `mcp github {
				command: "npx",
				args: ["-y", "@modelcontextprotocol/server-github"],
				env: {
					GITHUB_TOKEN: env.GITHUB_TOKEN,
				},
			}`,
			expected: func(t *testing.T, stmt Statement) {
				mcpDecl, ok := stmt.(*McpDeclaration)
				if !ok {
					t.Fatalf("stmt is not *McpDeclaration. got=%T", stmt)
				}

				if mcpDecl.Name.Value != "github" {
					t.Errorf("mcpDecl.Name.Value not 'github'. got=%q", mcpDecl.Name.Value)
				}

				if len(mcpDecl.Config) != 3 {
					t.Fatalf("mcpDecl.Config should have 3 entries. got=%d", len(mcpDecl.Config))
				}

				// Check env
				envVal, ok := mcpDecl.Config["env"]
				if !ok {
					t.Fatalf("mcpDecl.Config missing 'env' key")
				}

				objLit, ok := envVal.(*ObjectLiteral)
				if !ok {
					t.Fatalf("env value is not *ObjectLiteral. got=%T", envVal)
				}

				if len(objLit.Pairs) != 1 {
					t.Fatalf("env object should have 1 pair. got=%d", len(objLit.Pairs))
				}

				tokenVal, ok := objLit.Pairs["GITHUB_TOKEN"]
				if !ok {
					t.Fatalf("env object missing 'GITHUB_TOKEN' key")
				}

				memberExpr, ok := tokenVal.(*MemberExpression)
				if !ok {
					t.Fatalf("GITHUB_TOKEN value is not *MemberExpression. got=%T", tokenVal)
				}

				if memberExpr.Object.String() != "env" {
					t.Errorf("member object not 'env'. got=%q", memberExpr.Object.String())
				}

				if memberExpr.Property.Value != "GITHUB_TOKEN" {
					t.Errorf("member property not 'GITHUB_TOKEN'. got=%q", memberExpr.Property.Value)
				}
			},
		},
		{
			name: "MCP declaration with remote URL",
			input: `mcp database {
				url: "http://localhost:3000/mcp",
				headers: {
					Authorization: "Bearer token123",
				},
			}`,
			expected: func(t *testing.T, stmt Statement) {
				mcpDecl, ok := stmt.(*McpDeclaration)
				if !ok {
					t.Fatalf("stmt is not *McpDeclaration. got=%T", stmt)
				}

				if mcpDecl.Name.Value != "database" {
					t.Errorf("mcpDecl.Name.Value not 'database'. got=%q", mcpDecl.Name.Value)
				}

				if len(mcpDecl.Config) != 2 {
					t.Fatalf("mcpDecl.Config should have 2 entries. got=%d", len(mcpDecl.Config))
				}

				// Check url
				urlVal, ok := mcpDecl.Config["url"]
				if !ok {
					t.Fatalf("mcpDecl.Config missing 'url' key")
				}

				strLit, ok := urlVal.(*StringLiteral)
				if !ok {
					t.Fatalf("url value is not *StringLiteral. got=%T", urlVal)
				}

				if strLit.Value != "http://localhost:3000/mcp" {
					t.Errorf("url value not 'http://localhost:3000/mcp'. got=%q", strLit.Value)
				}

				// Check headers
				headersVal, ok := mcpDecl.Config["headers"]
				if !ok {
					t.Fatalf("mcpDecl.Config missing 'headers' key")
				}

				objLit, ok := headersVal.(*ObjectLiteral)
				if !ok {
					t.Fatalf("headers value is not *ObjectLiteral. got=%T", headersVal)
				}

				if len(objLit.Pairs) != 1 {
					t.Fatalf("headers object should have 1 pair. got=%d", len(objLit.Pairs))
				}
			},
		},
		{
			name:  "MCP declaration with template literal in args",
			input: "mcp filesystem {\n\tcommand: \"npx\",\n\targs: [\"-y\", \"@modelcontextprotocol/server-filesystem\", `/home/${env.USER}/project`],\n}",
			expected: func(t *testing.T, stmt Statement) {
				mcpDecl, ok := stmt.(*McpDeclaration)
				if !ok {
					t.Fatalf("stmt is not *McpDeclaration. got=%T", stmt)
				}

				if mcpDecl.Name.Value != "filesystem" {
					t.Errorf("mcpDecl.Name.Value not 'filesystem'. got=%q", mcpDecl.Name.Value)
				}

				argsVal, ok := mcpDecl.Config["args"]
				if !ok {
					t.Fatalf("mcpDecl.Config missing 'args' key")
				}

				arrayLit, ok := argsVal.(*ArrayLiteral)
				if !ok {
					t.Fatalf("args value is not *ArrayLiteral. got=%T", argsVal)
				}

				if len(arrayLit.Elements) != 3 {
					t.Fatalf("args array should have 3 elements. got=%d", len(arrayLit.Elements))
				}

				// Check that the third element is a template literal (stored as StringLiteral by lexer)
				arg2, ok := arrayLit.Elements[2].(*StringLiteral)
				if !ok {
					t.Fatalf("args[2] is not *StringLiteral. got=%T", arrayLit.Elements[2])
				}

				// Template literal value contains interpolation placeholder
				if arg2.Value != "/home/${env.USER}/project" {
					t.Errorf("args[2] not '/home/${env.USER}/project'. got=%q", arg2.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			checkParserErrors(t, p)

			if len(program.Statements) != 1 {
				t.Fatalf("program.Statements does not contain 1 statement. got=%d",
					len(program.Statements))
			}

			tt.expected(t, program.Statements[0])
		})
	}
}

func TestMcpDeclarationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "missing MCP name",
			input:         `mcp { command: "npx" }`,
			expectedError: "expected next token to be IDENT",
		},
		{
			name:          "missing opening brace",
			input:         `mcp filesystem command: "npx" }`,
			expectedError: "expected next token to be LBRACE",
		},
		{
			name:          "missing closing brace",
			input:         `mcp filesystem { command: "npx"`,
			expectedError: "expected '}'",
		},
		{
			name:          "invalid config key",
			input:         `mcp filesystem { 123: "npx" }`,
			expectedError: "expected identifier for config key",
		},
		{
			name:          "missing colon after key",
			input:         `mcp filesystem { command "npx" }`,
			expectedError: "expected next token to be COLON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			p.ParseProgram()

			errors := p.Errors()
			if len(errors) == 0 {
				t.Fatalf("expected parser errors, got none")
			}

			found := false
			for _, err := range errors {
				if contains(err, tt.expectedError) {
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

func TestMultipleMcpDeclarations(t *testing.T) {
	input := `
	mcp filesystem {
		command: "npx",
		args: ["-y", "@modelcontextprotocol/server-filesystem"],
	}

	mcp github {
		command: "npx",
		args: ["-y", "@modelcontextprotocol/server-github"],
		env: {
			GITHUB_TOKEN: env.GITHUB_TOKEN,
		},
	}
	`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Statements) != 2 {
		t.Fatalf("program.Statements does not contain 2 statements. got=%d",
			len(program.Statements))
	}

	// Check first declaration
	mcpDecl1, ok := program.Statements[0].(*McpDeclaration)
	if !ok {
		t.Fatalf("program.Statements[0] is not *McpDeclaration. got=%T", program.Statements[0])
	}

	if mcpDecl1.Name.Value != "filesystem" {
		t.Errorf("mcpDecl1.Name.Value not 'filesystem'. got=%q", mcpDecl1.Name.Value)
	}

	// Check second declaration
	mcpDecl2, ok := program.Statements[1].(*McpDeclaration)
	if !ok {
		t.Fatalf("program.Statements[1] is not *McpDeclaration. got=%T", program.Statements[1])
	}

	if mcpDecl2.Name.Value != "github" {
		t.Errorf("mcpDecl2.Name.Value not 'github'. got=%q", mcpDecl2.Name.Value)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}
