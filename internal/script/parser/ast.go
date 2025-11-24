package parser

import (
	"strings"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

// Node represents any node in the AST
type Node interface {
	TokenLiteral() string
	String() string
}

// Statement represents a statement node
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of every AST
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// Identifier represents an identifier expression
type Identifier struct {
	Token lexer.Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// NumberLiteral represents a number literal
type NumberLiteral struct {
	Token lexer.Token // the token.NUMBER token
	Value string
}

func (n *NumberLiteral) expressionNode()      {}
func (n *NumberLiteral) TokenLiteral() string { return n.Token.Literal }
func (n *NumberLiteral) String() string       { return n.Value }

// StringLiteral represents a string literal
type StringLiteral struct {
	Token lexer.Token // the token.STRING token
	Value string
}

func (s *StringLiteral) expressionNode()      {}
func (s *StringLiteral) TokenLiteral() string { return s.Token.Literal }
func (s *StringLiteral) String() string       { return "\"" + s.Value + "\"" }

// BooleanLiteral represents a boolean literal (true/false)
type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (b *BooleanLiteral) expressionNode()      {}
func (b *BooleanLiteral) TokenLiteral() string { return b.Token.Literal }
func (b *BooleanLiteral) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// BinaryExpression represents a binary operation (e.g., x + y)
type BinaryExpression struct {
	Token    lexer.Token // the operator token
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpression) expressionNode()      {}
func (b *BinaryExpression) TokenLiteral() string { return b.Token.Literal }
func (b *BinaryExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(b.Left.String())
	out.WriteString(" " + b.Operator + " ")
	out.WriteString(b.Right.String())
	out.WriteString(")")
	return out.String()
}

// UnaryExpression represents a unary operation (e.g., !x, -x)
type UnaryExpression struct {
	Token    lexer.Token // the operator token
	Operator string
	Right    Expression
}

func (u *UnaryExpression) expressionNode()      {}
func (u *UnaryExpression) TokenLiteral() string { return u.Token.Literal }
func (u *UnaryExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(u.Operator)
	out.WriteString(u.Right.String())
	out.WriteString(")")
	return out.String()
}

// AssignmentStatement represents a variable assignment
type AssignmentStatement struct {
	Token          lexer.Token // the '=' token or identifier token
	Name           *Identifier
	TypeAnnotation *Identifier // optional type annotation (e.g., ": string")
	Value          Expression
}

func (a *AssignmentStatement) statementNode()       {}
func (a *AssignmentStatement) TokenLiteral() string { return a.Token.Literal }
func (a *AssignmentStatement) String() string {
	var out strings.Builder
	out.WriteString(a.Name.String())
	if a.TypeAnnotation != nil {
		out.WriteString(": ")
		out.WriteString(a.TypeAnnotation.String())
	}
	out.WriteString(" = ")
	if a.Value != nil {
		out.WriteString(a.Value.String())
	}
	return out.String()
}

// ExpressionStatement wraps an expression as a statement
type ExpressionStatement struct {
	Token      lexer.Token // the first token of the expression
	Expression Expression
}

func (e *ExpressionStatement) statementNode()       {}
func (e *ExpressionStatement) TokenLiteral() string { return e.Token.Literal }
func (e *ExpressionStatement) String() string {
	if e.Expression != nil {
		return e.Expression.String()
	}
	return ""
}

// BlockStatement represents a block of statements
type BlockStatement struct {
	Token      lexer.Token // the '{' token
	Statements []Statement
}

func (b *BlockStatement) statementNode()       {}
func (b *BlockStatement) TokenLiteral() string { return b.Token.Literal }
func (b *BlockStatement) String() string {
	var out strings.Builder
	out.WriteString("{\n")
	for _, s := range b.Statements {
		out.WriteString("  ")
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	out.WriteString("}")
	return out.String()
}

// CallExpression represents a function/tool call
type CallExpression struct {
	Token     lexer.Token // the '(' token
	Function  Expression  // Identifier or MemberExpression
	Arguments []Expression
}

func (c *CallExpression) expressionNode()      {}
func (c *CallExpression) TokenLiteral() string { return c.Token.Literal }
func (c *CallExpression) String() string {
	var out strings.Builder
	out.WriteString(c.Function.String())
	out.WriteString("(")
	for i, arg := range c.Arguments {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(arg.String())
	}
	out.WriteString(")")
	return out.String()
}

// MemberExpression represents member access (e.g., env.HOME, filesystem.read_file)
type MemberExpression struct {
	Token    lexer.Token // the '.' token
	Object   Expression
	Property *Identifier
}

func (m *MemberExpression) expressionNode()      {}
func (m *MemberExpression) TokenLiteral() string { return m.Token.Literal }
func (m *MemberExpression) String() string {
	var out strings.Builder
	out.WriteString(m.Object.String())
	out.WriteString(".")
	out.WriteString(m.Property.String())
	return out.String()
}

// PipeExpression represents a pipe operation (e.g., "prompt" | Agent)
// Used for agent chaining and conversation management
type PipeExpression struct {
	Token lexer.Token // the '|' token
	Left  Expression
	Right Expression
}

func (p *PipeExpression) expressionNode()      {}
func (p *PipeExpression) TokenLiteral() string { return p.Token.Literal }
func (p *PipeExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(p.Left.String())
	out.WriteString(" | ")
	out.WriteString(p.Right.String())
	out.WriteString(")")
	return out.String()
}

// ArrayLiteral represents an array literal (e.g., [1, 2, 3])
type ArrayLiteral struct {
	Token    lexer.Token // the '[' token
	Elements []Expression
}

func (a *ArrayLiteral) expressionNode()      {}
func (a *ArrayLiteral) TokenLiteral() string { return a.Token.Literal }
func (a *ArrayLiteral) String() string {
	var out strings.Builder
	out.WriteString("[")
	for i, el := range a.Elements {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(el.String())
	}
	out.WriteString("]")
	return out.String()
}

// ObjectLiteral represents an object literal (e.g., {key: value})
type ObjectLiteral struct {
	Token lexer.Token // the '{' token
	Pairs map[string]Expression
	Order []string // preserve insertion order for String()
}

func (o *ObjectLiteral) expressionNode()      {}
func (o *ObjectLiteral) TokenLiteral() string { return o.Token.Literal }
func (o *ObjectLiteral) String() string {
	var out strings.Builder
	out.WriteString("{")
	for i, key := range o.Order {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(o.Pairs[key].String())
	}
	out.WriteString("}")
	return out.String()
}

// IfStatement represents an if/else statement
type IfStatement struct {
	Token       lexer.Token // the 'if' token
	Condition   Expression
	Consequence *BlockStatement
	Alternative Statement // can be another IfStatement (for else if) or BlockStatement (for else)
}

func (i *IfStatement) statementNode()       {}
func (i *IfStatement) TokenLiteral() string { return i.Token.Literal }
func (i *IfStatement) String() string {
	var out strings.Builder
	out.WriteString("if (")
	out.WriteString(i.Condition.String())
	out.WriteString(") ")
	out.WriteString(i.Consequence.String())
	if i.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(i.Alternative.String())
	}
	return out.String()
}

// WhileStatement represents a while loop
type WhileStatement struct {
	Token     lexer.Token // the 'while' token
	Condition Expression
	Body      *BlockStatement
}

func (w *WhileStatement) statementNode()       {}
func (w *WhileStatement) TokenLiteral() string { return w.Token.Literal }
func (w *WhileStatement) String() string {
	var out strings.Builder
	out.WriteString("while (")
	out.WriteString(w.Condition.String())
	out.WriteString(") ")
	out.WriteString(w.Body.String())
	return out.String()
}

// ForOfStatement represents a for-of loop
type ForOfStatement struct {
	Token    lexer.Token // the 'for' token
	Variable *Identifier
	Iterable Expression
	Body     *BlockStatement
}

func (f *ForOfStatement) statementNode()       {}
func (f *ForOfStatement) TokenLiteral() string { return f.Token.Literal }
func (f *ForOfStatement) String() string {
	var out strings.Builder
	out.WriteString("for (")
	out.WriteString(f.Variable.String())
	out.WriteString(" of ")
	out.WriteString(f.Iterable.String())
	out.WriteString(") ")
	out.WriteString(f.Body.String())
	return out.String()
}

// BreakStatement represents a break statement
type BreakStatement struct {
	Token lexer.Token // the 'break' token
}

func (b *BreakStatement) statementNode()       {}
func (b *BreakStatement) TokenLiteral() string { return b.Token.Literal }
func (b *BreakStatement) String() string       { return "break" }

// ContinueStatement represents a continue statement
type ContinueStatement struct {
	Token lexer.Token // the 'continue' token
}

func (c *ContinueStatement) statementNode()       {}
func (c *ContinueStatement) TokenLiteral() string { return c.Token.Literal }
func (c *ContinueStatement) String() string       { return "continue" }

// ReturnStatement represents a return statement
type ReturnStatement struct {
	Token       lexer.Token // the 'return' token
	ReturnValue Expression  // optional return value
}

func (r *ReturnStatement) statementNode()       {}
func (r *ReturnStatement) TokenLiteral() string { return r.Token.Literal }
func (r *ReturnStatement) String() string {
	var out strings.Builder
	out.WriteString("return")
	if r.ReturnValue != nil {
		out.WriteString(" ")
		out.WriteString(r.ReturnValue.String())
	}
	return out.String()
}

// TryStatement represents a try/catch/finally block
type TryStatement struct {
	Token         lexer.Token // the 'try' token
	Block         *BlockStatement
	CatchClause   *CatchClause
	FinallyClause *FinallyClause
}

func (t *TryStatement) statementNode()       {}
func (t *TryStatement) TokenLiteral() string { return t.Token.Literal }
func (t *TryStatement) String() string {
	var out strings.Builder
	out.WriteString("try ")
	out.WriteString(t.Block.String())
	if t.CatchClause != nil {
		out.WriteString(" ")
		out.WriteString(t.CatchClause.String())
	}
	if t.FinallyClause != nil {
		out.WriteString(" ")
		out.WriteString(t.FinallyClause.String())
	}
	return out.String()
}

// CatchClause represents a catch clause
type CatchClause struct {
	Token     lexer.Token // the 'catch' token
	Parameter *Identifier // error parameter
	Block     *BlockStatement
}

func (c *CatchClause) String() string {
	var out strings.Builder
	out.WriteString("catch (")
	if c.Parameter != nil {
		out.WriteString(c.Parameter.String())
	}
	out.WriteString(") ")
	out.WriteString(c.Block.String())
	return out.String()
}

// FinallyClause represents a finally clause
type FinallyClause struct {
	Token lexer.Token // the 'finally' token
	Block *BlockStatement
}

func (f *FinallyClause) String() string {
	var out strings.Builder
	out.WriteString("finally ")
	out.WriteString(f.Block.String())
	return out.String()
}

// McpDeclaration represents an MCP server declaration
type McpDeclaration struct {
	Token  lexer.Token // the 'mcp' token
	Name   *Identifier
	Config map[string]Expression
}

func (m *McpDeclaration) statementNode()       {}
func (m *McpDeclaration) TokenLiteral() string { return m.Token.Literal }
func (m *McpDeclaration) String() string {
	var out strings.Builder
	out.WriteString("mcp ")
	out.WriteString(m.Name.String())
	out.WriteString(" {\n")
	for key, value := range m.Config {
		out.WriteString("  ")
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(value.String())
		out.WriteString(",\n")
	}
	out.WriteString("}")
	return out.String()
}

// ModelDeclaration represents a model declaration
type ModelDeclaration struct {
	Token  lexer.Token // the 'model' token
	Name   *Identifier
	Config map[string]Expression
}

func (m *ModelDeclaration) statementNode()       {}
func (m *ModelDeclaration) TokenLiteral() string { return m.Token.Literal }
func (m *ModelDeclaration) String() string {
	var out strings.Builder
	out.WriteString("model ")
	out.WriteString(m.Name.String())
	out.WriteString(" {\n")
	for key, value := range m.Config {
		out.WriteString("  ")
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(value.String())
		out.WriteString(",\n")
	}
	out.WriteString("}")
	return out.String()
}

// AgentDeclaration represents an agent declaration
type AgentDeclaration struct {
	Token  lexer.Token // the 'agent' token
	Name   *Identifier
	Config map[string]Expression
}

func (a *AgentDeclaration) statementNode()       {}
func (a *AgentDeclaration) TokenLiteral() string { return a.Token.Literal }
func (a *AgentDeclaration) String() string {
	var out strings.Builder
	out.WriteString("agent ")
	out.WriteString(a.Name.String())
	out.WriteString(" {\n")
	for key, value := range a.Config {
		out.WriteString("  ")
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(value.String())
		out.WriteString(",\n")
	}
	out.WriteString("}")
	return out.String()
}

// ToolParameter represents a parameter in a tool declaration
type ToolParameter struct {
	Name *Identifier
	Type *Identifier // optional type annotation
}

// ToolDeclaration represents a tool declaration
type ToolDeclaration struct {
	Token      lexer.Token // the 'tool' token
	Name       *Identifier
	Parameters []*ToolParameter
	ReturnType *Identifier // optional return type annotation
	Body       *BlockStatement
}

func (t *ToolDeclaration) statementNode()       {}
func (t *ToolDeclaration) TokenLiteral() string { return t.Token.Literal }
func (t *ToolDeclaration) String() string {
	var out strings.Builder
	out.WriteString("tool ")
	out.WriteString(t.Name.String())
	out.WriteString("(")
	for i, param := range t.Parameters {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(param.Name.String())
		if param.Type != nil {
			out.WriteString(": ")
			out.WriteString(param.Type.String())
		}
	}
	out.WriteString(")")
	if t.ReturnType != nil {
		out.WriteString(": ")
		out.WriteString(t.ReturnType.String())
	}
	out.WriteString(" ")
	out.WriteString(t.Body.String())
	return out.String()
}
