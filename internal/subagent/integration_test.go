package subagent

import (
	"testing"
)

func TestIntegrationParseExampleConfigs(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
		expected int // Expected number of subagents
	}{
		{
			name:     "Claude code-reviewer",
			filePath: "../../.claude/agents/code-reviewer.md",
			expected: 1,
		},
		{
			name:     "Claude test-writer",
			filePath: "../../.claude/agents/test-writer.md",
			expected: 1,
		},
		{
			name:     "Claude docs-writer",
			filePath: "../../.claude/agents/docs-writer.md",
			expected: 1,
		},
		{
			name:     "Roo dev_modes",
			filePath: "../../.roo/modes/dev_modes.yaml",
			expected: 4, // git-helper, golang-dev, debug-assistant, security-auditor
		},
		{
			name:     "Roo writing_modes",
			filePath: "../../.roo/modes/writing_modes.yaml",
			expected: 3, // technical-writer, commit-helper, changelog-writer
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subagents, err := ParseConfigFile(tc.filePath)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", tc.filePath, err)
			}

			if len(subagents) != tc.expected {
				t.Errorf("Expected %d subagents from %s, got %d", tc.expected, tc.filePath, len(subagents))
			}

			// Validate each subagent
			for i, subagent := range subagents {
				if err := ValidateSubagent(subagent); err != nil {
					t.Errorf("Subagent %d from %s failed validation: %v", i, tc.filePath, err)
				}

				// Basic checks
				if subagent.ID == "" {
					t.Errorf("Subagent %d from %s has empty ID", i, tc.filePath)
				}
				if subagent.Name == "" {
					t.Errorf("Subagent %d from %s has empty name", i, tc.filePath)
				}
				if subagent.SystemPrompt == "" {
					t.Errorf("Subagent %d from %s has empty system prompt", i, tc.filePath)
				}
				if len(subagent.AllowedTools) == 0 {
					t.Errorf("Subagent %d from %s has no allowed tools", i, tc.filePath)
				}

				t.Logf("✓ %s: ID=%s, Name=%s, Type=%s, Tools=%v",
					tc.name, subagent.ID, subagent.Name, subagent.Type, subagent.AllowedTools)
			}
		})
	}
}