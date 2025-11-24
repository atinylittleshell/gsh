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

func TestParseModelDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(t *testing.T, stmt Statement)
	}{
		{
			name:  "Basic model declaration with Anthropic",
			input: "model claude {\n\tprovider: \"anthropic\",\n\tapiKey: env.ANTHROPIC_API_KEY,\n\tmodel: \"claude-3-5-sonnet-20241022\",\n}",
			expected: func(t *testing.T, stmt Statement) {
				modelDecl, ok := stmt.(*ModelDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ModelDeclaration. got=%T", stmt)
				}

				if modelDecl.Name.Value != "claude" {
					t.Errorf("modelDecl.Name.Value not 'claude'. got=%q", modelDecl.Name.Value)
				}

				if len(modelDecl.Config) != 3 {
					t.Errorf("modelDecl.Config should have 3 keys. got=%d", len(modelDecl.Config))
				}

				// Check provider
				providerVal, ok := modelDecl.Config["provider"]
				if !ok {
					t.Fatalf("modelDecl.Config missing 'provider' key")
				}
				providerStr, ok := providerVal.(*StringLiteral)
				if !ok {
					t.Fatalf("provider value is not *StringLiteral. got=%T", providerVal)
				}
				if providerStr.Value != "anthropic" {
					t.Errorf("provider not 'anthropic'. got=%q", providerStr.Value)
				}

				// Check apiKey
				apiKeyVal, ok := modelDecl.Config["apiKey"]
				if !ok {
					t.Fatalf("modelDecl.Config missing 'apiKey' key")
				}
				apiKeyMember, ok := apiKeyVal.(*MemberExpression)
				if !ok {
					t.Fatalf("apiKey value is not *MemberExpression. got=%T", apiKeyVal)
				}
				envIdent, ok := apiKeyMember.Object.(*Identifier)
				if !ok || envIdent.Value != "env" {
					t.Errorf("apiKey object not 'env'. got=%v", apiKeyMember.Object)
				}
				if apiKeyMember.Property.Value != "ANTHROPIC_API_KEY" {
					t.Errorf("apiKey property not 'ANTHROPIC_API_KEY'. got=%q", apiKeyMember.Property.Value)
				}

				// Check model
				modelVal, ok := modelDecl.Config["model"]
				if !ok {
					t.Fatalf("modelDecl.Config missing 'model' key")
				}
				modelStr, ok := modelVal.(*StringLiteral)
				if !ok {
					t.Fatalf("model value is not *StringLiteral. got=%T", modelVal)
				}
				if modelStr.Value != "claude-3-5-sonnet-20241022" {
					t.Errorf("model not 'claude-3-5-sonnet-20241022'. got=%q", modelStr.Value)
				}
			},
		},
		{
			name:  "Model declaration with OpenAI",
			input: "model gpt4 {\n\tprovider: \"openai\",\n\tapiKey: env.OPENAI_API_KEY,\n\tmodel: \"gpt-4\",\n\ttemperature: 0.5,\n}",
			expected: func(t *testing.T, stmt Statement) {
				modelDecl, ok := stmt.(*ModelDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ModelDeclaration. got=%T", stmt)
				}

				if modelDecl.Name.Value != "gpt4" {
					t.Errorf("modelDecl.Name.Value not 'gpt4'. got=%q", modelDecl.Name.Value)
				}

				if len(modelDecl.Config) != 4 {
					t.Errorf("modelDecl.Config should have 4 keys. got=%d", len(modelDecl.Config))
				}

				// Check temperature
				tempVal, ok := modelDecl.Config["temperature"]
				if !ok {
					t.Fatalf("modelDecl.Config missing 'temperature' key")
				}
				tempNum, ok := tempVal.(*NumberLiteral)
				if !ok {
					t.Fatalf("temperature value is not *NumberLiteral. got=%T", tempVal)
				}
				if tempNum.Value != "0.5" {
					t.Errorf("temperature not '0.5'. got=%q", tempNum.Value)
				}
			},
		},
		{
			name:  "Model declaration with Ollama (local)",
			input: "model llama {\n\tprovider: \"ollama\",\n\turl: \"http://localhost:11434\",\n\tmodel: \"llama3.2:3b\",\n}",
			expected: func(t *testing.T, stmt Statement) {
				modelDecl, ok := stmt.(*ModelDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ModelDeclaration. got=%T", stmt)
				}

				if modelDecl.Name.Value != "llama" {
					t.Errorf("modelDecl.Name.Value not 'llama'. got=%q", modelDecl.Name.Value)
				}

				// Check url
				urlVal, ok := modelDecl.Config["url"]
				if !ok {
					t.Fatalf("modelDecl.Config missing 'url' key")
				}
				urlStr, ok := urlVal.(*StringLiteral)
				if !ok {
					t.Fatalf("url value is not *StringLiteral. got=%T", urlVal)
				}
				if urlStr.Value != "http://localhost:11434" {
					t.Errorf("url not 'http://localhost:11434'. got=%q", urlStr.Value)
				}
			},
		},
		{
			name:  "Model declaration with minimal config",
			input: "model minimal {\n\tprovider: \"anthropic\",\n}",
			expected: func(t *testing.T, stmt Statement) {
				modelDecl, ok := stmt.(*ModelDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ModelDeclaration. got=%T", stmt)
				}

				if modelDecl.Name.Value != "minimal" {
					t.Errorf("modelDecl.Name.Value not 'minimal'. got=%q", modelDecl.Name.Value)
				}

				if len(modelDecl.Config) != 1 {
					t.Errorf("modelDecl.Config should have 1 key. got=%d", len(modelDecl.Config))
				}
			},
		},
		{
			name:  "Model declaration with template literal",
			input: "model dynamic {\n\tprovider: \"openai\",\n\tapiKey: `Bearer ${env.TOKEN}`,\n}",
			expected: func(t *testing.T, stmt Statement) {
				modelDecl, ok := stmt.(*ModelDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ModelDeclaration. got=%T", stmt)
				}

				if modelDecl.Name.Value != "dynamic" {
					t.Errorf("modelDecl.Name.Value not 'dynamic'. got=%q", modelDecl.Name.Value)
				}

				// Check apiKey with template literal
				apiKeyVal, ok := modelDecl.Config["apiKey"]
				if !ok {
					t.Fatalf("modelDecl.Config missing 'apiKey' key")
				}
				apiKeyStr, ok := apiKeyVal.(*StringLiteral)
				if !ok {
					t.Fatalf("apiKey value is not *StringLiteral. got=%T", apiKeyVal)
				}
				if apiKeyStr.Value != "Bearer ${env.TOKEN}" {
					t.Errorf("apiKey not 'Bearer ${env.TOKEN}'. got=%q", apiKeyStr.Value)
				}
			},
		},
		{
			name:  "Model declaration without trailing comma",
			input: "model nocomma {\n\tprovider: \"anthropic\"\n}",
			expected: func(t *testing.T, stmt Statement) {
				modelDecl, ok := stmt.(*ModelDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ModelDeclaration. got=%T", stmt)
				}

				if modelDecl.Name.Value != "nocomma" {
					t.Errorf("modelDecl.Name.Value not 'nocomma'. got=%q", modelDecl.Name.Value)
				}

				if len(modelDecl.Config) != 1 {
					t.Errorf("modelDecl.Config should have 1 key. got=%d", len(modelDecl.Config))
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
				t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
			}

			tt.expected(t, program.Statements[0])
		})
	}
}

func TestParseModelDeclarationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "Model declaration without name",
			input:         "model { provider: \"anthropic\" }",
			expectedError: "expected next token to be",
		},
		{
			name:          "Model declaration without opening brace",
			input:         "model claude provider: \"anthropic\" }",
			expectedError: "expected next token to be",
		},
		{
			name:          "Model declaration without closing brace",
			input:         "model claude { provider: \"anthropic\"",
			expectedError: "expected '}'",
		},
		{
			name:          "Model declaration with invalid config key",
			input:         "model claude { 123: \"value\" }",
			expectedError: "expected identifier for config key",
		},
		{
			name:          "Model declaration without colon after key",
			input:         "model claude { provider \"anthropic\" }",
			expectedError: "expected next token to be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			_ = p.ParseProgram()

			errors := p.Errors()
			if len(errors) == 0 {
				t.Fatalf("expected parser errors, but got none")
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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}
