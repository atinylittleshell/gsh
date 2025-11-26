package interpreter

import (
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// TestMcpDeclarationParsing tests that MCP declarations are parsed correctly
func TestMcpDeclarationParsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErr       bool
		skipExecution bool // Skip actual MCP server connection
	}{
		{
			name: "basic stdio MCP server",
			input: `
mcp filesystem {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
}
`,
			wantErr:       false,
			skipExecution: true, // Skip actual connection
		},
		{
			name: "MCP server with environment variables",
			input: `
mcp github {
	command: "npx",
	args: ["-y", "@modelcontextprotocol/server-github"],
	env: {
		GITHUB_TOKEN: "test_token",
	},
}
`,
			wantErr:       false,
			skipExecution: true, // Skip actual connection
		},
		{
			name: "MCP server with missing command",
			input: `
mcp invalid {
	args: ["-y", "something"],
}
`,
			wantErr: true,
		},
		{
			name: "duplicate MCP server name",
			input: `
mcp test {
	command: "test",
	args: [],
}
mcp test {
	command: "test2",
	args: [],
}
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Lex and parse the input
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				if !tt.wantErr {
					t.Fatalf("unexpected parse error: %v", p.Errors())
				}
				return
			}

			// For tests that skip execution, just verify parsing works
			if tt.skipExecution {
				// Just verify the program parsed correctly
				if len(program.Statements) == 0 {
					t.Fatal("expected statements, got none")
				}
				return
			}

			// Create interpreter and evaluate
			interp := New()
			defer interp.Close()

			_, err := interp.Eval(program)
			if (err != nil) != tt.wantErr {
				t.Errorf("evalMcpDeclaration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMcpProxyValue tests that MCP proxy values are created correctly
func TestMcpProxyValue(t *testing.T) {
	t.Skip("Skipping test that requires actual MCP server connection")

	input := `
mcp test {
	command: "echo",
	args: ["hello"],
}
`
	// Lex and parse
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}

	// Create interpreter and evaluate
	interp := New()
	defer interp.Close()

	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	// Check that the MCP server is registered in environment
	testVal, ok := interp.env.Get("test")
	if !ok {
		t.Fatal("MCP server 'test' not found in environment")
	}

	// Check that it's an MCP proxy
	proxy, ok := testVal.(*MCPProxyValue)
	if !ok {
		t.Fatalf("expected MCPProxyValue, got %T", testVal)
	}

	if proxy.ServerName != "test" {
		t.Errorf("expected server name 'test', got '%s'", proxy.ServerName)
	}

	// Check the result
	if result.FinalResult.Type() != ValueTypeObject {
		t.Errorf("expected object type result, got %s", result.FinalResult.Type())
	}
}

// TestMcpConfigValidation tests that MCP config fields are validated correctly
func TestMcpConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name: "command must be string",
			input: `
mcp test {
	command: 123,
	args: [],
}
`,
			wantErr: "must be a string",
		},
		{
			name: "args must be array",
			input: `
mcp test {
	command: "test",
	args: "not-an-array",
}
`,
			wantErr: "must be an array",
		},
		{
			name: "args must be array of strings",
			input: `
mcp test {
	command: "test",
	args: [1, 2, 3],
}
`,
			wantErr: "must be an array of strings",
		},
		{
			name: "env must be object",
			input: `
mcp test {
	command: "test",
	args: [],
	env: "not-an-object",
}
`,
			wantErr: "must be an object",
		},
		{
			name: "env values must be strings",
			input: `
mcp test {
	command: "test",
	args: [],
	env: {
		KEY: 123,
	},
}
`,
			wantErr: "must be strings",
		},
		{
			name: "unknown config field",
			input: `
mcp test {
	command: "test",
	args: [],
	unknown: "value",
}
`,
			wantErr: "unknown MCP config field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse error: %v", p.Errors())
			}

			interp := New()
			defer interp.Close()

			_, err := interp.Eval(program)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing '%s', got '%s'", tt.wantErr, err.Error())
			}
		})
	}
}

// TestMcpEnvironmentVariables tests that environment variables work in MCP declarations
func TestMcpEnvironmentVariables(t *testing.T) {
	t.Skip("Skipping test that requires actual MCP server connection")

	input := `
token = "my_token"
mcp test {
	command: "test",
	args: [],
	env: {
		TOKEN: token,
	},
}
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}

	interp := New()
	defer interp.Close()

	_, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	// Verify the server was registered
	server, err := interp.mcpManager.GetServer("test")
	if err != nil {
		t.Fatalf("failed to get server: %v", err)
	}

	if server.Config.Env["TOKEN"] != "my_token" {
		t.Errorf("expected env TOKEN='my_token', got '%s'", server.Config.Env["TOKEN"])
	}
}
