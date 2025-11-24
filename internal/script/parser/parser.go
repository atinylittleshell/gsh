package parser

import (
	"fmt"
	"strconv"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

// Parser represents the parser
type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

// Operator precedence levels
const (
	_ int = iota
	LOWEST
	PIPE        // | (pipe operator for agent chaining)
	NULLCOAL    // ??
	OR          // ||
	AND         // &&
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	MEMBER      // object.property
)

var precedences = map[lexer.TokenType]int{
	lexer.OP_PIPE:     PIPE,
	lexer.OP_NULLCOAL: NULLCOAL,
	lexer.OP_OR:       OR,
	lexer.OP_AND:      AND,
	lexer.OP_EQ:       EQUALS,
	lexer.OP_NEQ:      EQUALS,
	lexer.OP_LT:       LESSGREATER,
	lexer.OP_GT:       LESSGREATER,
	lexer.OP_LTE:      LESSGREATER,
	lexer.OP_GTE:      LESSGREATER,
	lexer.OP_PLUS:     SUM,
	lexer.OP_MINUS:    SUM,
	lexer.OP_SLASH:    PRODUCT,
	lexer.OP_ASTERISK: PRODUCT,
	lexer.OP_PERCENT:  PRODUCT,
	lexer.LPAREN:      CALL,
	lexer.DOT:         MEMBER,
}

type (
	prefixParseFn func() Expression
	infixParseFn  func(Expression) Expression
)

// New creates a new Parser instance
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// Register prefix parse functions
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.OP_BANG, p.parseUnaryExpression)
	p.registerPrefix(lexer.OP_MINUS, p.parseUnaryExpression)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(lexer.LBRACE, p.parseObjectLiteral)

	// Register infix parse functions
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.OP_PLUS, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_MINUS, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_SLASH, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_ASTERISK, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_PERCENT, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_EQ, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_NEQ, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_LT, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_GT, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_LTE, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_GTE, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_AND, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_OR, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_NULLCOAL, p.parseBinaryExpression)
	p.registerInfix(lexer.OP_PIPE, p.parseBinaryExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.DOT, p.parseMemberExpression)

	// Read two tokens to set both curToken and peekToken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// nextToken advances the parser to the next token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// Errors returns the list of parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// addError adds a parsing error
func (p *Parser) addError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	p.errors = append(p.errors, msg)
}

// curTokenIs checks if the current token is of the given type
func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs checks if the peek token is of the given type
func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek checks if the next token is of the expected type and advances if so
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t lexer.TokenType) {
	p.addError("expected next token to be %v, got %v instead at line %d, column %d",
		t, p.peekToken.Type, p.peekToken.Line, p.peekToken.Column)
}

// peekPrecedence returns the precedence of the peek token
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// curPrecedence returns the precedence of the current token
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	for !p.curTokenIs(lexer.EOF) {
		// Skip semicolons (they're optional statement terminators)
		if p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatement parses a statement
func (p *Parser) parseStatement() Statement {
	// Check if this is an assignment (identifier followed by '=' or ':')
	if p.curTokenIs(lexer.IDENT) {
		if p.peekTokenIs(lexer.OP_ASSIGN) || p.peekTokenIs(lexer.COLON) {
			return p.parseAssignmentStatement()
		}
	}

	// Otherwise, treat as expression statement
	return p.parseExpressionStatement()
}

// parseAssignmentStatement parses variable declarations and assignments
func (p *Parser) parseAssignmentStatement() Statement {
	stmt := &AssignmentStatement{Token: p.curToken}

	// Parse the identifier
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume identifier
		p.nextToken() // consume ':'

		// Parse type annotation
		if !p.curTokenIs(lexer.IDENT) {
			p.addError("expected type annotation after ':', got %v at line %d, column %d",
				p.curToken.Type, p.curToken.Line, p.curToken.Column)
			return nil
		}
		stmt.TypeAnnotation = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

		// Expect '=' after type annotation
		if !p.expectPeek(lexer.OP_ASSIGN) {
			return nil
		}
	} else if p.peekTokenIs(lexer.OP_ASSIGN) {
		p.nextToken() // consume identifier, now on '='
	} else {
		p.addError("expected '=' or ':', got %v at line %d, column %d",
			p.peekToken.Type, p.peekToken.Line, p.peekToken.Column)
		return nil
	}

	p.nextToken() // consume '=', now on value expression

	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

// parseExpressionStatement parses an expression statement
func (p *Parser) parseExpressionStatement() Statement {
	stmt := &ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	if stmt.Expression == nil {
		return nil
	}
	return stmt
}

// parseExpression parses an expression with operator precedence
func (p *Parser) parseExpression(precedence int) Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.addError("no prefix parse function for %v found at line %d, column %d",
			p.curToken.Type, p.curToken.Line, p.curToken.Column)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.EOF) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// parseIdentifier parses an identifier
