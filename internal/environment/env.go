package environment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/samber/lo"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

var (
	authorizedCommandsCache      []string
	authorizedCommandsCacheMutex sync.RWMutex
	lastFileModTime              time.Time
	configDir                    = filepath.Join(os.Getenv("HOME"), ".config", "gsh")
	authorizedCommandsFile       = filepath.Join(configDir, "authorized_commands")
)

// Helper functions for testing
func GetConfigDirForTesting() string {
	return configDir
}

func GetAuthorizedCommandsFileForTesting() string {
	return authorizedCommandsFile
}

func SetConfigDirForTesting(dir string) {
	configDir = dir
}

func SetAuthorizedCommandsFileForTesting(file string) {
	authorizedCommandsFile = file
}

const (
	DEFAULT_PROMPT       = "gsh> "
	DEFAULT_AGENT_PROMPT = "🤖> "
)

func GetHistoryContextLimit(runner *interp.Runner, logger *zap.Logger) int {
	historyContextLimit, err := strconv.ParseInt(
		runner.Vars["GSH_PAST_COMMANDS_CONTEXT_LIMIT"].String(), 10, 32)
	if err != nil {
		logger.Debug("error parsing GSH_PAST_COMMANDS_CONTEXT_LIMIT", zap.Error(err))
		historyContextLimit = 30
	}
	return int(historyContextLimit)
}

func GetLogLevel(runner *interp.Runner) zap.AtomicLevel {
	logLevel, err := zap.ParseAtomicLevel(runner.Vars["GSH_LOG_LEVEL"].String())
	if err != nil {
		logLevel = zap.NewAtomicLevel()
	}
	return logLevel
}

func ShouldCleanLogFile(runner *interp.Runner) bool {
	cleanLogFile := strings.ToLower(runner.Vars["GSH_CLEAN_LOG_FILE"].String())
	return cleanLogFile == "1" || cleanLogFile == "true"
}

// GetDefaultToYes returns whether prompts should default to "yes" when Enter is pressed.
// When true, prompts display [Y/n] and Enter confirms. When false, prompts display [y/N] and Enter denies.
func GetDefaultToYes(runner *interp.Runner) bool {
	defaultToYes := strings.ToLower(runner.Vars["GSH_DEFAULT_TO_YES"].String())
	return defaultToYes == "1" || defaultToYes == "true"
}

func GetPwd(runner *interp.Runner) string {
	return runner.Vars["PWD"].String()
}

func GetUser(runner *interp.Runner) string {
	u := runner.Vars["USER"].String()
	if u == "" {
		if usr, err := user.Current(); err == nil {
			return usr.Username
		}
		return "user"
	}
	return u
}

func GetPrompt(runner *interp.Runner, logger *zap.Logger) string {
	promptUpdater := runner.Funcs["GSH_UPDATE_PROMPT"]
	if promptUpdater != nil {
		err := runner.Run(context.Background(), promptUpdater)
		if err != nil {
			logger.Warn("error updating prompt", zap.Error(err))
		}
	}

	buildVersion := runner.Vars["GSH_BUILD_VERSION"].String()
	if buildVersion == "dev" {
		buildVersion = "[dev] "
	} else {
		buildVersion = ""
	}

	prompt := buildVersion + runner.Vars["GSH_PROMPT"].String()
	if prompt != "" {
		return prompt
	}
	return DEFAULT_PROMPT
}

// GetAgentPrompt returns the prompt to use when the agent displays commands
// If GSH_APROMPT is set, it uses that; otherwise uses DEFAULT_AGENT_PROMPT
// to differentiate agent commands from user commands
func GetAgentPrompt(runner *interp.Runner) string {
	agentPrompt := runner.Vars["GSH_APROMPT"].String()
	if agentPrompt != "" {
		return agentPrompt
	}
	return DEFAULT_AGENT_PROMPT
}

// GetAgentName returns the name of the active subagent or defaults to "gsh"
func GetAgentName(runner *interp.Runner) string {
	agentName := runner.Vars["GSH_AGENT_NAME"].String()
	if agentName != "" {
		return agentName
	}
	return "gsh"
}

func GetAgentContextWindowTokens(runner *interp.Runner, logger *zap.Logger) int {
	agentContextWindow, err := strconv.ParseInt(
		runner.Vars["GSH_AGENT_CONTEXT_WINDOW_TOKENS"].String(), 10, 32)
	if err != nil {
		logger.Debug("error parsing GSH_AGENT_CONTEXT_WINDOW_TOKENS", zap.Error(err))
		agentContextWindow = 32768
	}
	return int(agentContextWindow)
}

