package tools

import (
	"flag"
	"fmt"
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/styles"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/syntax"
)

// PermissionAtom represents a single permission rule for a command prefix
type PermissionAtom struct {
	Command string // The command prefix (e.g., "ls", "ls --foo", "ls --foo bar")
	Enabled bool   // Whether this permission is enabled
	IsNew   bool   // Whether this is a new permission being added
}

// PermissionsMenuState tracks the state of the permissions menu
type PermissionsMenuState struct {
	atoms           []PermissionAtom
	selectedIndex   int
	originalCommand string
	active          bool
}

// PermissionsCompletionProvider implements shellinput.CompletionProvider for the permissions menu
type PermissionsCompletionProvider struct {
	state  *PermissionsMenuState
	logger *zap.Logger
}

// GetCompletions returns the permission atoms as completion suggestions
func (p *PermissionsCompletionProvider) GetCompletions(line string, pos int) []string {
	// Return the command prefixes as suggestions
	suggestions := make([]string, len(p.state.atoms))
	for i, atom := range p.state.atoms {
		suggestions[i] = atom.Command
	}
	return suggestions
}

// GetHelpInfo returns help information for the permissions menu
func (p *PermissionsCompletionProvider) GetHelpInfo(line string, pos int) string {
	return "Use j/k to navigate, space to toggle, enter to apply, esc to cancel"
}

// GenerateCommandPrefixes generates limited prefixes of a command for permission atoms
// Returns 1-3 distinct options: base command, with first flag/arg, and full command (if different)
// Properly handles quoted arguments using shell parsing
func GenerateCommandPrefixes(command string) []string {
	command = strings.TrimSpace(command)
	if command == "" {
		return []string{}
	}

	// Use shell parser to properly tokenize the command, respecting quotes
	parser := syntax.NewParser()
	stmt, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		// Fallback to simple field splitting if parsing fails
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return []string{}
		}
		return generatePrefixesFromParts(parts, command)
	}

	// Extract the command and arguments from the parsed statement
	if len(stmt.Stmts) == 0 {
		return []string{}
	}

	firstStmt := stmt.Stmts[0]
	if firstStmt.Cmd == nil {
		return []string{}
	}

	callExpr, ok := firstStmt.Cmd.(*syntax.CallExpr)
	if !ok {
		// Fallback for non-call expressions
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return []string{}
		}
		return generatePrefixesFromParts(parts, command)
	}

	// Extract command name and arguments
	var parts []string
	for _, word := range callExpr.Args {
		// Convert the word back to string representation
		wordStr := wordToString(word)
		if wordStr != "" {
			parts = append(parts, wordStr)
		}
	}

	if len(parts) == 0 {
		return []string{}
	}

	return generatePrefixesFromParts(parts, command)
}

// generatePrefixesFromParts generates prefixes from tokenized parts
func generatePrefixesFromParts(parts []string, originalCommand string) []string {
	var prefixes []string

	// Always include the base command
	prefixes = append(prefixes, parts[0])

	// If there are more parts, include the command with first argument/flag
	if len(parts) > 1 {
		firstArg := parts[0] + " " + parts[1]
		if firstArg != prefixes[0] {
			prefixes = append(prefixes, firstArg)
		}
	}

	// If there are 3+ parts and the full command is different, include it
	if len(parts) >= 3 {
		if originalCommand != prefixes[len(prefixes)-1] {
			prefixes = append(prefixes, originalCommand)
		}
	}

	return prefixes
}

// wordToString converts a syntax.Word to its string representation
func wordToString(word *syntax.Word) string {
	if word == nil {
		return ""
	}

	var result strings.Builder
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			result.WriteString(p.Value)
		case *syntax.SglQuoted:
			result.WriteString("'")
			result.WriteString(p.Value)
			result.WriteString("'")
		case *syntax.DblQuoted:
			result.WriteString("\"")
			for _, dqPart := range p.Parts {
				if lit, ok := dqPart.(*syntax.Lit); ok {
					result.WriteString(lit.Value)
				}
			}
			result.WriteString("\"")
		}
	}
	return result.String()
}

