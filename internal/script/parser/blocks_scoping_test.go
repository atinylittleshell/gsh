package parser

import (
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

// TestBlocksInControlFlow verifies that blocks are correctly parsed
// in all control flow statements (if, while, for, try/catch/finally).
// This test confirms that Phase 2.2's "Parse blocks and scoping" is complete.
func TestBlocksInControlFlow(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedBlocks int // number of BlockStatement nodes expected
	}{
		{
			name: "if statement with block",
			input: `
if (x > 0) {
	y = x + 1
	print(y)
}
`,
			expectedBlocks: 1,
		},
		{
			name: "if-else with blocks",
			input: `
if (x > 0) {
	y = x + 1
} else {
	y = 0
}
`,
			expectedBlocks: 2,
		},
		{
			name: "while loop with block",
			input: `
while (count < 10) {
	count = count + 1
	print(count)
}
`,
			expectedBlocks: 1,
		},
		{
			name: "for-of loop with block",
			input: `
for (item of items) {
	total = total + item
	print(item)
}
`,
			expectedBlocks: 1,
		},
		{
			name: "try-catch with blocks",
			input: `
try {
	doSomething()
} catch (e) {
	handleError(e)
}
`,
			expectedBlocks: 2,
		},
		{
			name: "try-finally with blocks",
			input: `
try {
	doSomething()
} finally {
	cleanup()
}
`,
			expectedBlocks: 2,
		},
		{
			name: "try-catch-finally with blocks",
			input: `
try {
	doSomething()
} catch (e) {
	handleError(e)
} finally {
	cleanup()
}
`,
			expectedBlocks: 3,
		},
		{
			name: "nested if statements with blocks",
			input: `
if (a > 0) {
	if (b > 0) {
		c = a + b
	}
}
`,
			expectedBlocks: 2,
		},
		{
			name: "nested loops with blocks",
			input: `
for (i of outer) {
	for (j of inner) {
		print(i + j)
	}
}
`,
			expectedBlocks: 2,
		},
		{
			name: "complex nested control flow",
			input: `
if (condition) {
	for (item of items) {
		try {
			process(item)
		} catch (e) {
			log(e)
		}
	}
}
`,
			expectedBlocks: 4, // if block, for block, try block, catch block
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			// Count BlockStatement nodes
			blockCount := countBlockStatements(program)
			if blockCount != tt.expectedBlocks {
				t.Errorf("Expected %d blocks, got %d", tt.expectedBlocks, blockCount)
			}
		})
	}
}