func GetAssistantHeight(runner *interp.Runner, logger *zap.Logger) int {
	// Check for session override first (set via config UI, immune to bash script resets)
	rawValue := runner.Vars["GSH_ASSISTANT_HEIGHT"].String()
	if override, ok := getSessionConfigOverride("GSH_ASSISTANT_HEIGHT"); ok {
		rawValue = override
	}

	assistantHeight, err := strconv.ParseInt(rawValue, 10, 32)
	if err != nil {
		logger.Debug("error parsing GSH_ASSISTANT_HEIGHT", zap.Error(err))
		assistantHeight = 3
	}

	if assistantHeight < 0 {
		logger.Debug("GSH_ASSISTANT_HEIGHT is negative, clamping to 0",
			zap.Int64("assistantHeight", assistantHeight))
		assistantHeight = 0
	}
	return int(assistantHeight)
}

// sessionConfigOverrideGetter is set by the config package to allow cross-package access
var sessionConfigOverrideGetter func(key string) (string, bool)

// SetSessionConfigOverrideGetter sets the function to get session config overrides
func SetSessionConfigOverrideGetter(getter func(key string) (string, bool)) {
	sessionConfigOverrideGetter = getter
}

// getSessionConfigOverride gets a session config override if one exists
func getSessionConfigOverride(key string) (string, bool) {
	if sessionConfigOverrideGetter != nil {
		return sessionConfigOverrideGetter(key)
	}
	return "", false
}

func getContextTypes(runner *interp.Runner, key string) []string {
	contextTypes := strings.ToLower(runner.Vars[key].String())
	return lo.Map(strings.Split(contextTypes, ","), func(s string, _ int) string {
		return strings.TrimSpace(s)
	})
}

func GetContextTypesForAgent(runner *interp.Runner, logger *zap.Logger) []string {
	return getContextTypes(runner, "GSH_CONTEXT_TYPES_FOR_AGENT")
}

func GetContextTypesForPredictionWithPrefix(runner *interp.Runner, logger *zap.Logger) []string {
	return getContextTypes(runner, "GSH_CONTEXT_TYPES_FOR_PREDICTION_WITH_PREFIX")
}

func GetContextTypesForPredictionWithoutPrefix(runner *interp.Runner, logger *zap.Logger) []string {
	return getContextTypes(runner, "GSH_CONTEXT_TYPES_FOR_PREDICTION_WITHOUT_PREFIX")
}

func GetContextTypesForExplanation(runner *interp.Runner, logger *zap.Logger) []string {
	return getContextTypes(runner, "GSH_CONTEXT_TYPES_FOR_EXPLANATION")
}

func GetContextNumHistoryConcise(runner *interp.Runner, logger *zap.Logger) int {
	numHistoryConcise, err := strconv.ParseInt(
		runner.Vars["GSH_CONTEXT_NUM_HISTORY_CONCISE"].String(), 10, 32)
	if err != nil {
		logger.Debug("error parsing GSH_CONTEXT_NUM_HISTORY_CONCISE", zap.Error(err))
		numHistoryConcise = 30
	}
	return int(numHistoryConcise)
}

func GetContextNumHistoryVerbose(runner *interp.Runner, logger *zap.Logger) int {
	numHistoryVerbose, err := strconv.ParseInt(
		runner.Vars["GSH_CONTEXT_NUM_HISTORY_VERBOSE"].String(), 10, 32)
	if err != nil {
		logger.Debug("error parsing GSH_CONTEXT_NUM_HISTORY_VERBOSE", zap.Error(err))
		numHistoryVerbose = 30
	}
	return int(numHistoryVerbose)
}

// GetIdleSummaryTimeout returns the idle summary timeout in seconds.
// Returns 0 if disabled, otherwise defaults to 60 seconds.
func GetIdleSummaryTimeout(runner *interp.Runner, logger *zap.Logger) int {
	timeoutStr := runner.Vars["GSH_IDLE_SUMMARY_TIMEOUT_SECONDS"].String()
	if timeoutStr == "" {
		return 60 // Default to 60 seconds
	}

	timeout, err := strconv.ParseInt(timeoutStr, 10, 32)
	if err != nil {
		logger.Debug("error parsing GSH_IDLE_SUMMARY_TIMEOUT_SECONDS", zap.Error(err))
		return 60 // Default to 60 seconds on parse error
	}

	return int(timeout)
}