// showPermissionsMenuImpl is the actual implementation of ShowPermissionsMenu
// This is separated to allow for mocking in tests
func showPermissionsMenuImpl(logger *zap.Logger, command string) (string, error) {
	// First try to extract individual commands from compound statements
	individualCommands, err := ExtractCommands(command)
	if err != nil {
		logger.Warn("Failed to parse compound command, falling back to prefix generation", zap.Error(err))
		individualCommands = []string{command}
	}

	// Generate permission atoms for each individual command
	var atoms []PermissionAtom
	for _, cmd := range individualCommands {
		// Generate 1-3 prefixes for each individual command
		prefixes := GenerateCommandPrefixes(cmd)
		for _, prefix := range prefixes {
			// Check if the pattern that would be saved for this prefix exists in the authorized_commands file
			// Use literal pattern matching to avoid over-matching in the menu display
			regexPattern := GeneratePreselectionPattern(prefix)
			isAuthorized, err := environment.IsCommandPatternAuthorized(regexPattern)
			if err != nil {
				logger.Warn("Failed to check if command pattern is authorized", zap.String("command", prefix), zap.String("pattern", regexPattern), zap.Error(err))
				isAuthorized = false
			}

			atoms = append(atoms, PermissionAtom{
				Command: prefix,
				Enabled: isAuthorized,  // Pre-select if already authorized
				IsNew:   !isAuthorized, // Not new if already authorized
			})
		}
	}

	if len(atoms) == 0 {
		return "n", fmt.Errorf("no command prefixes found")
	}

	// Initialize menu state
	state := &PermissionsMenuState{
		atoms:           atoms,
		selectedIndex:   0,
		originalCommand: command,
		active:          true,
	}

	// Use a simpler approach - create a custom tea program that mimics gline behavior
	return runPermissionsMenuSimple(logger, state)
}

// ShowPermissionsMenu is a package-level variable that can be mocked in tests
// By default, it points to the real implementation
var ShowPermissionsMenu = showPermissionsMenuImpl

// runPermissionsMenuSimple runs a simple permissions menu using tea
func runPermissionsMenuSimple(logger *zap.Logger, state *PermissionsMenuState) (string, error) {
	// Check if we're running in test mode by checking if the test flag is set
	// This prevents the interactive menu from blocking during tests
	if flag.Lookup("test.v") != nil {
		logger.Debug("Running in test mode, skipping interactive menu")
		return "n", nil
	}

	// Create a simple tea program
	program := tea.NewProgram(&simplePermissionsModel{
		state:  state,
		logger: logger,
	})

	finalModel, err := program.Run()
	if err != nil {
		logger.Warn("Permissions menu program.Run() returned error", zap.Error(err))
		// Don't automatically return "n" - let the model handle the result
		// The error might be from normal termination
	}

	// Extract the result from the final model
	if result, ok := finalModel.(*simplePermissionsModel); ok {
		logger.Debug("Permissions menu completed", zap.String("result", result.result))
		return result.result, nil
	}

	logger.Warn("Permissions menu: unexpected model type, defaulting to 'n'")
	return "n", nil
}

// simplePermissionsModel is a simple tea model for the permissions menu
type simplePermissionsModel struct {
	state  *PermissionsMenuState
	logger *zap.Logger
	result string
}

// Init initializes the simple permissions model
func (m *simplePermissionsModel) Init() tea.Cmd {
	return nil
}

