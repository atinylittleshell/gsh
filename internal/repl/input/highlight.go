// Package input provides input handling for the gsh REPL.
package input

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/repl/render"
	"github.com/charmbracelet/lipgloss"
	"mvdan.cc/sh/v3/syntax"
)

// TokenType represents the type of a syntax token for highlighting.
type TokenType int

const (
	TokenDefault     TokenType = iota // Default text, no special highlighting
	TokenCommand                      // Command/executable name
	TokenCommandOK                    // Command that exists (green)
	TokenCommandErr                   // Command that doesn't exist (red)
	TokenString                       // Quoted strings
	TokenVariableOK                   // Variables with non-empty value (green)
	TokenVariableErr                  // Variables that are empty/unset (red)
	TokenOperator                     // Operators (|, &&, ||, ;, >, <)
	TokenFlag                         // Flags (-f, --help)
	TokenComment                      // Comments (# ...)
	TokenAgentPrefix                  // Agent mode prefix (#)
	TokenAgentCmd                     // Agent commands (/clear, /agents)
)

// builtinCommands contains shell built-ins and gsh built-ins that should be highlighted as valid
var builtinCommands = map[string]bool{
	// gsh built-ins
	"exit": true,
}

// StyledSpan represents a portion of text with a specific style.
type StyledSpan struct {
	Text  string
	Style lipgloss.Style
}

// Highlighter provides syntax highlighting for shell input.
type Highlighter struct {
	parser *syntax.Parser

	// aliasExists, if set, is consulted before checking PATH.
	// This is used to treat shell aliases and functions (e.g. from ~/.gshrc) as valid commands.
	aliasExists func(name string) bool

	// getEnv, if set, is used to get environment variables from the shell.
	// This allows the highlighter to use the shell's PATH (which may have been
	// modified in .gshenv or .gsh_profile) instead of the OS process's PATH.
	getEnv func(name string) string

	// getWorkingDir, if set, is used to resolve relative paths. This keeps
	// highlighting aligned with the shell's current working directory, which may
	// differ from the process working directory when users run `cd` inside gsh.
	getWorkingDir func() string

	// Styles for different token types
	styles map[TokenType]lipgloss.Style
}

// NewHighlighter creates a new syntax highlighter.
//
// If aliasExists is non-nil, any name for which it returns true is treated as an
// existing command (useful for shell aliases loaded at runtime).
//
// If getEnv is non-nil, it is used to get environment variables from the shell,
// allowing the highlighter to use the shell's PATH and variable values.
//
// If getWorkingDir is non-nil, it is used to resolve relative command paths so
// highlighting matches the shell's current working directory.
func NewHighlighter(
	aliasExists func(name string) bool,
	getEnv func(name string) string,
	getWorkingDir func() string,
) *Highlighter {
	h := &Highlighter{
		parser:        syntax.NewParser(),
		aliasExists:   aliasExists,
		getEnv:        getEnv,
		getWorkingDir: getWorkingDir,
		styles:        make(map[TokenType]lipgloss.Style),
	}

	// Initialize styles
	h.styles[TokenDefault] = lipgloss.NewStyle()
	h.styles[TokenCommandOK] = lipgloss.NewStyle().Foreground(render.ColorGreen)
	h.styles[TokenCommandErr] = lipgloss.NewStyle().Foreground(render.ColorRed)
	h.styles[TokenString] = lipgloss.NewStyle().Foreground(render.ColorMagenta)
	h.styles[TokenVariableOK] = lipgloss.NewStyle().Foreground(render.ColorGreen)
	h.styles[TokenVariableErr] = lipgloss.NewStyle().Foreground(render.ColorRed)
	h.styles[TokenOperator] = lipgloss.NewStyle().Foreground(render.ColorYellow)
	h.styles[TokenFlag] = lipgloss.NewStyle().Foreground(render.ColorBlue)
	h.styles[TokenComment] = lipgloss.NewStyle().Foreground(render.ColorGray)
	h.styles[TokenAgentPrefix] = lipgloss.NewStyle().Foreground(render.ColorYellow).Bold(true)
	h.styles[TokenAgentCmd] = lipgloss.NewStyle().Foreground(render.ColorYellow)

	return h
}

// commandExists checks if a command exists in PATH, is an alias, or is a built-in.
func (h *Highlighter) commandExists(cmd string) bool {
	// Check built-ins first
	if builtinCommands[cmd] {
		return true
	}

	// Treat aliases as valid commands
	if h.aliasExists != nil && h.aliasExists(cmd) {
		return true
	}

	// Get the current PATH from the shell (or fall back to OS PATH)
	currentPath := os.Getenv("PATH")
	if h.getEnv != nil {
		if shellPath := h.getEnv("PATH"); shellPath != "" {
			currentPath = shellPath
		}
	}

	// Check if command exists in the shell's PATH
	return h.lookupCommandInPath(cmd, currentPath)
}

