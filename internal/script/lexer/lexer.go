package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Lexer tokenizes gsh script source code
type Lexer struct {
	input        string
	position     int      // current position in input (points to current char)
	readPosition int      // current reading position in input (after current char)
	ch           byte     // current char under examination
	line         int      // current line number (1-indexed)
	column       int      // current column number (1-indexed)
	errors       []string // lexer errors
}

// New creates a new Lexer instance
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
		errors: []string{},
	}
	l.readChar()
	return l
}

// Errors returns the list of lexer errors
func (l *Lexer) Errors() []string {
	return l.errors
}

// addError adds a lexer error
func (l *Lexer) addError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.errors = append(l.errors, fmt.Sprintf("lexer error at line %d, column %d: %s", l.line, l.column, msg))
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	// Skip comments
	for l.ch == '#' {
		l.readLineComment()
		l.skipWhitespace()
	}

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OP_EQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(OP_ASSIGN, l.ch, tok.Line, tok.Column)
		}
	case '+':
		tok = newToken(OP_PLUS, l.ch, tok.Line, tok.Column)
	case '-':
		tok = newToken(OP_MINUS, l.ch, tok.Line, tok.Column)
	case '*':
		tok = newToken(OP_ASTERISK, l.ch, tok.Line, tok.Column)
	case '/':
		tok = newToken(OP_SLASH, l.ch, tok.Line, tok.Column)
	case '%':
		tok = newToken(OP_PERCENT, l.ch, tok.Line, tok.Column)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OP_NEQ, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(OP_BANG, l.ch, tok.Line, tok.Column)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OP_LTE, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(OP_LT, l.ch, tok.Line, tok.Column)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OP_GTE, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(OP_GT, l.ch, tok.Line, tok.Column)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OP_AND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Line: tok.Line, Column: tok.Column}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OP_OR, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(OP_PIPE, l.ch, tok.Line, tok.Column)
		}
	case '?':
		if l.peekChar() == '?' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OP_NULLCOAL, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(OP_QUESTION, l.ch, tok.Line, tok.Column)
		}
	case ',':
		tok = newToken(COMMA, l.ch, tok.Line, tok.Column)
	case ':':
		tok = newToken(COLON, l.ch, tok.Line, tok.Column)
	case ';':
		tok = newToken(SEMICOLON, l.ch, tok.Line, tok.Column)
	case '.':
		tok = newToken(DOT, l.ch, tok.Line, tok.Column)
	case '(':
		tok = newToken(LPAREN, l.ch, tok.Line, tok.Column)
	case ')':
		tok = newToken(RPAREN, l.ch, tok.Line, tok.Column)
	case '{':
		tok = newToken(LBRACE, l.ch, tok.Line, tok.Column)
	case '}':
		tok = newToken(RBRACE, l.ch, tok.Line, tok.Column)
	case '[':
		tok = newToken(LBRACKET, l.ch, tok.Line, tok.Column)
	case ']':
		tok = newToken(RBRACKET, l.ch, tok.Line, tok.Column)
	case '"':
		// Check for triple-quoted string
		if l.peekChar() == '"' && l.peekCharN(2) == '"' {
			tok.Type = STRING
			tok.Literal = l.readTripleQuotedString('"')
		} else {
			tok.Type = STRING
			tok.Literal = l.readString('"')
		}
		return tok
	case '\'':
		// Check for triple-quoted string
		if l.peekChar() == '\'' && l.peekCharN(2) == '\'' {
			tok.Type = STRING
			tok.Literal = l.readTripleQuotedString('\'')
		} else {
			tok.Type = STRING
			tok.Literal = l.readString('\'')
		}
		return tok
	case '`':
		tok.Type = TEMPLATE_LITERAL
		tok.Literal = l.readTemplateString()
		return tok
	case '#':
		tok.Type = COMMENT
		tok.Literal = l.readLineComment()
		return tok
	case 0:
		tok.Literal = ""
		tok.Type = EOF
		tok.Line = l.line
		tok.Column = l.column
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch, tok.Line, tok.Column)
		}
	}

	l.readChar()
	return tok
}

// readChar advances the lexer's position and updates the current character
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++

	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

