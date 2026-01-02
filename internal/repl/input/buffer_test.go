package input

import (
	"testing"
)

func TestNewBuffer(t *testing.T) {
	b := NewBuffer()
	if b.Text() != "" {
		t.Errorf("expected empty text, got %q", b.Text())
	}
	if b.Pos() != 0 {
		t.Errorf("expected pos 0, got %d", b.Pos())
	}
	if b.Len() != 0 {
		t.Errorf("expected len 0, got %d", b.Len())
	}
}

func TestNewBufferWithText(t *testing.T) {
	text := "hello world"
	b := NewBufferWithText(text)
	if b.Text() != text {
		t.Errorf("expected text %q, got %q", text, b.Text())
	}
	if b.Pos() != len([]rune(text)) {
		t.Errorf("expected pos %d, got %d", len([]rune(text)), b.Pos())
	}
	if b.Len() != len([]rune(text)) {
		t.Errorf("expected len %d, got %d", len([]rune(text)), b.Len())
	}
}

func TestBufferSetText(t *testing.T) {
	b := NewBuffer()
	text := "test"
	b.SetText(text)
	if b.Text() != text {
		t.Errorf("expected text %q, got %q", text, b.Text())
	}
	if b.Pos() != len([]rune(text)) {
		t.Errorf("expected cursor at end, got %d", b.Pos())
	}
}

func TestBufferClear(t *testing.T) {
	b := NewBufferWithText("hello")
	b.Clear()
	if b.Text() != "" {
		t.Errorf("expected empty text, got %q", b.Text())
	}
	if b.Pos() != 0 {
		t.Errorf("expected pos 0, got %d", b.Pos())
	}
}

func TestBufferSetPos(t *testing.T) {
	b := NewBufferWithText("hello")

	// Valid position
	b.SetPos(2)
	if b.Pos() != 2 {
		t.Errorf("expected pos 2, got %d", b.Pos())
	}

	// Negative position (clamped to 0)
	b.SetPos(-5)
	if b.Pos() != 0 {
		t.Errorf("expected pos 0, got %d", b.Pos())
	}

	// Position beyond end (clamped to len)
	b.SetPos(100)
	if b.Pos() != b.Len() {
		t.Errorf("expected pos %d, got %d", b.Len(), b.Pos())
	}
}

func TestBufferCursorStartEnd(t *testing.T) {
	b := NewBufferWithText("hello")

	b.CursorStart()
	if b.Pos() != 0 {
		t.Errorf("expected pos 0, got %d", b.Pos())
	}

	b.CursorEnd()
	if b.Pos() != 5 {
		t.Errorf("expected pos 5, got %d", b.Pos())
	}
}

func TestBufferInsert(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		insert   string
		expected string
		finalPos int
	}{
		{"insert at start", "world", 0, "hello ", "hello world", 6},
		{"insert at end", "hello", 5, " world", "hello world", 11},
		{"insert in middle", "helo", 2, "l", "hello", 3},
		{"insert empty", "test", 2, "", "test", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			b.Insert(tt.insert)
			if b.Text() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, b.Text())
			}
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferInsertRunes(t *testing.T) {
	b := NewBufferWithText("hello")
	b.SetPos(5)
	b.InsertRunes([]rune(" 世界"))
	if b.Text() != "hello 世界" {
		t.Errorf("expected %q, got %q", "hello 世界", b.Text())
	}
	if b.Pos() != 8 {
		t.Errorf("expected pos 8, got %d", b.Pos())
	}
}

func TestBufferDeleteCharBackward(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		expected string
		finalPos int
		deleted  bool
	}{
		{"delete middle char", "hello", 3, "helo", 2, true},
		{"delete first char", "hello", 1, "ello", 0, true},
		{"delete at start (no-op)", "hello", 0, "hello", 0, false},
		{"delete from empty", "", 0, "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			deleted := b.DeleteCharBackward()
			if deleted != tt.deleted {
				t.Errorf("expected deleted=%v, got %v", tt.deleted, deleted)
			}
			if b.Text() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, b.Text())
			}
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferDeleteCharForward(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		expected string
		finalPos int
		deleted  bool
	}{
		{"delete middle char", "hello", 2, "helo", 2, true},
		{"delete last char", "hello", 4, "hell", 4, true},
		{"delete at end (no-op)", "hello", 5, "hello", 5, false},
		{"delete from empty", "", 0, "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			deleted := b.DeleteCharForward()
			if deleted != tt.deleted {
				t.Errorf("expected deleted=%v, got %v", tt.deleted, deleted)
			}
			if b.Text() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, b.Text())
			}
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferDeleteBeforeCursor(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		expected string
		finalPos int
	}{
		{"delete before middle", "hello world", 6, "world", 0},
		{"delete before start (no-op)", "hello", 0, "hello", 0},
		{"delete all", "hello", 5, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			b.DeleteBeforeCursor()
			if b.Text() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, b.Text())
			}
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferDeleteAfterCursor(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		expected string
		finalPos int
	}{
		{"delete after middle", "hello world", 5, "hello", 5},
		{"delete after end (no-op)", "hello", 5, "hello", 5},
		{"delete all", "hello", 0, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			b.DeleteAfterCursor()
			if b.Text() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, b.Text())
			}
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferDeleteWordBackward(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		expected string
		finalPos int
	}{
		{"delete word from end", "hello world", 11, "hello ", 6},
		{"delete word from middle", "hello world", 5, " world", 0},
		{"delete with spaces", "hello   world", 13, "hello   ", 8},
		{"delete at start (no-op)", "hello", 0, "hello", 0},
		{"delete from empty (no-op)", "", 0, "", 0},
		{"delete single word", "test", 4, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			b.DeleteWordBackward()
			if b.Text() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, b.Text())
			}
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferDeleteWordForward(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		expected string
		finalPos int
	}{
		{"delete word from start", "hello world", 0, " world", 0},
		{"delete word from middle", "hello world", 6, "hello ", 6},
		{"delete with spaces", "hello   world", 5, "hello", 5},
		{"delete at end (no-op)", "hello", 5, "hello", 5},
		{"delete from empty (no-op)", "", 0, "", 0},
		{"delete single word", "test", 0, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			b.DeleteWordForward()
			if b.Text() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, b.Text())
			}
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferWordBackward(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		finalPos int
	}{
		{"from end to word start", "hello world", 11, 6},
		{"from word middle to start", "hello world", 8, 6},
		{"skip spaces", "hello   world", 13, 8},
		{"at start (no-op)", "hello", 0, 0},
		{"empty buffer", "", 0, 0},
		{"multiple words", "one two three", 13, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			b.WordBackward()
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferWordForward(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		pos      int
		finalPos int
	}{
		{"from start to word end", "hello world", 0, 5},
		{"from word middle to end", "hello world", 2, 5},
		{"skip spaces", "hello   world", 5, 13},
		{"at end (no-op)", "hello", 5, 5},
		{"empty buffer", "", 0, 0},
		{"multiple words", "one two three", 0, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferWithText(tt.initial)
			b.SetPos(tt.pos)
			b.WordForward()
			if b.Pos() != tt.finalPos {
				t.Errorf("expected pos %d, got %d", tt.finalPos, b.Pos())
			}
		})
	}
}

