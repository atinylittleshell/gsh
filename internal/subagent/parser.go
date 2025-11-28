package subagent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ParseConfigFile detects the format and parses either Claude or Roo Code configuration
func ParseConfigFile(filePath string) ([]*Subagent, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Check if this is a Roo rules directory
	if info.IsDir() && strings.Contains(filePath, "rules-") {
		return parseRooRulesDirectory(filePath, info.ModTime())
	}

	// Check if this is a .roomodes file
	if strings.HasSuffix(filepath.Base(filePath), ".roomodes") {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read .roomodes file %s: %w", filePath, err)
		}
		return parseRoomodesFile(filePath, content, info.ModTime())
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".md":
		subagent, err := parseClaudeConfig(filePath, content, info.ModTime())
		if err != nil {
			return nil, err
		}
		return []*Subagent{subagent}, nil

	case ".yaml", ".yml":
		return parseRooConfig(filePath, content, info.ModTime())

	default:
		return nil, fmt.Errorf("unsupported configuration file format: %s", ext)
	}
}

// parseClaudeConfig parses Claude-style .md files with YAML frontmatter
func parseClaudeConfig(filePath string, content []byte, modTime time.Time) (*Subagent, error) {
	contentStr := string(content)

	// Check if file has YAML frontmatter
	// Handle both Unix and Windows line endings
	if !strings.HasPrefix(contentStr, "---\n") && !strings.HasPrefix(contentStr, "---\r\n") {
		return nil, fmt.Errorf("claude configuration file must start with YAML frontmatter: %s", filePath)
	}

	// Find end of frontmatter
	// Handle both Unix and Windows line endings for the delimiter
	endIdx := strings.Index(contentStr[4:], "\n---\n")
	delimiterLength := 5 // \n---\n
	if endIdx == -1 {
		// Try Windows style line endings
		endIdx = strings.Index(contentStr[4:], "\r\n---\r\n")
		delimiterLength = 7 // \r\n---\r\n
	}

	if endIdx == -1 {
		return nil, fmt.Errorf("invalid YAML frontmatter in file: %s", filePath)
	}
	endIdx += 4 // Adjust for the initial offset

	// Extract frontmatter and content
	frontmatter := contentStr[4:endIdx]
	systemPrompt := strings.TrimSpace(contentStr[endIdx+delimiterLength:])

	// Parse YAML frontmatter
	var config ClaudeConfig
	if err := yaml.Unmarshal([]byte(frontmatter), &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter in %s: %w", filePath, err)
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("missing required 'name' field in %s", filePath)
	}
	if config.Description == "" {
		return nil, fmt.Errorf("missing required 'description' field in %s", filePath)
	}

	// Parse tools list
	allowedTools := parseToolsList(config.Tools)

	// Create subagent
	subagent := &Subagent{
		ID:           config.Name,
		Name:         config.Name,
		Description:  config.Description,
		Type:         ClaudeType,
		FilePath:     filePath,
		LastModified: modTime,
		SystemPrompt: systemPrompt,
		AllowedTools: allowedTools,
		Model:        config.Model,
		SourceConfig: config,
	}

	return subagent, nil
}

// parseRooConfig parses Roo Code-style YAML configuration files
func parseRooConfig(filePath string, content []byte, modTime time.Time) ([]*Subagent, error) {
	var config RooConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Roo Code YAML in %s: %w", filePath, err)
	}

	var subagents []*Subagent
	for _, mode := range config.CustomModes {
		// Validate required fields
		if mode.Slug == "" {
			return nil, fmt.Errorf("missing required 'slug' field in mode configuration in %s", filePath)
		}
		if mode.RoleDefinition == "" {
			return nil, fmt.Errorf("missing required 'roleDefinition' field for mode '%s' in %s", mode.Slug, filePath)
		}

		// Parse tool groups
		allowedTools, fileRegex := parseRooGroups(mode.Groups)

		// Build system prompt from role definition and optional fields
		systemPrompt := mode.RoleDefinition
		if mode.CustomInstructions != "" {
			systemPrompt += "\n\n" + mode.CustomInstructions
		}
		if mode.WhenToUse != "" {
			systemPrompt += "\n\nUse this mode when: " + mode.WhenToUse
		}

		// Use name if provided, otherwise use slug as display name
		displayName := mode.Name
		if displayName == "" {
			displayName = mode.Slug
		}

		subagent := &Subagent{
			ID:           mode.Slug,
			Name:         displayName,
			Description:  mode.Description,
			Type:         RooType,
			FilePath:     filePath,
			LastModified: modTime,
			SystemPrompt: systemPrompt,
			AllowedTools: allowedTools,
			FileRegex:    fileRegex,
			Model:        mode.Model,
			SourceConfig: mode,
		}

		subagents = append(subagents, subagent)
	}

	return subagents, nil
}

// parseToolsList parses a comma-separated list of tools for Claude format
func parseToolsList(toolsStr string) []string {
	if toolsStr == "" {
		// Return all available tools if none specified
		return []string{"bash", "view_file", "view_directory", "create_file", "edit_file"}
	}

	var tools []string
	for _, tool := range strings.Split(toolsStr, ",") {
		tool = strings.TrimSpace(tool)
		if tool != "" {
			tools = append(tools, tool)
		}
	}
	return tools
}

