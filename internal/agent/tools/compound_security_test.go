package tools

import (
	"path/filepath"
	"testing"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestCompoundCommandSecurity(t *testing.T) {
	// Create a test runner and logger
	runner, err := interp.New()
	assert.NoError(t, err)

	// Initialize the Vars map
	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}

	logger := zap.NewNop()

	// Create temporary directory for testing
	tempDir := t.TempDir()
	environment.SetConfigDirForTesting(tempDir)
	environment.SetAuthorizedCommandsFileForTesting(filepath.Join(tempDir, "authorized_commands"))
	defer func() {
		environment.SetConfigDirForTesting("")
		environment.SetAuthorizedCommandsFileForTesting("")
		environment.ResetCacheForTesting()
	}()

	tests := []struct {
		name                string
		approvedPatterns    []string
		command             string
		shouldBePreApproved bool
		description         string
	}{
		{
			name:                "simple approved command",
			approvedPatterns:    []string{"^ls.*"},
			command:             "ls -la",
			shouldBePreApproved: true,
			description:         "Single approved command should be pre-approved",
		},
		{
			name:                "compound approved commands",
			approvedPatterns:    []string{"^ls.*", "^pwd.*", "^echo.*"},
			command:             "ls && pwd && echo done",
			shouldBePreApproved: true,
			description:         "All commands in compound statement are approved",
		},
		{
			name:                "malicious injection with semicolon",
			approvedPatterns:    []string{"^ls.*"},
			command:             "ls; rm -rf /",
			shouldBePreApproved: false,
			description:         "Should block rm command even though ls is approved",
		},
		{
			name:                "malicious injection with AND operator",
			approvedPatterns:    []string{"^ls.*"},
			command:             "ls && rm -rf /",
			shouldBePreApproved: false,
			description:         "Should block rm command in AND chain",
		},
		{
			name:                "malicious injection with OR operator",
			approvedPatterns:    []string{"^ls.*"},
			command:             "ls || rm -rf /",
			shouldBePreApproved: false,
			description:         "Should block rm command in OR chain",
		},
		{
			name:                "malicious injection in pipe",
			approvedPatterns:    []string{"^ls.*"},
			command:             "ls | rm -rf /",
			shouldBePreApproved: false,
			description:         "Should block rm command in pipe",
		},
		{
			name:                "malicious injection in subshell",
			approvedPatterns:    []string{"^ls.*"},
			command:             "(ls && rm -rf /)",
			shouldBePreApproved: false,
			description:         "Should block rm command in subshell",
		},
		{
			name:                "malicious injection in command substitution",
			approvedPatterns:    []string{"^echo.*"},
			command:             "echo $(ls && rm -rf /)",
			shouldBePreApproved: false,
			description:         "Should block rm command in command substitution",
		},
		{
			name:                "complex nested injection",
			approvedPatterns:    []string{"^ls.*", "^echo.*"},
			command:             "ls && (echo safe && rm -rf /tmp/*)",
			shouldBePreApproved: false,
			description:         "Should block nested rm command",
		},
		{
			name:                "legitimate complex command",
			approvedPatterns:    []string{"^ls.*", "^echo.*", "^pwd.*"},
			command:             "ls && echo $(pwd) && (echo done && ls)",
			shouldBePreApproved: true,
			description:         "All nested commands are approved",
		},
		{
			name:                "user reported issue - compound pipe with all approved commands",
			approvedPatterns:    []string{"^ls.*", "^awk.*", "^sort.*"},
			command:             "ls -la | awk 'NR>1 {size=$5; name=$9; if(size>0) {logsize=log(size)/log(1024); if(logsize<1) unit=\"B\"; else if(logsize<2) unit=\"K\"; else if(logsize<3) unit=\"M\"; else unit=\"G\"; printf \"%8.1f%s %s\\n\", size/(1024^int(logsize)), unit, name}}' | sort -k2 -nr",
			shouldBePreApproved: true,
			description:         "Compound pipe command with all individual commands approved should be pre-approved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cache and setup environment
			environment.ResetCacheForTesting()

			// Set up approved patterns in environment variable
			if len(tt.approvedPatterns) > 0 {
				patternsJSON := `["` + tt.approvedPatterns[0]
				for i := 1; i < len(tt.approvedPatterns); i++ {
					patternsJSON += `", "` + tt.approvedPatterns[i]
				}
				patternsJSON += `"]`

				runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"] = expand.Variable{
					Kind: expand.String,
					Str:  patternsJSON,
				}
			} else {
				delete(runner.Vars, "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX")
			}

			// Test the validation
			approvedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)
			isPreApproved, err := ValidateCompoundCommand(tt.command, approvedPatterns)

			assert.NoError(t, err, "Validation should not error for: %s", tt.command)
			assert.Equal(t, tt.shouldBePreApproved, isPreApproved,
				"Command '%s' pre-approval mismatch: %s", tt.command, tt.description)
		})
	}
}