// peekChar returns the next character without advancing the position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// peekCharN returns the character n positions ahead without advancing
func (l *Lexer) peekCharN(n int) byte {
	pos := l.position + n
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// skipWhitespace skips over whitespace characters (but not newlines in all cases)
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

// readString reads a quoted string (single or double quotes)
func (l *Lexer) readString(quote byte) string {
	var result strings.Builder
	startLine := l.line
	startColumn := l.column
	l.readChar() // consume opening quote

	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case '\'':
				result.WriteByte('\'')
			case 'u':
				// Unicode escape sequence: \uXXXX (4 hex digits)
				l.readChar() // consume 'u'
				hex := ""
				for i := 0; i < 4 && isHexDigit(l.ch); i++ {
					hex += string(l.ch)
					l.readChar()
				}
				if len(hex) == 4 {
					if codePoint, err := strconv.ParseInt(hex, 16, 32); err == nil {
						result.WriteRune(rune(codePoint))
					} else {
						// Invalid hex, write literal
						result.WriteString("\\u")
						result.WriteString(hex)
					}
				} else {
					// Not enough hex digits, write literal
					result.WriteString("\\u")
					result.WriteString(hex)
				}
				continue // Already consumed the next character
			default:
				// For unknown escapes, keep the backslash and character
				result.WriteByte('\\')
				result.WriteByte(l.ch)
			}
			l.readChar()
		} else {
			result.WriteByte(l.ch)
			l.readChar()
		}
	}

	if l.ch == quote {
		l.readChar() // consume closing quote
	} else {
		// Reached EOF without closing quote
		l.addError("unterminated string literal starting at line %d, column %d", startLine, startColumn)
	}

	return result.String()
}

// readTripleQuotedString reads a triple-quoted string
func (l *Lexer) readTripleQuotedString(quote byte) string {
	startLine := l.line
	startColumn := l.column
	// Consume opening triple quotes
	l.readChar() // first quote
	l.readChar() // second quote
	l.readChar() // third quote

	var result strings.Builder
	foundClosing := false

	for l.ch != 0 {
		// Check for closing triple quotes
		if l.ch == quote && l.peekChar() == quote && l.peekCharN(2) == quote {
			l.readChar() // first closing quote
			l.readChar() // second closing quote
			l.readChar() // third closing quote
			foundClosing = true
			break
		}

		result.WriteByte(l.ch)
		l.readChar()
	}

	if !foundClosing {
		// Reached EOF without closing triple quotes
		l.addError("unterminated triple-quoted string starting at line %d, column %d", startLine, startColumn)
	}

	content := result.String()

	// Remove common leading whitespace (dedent)
	content = dedent(content)

	// Trim leading/trailing whitespace (newlines and spaces)
	content = strings.TrimSpace(content)

	return content
}

// readTemplateString reads a template string with interpolation
func (l *Lexer) readTemplateString() string {
	var result strings.Builder
	startLine := l.line
	startColumn := l.column
	l.readChar() // consume opening backtick

	for l.ch != '`' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '`':
				result.WriteByte('`')
			case '$':
				// Use a placeholder that won't be confused with real interpolation
				// The interpreter will convert this back to a literal $
				result.WriteString("\x00ESCAPED_DOLLAR\x00")
			case 'u':
				// Unicode escape sequence: \uXXXX (4 hex digits)
				l.readChar() // consume 'u'
				hex := ""
				for i := 0; i < 4 && isHexDigit(l.ch); i++ {
					hex += string(l.ch)
					l.readChar()
				}
				if len(hex) == 4 {
					if codePoint, err := strconv.ParseInt(hex, 16, 32); err == nil {
						result.WriteRune(rune(codePoint))
					} else {
						// Invalid hex, write literal
						result.WriteString("\\u")
						result.WriteString(hex)
					}
				} else {
					// Not enough hex digits, write literal
					result.WriteString("\\u")
					result.WriteString(hex)
				}
				continue // Already consumed the next character
			default:
				result.WriteByte('\\')
				result.WriteByte(l.ch)
			}
			l.readChar()
		} else {
			result.WriteByte(l.ch)
			l.readChar()
		}
	}

	if l.ch == '`' {
		l.readChar() // consume closing backtick
	} else {
		// Reached EOF without closing backtick
		l.addError("unterminated template string starting at line %d, column %d", startLine, startColumn)
	}

	return result.String()
}

// readLineComment reads a comment until end of line
func (l *Lexer) readLineComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

// isLetter checks if a character is a letter or underscore
func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

// isDigit checks if a character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

// newToken creates a new token from a single character
func newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{Type: tokenType, Literal: string(ch), Line: line, Column: column}
}

// dedent removes common leading whitespace from multi-line strings
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}

	// Find minimum indentation (ignoring empty lines)
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent++
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return s
	}

	// Remove common indentation
	var result strings.Builder
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			result.WriteString(line)
		} else if len(line) > minIndent {
			result.WriteString(line[minIndent:])
		}
		if i < len(lines)-1 {
			result.WriteByte('\n')
		}
	}

	return result.String()
}

// Error returns a formatted error message with line and column information
func (l *Lexer) Error(msg string) string {
	return fmt.Sprintf("lexer error at line %d, column %d: %s", l.line, l.column, msg)
}