// parseRooGroups converts Roo Code group configurations to gsh tool permissions
func parseRooGroups(groups []interface{}) ([]string, string) {
	var allowedTools []string
	var fileRegex string
	toolSet := make(map[string]bool)

	for _, group := range groups {
		switch g := group.(type) {
		case string:
			// Simple group name
			tools := mapRooGroupToTools(g)
			for _, tool := range tools {
				toolSet[tool] = true
			}

		case []interface{}:
			// Group with configuration [groupName, {config}]
			if len(g) >= 2 {
				if groupName, ok := g[0].(string); ok {
					tools := mapRooGroupToTools(groupName)
					for _, tool := range tools {
						toolSet[tool] = true
					}

					// Handle group configuration
					if len(g) > 1 {
						if configMap, ok := g[1].(map[string]interface{}); ok {
							if regex, ok := configMap["fileRegex"].(string); ok {
								fileRegex = regex
							}
						}
					}
				}
			}

		case map[string]interface{}:
			// Group configuration as a map
			if groupName, ok := g["group"].(string); ok {
				tools := mapRooGroupToTools(groupName)
				for _, tool := range tools {
					toolSet[tool] = true
				}

				if regex, ok := g["fileRegex"].(string); ok {
					fileRegex = regex
				}
			}
		}
	}

	// Convert set to slice
	for tool := range toolSet {
		allowedTools = append(allowedTools, tool)
	}

	// If no tools were specified, provide defaults
	if len(allowedTools) == 0 {
		allowedTools = []string{"view_file", "view_directory"}
	}

	return allowedTools, fileRegex
}

// mapRooGroupToTools maps Roo Code tool groups to gsh tools
func mapRooGroupToTools(group string) []string {
	switch group {
	case "read":
		return []string{"view_file", "view_directory"}
	case "edit":
		return []string{"create_file", "edit_file", "view_file", "view_directory"}
	case "command":
		return []string{"bash"}
	case "browser":
		// Not applicable in gsh context, but we could add web-related tools in future
		return []string{}
	case "mcp":
		// Future extension point for MCP tools
		return []string{}
	default:
		// Unknown group, return read permissions as fallback
		return []string{"view_file", "view_directory"}
	}
}

// ValidateSubagent performs validation checks on a parsed subagent configuration
func ValidateSubagent(subagent *Subagent) error {
	if subagent.ID == "" {
		return fmt.Errorf("subagent ID cannot be empty")
	}
	if subagent.Name == "" {
		return fmt.Errorf("subagent name cannot be empty")
	}
	if subagent.SystemPrompt == "" {
		return fmt.Errorf("subagent system prompt cannot be empty")
	}

	// Validate allowed tools against known gsh tools
	knownTools := map[string]bool{
		"bash":           true,
		"view_file":      true,
		"view_directory": true,
		"create_file":    true,
		"edit_file":      true,
	}

	for _, tool := range subagent.AllowedTools {
		if !knownTools[tool] {
			return fmt.Errorf("unknown tool '%s' in subagent '%s'", tool, subagent.ID)
		}
	}

	// Validate file regex if present
	if subagent.FileRegex != "" {
		if _, err := regexp.Compile(subagent.FileRegex); err != nil {
			return fmt.Errorf("invalid file regex '%s' in subagent '%s': %w", subagent.FileRegex, subagent.ID, err)
		}
	}

	return nil
}

// parseRooRulesDirectory parses a Roo rules directory (.roo/rules-{modeSlug}/)
func parseRooRulesDirectory(dirPath string, modTime time.Time) ([]*Subagent, error) {
	// Extract the mode slug from the directory name
	dirName := filepath.Base(dirPath)
	if !strings.HasPrefix(dirName, "rules-") {
		return nil, fmt.Errorf("invalid Roo rules directory name: %s", dirName)
	}

	modeSlug := strings.TrimPrefix(dirName, "rules-")
	if modeSlug == "" {
		return nil, fmt.Errorf("empty mode slug in directory: %s", dirName)
	}

	// Read all markdown files in the directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Roo rules directory %s: %w", dirPath, err)
	}

	var rulesContent strings.Builder
	hasRules := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.ToLower(filepath.Ext(entry.Name())) == ".md" {
			filePath := filepath.Join(dirPath, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read rule file %s: %w", filePath, err)
			}

			if hasRules {
				rulesContent.WriteString("\n\n")
			}
			rulesContent.WriteString(string(content))
			hasRules = true
		}
	}

	if !hasRules {
		return nil, fmt.Errorf("no markdown files found in Roo rules directory: %s", dirPath)
	}

	// Create a subagent from the combined rules
	subagent := &Subagent{
		ID:           modeSlug,
		Name:         strings.ToUpper(string(modeSlug[0])) + strings.ReplaceAll(modeSlug[1:], "-", " "),
		Description:  fmt.Sprintf("Roo custom mode: %s", modeSlug),
		Type:         RooType,
		FilePath:     dirPath,
		LastModified: modTime,
		SystemPrompt: rulesContent.String(),
		AllowedTools: []string{"bash", "view_file", "create_file", "edit_file"}, // Default Roo tools
		FileRegex:    "",                                                        // No file restrictions by default
		Model:        "",                                                        // Use default model
		SourceConfig: nil,                                                       // No structured config for directory-based rules
	}

	return []*Subagent{subagent}, nil
}

// parseRoomodesFile parses .roomodes files containing mode definitions
func parseRoomodesFile(filePath string, content []byte, modTime time.Time) ([]*Subagent, error) {
	// .roomodes files are expected to be YAML files with the same format as regular Roo configs
	// but with a .roomodes extension instead of .yaml/.yml
	return parseRooConfig(filePath, content, modTime)
}