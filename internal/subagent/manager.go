package subagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

// SubagentManagerInterface defines the interface for subagent management
// This allows other components to access subagent information without tight coupling
type SubagentManagerInterface interface {
	GetAllSubagents() map[string]*Subagent
	GetSubagent(id string) (*Subagent, bool)
	FindSubagentByName(name string) (*Subagent, bool)
}

const (
	DefaultScanInterval = 30 * time.Second
)

// NewSubagentManager creates a new SubagentManager with default configuration directories
func NewSubagentManager(runner *interp.Runner, logger *zap.Logger) *SubagentManager {
	currentPWD := runner.Vars["PWD"].String()
	manager := &SubagentManager{
		subagents:   make(map[string]*Subagent),
		directories: getDefaultDirectories(runner),
		runner:      runner,
		currentPWD:  currentPWD,
	}
	return manager
}

// getDefaultDirectories returns the default directories to scan for subagent configurations
func getDefaultDirectories(runner *interp.Runner) []string {
	homeDir := runner.Vars["HOME"].String()
	pwd := runner.Vars["PWD"].String()

	var directories []string

	// Project-level configurations (higher priority)
	directories = append(directories, filepath.Join(pwd, ".claude", "agents"))

	// Project-level .roomodes file
	projectRoomodesFile := filepath.Join(pwd, ".roomodes")
	if _, err := os.Stat(projectRoomodesFile); err == nil {
		directories = append(directories, projectRoomodesFile)
	}

	// Scan for Roo custom mode directories (.roo/rules-{modeSlug}/)
	rooBaseDir := filepath.Join(pwd, ".roo")
	if rooModesDirs := getRooModesDirectories(rooBaseDir); len(rooModesDirs) > 0 {
		directories = append(directories, rooModesDirs...)
	}

	// Scan for Roo YAML mode files (.roo/modes/)
	rooModesDir := filepath.Join(pwd, ".roo", "modes")
	if _, err := os.Stat(rooModesDir); err == nil {
		directories = append(directories, rooModesDir)
	}

	// User-level configurations (lower priority)
	directories = append(directories, filepath.Join(homeDir, ".claude", "agents"))

	// User-level .roomodes file
	userRoomodesFile := filepath.Join(homeDir, ".roomodes")
	if _, err := os.Stat(userRoomodesFile); err == nil {
		directories = append(directories, userRoomodesFile)
	}

	// User-level Roo modes
	userRooBaseDir := filepath.Join(homeDir, ".roo")
	if userRooModesDirs := getRooModesDirectories(userRooBaseDir); len(userRooModesDirs) > 0 {
		directories = append(directories, userRooModesDirs...)
	}

	// User-level Roo YAML mode files (.roo/modes/)
	userRooModesDir := filepath.Join(homeDir, ".roo", "modes")
	if _, err := os.Stat(userRooModesDir); err == nil {
		directories = append(directories, userRooModesDir)
	}

	return directories
}

// getRooModesDirectories scans the .roo directory for rules-{modeSlug} subdirectories
func getRooModesDirectories(rooBaseDir string) []string {
	var modesDirs []string

	// Check if .roo directory exists
	if _, err := os.Stat(rooBaseDir); os.IsNotExist(err) {
		return modesDirs
	}

	// Read the .roo directory
	entries, err := os.ReadDir(rooBaseDir)
	if err != nil {
		return modesDirs
	}

	// Look for directories matching the pattern rules-{modeSlug}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "rules-") {
			modeDir := filepath.Join(rooBaseDir, entry.Name())
			modesDirs = append(modesDirs, modeDir)
		}
	}

	return modesDirs
}

// LoadSubagents scans all configured directories and loads subagent configurations
func (m *SubagentManager) LoadSubagents(logger *zap.Logger) error {
	// Update directories if PWD has changed
	if m.hasDirectoryChanged() {
		logger.Debug("Directory changed, updating subagent scan paths",
			zap.String("oldPWD", m.currentPWD),
			zap.String("newPWD", m.runner.Vars["PWD"].String()))
		m.updateDirectories()
	}

	logger.Debug("Loading subagent configurations", zap.Strings("directories", m.directories))

	// Clear existing subagents
	m.subagents = make(map[string]*Subagent)

	for _, dir := range m.directories {
		if err := m.scanDirectory(dir, logger); err != nil {
			logger.Warn("Failed to scan directory for subagents",
				zap.String("directory", dir), zap.Error(err))
			// Continue with other directories even if one fails
		}
	}

	m.lastScan = time.Now()
	logger.Info("Loaded subagents", zap.Int("count", len(m.subagents)))

	return nil
}