func TestCompoundCommandRegexGeneration(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "simple command",
			command:  "ls -la",
			expected: []string{"^ls.*"},
		},
		{
			name:     "compound command with semicolon",
			command:  "ls; pwd; echo hello",
			expected: []string{"^ls.*", "^pwd.*", "^echo hello.*"},
		},
		{
			name:     "compound command with AND",
			command:  "ls && pwd && echo done",
			expected: []string{"^ls.*", "^pwd.*", "^echo done.*"},
		},
		{
			name:     "compound command with pipes",
			command:  "ls | grep txt | sort",
			expected: []string{"^ls.*", "^grep txt.*", "^sort.*"},
		},
		{
			name:     "subshell command",
			command:  "(cd /tmp && ls)",
			expected: []string{"^cd.*", "^ls.*"},
		},
		{
			name:     "command substitution",
			command:  "echo $(date)",
			expected: []string{"^echo.*", "^date.*"},
		},
		{
			name:     "complex mixed command",
			command:  "ls && echo $(pwd) && (cd /tmp; ls)",
			expected: []string{"^ls.*", "^echo.*", "^pwd.*", "^cd.*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateCompoundCommandRegex(tt.command)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestEndToEndSecurityScenario(t *testing.T) {
	// Create a test runner and logger
	runner, err := interp.New()
	assert.NoError(t, err)

	// Initialize the Vars map
	if runner.Vars == nil {
		runner.Vars = make(map[string]expand.Variable)
	}

	logger := zap.NewNop()

	// Create temporary directory for testing
	tempDir := t.TempDir()
	environment.SetConfigDirForTesting(tempDir)
	environment.SetAuthorizedCommandsFileForTesting(filepath.Join(tempDir, "authorized_commands"))
	defer func() {
		environment.SetConfigDirForTesting("")
		environment.SetAuthorizedCommandsFileForTesting("")
		environment.ResetCacheForTesting()
	}()

	// Set up initial approved patterns
	runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"] = expand.Variable{
		Kind: expand.String,
		Str:  `["^ls.*", "^echo.*"]`,
	}

	// Test 1: Legitimate compound command should be pre-approved
	legitimateCommand := "ls && echo done"
	approvedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)
	isPreApproved, err := ValidateCompoundCommand(legitimateCommand, approvedPatterns)
	assert.NoError(t, err)
	assert.True(t, isPreApproved, "Legitimate compound command should be pre-approved")

	// Test 2: Malicious injection should be blocked
	maliciousCommand := "ls; rm -rf /"
	isPreApproved, err = ValidateCompoundCommand(maliciousCommand, approvedPatterns)
	assert.NoError(t, err)
	assert.False(t, isPreApproved, "Malicious injection should be blocked")

	// Test 3: Add new patterns via "always allow" functionality
	newCommand := "pwd && date"
	patterns, err := GenerateCompoundCommandRegex(newCommand)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"^pwd.*", "^date.*"}, patterns)

	// Simulate saving patterns to file
	for _, pattern := range patterns {
		err = environment.AppendToAuthorizedCommands(pattern)
		assert.NoError(t, err)
	}

	// Test 4: Verify new patterns are loaded and work
	environment.ResetCacheForTesting() // Force reload from file
	updatedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)

	// Should now include both environment and file patterns
	expectedPatterns := []string{"^ls.*", "^echo.*", "^pwd.*", "^date.*"}
	assert.ElementsMatch(t, expectedPatterns, updatedPatterns)

	// Test 5: New command should now be pre-approved
	isPreApproved, err = ValidateCompoundCommand(newCommand, updatedPatterns)
	assert.NoError(t, err)
	assert.True(t, isPreApproved, "New command should be pre-approved after adding patterns")

	// Test 6: Injection with new patterns should still be blocked
	newMaliciousCommand := "pwd; rm -rf /"
	isPreApproved, err = ValidateCompoundCommand(newMaliciousCommand, updatedPatterns)
	assert.NoError(t, err)
	assert.False(t, isPreApproved, "Injection should still be blocked even with new patterns")
}