func TestBufferTextBeforeAfterCursor(t *testing.T) {
	b := NewBufferWithText("hello world")
	b.SetPos(6)

	before := b.TextBeforeCursor()
	if before != "hello " {
		t.Errorf("expected %q, got %q", "hello ", before)
	}

	after := b.TextAfterCursor()
	if after != "world" {
		t.Errorf("expected %q, got %q", "world", after)
	}
}

func TestBufferRuneAt(t *testing.T) {
	b := NewBufferWithText("hello")

	if r := b.RuneAt(0); r != 'h' {
		t.Errorf("expected 'h', got %c", r)
	}

	if r := b.RuneAt(4); r != 'o' {
		t.Errorf("expected 'o', got %c", r)
	}

	// Out of bounds
	if r := b.RuneAt(-1); r != 0 {
		t.Errorf("expected 0, got %c", r)
	}

	if r := b.RuneAt(10); r != 0 {
		t.Errorf("expected 0, got %c", r)
	}
}

func TestBufferRuneAtCursor(t *testing.T) {
	b := NewBufferWithText("hello")
	b.SetPos(1)

	if r := b.RuneAtCursor(); r != 'e' {
		t.Errorf("expected 'e', got %c", r)
	}

	// At end
	b.CursorEnd()
	if r := b.RuneAtCursor(); r != 0 {
		t.Errorf("expected 0, got %c", r)
	}
}

func TestBufferRunes(t *testing.T) {
	text := "hello"
	b := NewBufferWithText(text)

	runes := b.Runes()
	if string(runes) != text {
		t.Errorf("expected %q, got %q", text, string(runes))
	}

	// Verify it's a copy
	runes[0] = 'x'
	if b.Text() != text {
		t.Errorf("modifying returned runes should not affect buffer")
	}
}

func TestBufferSetRunes(t *testing.T) {
	b := NewBuffer()
	runes := []rune{'h', 'e', 'l', 'l', 'o'}
	b.SetRunes(runes)

	if b.Text() != "hello" {
		t.Errorf("expected %q, got %q", "hello", b.Text())
	}

	// Verify it's a copy
	runes[0] = 'x'
	if b.Text() != "hello" {
		t.Errorf("modifying input runes should not affect buffer")
	}
}

func TestBufferUnicodeSupport(t *testing.T) {
	// Test with multi-byte Unicode characters
	b := NewBufferWithText("Hello 世界")

	if b.Len() != 8 {
		t.Errorf("expected len 8, got %d", b.Len())
	}

	b.SetPos(6)
	b.Insert("美丽的")

	expected := "Hello 美丽的世界"
	if b.Text() != expected {
		t.Errorf("expected %q, got %q", expected, b.Text())
	}
}

func TestBufferComplexEditing(t *testing.T) {
	// Simulate a complex editing session
	b := NewBuffer()

	b.Insert("git commit")
	if b.Text() != "git commit" {
		t.Errorf("step 1 failed: %q", b.Text())
	}

	b.Insert(" -m")
	if b.Text() != "git commit -m" {
		t.Errorf("step 2 failed: %q", b.Text())
	}

	// Move back one word (to start of "-m") and delete the word before it (delete "commit ")
	b.WordBackward()
	b.DeleteWordBackward()
	if b.Text() != "git -m" {
		t.Errorf("step 3 failed: %q", b.Text())
	}

	b.SetPos(4) // After "git "
	b.Insert("status")
	if b.Text() != "git status-m" {
		t.Errorf("step 4 failed: %q", b.Text())
	}

	b.CursorEnd()
	b.DeleteWordBackward()
	if b.Text() != "git " {
		t.Errorf("step 5 failed: %q", b.Text())
	}

	b.Insert("log")
	if b.Text() != "git log" {
		t.Errorf("step 6 failed: %q", b.Text())
	}
}
