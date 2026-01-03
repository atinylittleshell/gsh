package acp

import "context"

// ACPSession is the interface for ACP session operations.
// This interface allows for mocking in tests.
type ACPSession interface {
	SendPrompt(ctx context.Context, text string, onUpdate func(*SessionUpdateParams)) (*SessionPromptResult, error)
	GetMessages() []Message
	GetLastMessage() *Message
	SessionID() string
	Close() error
}

// Ensure Session implements ACPSession
var _ ACPSession = (*Session)(nil)
