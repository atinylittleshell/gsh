package parser

import (
	"strconv"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

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
