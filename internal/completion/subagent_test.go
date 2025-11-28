package completion

import (
	"testing"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"github.com/stretchr/testify/assert"
	"mvdan.cc/sh/v3/interp"
)

// MockSubagentProvider implements SubagentProvider for testing
type MockSubagentProvider struct {
	subagents map[string]*SubagentInfo
}

func NewMockSubagentProvider() *MockSubagentProvider {
	return &MockSubagentProvider{
		subagents: make(map[string]*SubagentInfo),
	}
}

func (m *MockSubagentProvider) AddSubagent(info *SubagentInfo) {
	m.subagents[info.ID] = info
}

func (m *MockSubagentProvider) GetAllSubagents() map[string]*SubagentInfo {
	return m.subagents
}

func (m *MockSubagentProvider) GetSubagent(id string) (*SubagentInfo, bool) {
	subagent, exists := m.subagents[id]
	return subagent, exists
}

func TestSubagentCompletions(t *testing.T) {
	runner, _ := interp.New()
	manager := NewCompletionManager()
	provider := NewShellCompletionProvider(manager, runner)

	// Create mock subagent provider
	mockProvider := NewMockSubagentProvider()
	mockProvider.AddSubagent(&SubagentInfo{
		ID:           "code-reviewer",
		Name:         "Code Reviewer",
		Description:  "Review code for bugs and best practices",
		AllowedTools: []string{"view_file", "bash"},
		Model:        "inherit",
	})
	mockProvider.AddSubagent(&SubagentInfo{
		ID:           "test-writer",
		Name:         "Test Writer",
		Description:  "Write comprehensive tests",
		AllowedTools: []string{"create_file", "edit_file"},
		Model:        "gpt-4",
	})
	mockProvider.AddSubagent(&SubagentInfo{
		ID:           "docs-helper",
		Name:         "Documentation Helper",
		Description:  "Create and maintain documentation",
		AllowedTools: []string{"create_file", "edit_file", "view_file"},
		FileRegex:    "\\.(md|txt)$",
	})

	provider.SetSubagentProvider(mockProvider)

	testCases := []struct {
		name     string
		line     string
		pos      int
		expected []shellinput.CompletionCandidate
	}{
		{
			name: "complete all subagents with @",
			line: "@",
			pos:  1,
			expected: []shellinput.CompletionCandidate{
				{Value: "@code-reviewer"},
				{Value: "@docs-helper"},
				{Value: "@test-writer"},
			},
		},
		{
			name: "complete subagents starting with 'c'",
			line: "@c",
			pos:  2,
			expected: []shellinput.CompletionCandidate{
				{Value: "@code-reviewer"},
			},
		},
		{
			name: "complete subagents starting with 't'",
			line: "@t",
			pos:  2,
			expected: []shellinput.CompletionCandidate{
				{Value: "@test-writer"},
			},
		},
		{
			name: "complete subagents starting with 'd'",
			line: "@d",
			pos:  2,
			expected: []shellinput.CompletionCandidate{
				{Value: "@docs-helper"},
			},
		},
		{
			name:     "no completions for non-matching prefix",
			line:     "@xyz",
			pos:      4,
			expected: []shellinput.CompletionCandidate{},
		},
		{
			name: "subagent completion in middle of line",
			line: "some command @c and more text",
			pos:  14,
			expected: []shellinput.CompletionCandidate{
				{Value: "some command @code-reviewer and more text"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := provider.GetCompletions(tc.line, tc.pos)
			// Only compare values as descriptions might vary
			assert.Equal(t, len(tc.expected), len(result))
			for i := range result {
				assert.Equal(t, tc.expected[i].Value, result[i].Value)
			}
		})
	}
}

func TestSubagentHelp(t *testing.T) {
	runner, _ := interp.New()
	manager := NewCompletionManager()
	provider := NewShellCompletionProvider(manager, runner)

	// Create mock subagent provider
	mockProvider := NewMockSubagentProvider()
	mockProvider.AddSubagent(&SubagentInfo{
		ID:           "code-reviewer",
		Name:         "Code Reviewer",
		Description:  "Review code for bugs and best practices",
		AllowedTools: []string{"view_file", "bash"},
		Model:        "inherit",
	})
	mockProvider.AddSubagent(&SubagentInfo{
		ID:           "test-writer",
		Name:         "Test Writer",
		Description:  "Write comprehensive tests",
		AllowedTools: []string{"create_file", "edit_file"},
		Model:        "gpt-4",
	})

	provider.SetSubagentProvider(mockProvider)

	testCases := []struct {
		name     string
		line     string
		pos      int
		expected string
	}{
		{
			name:     "help for @ empty shows all subagents",
			line:     "@",
			pos:      1,
			expected: "**Subagents** - Specialized AI assistants with specific roles\n\nAvailable subagents:\n• **@code-reviewer** - Review code for bugs and best practices\n• **@test-writer** - Write comprehensive tests",
		},
		{
			name:     "help for specific subagent",
			line:     "@code-reviewer",
			pos:      14,
			expected: "**@code-reviewer** - Code Reviewer\n\nReview code for bugs and best practices\n**Tools:** [view_file bash]",
		},
		{
			name:     "help for subagent with model override",
			line:     "@test-writer",
			pos:      12,
			expected: "**@test-writer** - Test Writer\n\nWrite comprehensive tests\n**Tools:** [create_file edit_file]\n**Model:** gpt-4",
		},
		{
			name:     "help for partial match",
			line:     "@c",
			pos:      2,
			expected: "**Subagents** - Matching subagents:\n\n• **@code-reviewer** - Review code for bugs and best practices",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := provider.GetHelpInfo(tc.line, tc.pos)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSubagentCompletionWithoutProvider(t *testing.T) {
	runner, _ := interp.New()
	manager := NewCompletionManager()
	provider := NewShellCompletionProvider(manager, runner)
	// Note: No subagent provider set

	// Should return empty completions when no provider is set
	result := provider.GetCompletions("@", 1)
	assert.Equal(t, []shellinput.CompletionCandidate{}, result)

	// Should return generic help when no provider is set
	help := provider.GetHelpInfo("@", 1)
	expected := "**Subagents** - Specialized AI assistants with specific roles\n\nNo subagent manager configured. Use @<subagent-name> to invoke a subagent."
	assert.Equal(t, expected, help)
}
