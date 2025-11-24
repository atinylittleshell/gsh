package parser

import (
	"github.com/atinylittleshell/gsh/internal/script/lexer"
)

// parseStatement parses a statement
func (p *Parser) parseStatement() Statement {
	// Check for declaration keywords
	switch p.curToken.Type {
	case lexer.KW_MCP:
		return p.parseMcpDeclaration()
	case lexer.KW_IF:
		return p.parseIfStatement()
	case lexer.KW_WHILE:
		return p.parseWhileStatement()
	case lexer.KW_FOR:
		return p.parseForOfStatement()
	case lexer.KW_BREAK:
		return p.parseBreakStatement()
	case lexer.KW_CONTINUE:
		return p.parseContinueStatement()
	case lexer.KW_TRY:
		return p.parseTryStatement()
	}

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

// parseBlockStatement parses a block of statements
func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}

	p.nextToken() // move past '{'

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		// Skip semicolons (they're optional statement terminators)
		if p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.RBRACE) {
		p.addError("expected '}' at line %d, column %d", p.curToken.Line, p.curToken.Column)
		return nil
	}

	return block
}

// parseIfStatement parses an if/else statement
func (p *Parser) parseIfStatement() Statement {
	stmt := &IfStatement{Token: p.curToken}

	// Expect '(' after 'if'
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken() // move to condition expression

	// Parse condition
	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		return nil
	}

	// Expect ')' after condition
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Expect '{' after ')'
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse consequence block
	stmt.Consequence = p.parseBlockStatement()
	if stmt.Consequence == nil {
		return nil
	}

	// Check for 'else' or 'else if'
	if p.peekTokenIs(lexer.KW_ELSE) {
		p.nextToken() // consume 'else'

		// Check for 'else if'
		if p.peekTokenIs(lexer.KW_IF) {
			p.nextToken() // move to 'if'
			stmt.Alternative = p.parseIfStatement()
		} else {
			// Expect '{' for else block
			if !p.expectPeek(lexer.LBRACE) {
				return nil
			}
			stmt.Alternative = p.parseBlockStatement()
		}
	}

	return stmt
}

// parseWhileStatement parses a while loop
func (p *Parser) parseWhileStatement() Statement {
	stmt := &WhileStatement{Token: p.curToken}

	// Expect '(' after 'while'
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken() // move to condition expression

	// Parse condition
	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		return nil
	}

	// Expect ')' after condition
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Expect '{' after ')'
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse body block
	stmt.Body = p.parseBlockStatement()
	if stmt.Body == nil {
		return nil
	}

	return stmt
}

// parseForOfStatement parses a for-of loop
func (p *Parser) parseForOfStatement() Statement {
	stmt := &ForOfStatement{Token: p.curToken}

	// Expect '(' after 'for'
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Expect identifier for loop variable
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Variable = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect 'of' keyword
	if !p.expectPeek(lexer.KW_OF) {
		return nil
	}

	p.nextToken() // move to iterable expression

	// Parse iterable
	stmt.Iterable = p.parseExpression(LOWEST)
	if stmt.Iterable == nil {
		return nil
	}

	// Expect ')' after iterable
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Expect '{' after ')'
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse body block
	stmt.Body = p.parseBlockStatement()
	if stmt.Body == nil {
		return nil
	}

	return stmt
}

// parseBreakStatement parses a break statement
func (p *Parser) parseBreakStatement() Statement {
	return &BreakStatement{Token: p.curToken}
}

// parseContinueStatement parses a continue statement
func (p *Parser) parseContinueStatement() Statement {
	return &ContinueStatement{Token: p.curToken}
}

// parseTryStatement parses a try/catch/finally statement
func (p *Parser) parseTryStatement() Statement {
	stmt := &TryStatement{Token: p.curToken}

	// Expect '{' after 'try'
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse try block
	stmt.Block = p.parseBlockStatement()
	if stmt.Block == nil {
		return nil
	}

	// Check for catch clause
	if p.peekTokenIs(lexer.KW_CATCH) {
		p.nextToken() // move to 'catch'
		stmt.CatchClause = p.parseCatchClause()
		if stmt.CatchClause == nil {
			return nil
		}
	}

	// Check for finally clause
	if p.peekTokenIs(lexer.KW_FINALLY) {
		p.nextToken() // move to 'finally'
		stmt.FinallyClause = p.parseFinallyClause()
		if stmt.FinallyClause == nil {
			return nil
		}
	}

	// Validate that at least one of catch or finally is present
	if stmt.CatchClause == nil && stmt.FinallyClause == nil {
		p.addError("try statement must have at least one catch or finally clause at line %d, column %d",
			stmt.Token.Line, stmt.Token.Column)
		return nil
	}

	return stmt
}

// parseCatchClause parses a catch clause
func (p *Parser) parseCatchClause() *CatchClause {
	clause := &CatchClause{Token: p.curToken}

	// Expect '(' after 'catch'
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Expect identifier for error parameter
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	clause.Parameter = &Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect ')' after parameter
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Expect '{' after ')'
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse catch block
	clause.Block = p.parseBlockStatement()
	if clause.Block == nil {
		return nil
	}

	return clause
}

// parseFinallyClause parses a finally clause
func (p *Parser) parseFinallyClause() *FinallyClause {
	clause := &FinallyClause{Token: p.curToken}

	// Expect '{' after 'finally'
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse finally block
	clause.Block = p.parseBlockStatement()
	if clause.Block == nil {
		return nil
	}

	return clause
}
