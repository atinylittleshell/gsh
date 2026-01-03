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
			expectedError: "expected next token to be identifier",
		},
		{
			name:          "missing opening brace",
			input:         `mcp filesystem command: "npx" }`,
			expectedError: "expected next token to be '{'",
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
			expectedError: "expected next token to be ':'",
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
			input: "model llama {\n\tprovider: \"openai\",\n\tapiKey: \"ollama\",\n\tbaseURL: \"http://localhost:11434/v1\",\n\tmodel: \"llama3.2:3b\",\n}",
			expected: func(t *testing.T, stmt Statement) {
				modelDecl, ok := stmt.(*ModelDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ModelDeclaration. got=%T", stmt)
				}

				if modelDecl.Name.Value != "llama" {
					t.Errorf("modelDecl.Name.Value not 'llama'. got=%q", modelDecl.Name.Value)
				}

				// Check baseURL
				baseURLVal, ok := modelDecl.Config["baseURL"]
				if !ok {
					t.Fatalf("modelDecl.Config missing 'baseURL' key")
				}
				baseURLStr, ok := baseURLVal.(*StringLiteral)
				if !ok {
					t.Fatalf("baseURL value is not *StringLiteral. got=%T", baseURLVal)
				}
				if baseURLStr.Value != "http://localhost:11434/v1" {
					t.Errorf("baseURL not 'http://localhost:11434/v1'. got=%q", baseURLStr.Value)
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

func TestParseAgentDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(t *testing.T, stmt Statement)
	}{
		{
			name: "Basic agent declaration with model and systemPrompt",
			input: `agent DataAnalyst {
	model: claude,
	systemPrompt: "You are a data analyst",
}`,
			expected: func(t *testing.T, stmt Statement) {
				agentDecl, ok := stmt.(*AgentDeclaration)
				if !ok {
					t.Fatalf("stmt is not *AgentDeclaration. got=%T", stmt)
				}

				if agentDecl.Name.Value != "DataAnalyst" {
					t.Errorf("agentDecl.Name.Value not 'DataAnalyst'. got=%q", agentDecl.Name.Value)
				}

				if len(agentDecl.Config) != 2 {
					t.Errorf("agentDecl.Config should have 2 keys. got=%d", len(agentDecl.Config))
				}

				// Check model
				modelVal, ok := agentDecl.Config["model"]
				if !ok {
					t.Fatalf("agentDecl.Config missing 'model' key")
				}
				modelIdent, ok := modelVal.(*Identifier)
				if !ok {
					t.Fatalf("model value is not *Identifier. got=%T", modelVal)
				}
				if modelIdent.Value != "claude" {
					t.Errorf("model not 'claude'. got=%q", modelIdent.Value)
				}

				// Check systemPrompt
				promptVal, ok := agentDecl.Config["systemPrompt"]
				if !ok {
					t.Fatalf("agentDecl.Config missing 'systemPrompt' key")
				}
				promptStr, ok := promptVal.(*StringLiteral)
				if !ok {
					t.Fatalf("systemPrompt value is not *StringLiteral. got=%T", promptVal)
				}
				if promptStr.Value != "You are a data analyst" {
					t.Errorf("systemPrompt not 'You are a data analyst'. got=%q", promptStr.Value)
				}
			},
		},
		{
			name: "Agent declaration with tools array",
			input: `agent Helper {
	model: gpt4,
	systemPrompt: "You help users",
	tools: [filesystem.read_file, filesystem.write_file, analyzeData],
}`,
			expected: func(t *testing.T, stmt Statement) {
				agentDecl, ok := stmt.(*AgentDeclaration)
				if !ok {
					t.Fatalf("stmt is not *AgentDeclaration. got=%T", stmt)
				}

				if agentDecl.Name.Value != "Helper" {
					t.Errorf("agentDecl.Name.Value not 'Helper'. got=%q", agentDecl.Name.Value)
				}

				// Check tools array
				toolsVal, ok := agentDecl.Config["tools"]
				if !ok {
					t.Fatalf("agentDecl.Config missing 'tools' key")
				}
				toolsArray, ok := toolsVal.(*ArrayLiteral)
				if !ok {
					t.Fatalf("tools value is not *ArrayLiteral. got=%T", toolsVal)
				}
				if len(toolsArray.Elements) != 3 {
					t.Fatalf("tools array should have 3 elements. got=%d", len(toolsArray.Elements))
				}

				// Check first tool (MemberExpression)
				tool0, ok := toolsArray.Elements[0].(*MemberExpression)
				if !ok {
					t.Fatalf("tools[0] is not *MemberExpression. got=%T", toolsArray.Elements[0])
				}
				obj0, ok := tool0.Object.(*Identifier)
				if !ok || obj0.Value != "filesystem" {
					t.Errorf("tools[0] object not 'filesystem'. got=%v", tool0.Object)
				}
				if tool0.Property.Value != "read_file" {
					t.Errorf("tools[0] property not 'read_file'. got=%q", tool0.Property.Value)
				}

				// Check third tool (Identifier)
				tool2, ok := toolsArray.Elements[2].(*Identifier)
				if !ok {
					t.Fatalf("tools[2] is not *Identifier. got=%T", toolsArray.Elements[2])
				}
				if tool2.Value != "analyzeData" {
					t.Errorf("tools[2] not 'analyzeData'. got=%q", tool2.Value)
				}
			},
		},
		{
			name: "Agent declaration with multiline string and temperature override",
			input: `agent Analyst {
	model: claude,
	systemPrompt: """
		You are a data analyst. Analyze the provided data
		and generate insights using the available tools.
	""",
	tools: [analyzeData, formatReport],
	temperature: 0.5,
}`,
			expected: func(t *testing.T, stmt Statement) {
				agentDecl, ok := stmt.(*AgentDeclaration)
				if !ok {
					t.Fatalf("stmt is not *AgentDeclaration. got=%T", stmt)
				}

				if agentDecl.Name.Value != "Analyst" {
					t.Errorf("agentDecl.Name.Value not 'Analyst'. got=%q", agentDecl.Name.Value)
				}

				if len(agentDecl.Config) != 4 {
					t.Errorf("agentDecl.Config should have 4 keys. got=%d", len(agentDecl.Config))
				}

				// Check systemPrompt (multiline string)
				promptVal, ok := agentDecl.Config["systemPrompt"]
				if !ok {
					t.Fatalf("agentDecl.Config missing 'systemPrompt' key")
				}
				promptStr, ok := promptVal.(*StringLiteral)
				if !ok {
					t.Fatalf("systemPrompt value is not *StringLiteral. got=%T", promptVal)
				}
				// The multiline string should contain the text (whitespace handling done by lexer)
				if !contains(promptStr.Value, "You are a data analyst") {
					t.Errorf("systemPrompt doesn't contain expected text. got=%q", promptStr.Value)
				}

				// Check temperature
				tempVal, ok := agentDecl.Config["temperature"]
				if !ok {
					t.Fatalf("agentDecl.Config missing 'temperature' key")
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
			name: "Agent declaration with minimal config",
			input: `agent Minimal {
	model: gpt4,
}`,
			expected: func(t *testing.T, stmt Statement) {
				agentDecl, ok := stmt.(*AgentDeclaration)
				if !ok {
					t.Fatalf("stmt is not *AgentDeclaration. got=%T", stmt)
				}

				if agentDecl.Name.Value != "Minimal" {
					t.Errorf("agentDecl.Name.Value not 'Minimal'. got=%q", agentDecl.Name.Value)
				}

				if len(agentDecl.Config) != 1 {
					t.Errorf("agentDecl.Config should have 1 key. got=%d", len(agentDecl.Config))
				}
			},
		},
		{
			name: "Agent declaration without trailing comma",
			input: `agent NoComma {
	model: claude
}`,
			expected: func(t *testing.T, stmt Statement) {
				agentDecl, ok := stmt.(*AgentDeclaration)
				if !ok {
					t.Fatalf("stmt is not *AgentDeclaration. got=%T", stmt)
				}

				if agentDecl.Name.Value != "NoComma" {
					t.Errorf("agentDecl.Name.Value not 'NoComma'. got=%q", agentDecl.Name.Value)
				}

				if len(agentDecl.Config) != 1 {
					t.Errorf("agentDecl.Config should have 1 key. got=%d", len(agentDecl.Config))
				}
			},
		},
		{
			name: "Agent declaration with template literal in systemPrompt",
			input: `agent Dynamic {
	model: gpt4,
	systemPrompt: ` + "`You are a ${role} assistant`" + `,
}`,
			expected: func(t *testing.T, stmt Statement) {
				agentDecl, ok := stmt.(*AgentDeclaration)
				if !ok {
					t.Fatalf("stmt is not *AgentDeclaration. got=%T", stmt)
				}

				if agentDecl.Name.Value != "Dynamic" {
					t.Errorf("agentDecl.Name.Value not 'Dynamic'. got=%q", agentDecl.Name.Value)
				}

				// Check systemPrompt with template literal
				promptVal, ok := agentDecl.Config["systemPrompt"]
				if !ok {
					t.Fatalf("agentDecl.Config missing 'systemPrompt' key")
				}
				promptStr, ok := promptVal.(*StringLiteral)
				if !ok {
					t.Fatalf("systemPrompt value is not *StringLiteral. got=%T", promptVal)
				}
				if promptStr.Value != "You are a ${role} assistant" {
					t.Errorf("systemPrompt not 'You are a ${role} assistant'. got=%q", promptStr.Value)
				}
			},
		},
		{
			name: "Agent declaration from spec example",
			input: `agent PRAnalyzer {
	model: claude,
	systemPrompt: "You analyze pull requests for code quality and issues",
	tools: [github.get_pull_request_diff],
}`,
			expected: func(t *testing.T, stmt Statement) {
				agentDecl, ok := stmt.(*AgentDeclaration)
				if !ok {
					t.Fatalf("stmt is not *AgentDeclaration. got=%T", stmt)
				}

				if agentDecl.Name.Value != "PRAnalyzer" {
					t.Errorf("agentDecl.Name.Value not 'PRAnalyzer'. got=%q", agentDecl.Name.Value)
				}

				if len(agentDecl.Config) != 3 {
					t.Errorf("agentDecl.Config should have 3 keys. got=%d", len(agentDecl.Config))
				}

				// Verify tools array with MCP tool
				toolsVal, ok := agentDecl.Config["tools"]
				if !ok {
					t.Fatalf("agentDecl.Config missing 'tools' key")
				}
				toolsArray, ok := toolsVal.(*ArrayLiteral)
				if !ok {
					t.Fatalf("tools value is not *ArrayLiteral. got=%T", toolsVal)
				}
				if len(toolsArray.Elements) != 1 {
					t.Fatalf("tools array should have 1 element. got=%d", len(toolsArray.Elements))
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

func TestParseAgentDeclarationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "Agent declaration without name",
			input:         "agent { model: claude }",
			expectedError: "expected next token to be",
		},
		{
			name:          "Agent declaration without opening brace",
			input:         "agent DataAnalyst model: claude }",
			expectedError: "expected next token to be",
		},
		{
			name:          "Agent declaration without closing brace",
			input:         "agent DataAnalyst { model: claude",
			expectedError: "expected '}'",
		},
		{
			name:          "Agent declaration with invalid config key",
			input:         "agent DataAnalyst { 123: claude }",
			expectedError: "expected identifier for config key",
		},
		{
			name:          "Agent declaration without colon after key",
			input:         "agent DataAnalyst { model claude }",
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

func TestParseACPDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(t *testing.T, stmt Statement)
	}{
		{
			name: "Basic ACP declaration with command and args",
			input: `acp RovoDev {
	command: "acli",
	args: ["rovodev", "acp"],
}`,
			expected: func(t *testing.T, stmt Statement) {
				acpDecl, ok := stmt.(*ACPDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ACPDeclaration. got=%T", stmt)
				}

				if acpDecl.Name.Value != "RovoDev" {
					t.Errorf("acpDecl.Name.Value not 'RovoDev'. got=%q", acpDecl.Name.Value)
				}

				if len(acpDecl.Config) != 2 {
					t.Errorf("acpDecl.Config should have 2 keys. got=%d", len(acpDecl.Config))
				}

				// Check command
				cmdVal, ok := acpDecl.Config["command"]
				if !ok {
					t.Fatalf("acpDecl.Config missing 'command' key")
				}
				cmdStr, ok := cmdVal.(*StringLiteral)
				if !ok {
					t.Fatalf("command value is not *StringLiteral. got=%T", cmdVal)
				}
				if cmdStr.Value != "acli" {
					t.Errorf("command not 'acli'. got=%q", cmdStr.Value)
				}

				// Check args
				argsVal, ok := acpDecl.Config["args"]
				if !ok {
					t.Fatalf("acpDecl.Config missing 'args' key")
				}
				argsArray, ok := argsVal.(*ArrayLiteral)
				if !ok {
					t.Fatalf("args value is not *ArrayLiteral. got=%T", argsVal)
				}
				if len(argsArray.Elements) != 2 {
					t.Fatalf("args array should have 2 elements. got=%d", len(argsArray.Elements))
				}
			},
		},
		{
			name: "ACP declaration with environment variables",
			input: `acp RovoDev {
	command: "acli",
	args: ["rovodev", "acp"],
	env: {
		ATLASSIAN_TOKEN: env.ATLASSIAN_TOKEN,
	},
}`,
			expected: func(t *testing.T, stmt Statement) {
				acpDecl, ok := stmt.(*ACPDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ACPDeclaration. got=%T", stmt)
				}

				if acpDecl.Name.Value != "RovoDev" {
					t.Errorf("acpDecl.Name.Value not 'RovoDev'. got=%q", acpDecl.Name.Value)
				}

				if len(acpDecl.Config) != 3 {
					t.Errorf("acpDecl.Config should have 3 keys. got=%d", len(acpDecl.Config))
				}

				// Check env
				envVal, ok := acpDecl.Config["env"]
				if !ok {
					t.Fatalf("acpDecl.Config missing 'env' key")
				}
				envObj, ok := envVal.(*ObjectLiteral)
				if !ok {
					t.Fatalf("env value is not *ObjectLiteral. got=%T", envVal)
				}
				if len(envObj.Pairs) != 1 {
					t.Errorf("env object should have 1 key. got=%d", len(envObj.Pairs))
				}
			},
		},
		{
			name: "ACP declaration with working directory",
			input: `acp RovoDev {
	command: "acli",
	args: ["rovodev", "acp"],
	cwd: "/path/to/project",
}`,
			expected: func(t *testing.T, stmt Statement) {
				acpDecl, ok := stmt.(*ACPDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ACPDeclaration. got=%T", stmt)
				}

				if acpDecl.Name.Value != "RovoDev" {
					t.Errorf("acpDecl.Name.Value not 'RovoDev'. got=%q", acpDecl.Name.Value)
				}

				// Check cwd
				cwdVal, ok := acpDecl.Config["cwd"]
				if !ok {
					t.Fatalf("acpDecl.Config missing 'cwd' key")
				}
				cwdStr, ok := cwdVal.(*StringLiteral)
				if !ok {
					t.Fatalf("cwd value is not *StringLiteral. got=%T", cwdVal)
				}
				if cwdStr.Value != "/path/to/project" {
					t.Errorf("cwd not '/path/to/project'. got=%q", cwdStr.Value)
				}
			},
		},
		{
			name: "ACP declaration without trailing comma",
			input: `acp RovoDev {
	command: "acli"
}`,
			expected: func(t *testing.T, stmt Statement) {
				acpDecl, ok := stmt.(*ACPDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ACPDeclaration. got=%T", stmt)
				}

				if acpDecl.Name.Value != "RovoDev" {
					t.Errorf("acpDecl.Name.Value not 'RovoDev'. got=%q", acpDecl.Name.Value)
				}

				if len(acpDecl.Config) != 1 {
					t.Errorf("acpDecl.Config should have 1 key. got=%d", len(acpDecl.Config))
				}
			},
		},
		{
			name: "ACP declaration with MCP servers",
			input: `acp RovoDev {
	command: "acli",
	args: ["rovodev", "acp"],
	mcpServers: [filesystem],
}`,
			expected: func(t *testing.T, stmt Statement) {
				acpDecl, ok := stmt.(*ACPDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ACPDeclaration. got=%T", stmt)
				}

				// Check mcpServers
				mcpVal, ok := acpDecl.Config["mcpServers"]
				if !ok {
					t.Fatalf("acpDecl.Config missing 'mcpServers' key")
				}
				mcpArray, ok := mcpVal.(*ArrayLiteral)
				if !ok {
					t.Fatalf("mcpServers value is not *ArrayLiteral. got=%T", mcpVal)
				}
				if len(mcpArray.Elements) != 1 {
					t.Fatalf("mcpServers array should have 1 element. got=%d", len(mcpArray.Elements))
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

func TestParseACPDeclarationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "ACP declaration without name",
			input:         "acp { command: \"acli\" }",
			expectedError: "expected next token to be",
		},
		{
			name:          "ACP declaration without opening brace",
			input:         "acp RovoDev command: \"acli\" }",
			expectedError: "expected next token to be",
		},
		{
			name:          "ACP declaration without closing brace",
			input:         "acp RovoDev { command: \"acli\"",
			expectedError: "expected '}'",
		},
		{
			name:          "ACP declaration with invalid config key",
			input:         "acp RovoDev { 123: \"acli\" }",
			expectedError: "expected identifier for config key",
		},
		{
			name:          "ACP declaration without colon after key",
			input:         "acp RovoDev { command \"acli\" }",
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

func TestParseToolDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(t *testing.T, stmt Statement)
	}{
		{
			name: "Basic tool declaration without parameters",
			input: `tool hello() {
	print("Hello, world!")
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "hello" {
					t.Errorf("toolDecl.Name.Value not 'hello'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 0 {
					t.Errorf("toolDecl.Parameters should be empty. got=%d", len(toolDecl.Parameters))
				}

				if toolDecl.ReturnType != nil {
					t.Errorf("toolDecl.ReturnType should be nil. got=%v", toolDecl.ReturnType)
				}

				if toolDecl.Body == nil {
					t.Fatal("toolDecl.Body is nil")
				}
			},
		},
		{
			name: "Tool declaration with single parameter without type",
			input: `tool processData(input) {
	content = filesystem.read_file(input)
	return JSON.parse(content)
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "processData" {
					t.Errorf("toolDecl.Name.Value not 'processData'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 1 {
					t.Fatalf("toolDecl.Parameters should have 1 parameter. got=%d", len(toolDecl.Parameters))
				}

				param := toolDecl.Parameters[0]
				if param.Name.Value != "input" {
					t.Errorf("param.Name.Value not 'input'. got=%q", param.Name.Value)
				}

				if param.Type != nil {
					t.Errorf("param.Type should be nil. got=%v", param.Type)
				}

				if toolDecl.ReturnType != nil {
					t.Errorf("toolDecl.ReturnType should be nil. got=%v", toolDecl.ReturnType)
				}
			},
		},
		{
			name: "Tool declaration with typed parameters",
			input: `tool calculateScore(points: number, multiplier: number) {
	return points * multiplier
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "calculateScore" {
					t.Errorf("toolDecl.Name.Value not 'calculateScore'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 2 {
					t.Fatalf("toolDecl.Parameters should have 2 parameters. got=%d", len(toolDecl.Parameters))
				}

				// Check first parameter
				param0 := toolDecl.Parameters[0]
				if param0.Name.Value != "points" {
					t.Errorf("param0.Name.Value not 'points'. got=%q", param0.Name.Value)
				}
				if param0.Type == nil {
					t.Fatal("param0.Type is nil")
				}
				if param0.Type.Value != "number" {
					t.Errorf("param0.Type.Value not 'number'. got=%q", param0.Type.Value)
				}

				// Check second parameter
				param1 := toolDecl.Parameters[1]
				if param1.Name.Value != "multiplier" {
					t.Errorf("param1.Name.Value not 'multiplier'. got=%q", param1.Name.Value)
				}
				if param1.Type == nil {
					t.Fatal("param1.Type is nil")
				}
				if param1.Type.Value != "number" {
					t.Errorf("param1.Type.Value not 'number'. got=%q", param1.Type.Value)
				}

				if toolDecl.ReturnType != nil {
					t.Errorf("toolDecl.ReturnType should be nil. got=%v", toolDecl.ReturnType)
				}
			},
		},
		{
			name: "Tool declaration with return type",
			input: `tool calculateScore(points: number, multiplier: number): number {
	return points * multiplier
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "calculateScore" {
					t.Errorf("toolDecl.Name.Value not 'calculateScore'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 2 {
					t.Fatalf("toolDecl.Parameters should have 2 parameters. got=%d", len(toolDecl.Parameters))
				}

				if toolDecl.ReturnType == nil {
					t.Fatal("toolDecl.ReturnType is nil")
				}
				if toolDecl.ReturnType.Value != "number" {
					t.Errorf("toolDecl.ReturnType.Value not 'number'. got=%q", toolDecl.ReturnType.Value)
				}
			},
		},
		{
			name: "Tool declaration with mixed typed and untyped parameters",
			input: `tool processFile(path: string, options) {
	return filesystem.read_file(path)
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "processFile" {
					t.Errorf("toolDecl.Name.Value not 'processFile'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 2 {
					t.Fatalf("toolDecl.Parameters should have 2 parameters. got=%d", len(toolDecl.Parameters))
				}

				// Check first parameter (typed)
				param0 := toolDecl.Parameters[0]
				if param0.Name.Value != "path" {
					t.Errorf("param0.Name.Value not 'path'. got=%q", param0.Name.Value)
				}
				if param0.Type == nil {
					t.Fatal("param0.Type is nil")
				}
				if param0.Type.Value != "string" {
					t.Errorf("param0.Type.Value not 'string'. got=%q", param0.Type.Value)
				}

				// Check second parameter (untyped)
				param1 := toolDecl.Parameters[1]
				if param1.Name.Value != "options" {
					t.Errorf("param1.Name.Value not 'options'. got=%q", param1.Name.Value)
				}
				if param1.Type != nil {
					t.Errorf("param1.Type should be nil. got=%v", param1.Type)
				}
			},
		},
		{
			name: "Tool declaration with string return type",
			input: `tool formatReport(content: string): string {
	return "# Report\n\n" + content
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "formatReport" {
					t.Errorf("toolDecl.Name.Value not 'formatReport'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 1 {
					t.Fatalf("toolDecl.Parameters should have 1 parameter. got=%d", len(toolDecl.Parameters))
				}

				if toolDecl.ReturnType == nil {
					t.Fatal("toolDecl.ReturnType is nil")
				}
				if toolDecl.ReturnType.Value != "string" {
					t.Errorf("toolDecl.ReturnType.Value not 'string'. got=%q", toolDecl.ReturnType.Value)
				}
			},
		},
		{
			name: "Tool declaration from spec example",
			input: `tool analyzeData(data: string): string {
	parsed = JSON.parse(data)
	return "Found " + parsed.length + " records"
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "analyzeData" {
					t.Errorf("toolDecl.Name.Value not 'analyzeData'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 1 {
					t.Fatalf("toolDecl.Parameters should have 1 parameter. got=%d", len(toolDecl.Parameters))
				}

				param := toolDecl.Parameters[0]
				if param.Name.Value != "data" {
					t.Errorf("param.Name.Value not 'data'. got=%q", param.Name.Value)
				}
				if param.Type == nil {
					t.Fatal("param.Type is nil")
				}
				if param.Type.Value != "string" {
					t.Errorf("param.Type.Value not 'string'. got=%q", param.Type.Value)
				}

				if toolDecl.ReturnType == nil {
					t.Fatal("toolDecl.ReturnType is nil")
				}
				if toolDecl.ReturnType.Value != "string" {
					t.Errorf("toolDecl.ReturnType.Value not 'string'. got=%q", toolDecl.ReturnType.Value)
				}

				if toolDecl.Body == nil {
					t.Fatal("toolDecl.Body is nil")
				}
				if len(toolDecl.Body.Statements) == 0 {
					t.Error("toolDecl.Body.Statements is empty")
				}
			},
		},
		{
			name: "Tool declaration with any type",
			input: `tool safeProcess(path: string): any {
	try {
		return processFile(path)
	} catch (error) {
		return null
	}
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "safeProcess" {
					t.Errorf("toolDecl.Name.Value not 'safeProcess'. got=%q", toolDecl.Name.Value)
				}

				if toolDecl.ReturnType == nil {
					t.Fatal("toolDecl.ReturnType is nil")
				}
				if toolDecl.ReturnType.Value != "any" {
					t.Errorf("toolDecl.ReturnType.Value not 'any'. got=%q", toolDecl.ReturnType.Value)
				}
			},
		},
		{
			name: "Tool declaration with multiple parameters and complex body",
			input: `tool analyzePR(repo: string, prNumber: number) {
	log.info("Analyzing PR #" + prNumber)
	diff = github.get_pull_request_diff(repo, prNumber)
	return diff
}`,
			expected: func(t *testing.T, stmt Statement) {
				toolDecl, ok := stmt.(*ToolDeclaration)
				if !ok {
					t.Fatalf("stmt is not *ToolDeclaration. got=%T", stmt)
				}

				if toolDecl.Name.Value != "analyzePR" {
					t.Errorf("toolDecl.Name.Value not 'analyzePR'. got=%q", toolDecl.Name.Value)
				}

				if len(toolDecl.Parameters) != 2 {
					t.Fatalf("toolDecl.Parameters should have 2 parameters. got=%d", len(toolDecl.Parameters))
				}

				// Check parameters
				param0 := toolDecl.Parameters[0]
				if param0.Name.Value != "repo" {
					t.Errorf("param0.Name.Value not 'repo'. got=%q", param0.Name.Value)
				}
				if param0.Type == nil {
					t.Fatal("param0.Type is nil")
				}
				if param0.Type.Value != "string" {
					t.Errorf("param0.Type.Value not 'string'. got=%q", param0.Type.Value)
				}

				param1 := toolDecl.Parameters[1]
				if param1.Name.Value != "prNumber" {
					t.Errorf("param1.Name.Value not 'prNumber'. got=%q", param1.Name.Value)
				}
				if param1.Type == nil {
					t.Fatal("param1.Type is nil")
				}
				if param1.Type.Value != "number" {
					t.Errorf("param1.Type.Value not 'number'. got=%q", param1.Type.Value)
				}

				if toolDecl.Body == nil {
					t.Fatal("toolDecl.Body is nil")
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

func TestParseToolDeclarationErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "Tool declaration without name",
			input:         "tool () { print(\"hello\") }",
			expectedError: "expected next token to be",
		},
		{
			name:          "Tool declaration without opening paren",
			input:         "tool hello { print(\"hello\") }",
			expectedError: "expected next token to be",
		},
		{
			name:          "Tool declaration without closing paren",
			input:         "tool hello(x { print(x) }",
			expectedError: "expected ')'",
		},
		{
			name:          "Tool declaration without opening brace",
			input:         "tool hello() print(\"hello\") }",
			expectedError: "expected next token to be",
		},
		{
			name:          "Tool declaration without closing brace",
			input:         "tool hello() { print(\"hello\")",
			expectedError: "expected '}'",
		},
		{
			name:          "Tool declaration with invalid parameter name",
			input:         "tool hello(123) { print(\"hello\") }",
			expectedError: "expected parameter name",
		},
		{
			name:          "Tool declaration with type but no colon",
			input:         "tool hello(x string) { print(x) }",
			expectedError: "expected ')'",
		},
		{
			name:          "Tool declaration with missing type after colon",
			input:         "tool hello(x:) { print(x) }",
			expectedError: "expected type annotation",
		},
		{
			name:          "Tool declaration with missing return type after colon",
			input:         "tool hello(): { print(\"hello\") }",
			expectedError: "expected return type",
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

func TestParseImportStatement(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedPath string
		expectedSyms []string
		expectError  bool
	}{
		{
			name:         "side-effect import",
			input:        `import "./helpers.gsh"`,
			expectedPath: "./helpers.gsh",
			expectedSyms: []string{},
		},
		{
			name:         "selective import single symbol",
			input:        `import { helper } from "./helpers.gsh"`,
			expectedPath: "./helpers.gsh",
			expectedSyms: []string{"helper"},
		},
		{
			name:         "selective import multiple symbols",
			input:        `import { foo, bar, baz } from "./lib.gsh"`,
			expectedPath: "./lib.gsh",
			expectedSyms: []string{"foo", "bar", "baz"},
		},
		{
			name:         "import with relative path",
			input:        `import { config } from "../config.gsh"`,
			expectedPath: "../config.gsh",
			expectedSyms: []string{"config"},
		},
		{
			name:        "missing path",
			input:       `import`,
			expectError: true,
		},
		{
			name:        "missing from keyword",
			input:       `import { foo } "./file.gsh"`,
			expectError: true,
		},
		{
			name:        "missing closing brace",
			input:       `import { foo from "./file.gsh"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if tt.expectError {
				if len(p.Errors()) == 0 {
					t.Fatalf("expected parsing error, got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			importStmt, ok := program.Statements[0].(*ImportStatement)
			if !ok {
				t.Fatalf("expected ImportStatement, got %T", program.Statements[0])
			}

			if importStmt.Path.Value != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, importStmt.Path.Value)
			}

			if len(importStmt.Symbols) != len(tt.expectedSyms) {
				t.Fatalf("expected %d symbols, got %d", len(tt.expectedSyms), len(importStmt.Symbols))
			}

			for i, sym := range tt.expectedSyms {
				if importStmt.Symbols[i] != sym {
					t.Errorf("symbol[%d] expected %q, got %q", i, sym, importStmt.Symbols[i])
				}
			}
		})
	}
}

func TestParseExportStatement(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
		expectError  bool
	}{
		{
			name:         "export variable",
			input:        `export myVar = 42`,
			expectedName: "myVar",
		},
		{
			name:         "export tool",
			input:        "export tool myFunc(x) {\n    return x * 2\n}",
			expectedName: "myFunc",
		},
		{
			name:         "export model",
			input:        "export model myModel {\n    provider: \"openai\",\n    model: \"gpt-4\",\n}",
			expectedName: "myModel",
		},
		{
			name:         "export agent",
			input:        "export agent myAgent {\n    model: myModel,\n}",
			expectedName: "myAgent",
		},
		{
			name:        "export without declaration",
			input:       `export`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if tt.expectError {
				if len(p.Errors()) == 0 {
					t.Fatalf("expected parsing error, got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			exportStmt, ok := program.Statements[0].(*ExportStatement)
			if !ok {
				t.Fatalf("expected ExportStatement, got %T", program.Statements[0])
			}

			if exportStmt.Name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, exportStmt.Name)
			}
		})
	}
}
