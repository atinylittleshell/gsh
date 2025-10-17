package subagent

import (
	"time"

	"mvdan.cc/sh/v3/interp"
)

// SubagentType represents the format type of the subagent configuration
type SubagentType string

const (
	ClaudeType SubagentType = "claude"
	RooType    SubagentType = "roo"
)

// Subagent represents a unified subagent configuration from either Claude or Roo Code formats
type Subagent struct {
	// Unified fields
	ID          string      `json:"id"`          // Unique identifier (name for Claude, slug for Roo)
	Name        string      `json:"name"`        // Display name
	Description string      `json:"description"` // Description of when to use this subagent
	Type        SubagentType `json:"type"`       // Configuration format type
	FilePath    string      `json:"filePath"`   // Path to configuration file
	LastModified time.Time  `json:"lastModified"` // File modification time for cache invalidation

	// System prompt content
	SystemPrompt string `json:"systemPrompt"`

	// Tool configuration
	AllowedTools []string `json:"allowedTools"` // List of allowed gsh tools
	FileRegex    string   `json:"fileRegex"`    // File access restriction pattern (from Roo Code)

	// Model configuration
	Model string `json:"model"` // Model override or "inherit"

	// Source configuration for debugging/display
	SourceConfig interface{} `json:"sourceConfig,omitempty"`
}

// ClaudeConfig represents the YAML frontmatter structure for Claude-style subagents
type ClaudeConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Tools       string `yaml:"tools,omitempty"`       // Comma-separated list
	Model       string `yaml:"model,omitempty"`       // Model override
}

// RooCustomMode represents a single custom mode from Roo Code configuration
type RooCustomMode struct {
	Slug            string                 `yaml:"slug"`
	Name            string                 `yaml:"name"`
	Description     string                 `yaml:"description,omitempty"`
	RoleDefinition  string                 `yaml:"roleDefinition"`
	WhenToUse       string                 `yaml:"whenToUse,omitempty"`
	CustomInstructions string             `yaml:"customInstructions,omitempty"`
	Groups          []interface{}          `yaml:"groups"`
	Model           string                 `yaml:"model,omitempty"`
}

// RooConfig represents the top-level Roo Code configuration structure
type RooConfig struct {
	CustomModes []RooCustomMode `yaml:"customModes"`
}

// RooGroupConfig represents group configurations with optional restrictions
type RooGroupConfig struct {
	Group     string `yaml:"group"`
	FileRegex string `yaml:"fileRegex,omitempty"`
}

// SubagentManager handles loading, parsing, and managing subagent configurations
type SubagentManager struct {
	subagents   map[string]*Subagent // Key: subagent ID
	directories []string             // Directories to scan for configurations
	lastScan    time.Time            // Last time directories were scanned
	runner      *interp.Runner       // Shell runner for accessing PWD
	currentPWD  string               // Current working directory at last scan
}