package input

import (
	"unicode"
)

// Buffer manages text content and cursor position for line input.
// It provides rune-based text storage with efficient insert/delete operations
// and word boundary detection for navigation.
type Buffer struct {
	// runes stores the text content as a slice of runes
	runes []rune
	// pos is the cursor position (index in runes)
	pos int
}

// NewBuffer creates a new empty buffer.
func NewBuffer() *Buffer {
	return &Buffer{
		runes: []rune{},
		pos:   0,
	}
}

// NewBufferWithText creates a buffer with initial text.
func NewBufferWithText(text string) *Buffer {
	runes := []rune(text)
	return &Buffer{
		runes: runes,
		pos:   len(runes),
	}
}

// Text returns the current text content as a string.
func (b *Buffer) Text() string {
	return string(b.runes)
}

// Runes returns the current text content as a slice of runes.
func (b *Buffer) Runes() []rune {
	result := make([]rune, len(b.runes))
	copy(result, b.runes)
	return result
}

// Len returns the length of the text in runes.
func (b *Buffer) Len() int {
	return len(b.runes)
}

// Pos returns the current cursor position.
func (b *Buffer) Pos() int {
	return b.pos
}

// SetText replaces the entire buffer content with new text.
// The cursor is moved to the end of the new text.
func (b *Buffer) SetText(text string) {
	b.runes = []rune(text)
	b.pos = len(b.runes)
}

// SetRunes replaces the entire buffer content with new runes.
// The cursor is moved to the end of the new content.
func (b *Buffer) SetRunes(runes []rune) {
	b.runes = make([]rune, len(runes))
	copy(b.runes, runes)
	b.pos = len(b.runes)
}

// Clear removes all text from the buffer and resets the cursor.
func (b *Buffer) Clear() {
	b.runes = []rune{}
	b.pos = 0
}

// SetPos sets the cursor position. If pos is out of bounds,
// it will be clamped to valid range [0, len(runes)].
func (b *Buffer) SetPos(pos int) {
	b.pos = clamp(pos, 0, len(b.runes))
}

// CursorStart moves the cursor to the start of the buffer.
func (b *Buffer) CursorStart() {
	b.pos = 0
}

// CursorEnd moves the cursor to the end of the buffer.
func (b *Buffer) CursorEnd() {
	b.pos = len(b.runes)
}

// Insert inserts text at the current cursor position.
// The cursor is moved to the end of the inserted text.
func (b *Buffer) Insert(text string) {
	b.InsertRunes([]rune(text))
}

// InsertRunes inserts runes at the current cursor position.
// The cursor is moved to the end of the inserted runes.
func (b *Buffer) InsertRunes(runes []rune) {
	if len(runes) == 0 {
		return
	}

	result := make([]rune, len(b.runes)+len(runes))
	copy(result, b.runes[:b.pos])
	copy(result[b.pos:], runes)
	copy(result[b.pos+len(runes):], b.runes[b.pos:])

	b.runes = result
	b.pos += len(runes)
}

// DeleteCharBackward deletes the character before the cursor.
// Returns true if a character was deleted.
func (b *Buffer) DeleteCharBackward() bool {
	if b.pos == 0 || len(b.runes) == 0 {
		return false
	}

	result := make([]rune, len(b.runes)-1)
	copy(result, b.runes[:b.pos-1])
	copy(result[b.pos-1:], b.runes[b.pos:])

	b.runes = result
	b.pos--
	return true
}

// DeleteCharForward deletes the character at the cursor.
// Returns true if a character was deleted.
func (b *Buffer) DeleteCharForward() bool {
	if b.pos >= len(b.runes) || len(b.runes) == 0 {
		return false
	}

	result := make([]rune, len(b.runes)-1)
	copy(result, b.runes[:b.pos])
	copy(result[b.pos:], b.runes[b.pos+1:])

	b.runes = result
	return true
}

// DeleteBeforeCursor deletes all text before the cursor.
func (b *Buffer) DeleteBeforeCursor() {
	if b.pos == 0 {
		return
	}

	result := make([]rune, len(b.runes)-b.pos)
	copy(result, b.runes[b.pos:])

	b.runes = result
	b.pos = 0
}

// DeleteAfterCursor deletes all text after the cursor.
func (b *Buffer) DeleteAfterCursor() {
	if b.pos >= len(b.runes) {
		return
	}

	b.runes = b.runes[:b.pos]
}

// DeleteWordBackward deletes the word to the left of the cursor.
func (b *Buffer) DeleteWordBackward() {
	if b.pos == 0 || len(b.runes) == 0 {
		return
	}

	oldPos := b.pos
	b.WordBackward()

	// Delete from new position to old position
	result := make([]rune, len(b.runes)-(oldPos-b.pos))
	copy(result, b.runes[:b.pos])
	copy(result[b.pos:], b.runes[oldPos:])

	b.runes = result
}

// DeleteWordForward deletes the word to the right of the cursor.
func (b *Buffer) DeleteWordForward() {
	if b.pos >= len(b.runes) || len(b.runes) == 0 {
		return
	}

	oldPos := b.pos
	b.WordForward()

	// Delete from old position to new position
	result := make([]rune, len(b.runes)-(b.pos-oldPos))
	copy(result, b.runes[:oldPos])
	copy(result[oldPos:], b.runes[b.pos:])

	b.runes = result
	b.pos = oldPos
}

// WordBackward moves the cursor one word to the left.
// A word is a sequence of non-whitespace characters.
func (b *Buffer) WordBackward() {
	if b.pos == 0 || len(b.runes) == 0 {
		return
	}

	// Move back past any whitespace
	i := b.pos - 1
	for i >= 0 && unicode.IsSpace(b.runes[i]) {
		i--
	}

	// Move back past the word
	for i >= 0 && !unicode.IsSpace(b.runes[i]) {
		i--
	}

	b.pos = i + 1
}

// WordForward moves the cursor one word to the right.
// A word is a sequence of non-whitespace characters.
func (b *Buffer) WordForward() {
	if b.pos >= len(b.runes) || len(b.runes) == 0 {
		return
	}

	// Move forward past any whitespace
	i := b.pos
	for i < len(b.runes) && unicode.IsSpace(b.runes[i]) {
		i++
	}

	// Move forward past the word
	for i < len(b.runes) && !unicode.IsSpace(b.runes[i]) {
		i++
	}

	b.pos = i
}

// TextBeforeCursor returns the text before the cursor.
func (b *Buffer) TextBeforeCursor() string {
	return string(b.runes[:b.pos])
}

// TextAfterCursor returns the text after the cursor.
func (b *Buffer) TextAfterCursor() string {
	return string(b.runes[b.pos:])
}

// RuneAt returns the rune at the given position.
// Returns 0 if position is out of bounds.
func (b *Buffer) RuneAt(pos int) rune {
	if pos < 0 || pos >= len(b.runes) {
		return 0
	}
	return b.runes[pos]
}

// RuneAtCursor returns the rune at the cursor position.
// Returns 0 if cursor is at the end or buffer is empty.
func (b *Buffer) RuneAtCursor() rune {
	return b.RuneAt(b.pos)
}

// clamp returns value clamped to the range [low, high].
func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