// scanDirectory scans a single directory or file for subagent configuration files
func (m *SubagentManager) scanDirectory(path string, logger *zap.Logger) error {
	// Check if path exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		logger.Debug("Subagent path does not exist, skipping", zap.String("path", path))
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	// Handle .roomodes files directly
	if !info.IsDir() && strings.HasSuffix(filepath.Base(path), ".roomodes") {
		return m.scanRoomodesFile(path, logger)
	}

	// Handle directories
	if info.IsDir() {
		// Special handling for Roo rules directories
		if strings.Contains(path, "rules-") {
			return m.scanRooRulesDirectory(path, logger)
		}
		return m.scanRegularDirectory(path, logger)
	}

	// Handle individual files
	return m.scanSingleFile(path, logger)
}

// scanRegularDirectory scans a regular directory for subagent configuration files
func (m *SubagentManager) scanRegularDirectory(dir string, logger *zap.Logger) error {

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn("Error walking directory", zap.String("path", path), zap.Error(err))
			return nil // Continue walking
		}

		// Skip directories (regular scanning doesn't recurse into subdirectories)
		if info.IsDir() {
			return nil
		}

		// Check for supported file extensions
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		logger.Debug("Found potential subagent configuration file", zap.String("path", path))

		// Parse the configuration file
		subagents, err := ParseConfigFile(path)
		if err != nil {
			logger.Warn("Failed to parse subagent configuration",
				zap.String("path", path), zap.Error(err))
			return nil // Continue with other files
		}

		// Add parsed subagents to manager
		for _, subagent := range subagents {
			if err := ValidateSubagent(subagent); err != nil {
				logger.Warn("Invalid subagent configuration",
					zap.String("path", path),
					zap.String("subagent", subagent.ID),
					zap.Error(err))
				continue
			}

			// Check for conflicts (project-level configs override user-level)
			if existing, exists := m.subagents[subagent.ID]; exists {
				logger.Debug("Subagent ID conflict, using higher priority configuration",
					zap.String("id", subagent.ID),
					zap.String("existing", existing.FilePath),
					zap.String("new", subagent.FilePath))
				// Since we scan project directories first, keep the existing one
				continue
			}

			m.subagents[subagent.ID] = subagent
			logger.Debug("Loaded subagent",
				zap.String("id", subagent.ID),
				zap.String("name", subagent.Name),
				zap.String("type", string(subagent.Type)),
				zap.String("path", path))
		}

		return nil
	})
}

// scanRooRulesDirectory scans a Roo rules directory for subagent configurations
func (m *SubagentManager) scanRooRulesDirectory(dir string, logger *zap.Logger) error {
	logger.Debug("Scanning Roo rules directory", zap.String("directory", dir))

	// Parse the Roo rules directory directly
	subagents, err := ParseConfigFile(dir)
	if err != nil {
		logger.Warn("Failed to parse Roo rules directory",
			zap.String("directory", dir), zap.Error(err))
		return nil // Don't fail the entire scan for one directory
	}

	// Add parsed subagents to manager
	for _, subagent := range subagents {
		if err := ValidateSubagent(subagent); err != nil {
			logger.Warn("Invalid subagent configuration",
				zap.String("directory", dir),
				zap.String("subagent", subagent.ID),
				zap.Error(err))
			continue
		}

		// Check for conflicts (project-level configs override user-level)
		if existing, exists := m.subagents[subagent.ID]; exists {
			logger.Debug("Subagent ID conflict, using higher priority configuration",
				zap.String("id", subagent.ID),
				zap.String("existing", existing.FilePath),
				zap.String("new", subagent.FilePath))
			// Since we scan project directories first, keep the existing one
			continue
		}

		m.subagents[subagent.ID] = subagent
		logger.Debug("Loaded Roo rules subagent",
			zap.String("id", subagent.ID),
			zap.String("name", subagent.Name),
			zap.String("type", string(subagent.Type)),
			zap.String("directory", dir))
	}

	return nil
}

// scanSingleFile scans a single file for subagent configurations
func (m *SubagentManager) scanSingleFile(path string, logger *zap.Logger) error {
	// Check for supported file extensions
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" && ext != ".yaml" && ext != ".yml" {
		logger.Debug("Unsupported file extension, skipping", zap.String("path", path))
		return nil
	}

	logger.Debug("Found potential subagent configuration file", zap.String("path", path))

	// Parse the configuration file
	subagents, err := ParseConfigFile(path)
	if err != nil {
		logger.Warn("Failed to parse subagent configuration",
			zap.String("path", path), zap.Error(err))
		return nil // Continue with other files
	}

	// Add parsed subagents to manager
	for _, subagent := range subagents {
		if err := ValidateSubagent(subagent); err != nil {
			logger.Warn("Invalid subagent configuration",
				zap.String("path", path),
				zap.String("subagent", subagent.ID),
				zap.Error(err))
			continue
		}

		// Check for conflicts (project-level configs override user-level)
		if existing, exists := m.subagents[subagent.ID]; exists {
			logger.Debug("Subagent ID conflict, using higher priority configuration",
				zap.String("id", subagent.ID),
				zap.String("existing", existing.FilePath),
				zap.String("new", subagent.FilePath))
			// Since we scan project directories first, keep the existing one
			continue
		}

		m.subagents[subagent.ID] = subagent
		logger.Debug("Loaded subagent",
			zap.String("id", subagent.ID),
			zap.String("name", subagent.Name),
			zap.String("type", string(subagent.Type)),
			zap.String("path", path))
	}

	return nil
}

