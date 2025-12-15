package gline

import (
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// emojiWidthCache stores the detected widths of emoji characters.
// Widths are detected once using terminal cursor position probing and cached.
var (
	emojiWidthCache   = make(map[rune]int)
	emojiWidthCacheMu sync.RWMutex
)

// GetLightningBoltWidth returns the width of the lightning bolt character.
// Uses the generic emoji width cache with terminal probing.
func GetLightningBoltWidth() int {
	return GetRuneWidth('⚡')
}

// GetRobotWidth returns the width of the robot emoji character.
// Uses the generic emoji width cache with terminal probing.
func GetRobotWidth() int {
	return GetRuneWidth('🤖')
}

// GetRuneWidth returns the display width of a rune, using terminal probing for emoji.
// For emoji characters, the width is detected once and cached. For other characters,
// it returns 1 for ASCII or 2 for wide characters.
func GetRuneWidth(r rune) int {
	// Fast path for ASCII
	if r < 128 {
		return 1
	}

	// Check if it's an emoji (simplified check for common emoji ranges)
	isEmoji := (r >= 0x1F300 && r <= 0x1F9FF) || // Misc Symbols and Pictographs, Emoticons, etc.
		(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
		(r >= 0x1F000 && r <= 0x1F02F) || // Mahjong Tiles
		(r >= 0x1F0A0 && r <= 0x1F0FF) // Playing Cards

	if !isEmoji {
		// For non-emoji wide characters, return 2
		if r >= 0x1100 {
			return 2
		}
		return 1
	}

	// Check cache first
	emojiWidthCacheMu.RLock()
	if width, ok := emojiWidthCache[r]; ok {
		emojiWidthCacheMu.RUnlock()
		return width
	}
	emojiWidthCacheMu.RUnlock()

	// Probe and cache
	emojiWidthCacheMu.Lock()
	defer emojiWidthCacheMu.Unlock()

	// Double-check after acquiring write lock
	if width, ok := emojiWidthCache[r]; ok {
		return width
	}

	width := probeTerminalCharWidth(r)
	emojiWidthCache[r] = width
	return width
}

// probeTerminalCharWidth uses terminal cursor position reporting to detect
// the actual rendered width of a character.
func probeTerminalCharWidth(char rune) int {
	// Default fallback - most western terminals render ambiguous width chars as 1
	const defaultWidth = 1

	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())

	// Check if both stdin and stdout are terminals.
	// We need stdin to read the DSR response, and stdout to write escape sequences.
	// If either is not a terminal, we risk hanging or writing garbage to a file/pipe.
	if !term.IsTerminal(stdinFd) || !term.IsTerminal(stdoutFd) {
		return defaultWidth
	}

	// Save terminal state and set raw mode
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return defaultWidth
	}
	defer func() {
		_ = term.Restore(stdinFd, oldState)
	}()

	// Save cursor position, move to column 1, print char, query position
	// DSR (Device Status Report): ESC[6n returns ESC[row;colR
	//
	// Sequence:
	// 1. ESC7 - save cursor position
	// 2. ESC[1G - move to column 1
	// 3. print the character
	// 4. ESC[6n - query cursor position
	// 5. ESC8 - restore cursor position
	// 6. ESC[K - clear to end of line (clean up the printed char)
	_, err = os.Stdout.WriteString("\x1b7\x1b[1G" + string(char) + "\x1b[6n")
	if err != nil {
		return defaultWidth
	}
	_ = os.Stdout.Sync()

	// Read the DSR response: ESC[row;colR
	response := make([]byte, 32)
	_ = os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	n, err := os.Stdin.Read(response)

	// Restore cursor and clear the character we wrote
	_, _ = os.Stdout.WriteString("\x1b8\x1b[K")
	_ = os.Stdout.Sync()

	if err != nil || n < 6 {
		return defaultWidth
	}

	// Parse the response: ESC[row;colR
	// We only care about col
	col := parseDSRResponse(response[:n])
	if col <= 0 {
		return defaultWidth
	}

	// Width is col - 1 (since we started at column 1)
	width := col - 1
	if width < 1 {
		return defaultWidth
	}
	if width > 2 {
		// Sanity check - most chars are 1 or 2 wide
		return defaultWidth
	}

	return width
}

// parseDSRResponse parses an ESC[row;colR response and returns the column.
func parseDSRResponse(response []byte) int {
	// Find ESC[
	start := -1
	for i := 0; i < len(response)-1; i++ {
		if response[i] == '\x1b' && response[i+1] == '[' {
			start = i + 2
			break
		}
	}
	if start < 0 {
		return -1
	}

	// Parse row;col
	row := 0
	col := 0
	parsingCol := false

	for i := start; i < len(response); i++ {
		b := response[i]
		if b == ';' {
			parsingCol = true
			continue
		}
		if b == 'R' {
			break
		}
		if b >= '0' && b <= '9' {
			if parsingCol {
				col = col*10 + int(b-'0')
			} else {
				row = row*10 + int(b-'0')
			}
		}
	}

	_ = row // We only need col
	return col
}