// lookupCommandInPath checks if a command exists in the given PATH.
func (h *Highlighter) lookupCommandInPath(cmd string, pathEnv string) bool {
	// If the command contains a path separator, check it directly
	if strings.Contains(cmd, "/") {
		pathToCheck := cmd
		if !filepath.IsAbs(cmd) {
			if wd := h.workingDir(); wd != "" {
				pathToCheck = filepath.Join(wd, cmd)
			}
		}

		info, err := os.Stat(pathToCheck)
		if err != nil {
			return false
		}
		// Check if it's executable (not a directory)
		return !info.IsDir() && info.Mode()&0111 != 0
	}

	// Search each directory in PATH
	for _, dir := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		baseDir := dir
		if baseDir == "" {
			baseDir = "."
		}

		path := filepath.Join(baseDir, cmd)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		// Check if it's executable (not a directory)
		if !info.IsDir() && info.Mode()&0111 != 0 {
			return true
		}
	}
	return false
}

func (h *Highlighter) workingDir() string {
	if h.getWorkingDir != nil {
		if dir := h.getWorkingDir(); dir != "" {
			return dir
		}
	}

	if dir, err := os.Getwd(); err == nil {
		return dir
	}

	return ""
}

// variableHasValue checks if an environment variable has a non-empty value.
// It uses the shell's environment if available, falling back to the OS environment.
func (h *Highlighter) variableHasValue(name string) bool {
	// Handle special variables that are always set
	switch name {
	case "?", "$", "!", "#", "*", "@", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		return true
	}

	// Try the shell's environment first
	if h.getEnv != nil {
		value := h.getEnv(name)
		if value != "" {
			return true
		}
	}

	// Fall back to OS environment
	value, exists := os.LookupEnv(name)
	return exists && value != ""
}

// extractVariableName extracts the variable name from a variable reference like $VAR or ${VAR}.
func (h *Highlighter) extractVariableName(varText string) string {
	if len(varText) < 2 {
		return ""
	}

	// Skip the leading $
	rest := varText[1:]

	// Handle ${VAR} format
	if len(rest) > 0 && rest[0] == '{' {
		// Find closing brace
		end := strings.IndexByte(rest, '}')
		if end > 1 {
			// Extract name, handling modifiers like ${VAR:-default}
			name := rest[1:end]
			// Find any modifier (:-  :+  :?  :=  etc.)
			for i, c := range name {
				if c == ':' || c == '-' || c == '+' || c == '=' || c == '?' || c == '#' || c == '%' {
					return name[:i]
				}
			}
			return name
		}
		return ""
	}

	// Handle $VAR format - just alphanumeric and underscore
	var name strings.Builder
	for _, c := range rest {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			name.WriteRune(c)
		} else {
			break
		}
	}
	return name.String()
}

// Highlight returns the input text with syntax highlighting applied.
func (h *Highlighter) Highlight(input string) string {
	if input == "" {
		return ""
	}

	// Check for agent mode (starts with #)
	trimmed := strings.TrimLeft(input, " \t")
	if strings.HasPrefix(trimmed, "#") {
		return h.highlightAgentMode(input)
	}

	// Parse as shell command
	return h.highlightShell(input)
}

// highlightAgentMode highlights input in agent mode.
func (h *Highlighter) highlightAgentMode(input string) string {
	// Find the # prefix (preserving leading whitespace)
	prefixEnd := strings.Index(input, "#") + 1

	var result strings.Builder

	// Leading whitespace (if any)
	if prefixEnd > 1 {
		result.WriteString(input[:prefixEnd-1])
	}

	// The # prefix in yellow bold
	result.WriteString(h.styles[TokenAgentPrefix].Render("#"))

	// Rest of the input
	rest := input[prefixEnd:]
	if rest == "" {
		return result.String()
	}

	// Check for agent commands like /clear, /agents
	trimmedRest := strings.TrimLeft(rest, " \t")
	if strings.HasPrefix(trimmedRest, "/") {
		// Find where the command ends
		leadingSpace := rest[:len(rest)-len(trimmedRest)]
		result.WriteString(leadingSpace)

		// Find end of the command (space or end of string)
		cmdEnd := strings.IndexAny(trimmedRest, " \t")
		if cmdEnd == -1 {
			result.WriteString(h.styles[TokenAgentCmd].Render(trimmedRest))
		} else {
			result.WriteString(h.styles[TokenAgentCmd].Render(trimmedRest[:cmdEnd]))
			result.WriteString(trimmedRest[cmdEnd:])
		}
	} else {
		// Regular agent message - no special highlighting
		result.WriteString(rest)
	}

	return result.String()
}

