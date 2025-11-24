package lexer

import "testing"

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		expected  string
	}{
		{ILLEGAL, "ILLEGAL"},
		{EOF, "EOF"},
		{COMMENT, "COMMENT"},
		{IDENT, "IDENT"},
		{NUMBER, "NUMBER"},
		{STRING, "STRING"},
		{KW_MCP, "KW_MCP"},
		{KW_MODEL, "KW_MODEL"},
		{KW_AGENT, "KW_AGENT"},
		{KW_TOOL, "KW_TOOL"},
		{KW_IF, "KW_IF"},
		{KW_ELSE, "KW_ELSE"},
		{KW_FOR, "KW_FOR"},
		{KW_OF, "KW_OF"},
		{KW_WHILE, "KW_WHILE"},
		{KW_BREAK, "KW_BREAK"},
		{KW_CONTINUE, "KW_CONTINUE"},
		{KW_TRY, "KW_TRY"},
		{KW_CATCH, "KW_CATCH"},
		{KW_RETURN, "KW_RETURN"},
		{OP_ASSIGN, "OP_ASSIGN"},
		{OP_PLUS, "OP_PLUS"},
		{OP_MINUS, "OP_MINUS"},
		{OP_ASTERISK, "OP_ASTERISK"},
		{OP_SLASH, "OP_SLASH"},
		{OP_PERCENT, "OP_PERCENT"},
		{OP_BANG, "OP_BANG"},
		{OP_EQ, "OP_EQ"},
		{OP_NEQ, "OP_NEQ"},
		{OP_LT, "OP_LT"},
		{OP_GT, "OP_GT"},
		{OP_LTE, "OP_LTE"},
		{OP_GTE, "OP_GTE"},
		{OP_AND, "OP_AND"},
		{OP_OR, "OP_OR"},
		{OP_PIPE, "OP_PIPE"},
		{OP_QUESTION, "OP_QUESTION"},
		{OP_NULLCOAL, "OP_NULLCOAL"},
		{COMMA, "COMMA"},
		{COLON, "COLON"},
		{SEMICOLON, "SEMICOLON"},
		{DOT, "DOT"},
		{LPAREN, "LPAREN"},
		{RPAREN, "RPAREN"},
		{LBRACE, "LBRACE"},
		{RBRACE, "RBRACE"},
		{LBRACKET, "LBRACKET"},
		{RBRACKET, "RBRACKET"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.tokenType.String()
			if result != tt.expected {
				t.Errorf("TokenType.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLookupIdent(t *testing.T) {
	tests := []struct {
		name     string
		ident    string
		expected TokenType
	}{
		// Keywords
		{"mcp keyword", "mcp", KW_MCP},
		{"model keyword", "model", KW_MODEL},
		{"agent keyword", "agent", KW_AGENT},
		{"tool keyword", "tool", KW_TOOL},
		{"if keyword", "if", KW_IF},
		{"else keyword", "else", KW_ELSE},
		{"for keyword", "for", KW_FOR},
		{"of keyword", "of", KW_OF},
		{"while keyword", "while", KW_WHILE},
		{"break keyword", "break", KW_BREAK},
		{"continue keyword", "continue", KW_CONTINUE},
		{"try keyword", "try", KW_TRY},
		{"catch keyword", "catch", KW_CATCH},
		{"return keyword", "return", KW_RETURN},

		// Regular identifiers
		{"variable name", "variableName", IDENT},
		{"function name", "myFunction", IDENT},
		{"underscore", "_private", IDENT},
		{"camelCase", "camelCase", IDENT},
		{"snake_case", "snake_case", IDENT},
		{"with numbers", "var123", IDENT},
		{"single char", "x", IDENT},

		// Case sensitivity - keywords are lowercase
		{"uppercase MCP", "MCP", IDENT},
		{"uppercase MODEL", "MODEL", IDENT},
		{"uppercase IF", "IF", IDENT},
		{"mixed case If", "If", IDENT},

		// Similar to keywords but not exact
		{"models", "models", IDENT},
		{"tools", "tools", IDENT},
		{"ifStatement", "ifStatement", IDENT},
		{"breaking", "breaking", IDENT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LookupIdent(tt.ident)
			if result != tt.expected {
				t.Errorf("LookupIdent(%q) = %v, want %v", tt.ident, result, tt.expected)
			}
		})
	}
}

func TestTokenCreation(t *testing.T) {
	token := Token{
		Type:    IDENT,
		Literal: "myVar",
		Line:    1,
		Column:  5,
	}

	if token.Type != IDENT {
		t.Errorf("Token.Type = %v, want IDENT", token.Type)
	}
	if token.Literal != "myVar" {
		t.Errorf("Token.Literal = %q, want %q", token.Literal, "myVar")
	}
	if token.Line != 1 {
		t.Errorf("Token.Line = %d, want 1", token.Line)
	}
	if token.Column != 5 {
		t.Errorf("Token.Column = %d, want 5", token.Column)
	}
}

func TestAllKeywordsAreDefined(t *testing.T) {
	// Ensure all keywords in the map are valid token types
	for keyword, tokenType := range keywords {
		// Check that the token type is not IDENT (should be a keyword type)
		if tokenType == IDENT {
			t.Errorf("Keyword %q maps to IDENT, should map to a keyword token type", keyword)
		}

		// Check that LookupIdent returns the correct type
		result := LookupIdent(keyword)
		if result != tokenType {
			t.Errorf("LookupIdent(%q) = %v, want %v", keyword, result, tokenType)
		}
	}
}

func TestTokenTypeUniqueness(t *testing.T) {
	// Ensure all token types have unique string representations
	seen := make(map[string]TokenType)

	// Test all token types up to RBRACKET
	for i := ILLEGAL; i <= RBRACKET; i++ {
		str := i.String()
		if existing, found := seen[str]; found {
			t.Errorf("Duplicate token type string %q for types %v and %v", str, existing, i)
		}
		seen[str] = i
	}
}

func TestKeywordCoverage(t *testing.T) {
	// Ensure we have tests for all expected gsh keywords based on the spec
	expectedKeywords := []string{
		"mcp", "model", "agent", "tool",
		"if", "else", "for", "of", "while",
		"break", "continue", "try", "catch", "return",
	}

	for _, keyword := range expectedKeywords {
		tokenType := LookupIdent(keyword)
		if tokenType == IDENT {
			t.Errorf("Expected keyword %q is not registered in keywords map", keyword)
		}
	}
}

func TestOperatorTokens(t *testing.T) {
	// Test that we have all necessary operator tokens defined
	operators := []TokenType{
		OP_ASSIGN, OP_PLUS, OP_MINUS, OP_ASTERISK, OP_SLASH, OP_PERCENT,
		OP_BANG, OP_EQ, OP_NEQ, OP_LT, OP_GT, OP_LTE, OP_GTE,
		OP_AND, OP_OR, OP_PIPE, OP_QUESTION, OP_NULLCOAL,
	}

	for _, op := range operators {
		str := op.String()
		// Ensure operator has a valid string representation
		if str == "" || (len(str) >= 7 && str[:7] == "UNKNOWN") {
			t.Errorf("Operator token %d has invalid string representation: %q", op, str)
		}
	}
}

func TestDelimiterTokens(t *testing.T) {
	// Test that we have all necessary delimiter tokens defined
	delimiters := []TokenType{
		COMMA, COLON, SEMICOLON, DOT,
		LPAREN, RPAREN, LBRACE, RBRACE, LBRACKET, RBRACKET,
	}

	for _, delim := range delimiters {
		str := delim.String()
		// Ensure delimiter has a valid string representation
		if str == "" || (len(str) >= 7 && str[:7] == "UNKNOWN") {
			t.Errorf("Delimiter token %d has invalid string representation: %q", delim, str)
		}
	}
}
