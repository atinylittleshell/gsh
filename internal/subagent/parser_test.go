package subagent

import (
	"os"
	"testing"
	"time"
)

func TestParseClaudeConfig(t *testing.T) {
	// Create a temporary file with Claude-style configuration
	content := `---
name: test-agent
description: A test agent for unit testing
tools: view_file, bash
model: inherit
---

You are a test agent used for unit testing the subagent system.
`

	tmpFile, err := os.CreateTemp("", "claude-config-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	// Parse the configuration
	subagents, err := ParseConfigFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse Claude config: %v", err)
	}

	if len(subagents) != 1 {
		t.Fatalf("Expected 1 subagent, got %d", len(subagents))
	}

	subagent := subagents[0]

	// Verify parsed values
	if subagent.ID != "test-agent" {
		t.Errorf("Expected ID 'test-agent', got '%s'", subagent.ID)
	}

	if subagent.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", subagent.Name)
	}

	if subagent.Type != ClaudeType {
		t.Errorf("Expected type %s, got %s", ClaudeType, subagent.Type)
	}

	if len(subagent.AllowedTools) != 2 {
		t.Errorf("Expected 2 allowed tools, got %d", len(subagent.AllowedTools))
	}

	expectedPrompt := "You are a test agent used for unit testing the subagent system."
	if subagent.SystemPrompt != expectedPrompt {
		t.Errorf("Expected system prompt '%s', got '%s'", expectedPrompt, subagent.SystemPrompt)
	}
}

func TestParseRooConfig(t *testing.T) {
	content := `customModes:
  - slug: test-mode
    name: Test Mode
    description: A test mode for unit testing
    roleDefinition: You are a test mode used for unit testing.
    groups:
      - read
      - command
`

	tmpFile, err := os.CreateTemp("", "roo-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tmpFile.Close()

	// Parse the configuration
	subagents, err := ParseConfigFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse Roo config: %v", err)
	}

	if len(subagents) != 1 {
		t.Fatalf("Expected 1 subagent, got %d", len(subagents))
	}

	subagent := subagents[0]

	// Verify parsed values
	if subagent.ID != "test-mode" {
		t.Errorf("Expected ID 'test-mode', got '%s'", subagent.ID)
	}

	if subagent.Name != "Test Mode" {
		t.Errorf("Expected name 'Test Mode', got '%s'", subagent.Name)
	}

	if subagent.Type != RooType {
		t.Errorf("Expected type %s, got %s", RooType, subagent.Type)
	}

	// Should have both view tools (from read) and bash (from command)
	if len(subagent.AllowedTools) < 2 {
		t.Errorf("Expected at least 2 allowed tools, got %d", len(subagent.AllowedTools))
	}

	if subagent.SystemPrompt != "You are a test mode used for unit testing." {
		t.Errorf("Unexpected system prompt: %s", subagent.SystemPrompt)
	}
}

func TestSubagentValidation(t *testing.T) {
	validSubagent := &Subagent{
		ID:           "test-agent",
		Name:         "Test Agent",
		Type:         ClaudeType,
		SystemPrompt: "You are a test agent.",
		AllowedTools: []string{"view_file", "bash"},
		LastModified: time.Now(),
	}

	if err := ValidateSubagent(validSubagent); err != nil {
		t.Errorf("Valid subagent failed validation: %v", err)
	}

	// Test invalid subagent (empty ID)
	invalidSubagent := &Subagent{
		Name:         "Test Agent",
		SystemPrompt: "You are a test agent.",
		AllowedTools: []string{"view_file"},
	}

	if err := ValidateSubagent(invalidSubagent); err == nil {
		t.Error("Expected validation to fail for empty ID")
	}

	// Test invalid tool
	invalidToolSubagent := &Subagent{
		ID:           "test-agent",
		Name:         "Test Agent",
		SystemPrompt: "You are a test agent.",
		AllowedTools: []string{"invalid_tool"},
	}

	if err := ValidateSubagent(invalidToolSubagent); err == nil {
		t.Error("Expected validation to fail for invalid tool")
	}
}
