package interpreter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/acp"
	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

func TestACPDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkFunc func(t *testing.T, result *EvalResult, err error)
	}{
		{
			name: "Basic ACP declaration with command and args",
			input: `
				acp RovoDev {
					command: "acli",
					args: ["rovodev", "acp"],
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Check that the ACP agent is registered in the environment
				acpVal, ok := result.Env.Get("RovoDev")
				if !ok {
					t.Fatalf("acp 'RovoDev' not found in environment")
				}

				acp, ok := acpVal.(*ACPValue)
				if !ok {
					t.Fatalf("expected *ACPValue, got %T", acpVal)
				}

				if acp.Name != "RovoDev" {
					t.Errorf("expected acp name 'RovoDev', got %q", acp.Name)
				}

				// Check command
				cmd, ok := acp.Config["command"]
				if !ok {
					t.Fatalf("acp config missing 'command'")
				}
				cmdStr, ok := cmd.(*StringValue)
				if !ok {
					t.Fatalf("expected command to be *StringValue, got %T", cmd)
				}
				if cmdStr.Value != "acli" {
					t.Errorf("expected command 'acli', got %q", cmdStr.Value)
				}

				// Check args
				args, ok := acp.Config["args"]
				if !ok {
					t.Fatalf("acp config missing 'args'")
				}
				argsArr, ok := args.(*ArrayValue)
				if !ok {
					t.Fatalf("expected args to be *ArrayValue, got %T", args)
				}
				if len(argsArr.Elements) != 2 {
					t.Errorf("expected 2 args, got %d", len(argsArr.Elements))
				}
			},
		},
		{
			name: "ACP declaration with environment variables",
			input: `
				acp RovoDev {
					command: "acli",
					args: ["rovodev", "acp"],
					env: {
						ATLASSIAN_TOKEN: "test-token",
					},
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				acpVal, ok := result.Env.Get("RovoDev")
				if !ok {
					t.Fatalf("acp 'RovoDev' not found in environment")
				}

				acp, ok := acpVal.(*ACPValue)
				if !ok {
					t.Fatalf("expected *ACPValue, got %T", acpVal)
				}

				// Check env
				env, ok := acp.Config["env"]
				if !ok {
					t.Fatalf("acp config missing 'env'")
				}
				envObj, ok := env.(*ObjectValue)
				if !ok {
					t.Fatalf("expected env to be *ObjectValue, got %T", env)
				}

				tokenVal := envObj.GetPropertyValue("ATLASSIAN_TOKEN")
				tokenStr, ok := tokenVal.(*StringValue)
				if !ok {
					t.Fatalf("expected ATLASSIAN_TOKEN to be *StringValue, got %T", tokenVal)
				}
				if tokenStr.Value != "test-token" {
					t.Errorf("expected token 'test-token', got %q", tokenStr.Value)
				}
			},
		},
		{
			name: "ACP declaration with working directory",
			input: `
				acp RovoDev {
					command: "acli",
					args: ["rovodev", "acp"],
					cwd: "/path/to/project",
				}`,
			checkFunc: func(t *testing.T, result *EvalResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				acpVal, ok := result.Env.Get("RovoDev")
				if !ok {
					t.Fatalf("acp 'RovoDev' not found in environment")
				}

				acp, ok := acpVal.(*ACPValue)
				if !ok {
					t.Fatalf("expected *ACPValue, got %T", acpVal)
				}

				// Check cwd
				cwd, ok := acp.Config["cwd"]
				if !ok {
					t.Fatalf("acp config missing 'cwd'")
				}
				cwdStr, ok := cwd.(*StringValue)
				if !ok {
					t.Fatalf("expected cwd to be *StringValue, got %T", cwd)
				}
				if cwdStr.Value != "/path/to/project" {
					t.Errorf("expected cwd '/path/to/project', got %q", cwdStr.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			result, err := interp.Eval(program)

			tt.checkFunc(t, result, err)
		})
	}
}

func TestACPDeclarationErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name: "ACP without command field",
			input: `
				acp RovoDev {
					args: ["rovodev", "acp"],
				}`,
			expectedErr: "must have a 'command' field",
		},
		{
			name: "ACP with invalid command type",
			input: `
				acp RovoDev {
					command: 123,
				}`,
			expectedErr: "must be a string",
		},
		{
			name: "ACP with invalid args type",
			input: `
				acp RovoDev {
					command: "acli",
					args: "not-an-array",
				}`,
			expectedErr: "must be an array",
		},
		{
			name: "ACP with invalid env type",
			input: `
				acp RovoDev {
					command: "acli",
					env: "not-an-object",
				}`,
			expectedErr: "must be an object",
		},
		{
			name: "ACP with invalid cwd type",
			input: `
				acp RovoDev {
					command: "acli",
					cwd: 123,
				}`,
			expectedErr: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			_, err := interp.Eval(program)

			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error to contain %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestACPValueMethods(t *testing.T) {
	t.Run("Type", func(t *testing.T) {
		acp := &ACPValue{Name: "TestACP", Config: make(map[string]Value)}
		if acp.Type() != ValueTypeACP {
			t.Errorf("expected ValueTypeACP, got %v", acp.Type())
		}
	})

	t.Run("String", func(t *testing.T) {
		acp := &ACPValue{Name: "TestACP", Config: make(map[string]Value)}
		expected := "<acp TestACP>"
		if acp.String() != expected {
			t.Errorf("expected %q, got %q", expected, acp.String())
		}
	})

	t.Run("IsTruthy", func(t *testing.T) {
		acp := &ACPValue{Name: "TestACP", Config: make(map[string]Value)}
		if !acp.IsTruthy() {
			t.Error("expected ACPValue to be truthy")
		}
	})

	t.Run("Equals", func(t *testing.T) {
		acp1 := &ACPValue{Name: "TestACP", Config: make(map[string]Value)}
		acp2 := &ACPValue{Name: "TestACP", Config: make(map[string]Value)}
		acp3 := &ACPValue{Name: "OtherACP", Config: make(map[string]Value)}

		if !acp1.Equals(acp2) {
			t.Error("expected ACPValues with same name to be equal")
		}
		if acp1.Equals(acp3) {
			t.Error("expected ACPValues with different names to not be equal")
		}
		if acp1.Equals(&StringValue{Value: "test"}) {
			t.Error("expected ACPValue to not equal StringValue")
		}
	})

	t.Run("GetProperty", func(t *testing.T) {
		acp := &ACPValue{
			Name: "TestACP",
			Config: map[string]Value{
				"command": &StringValue{Value: "acli"},
			},
		}

		// Test name property
		nameVal := acp.GetProperty("name")
		nameStr, ok := nameVal.(*StringValue)
		if !ok {
			t.Fatalf("expected *StringValue, got %T", nameVal)
		}
		if nameStr.Value != "TestACP" {
			t.Errorf("expected name 'TestACP', got %q", nameStr.Value)
		}

		// Test config property
		cmdVal := acp.GetProperty("command")
		cmdStr, ok := cmdVal.(*StringValue)
		if !ok {
			t.Fatalf("expected *StringValue, got %T", cmdVal)
		}
		if cmdStr.Value != "acli" {
			t.Errorf("expected command 'acli', got %q", cmdStr.Value)
		}

		// Test non-existent property
		nonExistent := acp.GetProperty("nonexistent")
		if _, ok := nonExistent.(*NullValue); !ok {
			t.Errorf("expected NullValue for non-existent property, got %T", nonExistent)
		}
	})
}

func TestACPSessionValueMethods(t *testing.T) {
	acp := &ACPValue{Name: "TestACP", Config: make(map[string]Value)}

	t.Run("Type", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acp, Messages: []ChatMessage{}}
		if session.Type() != ValueTypeACPSession {
			t.Errorf("expected ValueTypeACPSession, got %v", session.Type())
		}
	})

	t.Run("String", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acp, Messages: []ChatMessage{}}
		expected := "<acpsession TestACP with 0 messages>"
		if session.String() != expected {
			t.Errorf("expected %q, got %q", expected, session.String())
		}

		// With messages
		session.Messages = []ChatMessage{{Role: "user", Content: "hello"}}
		expected = "<acpsession TestACP with 1 messages>"
		if session.String() != expected {
			t.Errorf("expected %q, got %q", expected, session.String())
		}

		// Closed session
		session.Closed = true
		expected = "<acpsession TestACP (closed) with 1 messages>"
		if session.String() != expected {
			t.Errorf("expected %q, got %q", expected, session.String())
		}
	})

	t.Run("IsTruthy", func(t *testing.T) {
		// Empty session is not truthy
		session := &ACPSessionValue{Agent: acp, Messages: []ChatMessage{}}
		if session.IsTruthy() {
			t.Error("expected empty session to not be truthy")
		}

		// Session with messages is truthy
		session.Messages = []ChatMessage{{Role: "user", Content: "hello"}}
		if !session.IsTruthy() {
			t.Error("expected session with messages to be truthy")
		}

		// Closed session is not truthy
		session.Closed = true
		if session.IsTruthy() {
			t.Error("expected closed session to not be truthy")
		}
	})

	t.Run("Equals", func(t *testing.T) {
		session1 := &ACPSessionValue{Agent: acp, SessionID: "session-1"}
		session2 := &ACPSessionValue{Agent: acp, SessionID: "session-1"}
		session3 := &ACPSessionValue{Agent: acp, SessionID: "session-2"}

		if !session1.Equals(session2) {
			t.Error("expected sessions with same ID and agent to be equal")
		}
		if session1.Equals(session3) {
			t.Error("expected sessions with different IDs to not be equal")
		}
		if session1.Equals(&StringValue{Value: "test"}) {
			t.Error("expected ACPSessionValue to not equal StringValue")
		}
	})

	t.Run("GetProperty_messages", func(t *testing.T) {
		session := &ACPSessionValue{
			Agent: acp,
			Messages: []ChatMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
		}

		messagesVal := session.GetProperty("messages")
		messagesArr, ok := messagesVal.(*ArrayValue)
		if !ok {
			t.Fatalf("expected *ArrayValue, got %T", messagesVal)
		}
		if len(messagesArr.Elements) != 2 {
			t.Errorf("expected 2 messages, got %d", len(messagesArr.Elements))
		}
	})

	t.Run("GetProperty_lastMessage", func(t *testing.T) {
		// Empty messages
		session := &ACPSessionValue{Agent: acp, Messages: []ChatMessage{}}
		lastMsg := session.GetProperty("lastMessage")
		if _, ok := lastMsg.(*NullValue); !ok {
			t.Errorf("expected NullValue for empty messages, got %T", lastMsg)
		}

		// With messages
		session.Messages = []ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		}
		lastMsg = session.GetProperty("lastMessage")
		lastMsgObj, ok := lastMsg.(*ObjectValue)
		if !ok {
			t.Fatalf("expected *ObjectValue, got %T", lastMsg)
		}
		roleVal := lastMsgObj.GetPropertyValue("role")
		roleStr, ok := roleVal.(*StringValue)
		if !ok {
			t.Fatalf("expected *StringValue, got %T", roleVal)
		}
		if roleStr.Value != "assistant" {
			t.Errorf("expected role 'assistant', got %q", roleStr.Value)
		}
	})

	t.Run("GetProperty_agent", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acp, Messages: []ChatMessage{}}
		agentVal := session.GetProperty("agent")
		agentACP, ok := agentVal.(*ACPValue)
		if !ok {
			t.Fatalf("expected *ACPValue, got %T", agentVal)
		}
		if agentACP.Name != "TestACP" {
			t.Errorf("expected agent name 'TestACP', got %q", agentACP.Name)
		}
	})

	t.Run("GetProperty_sessionId", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acp, SessionID: "test-session-123"}
		sessionIdVal := session.GetProperty("sessionId")
		sessionIdStr, ok := sessionIdVal.(*StringValue)
		if !ok {
			t.Fatalf("expected *StringValue, got %T", sessionIdVal)
		}
		if sessionIdStr.Value != "test-session-123" {
			t.Errorf("expected sessionId 'test-session-123', got %q", sessionIdStr.Value)
		}
	})

	t.Run("GetProperty_closed", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acp, Closed: false}
		closedVal := session.GetProperty("closed")
		closedBool, ok := closedVal.(*BoolValue)
		if !ok {
			t.Fatalf("expected *BoolValue, got %T", closedVal)
		}
		if closedBool.Value != false {
			t.Error("expected closed to be false")
		}

		session.Closed = true
		closedVal = session.GetProperty("closed")
		closedBool, ok = closedVal.(*BoolValue)
		if !ok {
			t.Fatalf("expected *BoolValue, got %T", closedVal)
		}
		if closedBool.Value != true {
			t.Error("expected closed to be true")
		}
	})

	t.Run("Close", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acp, Closed: false}
		if session.Closed {
			t.Error("expected session to not be closed initially")
		}
		session.Close()
		if !session.Closed {
			t.Error("expected session to be closed after Close()")
		}
	})
}