// highlightShell highlights shell command input using the syntax parser.
func (h *Highlighter) highlightShell(input string) string {
	// Try to parse the input
	file, err := h.parser.Parse(strings.NewReader(input), "")
	if err != nil {
		// If parsing fails, try basic highlighting without full parse
		return h.highlightBasic(input)
	}

	// Build highlighted output by walking the AST
	spans := h.walkFile(file, input)
	return h.renderSpans(spans, input)
}

// highlightBasic provides basic highlighting when full parsing fails.
// This handles incomplete commands being typed.
func (h *Highlighter) highlightBasic(input string) string {
	var result strings.Builder
	runes := []rune(input)
	i := 0

	// Skip leading whitespace
	for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
		result.WriteRune(runes[i])
		i++
	}

	// First word is the command
	cmdStart := i
	for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
		i++
	}

	if cmdStart < i {
		cmd := string(runes[cmdStart:i])
		if h.commandExists(cmd) {
			result.WriteString(h.styles[TokenCommandOK].Render(cmd))
		} else {
			result.WriteString(h.styles[TokenCommandErr].Render(cmd))
		}
	}

	// Process the rest with basic token detection
	for i < len(runes) {
		r := runes[i]

		switch {
		case r == ' ' || r == '\t':
			result.WriteRune(r)
			i++

		case r == '#':
			// Comment - rest of line
			result.WriteString(h.styles[TokenComment].Render(string(runes[i:])))
			i = len(runes)

		case r == '"':
			// Double quoted string
			end := h.findStringEnd(runes, i, '"')
			result.WriteString(h.styles[TokenString].Render(string(runes[i:end])))
			i = end

		case r == '\'':
			// Single quoted string
			end := h.findStringEnd(runes, i, '\'')
			result.WriteString(h.styles[TokenString].Render(string(runes[i:end])))
			i = end

		case r == '$':
			// Variable
			end := h.findVariableEnd(runes, i)
			varText := string(runes[i:end])
			varName := h.extractVariableName(varText)
			if h.variableHasValue(varName) {
				result.WriteString(h.styles[TokenVariableOK].Render(varText))
			} else {
				result.WriteString(h.styles[TokenVariableErr].Render(varText))
			}
			i = end

		case r == '-' && i+1 < len(runes) && (runes[i+1] == '-' || isAlphaNum(runes[i+1])):
			// Flag
			end := h.findFlagEnd(runes, i)
			result.WriteString(h.styles[TokenFlag].Render(string(runes[i:end])))
			i = end

		case isOperator(r):
			// Operator
			end := h.findOperatorEnd(runes, i)
			result.WriteString(h.styles[TokenOperator].Render(string(runes[i:end])))
			i = end

		default:
			result.WriteRune(r)
			i++
		}
	}

	return result.String()
}

// tokenSpan represents a span of text with position info.
type tokenSpan struct {
	start int
	end   int
	style lipgloss.Style
}

// walkFile walks the AST and collects styled spans.
func (h *Highlighter) walkFile(file *syntax.File, input string) []tokenSpan {
	var spans []tokenSpan

	// Track which positions we've already covered
	syntax.Walk(file, func(node syntax.Node) bool {
		if node == nil {
			return true
		}

		pos := node.Pos()
		end := node.End()
		startOffset := int(pos.Offset())
		endOffset := int(end.Offset())

		switch n := node.(type) {
		case *syntax.Lit:
			// Check if this is the command (first word in a CallExpr)
			if parent, ok := h.getParentCallExpr(file, node); ok {
				if len(parent.Args) > 0 && parent.Args[0].Pos() == n.Pos() {
					// This is the command
					if h.commandExists(n.Value) {
						spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenCommandOK]})
					} else {
						spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenCommandErr]})
					}
					return true
				}
			}
			// Check if it's a flag
			if strings.HasPrefix(n.Value, "-") {
				spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenFlag]})
			}

		case *syntax.SglQuoted:
			spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenString]})

		case *syntax.DblQuoted:
			spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenString]})

		case *syntax.ParamExp:
			varName := n.Param.Value
			if h.variableHasValue(varName) {
				spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenVariableOK]})
			} else {
				spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenVariableErr]})
			}

		case *syntax.BinaryCmd:
			// Highlight the operator
			opStr := n.Op.String()
			opOffset := startOffset
			// Find the operator position in the input
			for i := startOffset; i < endOffset-len(opStr)+1; i++ {
				if i+len(opStr) <= len(input) && input[i:i+len(opStr)] == opStr {
					opOffset = i
					break
				}
			}
			spans = append(spans, tokenSpan{opOffset, opOffset + len(opStr), h.styles[TokenOperator]})

		case *syntax.Redirect:
			// Highlight redirect operators
			opStr := n.Op.String()
			spans = append(spans, tokenSpan{startOffset, startOffset + len(opStr), h.styles[TokenOperator]})

		case *syntax.Comment:
			spans = append(spans, tokenSpan{startOffset, endOffset, h.styles[TokenComment]})
		}

		return true
	})

	return spans
}

