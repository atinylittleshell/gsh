package parser

import (
	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

// parseMcpDeclaration parses an MCP server declaration
// mcp <name> { <config> }
func (p *Parser) parseMcpDeclaration() Statement {
	stmt := &McpDeclaration{Token: p.curToken}

	// Expect identifier for MCP server name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect '{' after name
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken() // move past '{'

	// Parse configuration object
	stmt.Config = p.parseDeclarationConfig()
	if stmt.Config == nil {
		return nil
	}

	// Expect '}' to close the declaration
	if !p.curTokenIs(lexer.RBRACE) {
		p.addError("expected '}' at line %d, column %d", p.curToken.Line, p.curToken.Column)
		return nil
	}

	return stmt
}

// parseModelDeclaration parses a model declaration
// model <name> { <config> }
func (p *Parser) parseModelDeclaration() Statement {
	stmt := &ModelDeclaration{Token: p.curToken}

	// Expect identifier for model name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect '{' after name
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken() // move past '{'

	// Parse configuration object
	stmt.Config = p.parseDeclarationConfig()
	if stmt.Config == nil {
		return nil
	}

	// Expect '}' to close the declaration
	if !p.curTokenIs(lexer.RBRACE) {
		p.addError("expected '}' at line %d, column %d", p.curToken.Line, p.curToken.Column)
		return nil
	}

	return stmt
}

// parseDeclarationConfig parses a configuration object inside a declaration
// This is similar to object literal but uses a different structure for declarations
func (p *Parser) parseDeclarationConfig() map[string]Expression {
	config := make(map[string]Expression)

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		// Skip commas
		if p.curTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}

		// Expect identifier or keyword for config key (keywords like "model" can be used as keys)
		if !p.curTokenIs(lexer.IDENT) && !p.isKeyword(p.curToken.Type) {
			p.addError("expected identifier for config key, got %v at line %d, column %d",
				p.curToken.Type, p.curToken.Line, p.curToken.Column)
			return nil
		}

		key := p.curToken.Literal

		// Expect ':' after key
		if !p.expectPeek(lexer.COLON) {
			return nil
		}

		p.nextToken() // move to value expression

		// Parse value
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}

		config[key] = value

		// Optional comma or closing brace
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
		}

		p.nextToken() // move to next key or closing brace
	}

	return config
}

// isKeyword checks if a token type is a keyword
func (p *Parser) isKeyword(t lexer.TokenType) bool {
	return t == lexer.KW_MCP || t == lexer.KW_MODEL || t == lexer.KW_AGENT ||
		t == lexer.KW_TOOL || t == lexer.KW_IF || t == lexer.KW_ELSE ||
		t == lexer.KW_FOR || t == lexer.KW_OF || t == lexer.KW_WHILE ||
		t == lexer.KW_BREAK || t == lexer.KW_CONTINUE || t == lexer.KW_TRY ||
		t == lexer.KW_CATCH || t == lexer.KW_FINALLY || t == lexer.KW_RETURN
}