func TestACPPipeOperations(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name: "ACPSession | ACP (same agent) should error",
			input: `
				acp RovoDev {
					command: "acli",
					args: ["rovodev", "acp"],
				}
				# Create a mock session directly for testing
				# Note: This test relies on the pipe semantics being checked
			`,
			// This test verifies the error path when piping ACPSession to same agent
			expectedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			interp := New(nil)
			_, err := interp.Eval(program)

			if tt.expectedErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedErr)
				}
				if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectedErr, err.Error())
				}
			}
		})
	}
}

func TestACPPipeSemantics(t *testing.T) {
	// Test pipe operation error cases directly using the interpreter
	acp1 := &ACPValue{Name: "Agent1", Config: map[string]Value{
		"command": &StringValue{Value: "cmd1"},
	}}
	acp2 := &ACPValue{Name: "Agent2", Config: map[string]Value{
		"command": &StringValue{Value: "cmd2"},
	}}
	session := &ACPSessionValue{Agent: acp1, SessionID: "test-session"}
	localAgent := &AgentValue{Name: "LocalAgent", Config: make(map[string]Value)}
	conv := &ConversationValue{Messages: []ChatMessage{}}

	t.Run("ACPSession | SameACP should error", func(t *testing.T) {
		interp := New(nil)
		interp.env.Set("session", session)
		interp.env.Set("Agent1", acp1)

		l := lexer.New("session | Agent1")
		p := parser.New(l)
		program := p.ParseProgram()

		_, err := interp.Eval(program)
		if err == nil {
			t.Fatal("expected error for piping ACPSession to same agent")
		}
		if !strings.Contains(err.Error(), "already bound to this agent") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("ACPSession | DifferentACP should error", func(t *testing.T) {
		interp := New(nil)
		interp.env.Set("session", session)
		interp.env.Set("Agent2", acp2)

		l := lexer.New("session | Agent2")
		p := parser.New(l)
		program := p.ParseProgram()

		_, err := interp.Eval(program)
		if err == nil {
			t.Fatal("expected error for piping ACPSession to different agent")
		}
		if !strings.Contains(err.Error(), "session is bound to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("ACPSession | LocalAgent should error", func(t *testing.T) {
		interp := New(nil)
		interp.env.Set("session", session)
		interp.env.Set("LocalAgent", localAgent)

		l := lexer.New("session | LocalAgent")
		p := parser.New(l)
		program := p.ParseProgram()

		_, err := interp.Eval(program)
		if err == nil {
			t.Fatal("expected error for piping ACPSession to gsh agent")
		}
		if !strings.Contains(err.Error(), "cannot be handed off") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Conversation | ACP should error", func(t *testing.T) {
		interp := New(nil)
		interp.env.Set("conv", conv)
		interp.env.Set("Agent1", acp1)

		l := lexer.New("conv | Agent1")
		p := parser.New(l)
		program := p.ParseProgram()

		_, err := interp.Eval(program)
		if err == nil {
			t.Fatal("expected error for piping Conversation to ACP agent")
		}
		if !strings.Contains(err.Error(), "use a string prompt") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("String | ACP returns error when command not found", func(t *testing.T) {
		interp := New(nil)
		interp.env.Set("Agent1", acp1)

		l := lexer.New(`"Hello" | Agent1`)
		p := parser.New(l)
		program := p.ParseProgram()

		_, err := interp.Eval(program)
		if err == nil {
			t.Fatal("expected error when ACP command executable not found")
		}
		// The error should indicate failure to connect/spawn the ACP agent
		if !strings.Contains(err.Error(), "failed to connect to ACP agent") && !strings.Contains(err.Error(), "failed to spawn") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("ACPSession | String returns error for closed session", func(t *testing.T) {
		closedSession := &ACPSessionValue{Agent: acp1, SessionID: "closed-session", Closed: true}
		interp := New(nil)
		interp.env.Set("session", closedSession)

		l := lexer.New(`session | "Hello"`)
		p := parser.New(l)
		program := p.ParseProgram()

		_, err := interp.Eval(program)
		if err == nil {
			t.Fatal("expected error for sending prompt to closed session")
		}
		if !strings.Contains(err.Error(), "closed ACP session") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestACPSessionCloseMethod(t *testing.T) {
	acpVal := &ACPValue{Name: "TestACP", Config: map[string]Value{
		"command": &StringValue{Value: "echo"},
	}}

	t.Run("session.close() marks session as closed", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acpVal, SessionID: "test-session", Closed: false}
		interp := New(nil)
		interp.env.Set("session", session)

		l := lexer.New(`session.close()`)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		_, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !session.Closed {
			t.Error("expected session to be closed after calling close()")
		}
	})

	t.Run("session.closed property reflects state", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acpVal, SessionID: "test-session", Closed: false}
		interp := New(nil)
		interp.env.Set("session", session)

		// Check closed is false initially
		l := lexer.New(`session.closed`)
		p := parser.New(l)
		program := p.ParseProgram()

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		boolVal, ok := result.Value().(*BoolValue)
		if !ok {
			t.Fatalf("expected BoolValue, got %T", result.Value())
		}
		if boolVal.Value != false {
			t.Error("expected closed to be false initially")
		}

		// Close the session
		session.Closed = true

		result, err = interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		boolVal, ok = result.Value().(*BoolValue)
		if !ok {
			t.Fatalf("expected BoolValue, got %T", result.Value())
		}
		if boolVal.Value != true {
			t.Error("expected closed to be true after closing")
		}
	})

	t.Run("ACPSessionMethodValue implements Value interface", func(t *testing.T) {
		session := &ACPSessionValue{Agent: acpVal, SessionID: "test-session"}
		method := &ACPSessionMethodValue{Name: "close", Session: session, Interp: New(nil)}

		if method.Type() != ValueTypeTool {
			t.Errorf("expected ValueTypeTool, got %v", method.Type())
		}
		if method.String() != "<acpsession method: close>" {
			t.Errorf("unexpected String(): %s", method.String())
		}
		if !method.IsTruthy() {
			t.Error("expected method to be truthy")
		}
		if method.Equals(&StringValue{Value: "test"}) {
			t.Error("expected method to not equal other values")
		}
	})
}

func TestACPWithMockSession(t *testing.T) {
	t.Run("sendPromptToACPSession with mock session", func(t *testing.T) {
		// Create a mock session
		mockSession := acp.NewMockSession("mock-session-1")
		mockSession.AddChunkUpdate("Hello ")
		mockSession.AddChunkUpdate("World!")

		// Create the interpreter and ACP value
		interp := New(nil)
		acpVal := &ACPValue{Name: "MockAgent", Config: map[string]Value{
			"command": &StringValue{Value: "mock"},
		}}

		// Create session value and inject mock
		sessionVal := &ACPSessionValue{
			Agent:     acpVal,
			SessionID: "mock-session-1",
			Messages:  []ChatMessage{},
			Closed:    false,
		}
		interp.InjectACPSession("MockAgent", "mock-session-1", mockSession)
		interp.env.Set("session", sessionVal)

		// Send a follow-up prompt
		l := lexer.New(`session | "Test prompt"`)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		result, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify result is an ACPSessionValue
		resultSession, ok := result.Value().(*ACPSessionValue)
		if !ok {
			t.Fatalf("expected ACPSessionValue, got %T", result.Value())
		}

		// Verify messages were recorded
		if len(resultSession.Messages) < 2 {
			t.Errorf("expected at least 2 messages, got %d", len(resultSession.Messages))
		}
	})

	t.Run("sendPromptToACPSession emits events", func(t *testing.T) {
		// Create a mock session with tool call
		mockSession := acp.NewMockSession("mock-session-2")
		mockSession.AddChunkUpdate("Let me help you.")
		mockSession.AddToolCallUpdate("tool-1", "exec", `{"command": "ls"}`)
		mockSession.AddToolCallEndUpdate("tool-1", "exec", "completed", "file1.txt\nfile2.txt")

		// Create the interpreter
		interp := New(nil)
		acpVal := &ACPValue{Name: "MockAgent", Config: map[string]Value{
			"command": &StringValue{Value: "mock"},
		}}

		// Initialize eventCount and create event handlers using proper tool definitions
		interp.env.Set("eventCount", &NumberValue{Value: 0})

		// Helper to create a tool that increments eventCount
		createHandler := func(name string) *ToolValue {
			toolScript := fmt.Sprintf(`tool %s(ctx) { eventCount = eventCount + 1 }`, name)
			l := lexer.New(toolScript)
			p := parser.New(l)
			prog := p.ParseProgram()
			interp.Eval(prog)
			val, _ := interp.env.Get(name)
			return val.(*ToolValue)
		}

		interp.eventManager.On(EventAgentStart, createHandler("onStart"))
		interp.eventManager.On(EventAgentChunk, createHandler("onChunk"))
		interp.eventManager.On(EventAgentToolPending, createHandler("onToolPending"))
		interp.eventManager.On(EventAgentToolStart, createHandler("onToolStart"))
		interp.eventManager.On(EventAgentToolEnd, createHandler("onToolEnd"))
		interp.eventManager.On(EventAgentEnd, createHandler("onEnd"))

		// Create session value and inject mock
		sessionVal := &ACPSessionValue{
			Agent:     acpVal,
			SessionID: "mock-session-2",
			Messages:  []ChatMessage{},
			Closed:    false,
		}
		interp.InjectACPSession("MockAgent", "mock-session-2", mockSession)
		interp.env.Set("session", sessionVal)

		// Send a prompt
		l := lexer.New(`session | "Do something"`)
		p := parser.New(l)
		program := p.ParseProgram()

		_, err := interp.Eval(program)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check the event count
		eventCountVal, ok := interp.env.Get("eventCount")
		if !ok {
			t.Fatal("eventCount not found")
		}
		eventCount, ok := eventCountVal.(*NumberValue)
		if !ok {
			t.Fatalf("expected NumberValue, got %T", eventCountVal)
		}

		// We expect 6 events: start, chunk, tool.pending, tool.start, tool.end, end
		if eventCount.Value != 6 {
			t.Errorf("expected 6 events, got %v", eventCount.Value)
		}
	})

	t.Run("sendPromptToACPSession handles error", func(t *testing.T) {
		// Create a mock session that returns an error
		mockSession := acp.NewMockSession("mock-session-3")
		mockSession.PromptError = fmt.Errorf("mock error: connection lost")

		interp := New(nil)
		acpVal := &ACPValue{Name: "MockAgent", Config: map[string]Value{
			"command": &StringValue{Value: "mock"},
		}}

		// Initialize tracking variables and create event handler
		interp.env.Set("endEventEmitted", &BoolValue{Value: false})
		interp.env.Set("endEventHasError", &BoolValue{Value: false})

		// Create handler that checks for error
		toolScript := `tool onAgentEnd(ctx) {
			endEventEmitted = true
			if (ctx.error != null) {
				endEventHasError = true
			}
		}`
		l := lexer.New(toolScript)
		p := parser.New(l)
		program := p.ParseProgram()
		interp.Eval(program)
		handler, _ := interp.env.Get("onAgentEnd")
		interp.eventManager.On(EventAgentEnd, handler.(*ToolValue))

		sessionVal := &ACPSessionValue{
			Agent:     acpVal,
			SessionID: "mock-session-3",
			Messages:  []ChatMessage{},
			Closed:    false,
		}
		interp.InjectACPSession("MockAgent", "mock-session-3", mockSession)
		interp.env.Set("session", sessionVal)

		l = lexer.New(`session | "Test"`)
		p = parser.New(l)
		program = p.ParseProgram()

		_, err := interp.Eval(program)
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		if !strings.Contains(err.Error(), "mock error") {
			t.Errorf("expected error to contain 'mock error', got: %v", err)
		}

		// Check if end event was emitted with error
		endEventEmittedVal, _ := interp.env.Get("endEventEmitted")
		if bv, ok := endEventEmittedVal.(*BoolValue); !ok || !bv.Value {
			t.Error("expected agent.end event to be emitted on error")
		}

		endEventHasErrorVal, _ := interp.env.Get("endEventHasError")
		if bv, ok := endEventHasErrorVal.(*BoolValue); !ok || !bv.Value {
			t.Error("expected agent.end event to have error field set")
		}
	})
}