// getParentCallExpr finds the parent CallExpr for a node if it exists.
func (h *Highlighter) getParentCallExpr(file *syntax.File, target syntax.Node) (*syntax.CallExpr, bool) {
	var result *syntax.CallExpr
	var found bool

	syntax.Walk(file, func(node syntax.Node) bool {
		if found {
			return false
		}
		if call, ok := node.(*syntax.CallExpr); ok {
			for _, arg := range call.Args {
				syntax.Walk(arg, func(n syntax.Node) bool {
					if n == target {
						result = call
						found = true
						return false
					}
					return true
				})
				if found {
					return false
				}
			}
		}
		return true
	})

	return result, found
}

// renderSpans renders the input with the collected spans applied.
func (h *Highlighter) renderSpans(spans []tokenSpan, input string) string {
	if len(spans) == 0 {
		return h.highlightBasic(input)
	}

	// Sort spans by start position
	sortSpans(spans)

	var result strings.Builder
	lastEnd := 0

	for _, span := range spans {
		// Skip overlapping spans
		if span.start < lastEnd {
			continue
		}

		// Add unstyled text before this span
		if span.start > lastEnd {
			result.WriteString(input[lastEnd:span.start])
		}

		// Add styled span
		if span.end <= len(input) {
			result.WriteString(span.style.Render(input[span.start:span.end]))
			lastEnd = span.end
		}
	}

	// Add remaining unstyled text
	if lastEnd < len(input) {
		result.WriteString(input[lastEnd:])
	}

	return result.String()
}

// sortSpans sorts spans by start position using simple insertion sort.
func sortSpans(spans []tokenSpan) {
	for i := 1; i < len(spans); i++ {
		key := spans[i]
		j := i - 1
		for j >= 0 && spans[j].start > key.start {
			spans[j+1] = spans[j]
			j--
		}
		spans[j+1] = key
	}
}

// Helper functions for basic highlighting

func (h *Highlighter) findStringEnd(runes []rune, start int, quote rune) int {
	i := start + 1
	for i < len(runes) {
		if runes[i] == '\\' && i+1 < len(runes) {
			i += 2 // Skip escaped character
			continue
		}
		if runes[i] == quote {
			return i + 1
		}
		i++
	}
	return len(runes) // Unclosed string
}

func (h *Highlighter) findVariableEnd(runes []rune, start int) int {
	i := start + 1
	if i >= len(runes) {
		return i
	}

	// Handle ${...}
	if runes[i] == '{' {
		i++
		for i < len(runes) && runes[i] != '}' {
			i++
		}
		if i < len(runes) {
			i++ // Include closing brace
		}
		return i
	}

	// Handle $VAR
	for i < len(runes) && (isAlphaNum(runes[i]) || runes[i] == '_') {
		i++
	}
	return i
}

func (h *Highlighter) findFlagEnd(runes []rune, start int) int {
	i := start + 1

	// Handle -- prefix
	if i < len(runes) && runes[i] == '-' {
		i++
	}

	// Read flag name
	for i < len(runes) && (isAlphaNum(runes[i]) || runes[i] == '-' || runes[i] == '_') {
		i++
	}

	return i
}

func (h *Highlighter) findOperatorEnd(runes []rune, start int) int {
	i := start
	r := runes[i]

	// Handle multi-character operators
	switch r {
	case '|':
		if i+1 < len(runes) && runes[i+1] == '|' {
			return i + 2 // ||
		}
		return i + 1 // |

	case '&':
		if i+1 < len(runes) && runes[i+1] == '&' {
			return i + 2 // &&
		}
		return i + 1 // &

	case '>':
		if i+1 < len(runes) && runes[i+1] == '>' {
			return i + 2 // >>
		}
		return i + 1 // >

	case '<':
		if i+1 < len(runes) && runes[i+1] == '<' {
			return i + 2 // <<
		}
		return i + 1 // <

	default:
		return i + 1
	}
}

func isOperator(r rune) bool {
	switch r {
	case '|', '&', ';', '>', '<':
		return true
	default:
		return false
	}
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
