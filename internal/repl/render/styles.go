// Package render provides agent output rendering functionality for the REPL.
package render

import (
	"github.com/charmbracelet/lipgloss"
)

// ANSI color codes as defined in the spec
const (
	ColorCyan   = lipgloss.Color("12") // Agent header/footer
	ColorYellow = lipgloss.Color("11") // Tool pending
	ColorGreen  = lipgloss.Color("10") // Success indicator
	ColorRed    = lipgloss.Color("9")  // Error indicator
	ColorGray   = lipgloss.Color("8")  // Dim/secondary (timing, meta info)
)

// Symbols as defined in the spec
const (
	SymbolExec          = "▶" // Exec tool (shell command) start
	SymbolToolPending   = "○" // Non-exec tool pending/executing
	SymbolToolComplete  = "●" // Non-exec tool complete
	SymbolSuccess       = "✓" // Success
	SymbolError         = "✗" // Error
	SymbolSystemMessage = "→" // System message
)

// Style definitions using Lip Gloss
var (
	// HeaderStyle is used for agent header/footer lines
	HeaderStyle = lipgloss.NewStyle().Foreground(ColorCyan)

	// ExecStartStyle is used for the exec tool start symbol
	ExecStartStyle = lipgloss.NewStyle().Foreground(ColorYellow)

	// ToolPendingStyle is used for pending/executing tool status
	ToolPendingStyle = lipgloss.NewStyle().Foreground(ColorYellow)

	// SuccessStyle is used for success indicators
	SuccessStyle = lipgloss.NewStyle().Foreground(ColorGreen)

	// ErrorStyle is used for error indicators
	ErrorStyle = lipgloss.NewStyle().Foreground(ColorRed)

	// DimStyle is used for secondary information like timing
	DimStyle = lipgloss.NewStyle().Foreground(ColorGray)

	// SystemMessageStyle is used for system/status messages
	SystemMessageStyle = lipgloss.NewStyle().Foreground(ColorGray)
)

// StyledSymbol returns a symbol with appropriate styling applied
func StyledSymbol(symbol string, success bool) string {
	switch symbol {
	case SymbolExec:
		return ExecStartStyle.Render(symbol)
	case SymbolToolPending:
		return ToolPendingStyle.Render(symbol)
	case SymbolToolComplete:
		if success {
			return SuccessStyle.Render(symbol)
		}
		return ErrorStyle.Render(symbol)
	case SymbolSuccess:
		return SuccessStyle.Render(symbol)
	case SymbolError:
		return ErrorStyle.Render(symbol)
	case SymbolSystemMessage:
		return SystemMessageStyle.Render(symbol)
	default:
		return symbol
	}
}
