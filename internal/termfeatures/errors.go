package termfeatures

import "errors"

// ErrDumbTerminal indicates the terminal doesn't support escape sequences.
var ErrDumbTerminal = errors.New("dumb terminal: no escape sequence support")