// TestBlockStatementStructure verifies the structure of BlockStatement nodes
func TestBlockStatementStructure(t *testing.T) {
	input := `
if (x > 0) {
	a = 1
	b = 2
	c = a + b
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program should have 1 statement, got=%d", len(program.Statements))
	}

	ifStmt, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatalf("statement is not *IfStatement, got=%T", program.Statements[0])
	}

	if ifStmt.Consequence == nil {
		t.Fatal("ifStmt.Consequence is nil")
	}

	// Verify the block contains 3 statements
	if len(ifStmt.Consequence.Statements) != 3 {
		t.Fatalf("block should have 3 statements, got=%d", len(ifStmt.Consequence.Statements))
	}

	// Verify all are assignment statements
	for i, stmt := range ifStmt.Consequence.Statements {
		if _, ok := stmt.(*AssignmentStatement); !ok {
			t.Errorf("statement %d is not *AssignmentStatement, got=%T", i, stmt)
		}
	}
}

// TestEmptyBlocks verifies that empty blocks are handled correctly
func TestEmptyBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty if block",
			input: "if (x > 0) {}",
		},
		{
			name:  "empty while block",
			input: "while (true) {}",
		},
		{
			name:  "empty for block",
			input: "for (item of items) {}",
		},
		{
			name:  "empty try block",
			input: "try {} catch (e) { print(e) }",
		},
		{
			name:  "empty catch block",
			input: "try { doSomething() } catch (e) {}",
		},
		{
			name:  "empty finally block",
			input: "try { doSomething() } finally {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			if len(program.Statements) != 1 {
				t.Fatalf("program should have 1 statement, got=%d", len(program.Statements))
			}
		})
	}
}

// TestBlocksPreserveStatementOrder verifies that statements within blocks
// maintain their original order
func TestBlocksPreserveStatementOrder(t *testing.T) {
	input := `
if (true) {
	first = 1
	second = 2
	third = 3
	fourth = 4
	fifth = 5
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ifStmt := program.Statements[0].(*IfStatement)
	block := ifStmt.Consequence

	expectedVars := []string{"first", "second", "third", "fourth", "fifth"}

	if len(block.Statements) != len(expectedVars) {
		t.Fatalf("expected %d statements, got=%d", len(expectedVars), len(block.Statements))
	}

	for i, expectedVar := range expectedVars {
		assignStmt, ok := block.Statements[i].(*AssignmentStatement)
		if !ok {
			t.Fatalf("statement %d is not *AssignmentStatement", i)
		}
		if assignStmt.Name.Value != expectedVar {
			t.Errorf("statement %d: expected variable '%s', got '%s'",
				i, expectedVar, assignStmt.Name.Value)
		}
	}
}

// TestBlockTokenLiterals verifies that blocks have correct token literals
func TestBlockTokenLiterals(t *testing.T) {
	input := `if (x > 0) { y = 1 }`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ifStmt := program.Statements[0].(*IfStatement)
	block := ifStmt.Consequence

	if block.TokenLiteral() != "{" {
		t.Errorf("block.TokenLiteral() = %q, want '{'", block.TokenLiteral())
	}
}

// TestBlocksWithMixedStatements verifies blocks can contain
// different types of statements
func TestBlocksWithMixedStatements(t *testing.T) {
	input := `
if (true) {
	x = 5
	print(x)
	for (i of items) {
		process(i)
	}
	y = x + 10
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	ifStmt := program.Statements[0].(*IfStatement)
	block := ifStmt.Consequence

	if len(block.Statements) != 4 {
		t.Fatalf("expected 4 statements, got=%d", len(block.Statements))
	}

	// First: assignment
	if _, ok := block.Statements[0].(*AssignmentStatement); !ok {
		t.Errorf("statement 0 is not *AssignmentStatement, got=%T", block.Statements[0])
	}

	// Second: expression statement (function call)
	if _, ok := block.Statements[1].(*ExpressionStatement); !ok {
		t.Errorf("statement 1 is not *ExpressionStatement, got=%T", block.Statements[1])
	}

	// Third: for-of loop
	if _, ok := block.Statements[2].(*ForOfStatement); !ok {
		t.Errorf("statement 2 is not *ForOfStatement, got=%T", block.Statements[2])
	}

	// Fourth: assignment
	if _, ok := block.Statements[3].(*AssignmentStatement); !ok {
		t.Errorf("statement 3 is not *AssignmentStatement, got=%T", block.Statements[3])
	}
}

// Helper function to recursively count BlockStatement nodes in the AST
func countBlockStatements(node Node) int {
	count := 0

	switch n := node.(type) {
	case *Program:
		for _, stmt := range n.Statements {
			count += countBlockStatements(stmt)
		}
	case *BlockStatement:
		count = 1 // Count this block
		for _, stmt := range n.Statements {
			count += countBlockStatements(stmt)
		}
	case *IfStatement:
		if n.Consequence != nil {
			count += countBlockStatements(n.Consequence)
		}
		if n.Alternative != nil {
			count += countBlockStatements(n.Alternative)
		}
	case *WhileStatement:
		if n.Body != nil {
			count += countBlockStatements(n.Body)
		}
	case *ForOfStatement:
		if n.Body != nil {
			count += countBlockStatements(n.Body)
		}
	case *TryStatement:
		if n.Block != nil {
			count += countBlockStatements(n.Block)
		}
		if n.CatchClause != nil && n.CatchClause.Block != nil {
			count += countBlockStatements(n.CatchClause.Block)
		}
		if n.FinallyClause != nil && n.FinallyClause.Block != nil {
			count += countBlockStatements(n.FinallyClause.Block)
		}
	}

	return count
}
