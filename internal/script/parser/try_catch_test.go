package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

func TestTryStatementWithCatch(t *testing.T) {
	input := `
try {
	x = 1
	y = 2
} catch (error) {
	print(error)
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*TryStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *TryStatement. got=%T", program.Statements[0])
	}

	if stmt.Token.Type != lexer.KW_TRY {
		t.Errorf("stmt.Token.Type not KW_TRY. got=%v", stmt.Token.Type)
	}

	// Check try block
	if stmt.Block == nil {
		t.Fatal("stmt.Block is nil")
	}

	if len(stmt.Block.Statements) != 2 {
		t.Fatalf("stmt.Block.Statements does not contain 2 statements. got=%d", len(stmt.Block.Statements))
	}

	// Check catch clause
	if stmt.CatchClause == nil {
		t.Fatal("stmt.CatchClause is nil")
	}

	if stmt.CatchClause.Token.Type != lexer.KW_CATCH {
		t.Errorf("stmt.CatchClause.Token.Type not KW_CATCH. got=%v", stmt.CatchClause.Token.Type)
	}

	if stmt.CatchClause.Parameter == nil {
		t.Fatal("stmt.CatchClause.Parameter is nil")
	}

	if stmt.CatchClause.Parameter.Value != "error" {
		t.Errorf("stmt.CatchClause.Parameter.Value not 'error'. got=%s", stmt.CatchClause.Parameter.Value)
	}

	if stmt.CatchClause.Block == nil {
		t.Fatal("stmt.CatchClause.Block is nil")
	}

	if len(stmt.CatchClause.Block.Statements) != 1 {
		t.Fatalf("stmt.CatchClause.Block.Statements does not contain 1 statement. got=%d", len(stmt.CatchClause.Block.Statements))
	}
}

func TestTryStatementWithFinally(t *testing.T) {
	input := `
try {
	x = 1
} finally {
	cleanup()
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*TryStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *TryStatement. got=%T", program.Statements[0])
	}

	// Check try block
	if stmt.Block == nil {
		t.Fatal("stmt.Block is nil")
	}

	if len(stmt.Block.Statements) != 1 {
		t.Fatalf("stmt.Block.Statements does not contain 1 statement. got=%d", len(stmt.Block.Statements))
	}

	// Check catch clause is nil
	if stmt.CatchClause != nil {
		t.Error("stmt.CatchClause should be nil when no catch clause is present")
	}

	// Check finally clause
	if stmt.FinallyClause == nil {
		t.Fatal("stmt.FinallyClause is nil")
	}

	if stmt.FinallyClause.Token.Type != lexer.KW_FINALLY {
		t.Errorf("stmt.FinallyClause.Token.Type not KW_FINALLY. got=%v", stmt.FinallyClause.Token.Type)
	}

	if stmt.FinallyClause.Block == nil {
		t.Fatal("stmt.FinallyClause.Block is nil")
	}

	if len(stmt.FinallyClause.Block.Statements) != 1 {
		t.Fatalf("stmt.FinallyClause.Block.Statements does not contain 1 statement. got=%d", len(stmt.FinallyClause.Block.Statements))
	}
}