// scanRoomodesFile scans a .roomodes file for subagent configurations
func (m *SubagentManager) scanRoomodesFile(path string, logger *zap.Logger) error {
	logger.Debug("Scanning .roomodes file", zap.String("path", path))

	// Parse the .roomodes file
	subagents, err := ParseConfigFile(path)
	if err != nil {
		logger.Warn("Failed to parse .roomodes file",
			zap.String("path", path), zap.Error(err))
		return nil // Don't fail the entire scan for one file
	}

	// Add parsed subagents to manager
	for _, subagent := range subagents {
		if err := ValidateSubagent(subagent); err != nil {
			logger.Warn("Invalid subagent configuration",
				zap.String("path", path),
				zap.String("subagent", subagent.ID),
				zap.Error(err))
			continue
		}

		// Check for conflicts (project-level configs override user-level)
		if existing, exists := m.subagents[subagent.ID]; exists {
			logger.Debug("Subagent ID conflict, using higher priority configuration",
				zap.String("id", subagent.ID),
				zap.String("existing", existing.FilePath),
				zap.String("new", subagent.FilePath))
			// Since we scan project directories first, keep the existing one
			continue
		}

		m.subagents[subagent.ID] = subagent
		logger.Debug("Loaded subagent from .roomodes file",
			zap.String("id", subagent.ID),
			zap.String("name", subagent.Name),
			zap.String("type", string(subagent.Type)),
			zap.String("path", path))
	}

	return nil
}

// GetSubagent retrieves a subagent by ID
func (m *SubagentManager) GetSubagent(id string) (*Subagent, bool) {
	subagent, exists := m.subagents[id]
	return subagent, exists
}

// GetAllSubagents returns all loaded subagents
func (m *SubagentManager) GetAllSubagents() map[string]*Subagent {
	// Return a copy to prevent external modification
	result := make(map[string]*Subagent, len(m.subagents))
	for id, subagent := range m.subagents {
		result[id] = subagent
	}
	return result
}

// FindSubagentByName searches for a subagent by name or partial name match
func (m *SubagentManager) FindSubagentByName(name string) (*Subagent, bool) {
	name = strings.ToLower(name)

	// First try exact ID match
	if subagent, exists := m.subagents[name]; exists {
		return subagent, true
	}

	// Then try exact name match (case-insensitive)
	for _, subagent := range m.subagents {
		if strings.ToLower(subagent.Name) == name {
			return subagent, true
		}
	}

	// Finally try partial name match
	for _, subagent := range m.subagents {
		if strings.Contains(strings.ToLower(subagent.Name), name) ||
			strings.Contains(strings.ToLower(subagent.ID), name) {
			return subagent, true
		}
	}

	return nil, false
}

// ShouldReload checks if configurations should be reloaded based on file modifications or directory changes
func (m *SubagentManager) ShouldReload() bool {
	// Check if directory has changed
	currentPWD := m.runner.Vars["PWD"].String()
	if currentPWD != m.currentPWD {
		return true
	}

	// Simple time-based check for now
	return time.Since(m.lastScan) > DefaultScanInterval
}

// hasDirectoryChanged checks if the current working directory has changed
func (m *SubagentManager) hasDirectoryChanged() bool {
	currentPWD := m.runner.Vars["PWD"].String()
	return currentPWD != m.currentPWD
}

// updateDirectories refreshes the directory list based on the current PWD
func (m *SubagentManager) updateDirectories() {
	currentPWD := m.runner.Vars["PWD"].String()
	m.currentPWD = currentPWD
	m.directories = getDefaultDirectories(m.runner)
}

// Reload reloads all subagent configurations
func (m *SubagentManager) Reload(logger *zap.Logger) error {
	logger.Info("Reloading subagent configurations")
	return m.LoadSubagents(logger)
}

// GetSubagentsSummary returns a formatted summary of all loaded subagents
func (m *SubagentManager) GetSubagentsSummary() string {
	if len(m.subagents) == 0 {
		return "No subagents configured."
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Loaded %d subagent(s):\n", len(m.subagents)))

	for id, subagent := range m.subagents {
		summary.WriteString(fmt.Sprintf("  • %s (%s): %s\n",
			subagent.Name, id, subagent.Description))
		summary.WriteString(fmt.Sprintf("    Type: %s, Tools: %v\n",
			subagent.Type, subagent.AllowedTools))
		if subagent.FileRegex != "" {
			summary.WriteString(fmt.Sprintf("    File Access: %s\n", subagent.FileRegex))
		}
		summary.WriteString("\n")
	}

	return summary.String()
}
