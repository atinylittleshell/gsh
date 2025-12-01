package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// homeDir returns the user's home directory, using os.UserHomeDir() for portability
// across different platforms (including Windows where HOME is not typically set).
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to HOME env var if os.UserHomeDir() fails
		return os.Getenv("HOME")
	}
	return home
}

var (
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	// Full-screen box styles (matching ctrl-r history search)
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red for errors
)

// sessionConfigOverrides stores config values set via the UI that should override shell variables
// This prevents user's bash scripts from resetting values we just set
var sessionConfigOverrides = make(map[string]string)

// GetSessionOverride returns a session config override if one exists
func GetSessionOverride(key string) (string, bool) {
	val, ok := sessionConfigOverrides[key]
	return val, ok
}

type model struct {
	runner         *interp.Runner
	list           list.Model
	submenuList    list.Model
	selectionList  list.Model
	state          state
	items          []settingItem
	textInput      textinput.Model
	activeSetting  *settingItem
	activeSubmenu  *menuItem
	quitting       bool
	width          int
	height         int
	errorMsg       string // Temporary error message to display
}

type state int

const (
	stateList state = iota
	stateSubmenu
	stateEditing
	stateSelection
)

// menuItem represents a top-level menu entry (may have submenu)
type menuItem struct {
	title       string
	description string
	submenu     []settingItem // nil if this is a direct setting
	setting     *settingItem  // non-nil if this is a direct setting (no submenu)
}

func (m menuItem) Title() string       { return m.title }
func (m menuItem) Description() string { return m.description }
func (m menuItem) FilterValue() string { return m.title }

type settingItem struct {
	title       string
	description string
	envVar      string
	itemType    settingType
	options     []string // For list type
}

type settingType int

const (
	typeText settingType = iota
	typeList
	typeToggle
)

func (s settingItem) Title() string       { return s.title }
func (s settingItem) Description() string { return s.description }
func (s settingItem) FilterValue() string { return s.title }

// simpleItem implements list.Item
type simpleItem string

func (s simpleItem) Title() string       { return string(s) }
func (s simpleItem) Description() string { return "" }
func (s simpleItem) FilterValue() string { return string(s) }

