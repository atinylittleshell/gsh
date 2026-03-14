package repl

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// emitOSC7 writes an OSC 7 escape sequence to inform the terminal of the
// current working directory. This allows terminal emulators (Terminal.app,
// iTerm2, etc.) to open new tabs/windows in the same directory.
// Format: ESC ] 7 ; file://HOSTNAME/url-encoded-path ESC \
func emitOSC7(w io.Writer, hostname, dir string) {
	// Encode each path segment individually to preserve '/' separators.
	// url.PathEscape encodes '/' to '%2F' which breaks the file URL.
	segments := strings.Split(dir, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	encodedPath := strings.Join(segments, "/")
	fmt.Fprintf(w, "\033]7;file://%s%s\033\\", hostname, encodedPath)
}