func GetHomeDir(runner *interp.Runner) string {
	return runner.Vars["HOME"].String()
}

func GetAgentMacros(runner *interp.Runner, logger *zap.Logger) map[string]string {
	macrosStr := runner.Vars["GSH_AGENT_MACROS"].String()
	if macrosStr == "" {
		return map[string]string{}
	}

	var macros map[string]string
	err := json.Unmarshal([]byte(macrosStr), &macros)
	if err != nil {
		logger.Debug("error parsing GSH_AGENT_MACROS", zap.Error(err))
		return map[string]string{}
	}
	return macros
}

// AppendToAuthorizedCommands appends a command regex to the authorized_commands file
func AppendToAuthorizedCommands(commandRegex string) error {
	// Create config directory if it doesn't exist with secure permissions (owner only)
	// Explicitly set to 0700 for secure directory permissions
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Ensure directory has correct permissions even if it already existed
	if err := os.Chmod(configDir, 0700); err != nil {
		return fmt.Errorf("failed to set config directory permissions: %w", err)
	}

	// Check if file exists to determine if we need to set permissions
	fileExists := true
	if _, err := os.Stat(authorizedCommandsFile); os.IsNotExist(err) {
		fileExists = false
	}

	// Open file in append mode, create if doesn't exist with secure permissions (owner only - 0600)
	file, err := os.OpenFile(authorizedCommandsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open authorized_commands file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// If file already existed, ensure it has secure permissions
	if fileExists {
		if err := os.Chmod(authorizedCommandsFile, 0600); err != nil {
			return fmt.Errorf("failed to set authorized_commands file permissions: %w", err)
		}
	}

	// Write the regex pattern followed by newline
	if _, err := file.WriteString(commandRegex + "\n"); err != nil {
		return fmt.Errorf("failed to write to authorized_commands file: %w", err)
	}

	return nil
}

// LoadAuthorizedCommandsFromFile loads authorized command regex patterns from file
func LoadAuthorizedCommandsFromFile() ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(authorizedCommandsFile); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read file contents
	content, err := os.ReadFile(authorizedCommandsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read authorized_commands file: %w", err)
	}

	// Split by newlines and filter out empty lines
	lines := strings.Split(string(content), "\n")
	var patterns []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			patterns = append(patterns, trimmed)
		}
	}

	// Ensure we return an empty slice rather than nil
	if patterns == nil {
		patterns = []string{}
	}

	return patterns, nil
}

// WriteAuthorizedCommandsToFile writes a list of regex patterns to the authorized_commands file
// This replaces the entire file content and deduplicates entries
func WriteAuthorizedCommandsToFile(patterns []string) error {
	// Create config directory if it doesn't exist with secure permissions (owner only)
	// Explicitly set to 0700 for secure directory permissions
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Ensure directory has correct permissions even if it already existed
	if err := os.Chmod(configDir, 0700); err != nil {
		return fmt.Errorf("failed to set config directory permissions: %w", err)
	}

	// Deduplicate patterns while preserving order
	seen := make(map[string]bool)
	var uniquePatterns []string
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			uniquePatterns = append(uniquePatterns, trimmed)
		}
	}

	// Create file with secure permissions (owner only - 0600)
	file, err := os.OpenFile(authorizedCommandsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open authorized_commands file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Write all patterns
	for _, pattern := range uniquePatterns {
		if _, err := file.WriteString(pattern + "\n"); err != nil {
			return fmt.Errorf("failed to write to authorized_commands file: %w", err)
		}
	}

	return nil
}