// Update handles key presses and updates the model
func (m *simplePermissionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// User pressed Ctrl+C - treat as "n" (decline)
			m.result = "n"
			return m, tea.Quit
		case "esc":
			// User pressed Esc - just cancel the menu
			m.result = "n"
			return m, tea.Quit

		case "enter":
			// Apply the current selections
			m.result = m.processMenuSelection()
			return m, tea.Quit

		case "j", "down":
			// Navigate down
			if m.state.selectedIndex < len(m.state.atoms)-1 {
				m.state.selectedIndex++
			}
			return m, nil

		case "k", "up":
			// Navigate up
			if m.state.selectedIndex > 0 {
				m.state.selectedIndex--
			}
			return m, nil

		case " ":
			// Toggle current selection
			if m.state.selectedIndex >= 0 && m.state.selectedIndex < len(m.state.atoms) {
				m.state.atoms[m.state.selectedIndex].Enabled = !m.state.atoms[m.state.selectedIndex].Enabled
			}
			return m, nil

		case "y":
			m.result = "y"
			return m, tea.Quit

		case "n":
			m.result = "n"
			return m, tea.Quit

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Jump to specific item
			index := int(msg.String()[0] - '1')
			if index >= 0 && index < len(m.state.atoms) {
				m.state.selectedIndex = index
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the permissions menu
func (m *simplePermissionsModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(styles.AGENT_QUESTION(fmt.Sprintf("Managing permissions for: %s\n\n", m.state.originalCommand)))
	content.WriteString("Permission Management - Toggle permissions for command prefixes:\n\n")

	// Show each option with clear formatting like the completion box
	for i, atom := range m.state.atoms {
		// Selection indicator (like completion box)
		indicator := "     "
		if i == m.state.selectedIndex {
			indicator = ">    "
		}

		// Checkbox
		checkbox := "[ ]"
		if atom.Enabled {
			checkbox = "[✓]"
		}

		// Command (truncate if needed)
		command := atom.Command
		if len(command) > 60 {
			command = command[:57] + "..."
		}

		content.WriteString(fmt.Sprintf("%s%s %s\n", indicator, checkbox, command))
	}

	content.WriteString("\n")
	content.WriteString("Controls: j/k=navigate, space=toggle, enter=apply, esc=cancel\n")
	content.WriteString("Direct: y=yes (one-time), n=no (deny)\n")

	// Current state
	currentCmd := m.state.atoms[m.state.selectedIndex].Command
	if len(currentCmd) > 50 {
		currentCmd = currentCmd[:47] + "..."
	}
	content.WriteString(fmt.Sprintf("\nCurrent: %s\n", currentCmd))
	content.WriteString(fmt.Sprintf("Enabled: %s\n", m.getEnabledPermissionsList()))

	return content.String()
}

// processMenuSelection handles the final selection and returns the appropriate response
func (m *simplePermissionsModel) processMenuSelection() string {
	m.state.active = false

	// Load existing authorized commands
	existingPatterns, err := environment.LoadAuthorizedCommandsFromFile()
	if err != nil {
		m.logger.Error("Failed to load existing authorized commands", zap.Error(err))
		existingPatterns = []string{}
	}

	// Create a map of existing patterns for easy lookup
	existingPatternsMap := make(map[string]bool)
	for _, pattern := range existingPatterns {
		existingPatternsMap[pattern] = true
	}

	// Build the new list of patterns based on current menu state
	var newPatterns []string

	// First, add all existing patterns that are not related to the current menu atoms
	menuRegexes := make(map[string]bool)
	for _, atom := range m.state.atoms {
		// Use GenerateCommandRegex for runtime matching patterns
		regex := GenerateCommandRegex(atom.Command)
		menuRegexes[regex] = true
		// Also track the specific pattern for pre-selection
		specificRegex := GenerateSpecificCommandRegex(atom.Command)
		menuRegexes[specificRegex] = true
	}

	// Keep existing patterns that are not being managed by this menu
	for _, pattern := range existingPatterns {
		if !menuRegexes[pattern] {
			newPatterns = append(newPatterns, pattern)
		}
	}

	// Add enabled permissions from the current menu
	hasEnabledPermissions := false
	for _, atom := range m.state.atoms {
		if atom.Enabled {
			hasEnabledPermissions = true
			// Use GenerateCommandRegex for runtime matching (e.g., ^ls.* instead of ^ls$)
			regex := GenerateCommandRegex(atom.Command)
			newPatterns = append(newPatterns, regex)
		}
	}

	// Write the updated patterns to file (this will deduplicate automatically)
	if err := environment.WriteAuthorizedCommandsToFile(newPatterns); err != nil {
		m.logger.Error("Failed to update authorized commands file", zap.Error(err))
	}

	if hasEnabledPermissions {
		// Return "manage" to indicate we want to manage permissions
		return "manage"
	} else {
		// No permissions enabled, treat as "yes" for one-time execution
		return "y"
	}
}

// getEnabledPermissionsList returns a string representation of enabled permissions
func (m *simplePermissionsModel) getEnabledPermissionsList() string {
	var enabled []string
	for _, atom := range m.state.atoms {
		if atom.Enabled {
			command := atom.Command
			// Truncate long commands for display
			if len(command) > 30 {
				command = command[:27] + "..."
			}
			enabled = append(enabled, command)
		}
	}
	if len(enabled) == 0 {
		return "none"
	}
	return strings.Join(enabled, ", ")
}

// renderPermissionsMenu creates the visual representation of the permissions menu (for tests)
func renderPermissionsMenu(state *PermissionsMenuState) string {
	var content strings.Builder

	content.WriteString("┌─ Permission Management ─────────────────────────────────────┐\n")
	content.WriteString("│ Toggle permissions for command prefixes:                    │\n")
	content.WriteString("├──────────────────────────────────────────────────────────────┤\n")

	for i, atom := range state.atoms {
		var line strings.Builder
		line.WriteString("│ ")

		// Selection indicator
		if i == state.selectedIndex {
			line.WriteString("> ")
		} else {
			line.WriteString("  ")
		}

		// Checkbox
		if atom.Enabled {
			line.WriteString("[✓] ")
		} else {
			line.WriteString("[ ] ")
		}

		// Command (truncate if too long)
		command := atom.Command
		maxCommandWidth := 50
		if len(command) > maxCommandWidth {
			command = command[:maxCommandWidth-3] + "..."
		}
		line.WriteString(command)

		// Pad to consistent width
		padding := maxCommandWidth - len(command)
		if padding > 0 {
			line.WriteString(strings.Repeat(" ", padding))
		}

		line.WriteString(" │\n")
		content.WriteString(line.String())
	}

	content.WriteString("├──────────────────────────────────────────────────────────────┤\n")
	content.WriteString("│ ↑/↓: Navigate  SPACE: Toggle  ENTER: Apply  ESC: Cancel     │\n")
	content.WriteString("└──────────────────────────────────────────────────────────────┘")

	return content.String()
}

// handleMenuInput processes user input and returns action or empty string to continue (for tests)
func handleMenuInput(state *PermissionsMenuState, input string) string {
	// Handle space character specially (before trimming)
	if input == " " {
		if state.selectedIndex >= 0 && state.selectedIndex < len(state.atoms) {
			state.atoms[state.selectedIndex].Enabled = !state.atoms[state.selectedIndex].Enabled
		}
		return ""
	}

	input = strings.TrimSpace(input)

	// Handle common single character and word inputs
	switch strings.ToLower(input) {
	case "k", "up": // Navigate up
		if state.selectedIndex > 0 {
			state.selectedIndex--
		}
		return ""

	case "j", "down": // Navigate down
		if state.selectedIndex < len(state.atoms)-1 {
			state.selectedIndex++
		}
		return ""

	case "space", "toggle": // Toggle current selection (word versions)
		if state.selectedIndex >= 0 && state.selectedIndex < len(state.atoms) {
			state.atoms[state.selectedIndex].Enabled = !state.atoms[state.selectedIndex].Enabled
		}
		return ""

	case "", "enter", "apply": // Apply selections
		return processMenuSelection(state)

	case "esc", "escape", "cancel", "q", "quit": // Cancel
		state.active = false
		return "n"

	case "y", "yes": // Direct yes (skip menu)
		return "y"

	case "n", "no": // Direct no
		return "n"

	case "h", "help", "?": // Show help
		return ""

	default:
		// Check for numeric input to jump to specific item
		if len(input) == 1 && input >= "1" && input <= "9" {
			index := int(input[0] - '1')
			if index >= 0 && index < len(state.atoms) {
				state.selectedIndex = index
			}
			return ""
		}

		// Any other input is treated as freeform response
		if input != "" {
			return input
		}
		return ""
	}
}

// processMenuSelection handles the final selection and returns the appropriate response (for tests)
func processMenuSelection(state *PermissionsMenuState) string {
	state.active = false

	// Check if any permissions are enabled
	hasEnabledPermissions := false
	for _, atom := range state.atoms {
		if atom.Enabled {
			hasEnabledPermissions = true
			break
		}
	}

	if hasEnabledPermissions {
		// Return "manage" to indicate we want to manage permissions
		return "manage"
	} else {
		// No permissions enabled, treat as "yes" for one-time execution
		return "y"
	}
}

// GetEnabledPermissions returns the list of enabled permission atoms
func GetEnabledPermissions(state *PermissionsMenuState) []PermissionAtom {
	var enabled []PermissionAtom
	for _, atom := range state.atoms {
		if atom.Enabled {
			enabled = append(enabled, atom)
		}
	}
	return enabled
}