func initialModel(runner *interp.Runner) model {
	// Define submenu items for slow model (chat/agent)
	slowModelSettings := []settingItem{
		{
			title:       "Provider",
			description: "LLM provider to use",
			envVar:      "GSH_SLOW_MODEL_PROVIDER",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		{
			title:       "API Key",
			description: "API key for the provider",
			envVar:      "GSH_SLOW_MODEL_API_KEY",
			itemType:    typeText,
		},
		{
			title:       "Model ID",
			description: "Model identifier (e.g., qwen2.5:32b)",
			envVar:      "GSH_SLOW_MODEL_ID",
			itemType:    typeText,
		},
		{
			title:       "Base URL",
			description: "API endpoint URL (optional override)",
			envVar:      "GSH_SLOW_MODEL_BASE_URL",
			itemType:    typeText,
		},
	}

	// Define submenu items for fast model (completion/suggestions)
	fastModelSettings := []settingItem{
		{
			title:       "Provider",
			description: "LLM provider to use",
			envVar:      "GSH_FAST_MODEL_PROVIDER",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		{
			title:       "API Key",
			description: "API key for the provider",
			envVar:      "GSH_FAST_MODEL_API_KEY",
			itemType:    typeText,
		},
		{
			title:       "Model ID",
			description: "Model identifier (e.g., qwen2.5)",
			envVar:      "GSH_FAST_MODEL_ID",
			itemType:    typeText,
		},
		{
			title:       "Base URL",
			description: "API endpoint URL (optional override)",
			envVar:      "GSH_FAST_MODEL_BASE_URL",
			itemType:    typeText,
		},
	}

	// Direct settings (no submenu)
	assistantHeightSetting := settingItem{
		title:       "Assistant Height",
		description: "Height of the bottom assistant box",
		envVar:      "GSH_ASSISTANT_HEIGHT",
		itemType:    typeText,
	}
	safetyChecksSetting := settingItem{
		title:       "Safety Checks",
		description: "Enable/Disable approved command checks",
		envVar:      "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX",
		itemType:    typeToggle,
	}

	// Top-level menu items
	items := []list.Item{
		menuItem{
			title:       "Configure Slow Model",
			description: "Chat and agent operations",
			submenu:     slowModelSettings,
		},
		menuItem{
			title:       "Configure Fast Model",
			description: "Auto-completion and suggestions",
			submenu:     fastModelSettings,
		},
		menuItem{
			title:       "Assistant Height",
			description: "Height of the bottom assistant box",
			setting:     &assistantHeightSetting,
		},
		menuItem{
			title:       "Safety Checks",
			description: "Enable/Disable approved command checks",
			setting:     &safetyChecksSetting,
		},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle.Foreground(lipgloss.Color("240"))

	l := list.New(items, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)

	subL := list.New([]list.Item{}, delegate, 0, 0)
	subL.SetShowStatusBar(false)
	subL.SetFilteringEnabled(false)
	subL.SetShowTitle(false)
	subL.SetShowHelp(false)

	selL := list.New([]list.Item{}, delegate, 0, 0)
	selL.SetShowStatusBar(false)
	selL.SetFilteringEnabled(false)
	selL.SetShowTitle(false)
	selL.SetShowHelp(false)

	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ti.Focus()

	return model{
		runner:        runner,
		list:          l,
		submenuList:   subL,
		selectionList: selL,
		state:         stateList,
		items:         []settingItem{},
		textInput:     ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
		m.submenuList.SetWidth(msg.Width)
		m.submenuList.SetHeight(msg.Height)
		m.selectionList.SetWidth(msg.Width)
		m.selectionList.SetHeight(msg.Height)

	case tea.KeyMsg:
		// Clear any previous error message on new key press
		m.errorMsg = ""

		// Handle text editing state
		if m.state == stateEditing {
			// Check for quit keys first, before delegating to text input
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			}
			switch msg.Type {
			case tea.KeyEsc:
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			case tea.KeyEnter:
				newValue := m.textInput.Value()
				if err := saveConfig(m.activeSetting.envVar, newValue, m.runner); err != nil {
					m.errorMsg = fmt.Sprintf("Failed to save %s: %v", m.activeSetting.envVar, err)
					return m, nil
				}
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// Handle selection list state
		if m.state == stateSelection {
			// Check for quit keys first, before delegating to selection list
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			}
			switch msg.Type {
			case tea.KeyEsc:
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			case tea.KeyEnter:
				if i, ok := m.selectionList.SelectedItem().(simpleItem); ok {
					newValue := string(i)
					if err := saveConfig(m.activeSetting.envVar, newValue, m.runner); err != nil {
						m.errorMsg = fmt.Sprintf("Failed to save %s: %v", m.activeSetting.envVar, err)
						return m, nil
					}
					if m.activeSubmenu != nil {
						m.state = stateSubmenu
					} else {
						m.state = stateList
					}
					return m, nil
				}
			}
			m.selectionList, cmd = m.selectionList.Update(msg)
			return m, cmd
		}

		// Handle submenu state
		if m.state == stateSubmenu {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.activeSubmenu = nil
				m.state = stateList
				return m, nil
			case "enter":
				if i, ok := m.submenuList.SelectedItem().(settingItem); ok {
					m.activeSetting = &i
					return m, m.handleSettingAction(&i)
				}
			}
			m.submenuList, cmd = m.submenuList.Update(msg)
			return m, cmd
		}

		// Handle main list state
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(menuItem); ok {
				// If this menu item has a submenu, navigate to it
				if item.submenu != nil {
					m.activeSubmenu = &item
					subItems := make([]list.Item, len(item.submenu))
					for idx, s := range item.submenu {
						subItems[idx] = s
					}
					m.submenuList.SetItems(subItems)
					m.submenuList.Title = item.title
					m.state = stateSubmenu
					return m, nil
				}
				// If this is a direct setting, handle it
				if item.setting != nil {
					m.activeSetting = item.setting
					return m, m.handleSettingAction(item.setting)
				}
			}
		}
	}

	if m.state == stateList {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleSettingAction processes the action for a setting item
func (m *model) handleSettingAction(s *settingItem) tea.Cmd {
	if s.itemType == typeToggle {
		curr := getEnv(m.runner, s.envVar)
		var newVal string
		if s.envVar == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
			if strings.Contains(curr, `".*"`) || strings.Contains(curr, `".+"`) {
				newVal = "[]"
			} else {
				newVal = `[".*"]`
			}
		} else {
			if curr == "true" {
				newVal = "false"
			} else {
				newVal = "true"
			}
		}
		if err := saveConfig(s.envVar, newVal, m.runner); err != nil {
			m.errorMsg = fmt.Sprintf("Failed to save %s: %v", s.envVar, err)
		}
		return nil
	}

	if s.itemType == typeList {
		items := make([]list.Item, len(s.options))
		for idx, opt := range s.options {
			items[idx] = simpleItem(opt)
		}
		m.selectionList.SetItems(items)
		m.selectionList.Title = "Select " + s.title
		m.state = stateSelection
		return nil
	}

	// typeText
	m.textInput.SetValue(getEnv(m.runner, s.envVar))
	m.state = stateEditing
	return nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Calculate available dimensions for content
	// Leave room for header (1), footer (1), and border (2)
	availableHeight := m.height - 4
	if availableHeight < 5 {
		availableHeight = 5
	}
	availableWidth := m.width - 4
	if availableWidth < 20 {
		availableWidth = 20
	}

	var content strings.Builder
	var title string
	var helpText string

	switch m.state {
	case stateEditing:
		title = fmt.Sprintf("Edit %s", m.activeSetting.title)
		helpText = "Enter: Save | Esc: Cancel | q: Quit"
		content.WriteString("\n" + m.textInput.View() + "\n")
	case stateSelection:
		title = "Select " + m.activeSetting.title
		helpText = "↑/↓: Navigate | Enter: Select | Esc: Back | q: Quit"
		content.WriteString(m.selectionList.View())
	case stateSubmenu:
		title = m.activeSubmenu.title
		helpText = "↑/↓: Navigate | Enter: Edit | Esc: Back | q: Quit"
		// Update submenu descriptions with current values
		items := m.submenuList.Items()
		for i, item := range items {
			if s, ok := item.(settingItem); ok {
				val := getEnv(m.runner, s.envVar)
				if val == "" {
					val = "(not set)"
				}
				s.description = fmt.Sprintf("Current: %s", val)
				items[i] = s
			}
		}
		m.submenuList.SetItems(items)
		content.WriteString(m.submenuList.View())
	default:
		title = "Config Menu"
		helpText = "↑/↓: Navigate | Enter: Select | q: Quit"
		// Update main menu descriptions with current values for direct settings
		items := m.list.Items()
		for i, item := range items {
			if mi, ok := item.(menuItem); ok {
				if mi.setting != nil {
					val := getEnv(m.runner, mi.setting.envVar)
					if mi.setting.envVar == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
						if strings.Contains(val, `".*"`) || strings.Contains(val, `".+"`) {
							val = "Disabled (All commands allowed)"
						} else {
							val = "Enabled"
						}
					}
					if val == "" {
						val = "(not set)"
					}
					mi.description = fmt.Sprintf("Current: %s", val)
					items[i] = mi
				}
			}
		}
		m.list.SetItems(items)
		content.WriteString(m.list.View())
	}

	// Build the full-screen box content
	var boxContent strings.Builder

	// Header with centered title
	titlePadding := (availableWidth - len(title)) / 2
	if titlePadding < 0 {
		titlePadding = 0
	}
	centeredTitle := strings.Repeat(" ", titlePadding) + title
	boxContent.WriteString(headerStyle.Render(centeredTitle) + "\n")

	// Content area - truncate to available height
	contentLines := strings.Split(content.String(), "\n")
	contentHeight := availableHeight - 2 // Leave room for header and footer
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	boxContent.WriteString(strings.Join(contentLines, "\n"))

	// Pad to fill available height
	currentLines := len(contentLines) + 1 // +1 for header
	for i := currentLines; i < availableHeight-1; i++ {
		boxContent.WriteString("\n")
	}

	// Footer with help text and error message
	footerContent := helpStyle.Render(helpText)
	if m.errorMsg != "" {
		footerContent = errorStyle.Render(m.errorMsg) + "\n" + footerContent
	}
	boxContent.WriteString("\n" + footerContent)

	// Render in a box with rounded border (matching ctrl-r style)
	boxStyle := lipgloss.NewStyle().
		Width(availableWidth).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return boxStyle.Render(boxContent.String())
}

// RunConfigUI launches the interactive configuration UI
func RunConfigUI(runner *interp.Runner) error {
	p := tea.NewProgram(initialModel(runner), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func getEnv(runner *interp.Runner, key string) string {
	// Safety Checks uses a session-only flag GSH_SAFETY_CHECKS_DISABLED
	if key == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
		if runner.Vars["GSH_SAFETY_CHECKS_DISABLED"].String() == "true" {
			return `[".*"]` // Disabled for this session
		}
		return "[]" // Enabled (default)
	}

	if v, ok := runner.Vars[key]; ok {
		return v.String()
	}
	return ""
}

func saveConfig(key, value string, runner *interp.Runner) error {
	// Handle Safety Checks specially - only affects current session, not persisted
	// Uses GSH_SAFETY_CHECKS_DISABLED flag which is checked in GetApprovedBashCommandRegex
	if key == "GSH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
		// value is either '[".*"]' (disabled) or '[]' (enabled)
		if strings.Contains(value, `".*"`) || strings.Contains(value, `".+"`) {
			// Disable safety checks for this session only
			runner.Vars["GSH_SAFETY_CHECKS_DISABLED"] = expand.Variable{
				Exported: true,
				Kind:     expand.String,
				Str:      "true",
			}
		} else {
			// Enable safety checks - remove the session flag
			delete(runner.Vars, "GSH_SAFETY_CHECKS_DISABLED")
		}
		// Don't persist this setting - it only affects the current session
		return nil
	}

	// For other settings, update current session
	runner.Vars[key] = expand.Variable{
		Exported: true,
		Kind:     expand.String,
		Str:      value,
	}

	// Store in session overrides to prevent bash scripts from resetting the value
	sessionConfigOverrides[key] = value

	// Sync to environment so changes take effect immediately
	environment.SyncVariableToEnv(runner, key)

	// Persist to file for future sessions
	configPath := filepath.Join(homeDir(), ".gsh_config_ui")
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Write export command
	safeValue := strings.ReplaceAll(value, "'", "'\\''")
	line := fmt.Sprintf("export %s='%s'\n", key, safeValue)

	if _, err := f.WriteString(line); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// Ensure sourced in .gshrc
	gshrcPath := filepath.Join(homeDir(), ".gshrc")
	sourceSnippet := "\n# Source UI configuration\n[ -f ~/.gsh_config_ui ] && source ~/.gsh_config_ui\n"

	content, err := os.ReadFile(gshrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", gshrcPath, err)
	}

	// Check if already contains the source snippet
	if err == nil && strings.Contains(string(content), ".gsh_config_ui") {
		return nil // Already configured
	}

	// Need to add the source snippet - either append to existing or create new
	var f2 *os.File
	if os.IsNotExist(err) {
		f2, err = os.Create(gshrcPath)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", gshrcPath, err)
		}
	} else {
		f2, err = os.OpenFile(gshrcPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s for appending: %w", gshrcPath, err)
		}
	}

	var writeErr error
	defer func() {
		if closeErr := f2.Close(); closeErr != nil && writeErr == nil {
			writeErr = fmt.Errorf("failed to close %s: %w", gshrcPath, closeErr)
		}
	}()

	if _, err := f2.WriteString(sourceSnippet); err != nil {
		writeErr = fmt.Errorf("failed to write to %s: %w", gshrcPath, err)
		return writeErr
	}

	return writeErr
}
