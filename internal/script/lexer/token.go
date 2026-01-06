package lexer

//go:generate stringer -type=TokenType -trimprefix=TokenType

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Identifiers and literals
	IDENT            // variable names, function names
	NUMBER           // 123, 45.67
	STRING           // "hello", 'world', """multiline"""
	TEMPLATE_LITERAL // `template ${expr}`

	// Keywords
	KW_MCP
	KW_MODEL
	KW_AGENT
	KW_ACP
	KW_TOOL
	KW_IF
	KW_ELSE
	KW_FOR
	KW_OF
	KW_WHILE
	KW_BREAK
	KW_CONTINUE
	KW_TRY
	KW_CATCH
	KW_FINALLY
	KW_RETURN
	KW_IMPORT
	KW_EXPORT
	KW_FROM
	KW_GO // Reserved for future concurrency support (fire-and-forget)

	// Operators
	OP_ASSIGN   // =
	OP_PLUS     // +
	OP_MINUS    // -
	OP_ASTERISK // *
	OP_SLASH    // /
	OP_PERCENT  // %
	OP_BANG     // !
	OP_EQ       // ==
	OP_NEQ      // !=
	OP_LT       // <
	OP_GT       // >
	OP_LTE      // <=
	OP_GTE      // >=
	OP_AND      // &&
	OP_OR       // ||
	OP_PIPE     // |
	OP_QUESTION // ?
	OP_NULLCOAL // ??

	// Delimiters
	COMMA     // ,
	COLON     // :
	SEMICOLON // ;
	DOT       // .
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
)

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// keywords maps keyword strings to their token types
var keywords = map[string]TokenType{
	"mcp":      KW_MCP,
	"model":    KW_MODEL,
	"agent":    KW_AGENT,
	"acp":      KW_ACP,
	"tool":     KW_TOOL,
	"if":       KW_IF,
	"else":     KW_ELSE,
	"for":      KW_FOR,
	"of":       KW_OF,
	"while":    KW_WHILE,
	"break":    KW_BREAK,
	"continue": KW_CONTINUE,
	"try":      KW_TRY,
	"catch":    KW_CATCH,
	"finally":  KW_FINALLY,
	"return":   KW_RETURN,
	"import":   KW_IMPORT,
	"export":   KW_EXPORT,
	"from":     KW_FROM,
	"go":       KW_GO, // Reserved for future concurrency support (fire-and-forget)
}

// LookupIdent checks if an identifier is a keyword and returns the appropriate token type
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// keywordTypes is a set of all keyword token types, derived from the keywords map
var keywordTypes = func() map[TokenType]bool {
	m := make(map[TokenType]bool)
	for _, tokenType := range keywords {
		m[tokenType] = true
	}
	return m
}()

// IsKeyword returns true if the token type is a keyword
func IsKeyword(t TokenType) bool {
	return keywordTypes[t]
}
