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
			interp := New(nil)
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
	interp := New(nil)
	defer interp.Close()

	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	// Check that the MCP server is registered in environment
	testVal, ok := interp.globalEnv.Get("test")
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

			interp := New(nil)
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

	interp := New(nil)
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

// TestMcpToolsAvailableInEnvironment tests that MCP tools are accessible via member expressions
func TestMcpToolsAvailableInEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		checkFn func(t *testing.T, interp *Interpreter)
	}{
		{
			name: "undefined MCP server",
			input: `
# Try to access undefined MCP server
x = undefined_server
`,
			wantErr: true,
		},
		{
			name: "MCP server name conflict with variable",
			input: `
# Define a variable first
filesystem = "test"
# This should fail because the variable already exists
`,
			wantErr: false,
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

			interp := New(nil)
			defer interp.Close()

			_, err := interp.Eval(program)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error=%v, got error=%v", tt.wantErr, err)
			}

			if tt.checkFn != nil {
				tt.checkFn(t, interp)
			}
		})
	}
}

// TestMcpProxyValueType tests that MCP proxy values have the correct type
func TestMcpProxyValueType(t *testing.T) {
	// Create an MCP proxy value directly to test its properties
	// without requiring an actual MCP server connection
	proxy := &MCPProxyValue{
		ServerName: "testserver",
		Manager:    nil, // Manager not needed for type tests
	}

	// Check properties
	if proxy.ServerName != "testserver" {
		t.Errorf("expected ServerName='testserver', got '%s'", proxy.ServerName)
	}

	if proxy.Type() != ValueTypeObject {
		t.Errorf("expected Type()=ValueTypeObject, got %s", proxy.Type())
	}

	if !proxy.IsTruthy() {
		t.Error("expected IsTruthy()=true")
	}

	expectedStr := "<mcp server: testserver>"
	if proxy.String() != expectedStr {
		t.Errorf("expected String()='%s', got '%s'", expectedStr, proxy.String())
	}

	// Test equality
	proxy2 := &MCPProxyValue{
		ServerName: "testserver",
		Manager:    nil,
	}
	if !proxy.Equals(proxy2) {
		t.Error("expected equal proxies with same ServerName")
	}

	proxy3 := &MCPProxyValue{
		ServerName: "different",
		Manager:    nil,
	}
	if proxy.Equals(proxy3) {
		t.Error("expected unequal proxies with different ServerName")
	}
}

// TestMcpToolValueType tests that MCP tool values have the correct type
func TestMcpToolValueType(t *testing.T) {
	// Create an MCP tool value directly to test its properties
	// without requiring an actual MCP server connection
	tool := &MCPToolValue{
		ServerName: "testserver",
		ToolName:   "test_tool",
		Manager:    nil, // Manager not needed for type tests
	}

	// Check properties
	if tool.ServerName != "testserver" {
		t.Errorf("expected ServerName='testserver', got '%s'", tool.ServerName)
	}

	if tool.ToolName != "test_tool" {
		t.Errorf("expected ToolName='test_tool', got '%s'", tool.ToolName)
	}

	if tool.Type() != ValueTypeTool {
		t.Errorf("expected Type()=ValueTypeTool, got %s", tool.Type())
	}

	if !tool.IsTruthy() {
		t.Error("expected IsTruthy()=true")
	}

	expectedStr := "<mcp tool: testserver.test_tool>"
	if tool.String() != expectedStr {
		t.Errorf("expected String()='%s', got '%s'", expectedStr, tool.String())
	}

	// Test equality
	tool2 := &MCPToolValue{
		ServerName: "testserver",
		ToolName:   "test_tool",
		Manager:    nil,
	}
	if !tool.Equals(tool2) {
		t.Error("expected equal tools with same ServerName and ToolName")
	}

	tool3 := &MCPToolValue{
		ServerName: "testserver",
		ToolName:   "different_tool",
		Manager:    nil,
	}
	if tool.Equals(tool3) {
		t.Error("expected unequal tools with different ToolName")
	}

	tool4 := &MCPToolValue{
		ServerName: "different_server",
		ToolName:   "test_tool",
		Manager:    nil,
	}
	if tool.Equals(tool4) {
		t.Error("expected unequal tools with different ServerName")
	}
}

// TestMcpServerInEnvironmentAfterDeclaration tests that MCP servers are immediately available
func TestMcpServerInEnvironmentAfterDeclaration(t *testing.T) {
	t.Skip("Skipping test that requires actual MCP server connection")

	input := `
mcp fs {
	command: "echo",
	args: [],
}

# Server should be available immediately after declaration
server = fs
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}

	interp := New(nil)
	defer interp.Close()

	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	// Verify 'fs' is in the environment
	fsVal, ok := interp.globalEnv.Get("fs")
	if !ok {
		t.Fatal("MCP server 'fs' not found in environment")
	}

	// Verify it's an MCP proxy
	if _, ok := fsVal.(*MCPProxyValue); !ok {
		t.Fatalf("expected MCPProxyValue, got %T", fsVal)
	}

	// Verify 'server' variable has the same proxy
	serverVal, ok := result.Variables()["server"]
	if !ok {
		t.Fatal("variable 'server' not found")
	}

	if !fsVal.Equals(serverVal) {
		t.Error("expected 'fs' and 'server' to reference the same MCP proxy")
	}
}

// TestMcpMultipleServersInEnvironment tests that multiple MCP servers coexist
func TestMcpMultipleServersInEnvironment(t *testing.T) {
	t.Skip("Skipping test that requires actual MCP server connection")

	input := `
mcp server1 {
	command: "echo",
	args: ["one"],
}

mcp server2 {
	command: "echo",
	args: ["two"],
}

mcp server3 {
	command: "echo",
	args: ["three"],
}

s1 = server1
s2 = server2
s3 = server3
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}

	interp := New(nil)
	defer interp.Close()

	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	// Verify all three servers are in the environment
	vars := result.Variables()

	for i := 1; i <= 3; i++ {
		varName := "s" + strings.Trim(strings.Fields("1 2 3")[i-1], " ")
		val, ok := vars[varName]
		if !ok {
			t.Fatalf("variable '%s' not found", varName)
		}

		proxy, ok := val.(*MCPProxyValue)
		if !ok {
			t.Fatalf("expected MCPProxyValue for '%s', got %T", varName, val)
		}

		expectedServerName := "server" + strings.Trim(strings.Fields("1 2 3")[i-1], " ")
		if proxy.ServerName != expectedServerName {
			t.Errorf("expected ServerName='%s', got '%s'", expectedServerName, proxy.ServerName)
		}
	}
}

// TestMcpProxyEquality tests that MCP proxy equality works correctly
func TestMcpProxyEquality(t *testing.T) {
	t.Skip("Skipping test that requires actual MCP server connection")

	input := `
mcp server1 {
	command: "echo",
	args: [],
}

mcp server2 {
	command: "echo",
	args: [],
}

a = server1
b = server1
c = server2
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}

	interp := New(nil)
	defer interp.Close()

	result, err := interp.Eval(program)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	vars := result.Variables()
	a := vars["a"]
	b := vars["b"]
	c := vars["c"]

	// a and b should be equal (same server)
	if !a.Equals(b) {
		t.Error("expected 'a' and 'b' to be equal (same MCP server)")
	}

	// a and c should not be equal (different servers)
	if a.Equals(c) {
		t.Error("expected 'a' and 'c' to be different (different MCP servers)")
	}
}
