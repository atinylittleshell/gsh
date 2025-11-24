package parser

import (
	"fmt"

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