// IsCommandAuthorized checks if a command matches any of the authorized patterns
func IsCommandAuthorized(command string) (bool, error) {
	patterns, err := LoadAuthorizedCommandsFromFile()
	if err != nil {
		return false, err
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, command)
		if err != nil {
			// Skip invalid regex patterns
			continue
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}

// IsCommandPatternAuthorized checks if a specific command pattern already exists in the authorized_commands file
// This is used for pre-selecting permissions in the menu - only exact pattern matches should be pre-selected
func IsCommandPatternAuthorized(commandPattern string) (bool, error) {
	patterns, err := LoadAuthorizedCommandsFromFile()
	if err != nil {
		return false, err
	}

	for _, pattern := range patterns {
		if pattern == commandPattern {
			return true, nil
		}
	}

	return false, nil
}

// getFileModTime returns the modification time of the authorized_commands file
func getFileModTime() (time.Time, error) {
	info, err := os.Stat(authorizedCommandsFile)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// shouldReloadAuthorizedCommands checks if the file has been modified since last read
func shouldReloadAuthorizedCommands() bool {
	currentModTime, err := getFileModTime()
	if err != nil {
		return false
	}
	return currentModTime.After(lastFileModTime)
}

// GetApprovedBashCommandRegex returns approved bash command regex patterns from both env var and file
// filterDangerousPatterns removes overly broad patterns that could bypass file-based security
func filterDangerousPatterns(patterns []string, logger *zap.Logger) []string {
	dangerousPatterns := []string{
		".*",          // Matches everything
		"^.*$",        // Matches everything with anchors
		".+",          // Matches any non-empty string
		"^.+$",        // Matches any non-empty string with anchors
		"[\\s\\S]*",   // Matches everything including newlines
		"^[\\s\\S]*$", // Matches everything including newlines with anchors
	}

	var filtered []string
	for _, pattern := range patterns {
		isDangerous := false
		for _, dangerous := range dangerousPatterns {
			if pattern == dangerous {
				isDangerous = true
				logger.Warn("Filtered out dangerous environment pattern that could bypass file-based security",
					zap.String("pattern", pattern))
				break
			}
		}
		if !isDangerous {
			filtered = append(filtered, pattern)
		}
	}

	// Ensure we return an empty slice rather than nil
	if filtered == nil {
		filtered = []string{}
	}

	return filtered
}

func GetApprovedBashCommandRegex(runner *interp.Runner, logger *zap.Logger) []string {
	// Check if safety checks are disabled for this session (set via config UI)
	// This is a session-only setting that is not persisted
	if runner.Vars["GSH_SAFETY_CHECKS_DISABLED"].String() == "true" {
		logger.Debug("safety checks disabled for this session")
		return []string{".*"}
	}

	// Get patterns from environment variable
	regexStr := runner.Vars["GSH_AGENT_APPROVED_BASH_COMMAND_REGEX"].String()
	logger.Debug("GSH_AGENT_APPROVED_BASH_COMMAND_REGEX value", zap.String("value", regexStr))
	var envPatterns []string
	if regexStr != "" {
		err := json.Unmarshal([]byte(regexStr), &envPatterns)
		if err != nil {
			logger.Debug("error parsing GSH_AGENT_APPROVED_BASH_COMMAND_REGEX", zap.Error(err))
			envPatterns = []string{}
		} else {
			logger.Debug("successfully parsed environment patterns", zap.Any("patterns", envPatterns))
		}
	} else {
		logger.Debug("GSH_AGENT_APPROVED_BASH_COMMAND_REGEX is empty")
		envPatterns = []string{}
	}

	// Check if we should reload from file
	authorizedCommandsCacheMutex.RLock()
	shouldReload := shouldReloadAuthorizedCommands()
	cachedPatterns := make([]string, len(authorizedCommandsCache))
	copy(cachedPatterns, authorizedCommandsCache)
	authorizedCommandsCacheMutex.RUnlock()

	var filePatterns []string
	if shouldReload {
		// Reload from file
		var err error
		filePatterns, err = LoadAuthorizedCommandsFromFile()
		if err != nil {
			logger.Debug("error loading authorized commands from file", zap.Error(err))
			filePatterns = []string{}
		} else {
			// Update cache
			authorizedCommandsCacheMutex.Lock()
			authorizedCommandsCache = make([]string, len(filePatterns))
			copy(authorizedCommandsCache, filePatterns)
			if modTime, err := getFileModTime(); err == nil {
				lastFileModTime = modTime
			}
			authorizedCommandsCacheMutex.Unlock()
		}
	} else {
		// Use cached patterns
		filePatterns = cachedPatterns
	}

	// Filter out overly broad environment patterns that could bypass file-based security
	filteredEnvPatterns := filterDangerousPatterns(envPatterns, logger)

	// Combine filtered environment and file patterns
	allPatterns := append(filteredEnvPatterns, filePatterns...)

	// Ensure we return an empty slice rather than nil
	if allPatterns == nil {
		allPatterns = []string{}
	}

	return allPatterns
}

// ResetCacheForTesting resets the authorized commands cache for testing
func ResetCacheForTesting() {
	authorizedCommandsCacheMutex.Lock()
	authorizedCommandsCache = []string{}
	lastFileModTime = time.Time{}
	authorizedCommandsCacheMutex.Unlock()
}