func (p *Parser) parseIdentifier() Expression {
	// Handle boolean literals
	if p.curToken.Literal == "true" {
		return &BooleanLiteral{Token: p.curToken, Value: true}
	}
	if p.curToken.Literal == "false" {
		return &BooleanLiteral{Token: p.curToken, Value: false}
	}

	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseNumberLiteral parses a number literal
func (p *Parser) parseNumberLiteral() Expression {
	lit := &NumberLiteral{Token: p.curToken, Value: p.curToken.Literal}

	// Validate that it's a valid number
	_, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.addError("could not parse %q as number at line %d, column %d",
			p.curToken.Literal, p.curToken.Line, p.curToken.Column)
		return nil
	}

	return lit
}

// parseStringLiteral parses a string literal
func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parseUnaryExpression parses unary expressions (!, -)
func (p *Parser) parseUnaryExpression() Expression {
	expression := &UnaryExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// parseBinaryExpression parses binary expressions
func (p *Parser) parseBinaryExpression(left Expression) Expression {
	expression := &BinaryExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parseGroupedExpression parses grouped expressions (parentheses)
func (p *Parser) parseGroupedExpression() Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return exp
}

// parseCallExpression parses function/tool call expressions
func (p *Parser) parseCallExpression(function Expression) Expression {
	exp := &CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(lexer.RPAREN)
	return exp
}

// parseMemberExpression parses member access expressions
func (p *Parser) parseMemberExpression(object Expression) Expression {
	exp := &MemberExpression{Token: p.curToken, Object: object}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	exp.Property = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

// parseArrayLiteral parses array literals
func (p *Parser) parseArrayLiteral() Expression {
	array := &ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(lexer.RBRACKET)
	return array
}

// parseObjectLiteral parses object literals
func (p *Parser) parseObjectLiteral() Expression {
	obj := &ObjectLiteral{Token: p.curToken}
	obj.Pairs = make(map[string]Expression)
	obj.Order = []string{}

	// Empty object
	if p.peekTokenIs(lexer.RBRACE) {
		p.nextToken()
		return obj
	}

	p.nextToken() // move to first key

	for {
		// Parse key (must be identifier or string)
		var key string
		if p.curTokenIs(lexer.IDENT) {
			key = p.curToken.Literal
		} else if p.curTokenIs(lexer.STRING) {
			key = p.curToken.Literal
		} else {
			p.addError("expected object key (identifier or string), got %v at line %d, column %d",
				p.curToken.Type, p.curToken.Line, p.curToken.Column)
			return nil
		}

		// Expect ':'
		if !p.expectPeek(lexer.COLON) {
			return nil
		}

		p.nextToken() // move to value

		// Parse value
		value := p.parseExpression(LOWEST)
		obj.Pairs[key] = value
		obj.Order = append(obj.Order, key)

		// Check for comma or closing brace
		if p.peekTokenIs(lexer.RBRACE) {
			p.nextToken()
			break
		}

		if !p.expectPeek(lexer.COMMA) {
			return nil
		}

		p.nextToken() // move to next key
	}

	return obj
}

// parseExpressionList parses a comma-separated list of expressions
func (p *Parser) parseExpressionList(end lexer.TokenType) []Expression {
	list := []Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next expression
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}
