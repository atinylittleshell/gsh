// Package termfeatures provides safe, cross-platform terminal feature detection
// and operations for window title management.
package termfeatures

import (
	"fmt"
	"os"
	"strings"

	"github.com/muesli/termenv"
)

// FeatureSupport indicates the level of support for a terminal feature.
type FeatureSupport int

const (
	// FeatureUnsupported indicates the feature is not supported.
	FeatureUnsupported FeatureSupport = iota
	// FeatureNative indicates the feature is supported via escape sequences.
	FeatureNative
	// FeatureUnknown indicates the feature may work but is not confirmed.
	FeatureUnknown
)

// String returns a string representation of FeatureSupport.
func (f FeatureSupport) String() string {
	switch f {
	case FeatureUnsupported:
		return "unsupported"
	case FeatureNative:
		return "native"
	case FeatureUnknown:
		return "unknown"
	default:
		return "invalid"
	}
}

// Capabilities describes the terminal's feature support.
type Capabilities struct {
	// Terminal identification
	Term        string // TERM environment variable
	TermProgram string // TERM_PROGRAM environment variable

	// Feature support
	WindowTitle FeatureSupport

	// Environment context
	IsSSH    bool
	IsTmux   bool
	IsScreen bool
	IsDumb   bool
}

// Terminal provides safe terminal operations with automatic capability detection.
type Terminal struct {
	output       *termenv.Output
	capabilities Capabilities
}

// TitleResult contains information about a window title operation.
type TitleResult struct {
	Success bool
	Method  string // "osc2", "none"
	Error   error
}

// New creates a new Terminal with automatic capability detection.
func New() *Terminal {
	output := termenv.DefaultOutput()
	return NewWithOutput(output)
}

// NewWithOutput creates a new Terminal with the specified termenv output.
func NewWithOutput(output *termenv.Output) *Terminal {
	return &Terminal{
		output:       output,
		capabilities: detectCapabilities(),
	}
}

// Capabilities returns the detected terminal capabilities.
func (t *Terminal) Capabilities() Capabilities {
	return t.capabilities
}

// SupportsWindowTitle returns true if the terminal supports window title setting.
func (t *Terminal) SupportsWindowTitle() bool {
	return t.capabilities.WindowTitle == FeatureNative ||
		t.capabilities.WindowTitle == FeatureUnknown
}

// detectCapabilities detects terminal capabilities based on environment variables.
func detectCapabilities() Capabilities {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	caps := Capabilities{
		Term:        term,
		TermProgram: termProgram,
		IsSSH:       os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "",
		IsTmux:      os.Getenv("TMUX") != "",
		IsScreen:    os.Getenv("STY") != "",
		IsDumb:      term == "dumb" || term == "",
	}

	// Detect window title support
	caps.WindowTitle = detectWindowTitleSupport(caps)

	return caps
}

// detectWindowTitleSupport determines if the terminal supports window title setting.
func detectWindowTitleSupport(caps Capabilities) FeatureSupport {
	if caps.IsDumb {
		return FeatureUnsupported
	}

	// Known terminals with window title support
	termProgram := strings.ToLower(caps.TermProgram)
	knownSupport := map[string]bool{
		"iterm.app":        true,
		"apple_terminal":   true,
		"wezterm":          true,
		"kitty":            true,
		"alacritty":        true,
		"hyper":            true,
		"vscode":           true,
		"windows terminal": true,
		"gnome-terminal":   true,
		"konsole":          true,
		"xfce4-terminal":   true,
		"tilix":            true,
		"terminator":       true,
		"terminology":      true,
		"rxvt":             true,
		"urxvt":            true,
		"st":               true,
		"foot":             true,
	}

	if knownSupport[termProgram] {
		return FeatureNative
	}

	// Check TERM for xterm-compatible terminals
	term := strings.ToLower(caps.Term)
	if strings.HasPrefix(term, "xterm") ||
		strings.HasPrefix(term, "screen") ||
		strings.HasPrefix(term, "tmux") ||
		strings.HasPrefix(term, "rxvt") ||
		strings.HasPrefix(term, "linux") ||
		strings.Contains(term, "256color") ||
		strings.Contains(term, "color") {
		return FeatureNative
	}

	// tmux and screen pass through title commands
	if caps.IsTmux || caps.IsScreen {
		return FeatureNative
	}

	// Default to unknown for unrecognized terminals (may still work)
	if caps.Term != "" {
		return FeatureUnknown
	}

	return FeatureUnsupported
}

// SetWindowTitle sets the terminal window title.
// Safe to call even if unsupported (no-op).
func (t *Terminal) SetWindowTitle(title string) TitleResult {
	if t.capabilities.IsDumb {
		return TitleResult{
			Success: false,
			Method:  "none",
			Error:   ErrDumbTerminal,
		}
	}

	if t.capabilities.WindowTitle == FeatureUnsupported {
		// Silent no-op for unsupported terminals
		return TitleResult{Success: false, Method: "none", Error: nil}
	}

	// Sanitize title: remove control characters and limit length
	title = sanitizeTitle(title)

	// Build escape sequence based on terminal type
	var seq string
	if t.capabilities.IsTmux {
		// tmux passthrough: \ePtmux;\e\e]2;title\a\e\\
		seq = fmt.Sprintf("\x1bPtmux;\x1b\x1b]2;%s\x07\x1b\\", title)
	} else if t.capabilities.IsScreen {
		// screen: standard OSC 2
		seq = fmt.Sprintf("\x1b]2;%s\x07", title)
	} else {
		// Standard OSC 2
		seq = fmt.Sprintf("\x1b]2;%s\x07", title)
	}

	_, err := t.output.WriteString(seq)
	return TitleResult{
		Success: err == nil,
		Method:  "osc2",
		Error:   err,
	}
}

// SetWindowTitlef sets the title with fmt.Sprintf formatting.
func (t *Terminal) SetWindowTitlef(format string, args ...any) TitleResult {
	return t.SetWindowTitle(fmt.Sprintf(format, args...))
}

// ResetWindowTitle restores the default window title.
// Implementation varies by terminal; may be no-op on some terminals.
func (t *Terminal) ResetWindowTitle() TitleResult {
	// Most terminals will reset to their default when given an empty title
	// or we can try to restore using the terminal's default behavior
	return t.SetWindowTitle("")
}

// sanitizeTitle removes control characters and limits length.
func sanitizeTitle(title string) string {
	// Remove BEL, ESC, and other control characters
	var sanitized strings.Builder
	sanitized.Grow(len(title))

	for _, r := range title {
		// Allow printable characters and common whitespace
		if r >= 32 && r != 127 {
			sanitized.WriteRune(r)
		} else if r == '\t' {
			sanitized.WriteRune(' ') // Replace tabs with spaces
		}
	}

	result := sanitized.String()

	// Limit length to 255 runes (not bytes) to avoid splitting multi-byte UTF-8 characters
	runes := []rune(result)
	if len(runes) > 255 {
		result = string(runes[:255])
	}

	return result
}