func TestTryStatementWithCatchAndFinally(t *testing.T) {
	input := `
try {
	x = 1
} catch (error) {
	log.error(error)
} finally {
	cleanup()
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*TryStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *TryStatement. got=%T", program.Statements[0])
	}

	// Check try block
	if stmt.Block == nil {
		t.Fatal("stmt.Block is nil")
	}

	// Check catch clause
	if stmt.CatchClause == nil {
		t.Fatal("stmt.CatchClause is nil")
	}

	if stmt.CatchClause.Parameter.Value != "error" {
		t.Errorf("stmt.CatchClause.Parameter.Value not 'error'. got=%s", stmt.CatchClause.Parameter.Value)
	}

	// Check finally clause
	if stmt.FinallyClause == nil {
		t.Fatal("stmt.FinallyClause is nil")
	}

	if stmt.FinallyClause.Token.Type != lexer.KW_FINALLY {
		t.Errorf("stmt.FinallyClause.Token.Type not KW_FINALLY. got=%v", stmt.FinallyClause.Token.Type)
	}
}

func TestNestedTryCatch(t *testing.T) {
	input := `
try {
	try {
		x = dangerous()
	} catch (innerError) {
		log.error(innerError)
	}
} catch (outerError) {
	print(outerError)
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	outerTry, ok := program.Statements[0].(*TryStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *TryStatement. got=%T", program.Statements[0])
	}

	// Check outer try block contains inner try
	if len(outerTry.Block.Statements) != 1 {
		t.Fatalf("outerTry.Block.Statements does not contain 1 statement. got=%d", len(outerTry.Block.Statements))
	}

	innerTry, ok := outerTry.Block.Statements[0].(*TryStatement)
	if !ok {
		t.Fatalf("outerTry.Block.Statements[0] is not *TryStatement. got=%T", outerTry.Block.Statements[0])
	}

	// Check inner catch parameter
	if innerTry.CatchClause == nil {
		t.Fatal("innerTry.CatchClause is nil")
	}

	if innerTry.CatchClause.Parameter.Value != "innerError" {
		t.Errorf("innerTry.CatchClause.Parameter.Value not 'innerError'. got=%s", innerTry.CatchClause.Parameter.Value)
	}

	// Check outer catch parameter
	if outerTry.CatchClause == nil {
		t.Fatal("outerTry.CatchClause is nil")
	}

	if outerTry.CatchClause.Parameter.Value != "outerError" {
		t.Errorf("outerTry.CatchClause.Parameter.Value not 'outerError'. got=%s", outerTry.CatchClause.Parameter.Value)
	}
}

func TestTryCatchWithComplexCode(t *testing.T) {
	input := `
try {
	content = filesystem.read_file("data.json")
	data = JSON.parse(content)
	
	for (item of data) {
		if (item.valid) {
			process(item)
		}
	}
} catch (error) {
	log.error("Failed: " + error.message)
	data = getDefaultData()
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*TryStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *TryStatement. got=%T", program.Statements[0])
	}

	// Check try block has multiple statements
	if stmt.Block == nil {
		t.Fatal("stmt.Block is nil")
	}

	if len(stmt.Block.Statements) != 3 {
		t.Fatalf("stmt.Block.Statements does not contain 3 statements. got=%d", len(stmt.Block.Statements))
	}

	// Verify first statement is assignment
	_, ok = stmt.Block.Statements[0].(*AssignmentStatement)
	if !ok {
		t.Errorf("stmt.Block.Statements[0] is not *AssignmentStatement. got=%T", stmt.Block.Statements[0])
	}

	// Verify second statement is assignment
	_, ok = stmt.Block.Statements[1].(*AssignmentStatement)
	if !ok {
		t.Errorf("stmt.Block.Statements[1] is not *AssignmentStatement. got=%T", stmt.Block.Statements[1])
	}

	// Verify third statement is for-of loop
	_, ok = stmt.Block.Statements[2].(*ForOfStatement)
	if !ok {
		t.Errorf("stmt.Block.Statements[2] is not *ForOfStatement. got=%T", stmt.Block.Statements[2])
	}

	// Check catch block
	if stmt.CatchClause == nil {
		t.Fatal("stmt.CatchClause is nil")
	}

	if len(stmt.CatchClause.Block.Statements) != 2 {
		t.Fatalf("stmt.CatchClause.Block.Statements does not contain 2 statements. got=%d", len(stmt.CatchClause.Block.Statements))
	}
}

func TestTryCatchString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input: `try {
				x = 1
			} catch (e) {
				print(e)
			}`,
			expected: "try {\n  x = 1\n} catch (e) {\n  print(e)\n}",
		},
		{
			input: `try {
				doSomething()
			} finally {
				cleanup()
			}`,
			expected: "try {\n  doSomething()\n} finally {\n  cleanup()\n}",
		},
		{
			input: `try {
				x = 1
			} catch (e) {
				print(e)
			} finally {
				cleanup()
			}`,
			expected: "try {\n  x = 1\n} catch (e) {\n  print(e)\n} finally {\n  cleanup()\n}",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*TryStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not *TryStatement. got=%T", program.Statements[0])
		}

		if stmt.String() != tt.expected {
			t.Errorf("stmt.String() wrong.\nexpected:\n%s\ngot:\n%s", tt.expected, stmt.String())
		}
	}
}

func TestTryCatchErrorCases(t *testing.T) {
	tests := []struct {
		input       string
		expectedErr string
	}{
		{
			input:       "try x = 1",
			expectedErr: "expected next token to be '{'",
		},
		{
			input:       "try { x = 1 }",
			expectedErr: "try statement must have at least one 'catch' or 'finally' clause",
		},
		{
			input:       "try { x = 1 } catch",
			expectedErr: "expected next token to be '('",
		},
		{
			input:       "try { x = 1 } catch ()",
			expectedErr: "expected next token to be identifier",
		},
		{
			input:       "try { x = 1 } catch (error",
			expectedErr: "expected next token to be ')'",
		},
		{
			input:       "try { x = 1 } catch (error)",
			expectedErr: "expected next token to be '{'",
		},
		{
			input:       "try { x = 1 } finally",
			expectedErr: "expected next token to be '{'",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		p.ParseProgram()

		if len(p.Errors()) == 0 {
			t.Errorf("expected parser errors for input: %s", tt.input)
			continue
		}

		found := false
		for _, err := range p.Errors() {
			if containsSubstring(err, tt.expectedErr) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected error containing '%s', got: %v", tt.expectedErr, p.Errors())
		}
	}
}
