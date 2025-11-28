package agent

import (
	"github.com/stretchr/testify/assert"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestResetChat(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	runner, _ := interp.New(
		interp.StdIO(nil, nil, nil),
	)

	agent := &Agent{
		runner: runner,
		logger: logger,
		messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "Old system message"},
			{Role: "user", Content: "User message 1"},
			{Role: "assistant", Content: "Assistant message 1"},
			{Role: "user", Content: "User message 2"},
			{Role: "assistant", Content: "Assistant message 2"},
		},
	}

	// Reset the chat
	agent.ResetChat()

	// Verify that only one system message remains
	assert.Len(t, agent.messages, 1, "Expected only one message after reset")
	assert.Equal(t, "system", agent.messages[0].Role, "Expected the remaining message to be 'system'")

	// Verify that the system message contains the latest context
	assert.Contains(t, agent.messages[0].Content, "You are gsh", "Expected system message to contain the latest context")
}

func TestPruneMessages(t *testing.T) {
	tests := []struct {
		name          string
		contextWindow string
		inputMessages []openai.ChatCompletionMessage
		expectedCount int
		verifyFunc    func(*testing.T, []openai.ChatCompletionMessage)
	}{
		{
			name:          "no pruning needed for small conversation",
			contextWindow: "1000", // Large enough to keep all messages
			inputMessages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "System message"},
				{Role: "user", Content: "User message 1"},
				{Role: "assistant", Content: "Assistant message 1"},
			},
			expectedCount: 3,
			verifyFunc: func(t *testing.T, messages []openai.ChatCompletionMessage) {
				assert.Equal(t, "System message", messages[0].Content)
				assert.Equal(t, "User message 1", messages[1].Content)
				assert.Equal(t, "Assistant message 1", messages[2].Content)
			},
		},
		{
			name:          "prune with 2/3 recent and 1/3 early distribution",
			contextWindow: "50", // Small enough to force pruning
			inputMessages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "System message"},
				{Role: "user", Content: "Early message 1"},
				{Role: "assistant", Content: "Early message 2"},
				{Role: "user", Content: "Middle message 1"},
				{Role: "assistant", Content: "Middle message 2"},
				{Role: "user", Content: "Recent message 1"},
				{Role: "assistant", Content: "Recent message 2"},
				{Role: "user", Content: "Recent message 3"},
			},
			expectedCount: 4, // system + 1 early + 2 recent
			verifyFunc: func(t *testing.T, messages []openai.ChatCompletionMessage) {
				// We expect:
				// - System message (always kept)
				// - One early message (1/3 of remaining budget)
				// - Two recent messages (2/3 of remaining budget)
				assert.Equal(t, "System message", messages[0].Content)
				assert.Equal(t, "Early message 1", messages[1].Content, "Expected to keep one early message")
				assert.Equal(t, "Recent message 2", messages[2].Content, "Expected to keep second-to-last recent message")
				assert.Equal(t, "Recent message 3", messages[3].Content, "Expected to keep last recent message")
			},
		},
		{
			name:          "very small context window",
			contextWindow: "10", // Extremely small to test minimal retention
			inputMessages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "System message"},
				{Role: "user", Content: "Message 1"},
				{Role: "assistant", Content: "Message 2"},
				{Role: "user", Content: "Message 3"},
				{Role: "assistant", Content: "Message 4"},
			},
			expectedCount: 1, // system message only (context window too small)
			verifyFunc: func(t *testing.T, messages []openai.ChatCompletionMessage) {
				// With extremely small context window, we should at least keep the system message
				assert.Equal(t, "System message", messages[0].Content)
			},
		},
		{
			name:          "single message besides system",
			contextWindow: "20",
			inputMessages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "System message"},
				{Role: "user", Content: "Single message"},
			},
			expectedCount: 2,
			verifyFunc: func(t *testing.T, messages []openai.ChatCompletionMessage) {
				assert.Equal(t, "System message", messages[0].Content)
				assert.Equal(t, "Single message", messages[1].Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			defer func() { _ = logger.Sync() }()

			runner, _ := interp.New(
				interp.StdIO(nil, nil, nil),
			)

			runner.Vars = map[string]expand.Variable{
				"GSH_AGENT_CONTEXT_WINDOW_TOKENS": {Kind: expand.String, Str: tt.contextWindow},
			}

			agent := &Agent{
				runner:   runner,
				logger:   logger,
				messages: tt.inputMessages,
			}

			agent.pruneMessages()

			assert.NotEmpty(t, agent.messages, "Expected some messages to be retained")
			assert.Equal(t, "system", agent.messages[0].Role, "Expected the first message to be 'system'")
			assert.Len(t, agent.messages, tt.expectedCount, "Unexpected number of messages after pruning")

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, agent.messages)
			}
		})
	}

}
