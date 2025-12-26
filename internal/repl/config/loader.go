// Package config provides configuration management for the gsh REPL.
package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"go.uber.org/zap"
)

// Loader handles loading and parsing of .gshrc.gsh configuration files.
type Loader struct {
	logger *zap.Logger
}

// NewLoader creates a new configuration loader.
func NewLoader(logger *zap.Logger) *Loader {
	return &Loader{
		logger: logger,
	}
}

// LoadResult contains the result of loading a configuration file.
type LoadResult struct {
	Config      *Config
	Interpreter *interpreter.Interpreter
	Errors      []error
}

// LoadFromFile loads configuration from a .gshrc.gsh file.
// Returns the configuration and any non-fatal errors encountered.
// If the file doesn't exist, returns default configuration with no error.
func (l *Loader) LoadFromFile(path string) (*LoadResult, error) {
	result := &LoadResult{
		Config: DefaultConfig(),
		Errors: []error{},
	}

	// Check if file exists
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return defaults
			return result, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return l.LoadFromString(string(content))
}

// LoadFromString loads configuration from a gsh script string.
func (l *Loader) LoadFromString(source string) (*LoadResult, error) {
	result := &LoadResult{
		Config: DefaultConfig(),
		Errors: []error{},
	}

	// Create interpreter and evaluate
	interp := interpreter.NewWithLogger(l.logger)
	_, err := interp.EvalString(source)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, nil
	}

	result.Interpreter = interp

	// Extract configuration from the interpreter
	l.extractConfigFromInterpreter(interp, result)

	return result, nil
}

// LoadDefaultConfigPath loads configuration from the default path (~/.gshrc.gsh).
// Loading order:
//  1. .gshrc.default.gsh (system defaults)
//  2. ~/.gshrc.gsh (user config)
//  3. .gshrc.starship.gsh (if starship is detected and user hasn't disabled it)
//
// Starship integration is loaded last so that user config can set starshipIntegration = false
// to disable it. Users who want a custom prompt should disable starship integration and
// define their own GSH_PROMPT tool.
//
// defaultContent is the embedded content of .gshrc.default.gsh (can be empty).
// starshipContent is the embedded content of .gshrc.starship.gsh (can be empty).
func (l *Loader) LoadDefaultConfigPath(defaultContent string, starshipContent string) (*LoadResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &LoadResult{
			Config: DefaultConfig(),
			Errors: []error{fmt.Errorf("failed to get home directory: %w", err)},
		}, nil
	}

	// Start with default configuration
	result := &LoadResult{
		Config: DefaultConfig(),
		Errors: []error{},
	}

	// Create ONE interpreter for all configs
	interp := interpreter.NewWithLogger(l.logger)

	// 1. Load default config first
	if defaultContent != "" {
		_, err := interp.EvalString(defaultContent)
		if err != nil {
			result.Errors = append(result.Errors, err)
			// Log but continue - user config might still work
			if l.logger != nil {
				l.logger.Warn("errors loading default config", zap.Error(err))
			}
		} else if l.logger != nil {
			l.logger.Debug("loaded embedded default configuration")
		}
	}

	// 2. Load user config into SAME interpreter (shadows defaults)
	userConfigPath := filepath.Join(homeDir, ".gshrc.gsh")
	userContent, err := os.ReadFile(userConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read user config: %w", err)
		}
		// File doesn't exist, that's fine - continue with defaults only
		if l.logger != nil {
			l.logger.Debug("no user configuration file found", zap.String("path", userConfigPath))
		}
	} else {
		_, err := interp.EvalString(string(userContent))
		if err != nil {
			result.Errors = append(result.Errors, err)
			if l.logger != nil {
				l.logger.Warn("errors loading user config", zap.Error(err))
			}
		} else if l.logger != nil {
			l.logger.Debug("loaded user configuration", zap.String("path", userConfigPath))
		}
	}

	// 3. Extract config to check starshipIntegration setting
	l.extractConfigFromInterpreter(interp, result)

	// 4. Load starship integration if enabled and starship is available
	if starshipContent != "" && result.Config.StarshipIntegrationEnabled() && isStarshipAvailable() {
		_, err := interp.EvalString(starshipContent)
		if err != nil {
			result.Errors = append(result.Errors, err)
			if l.logger != nil {
				l.logger.Warn("errors loading starship integration", zap.Error(err))
			}
		} else if l.logger != nil {
			l.logger.Debug("loaded starship integration (starship detected in PATH)")
		}
		// Re-extract config after loading starship integration
		l.extractConfigFromInterpreter(interp, result)
	}

	// 5. Set interpreter in result
	result.Interpreter = interp

	return result, nil
}

// isStarshipAvailable checks if starship is available in the system PATH.
func isStarshipAvailable() bool {
	_, err := exec.LookPath("starship")
	return err == nil
}

// LoadBashRC loads a bash configuration file (.gshrc) by executing it through
// a bash interpreter. This maintains compatibility with existing bash/zsh configurations.
// Returns any errors encountered during execution (non-fatal).
func LoadBashRC(ctx context.Context, executor BashExecutor, rcPath string) error {
	// Check if file exists
	stat, err := os.Stat(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, this is fine
			return nil
		}
		return fmt.Errorf("failed to stat %s: %w", rcPath, err)
	}

	// Skip empty files
	if stat.Size() == 0 {
		return nil
	}

	// Open and execute the file
	f, err := os.Open(rcPath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", rcPath, err)
	}
	defer f.Close()

	err = executor.RunBashScriptFromReader(ctx, f, rcPath)
	if err != nil {
		return fmt.Errorf("failed to execute %s: %w", rcPath, err)
	}

	return nil
}

// BashExecutor is the interface required for loading bash configuration files.
type BashExecutor interface {
	RunBashScriptFromReader(ctx context.Context, reader io.Reader, name string) error
}

// extractConfigFromInterpreter extracts configuration values from the interpreter's environment.
// This is used when we want to extract config from an interpreter that has already evaluated code.
func (l *Loader) extractConfigFromInterpreter(interp *interpreter.Interpreter, result *LoadResult) {
	// Get all variables from the interpreter's environment
	vars := interp.GetVariables()

	// Extract GSH_CONFIG if present
	if gshConfig, ok := vars["GSH_CONFIG"]; ok {
		l.extractGSHConfig(gshConfig, result)
	}

	// Extract all declarations (models, agents, tools, MCP servers)
	for name, value := range vars {
		switch v := value.(type) {
		case *interpreter.ModelValue:
			result.Config.Models[name] = v
		case *interpreter.AgentValue:
			result.Config.Agents[name] = v
		case *interpreter.ToolValue:
			result.Config.Tools[name] = v
		case *interpreter.MCPProxyValue:
			// For MCP proxy values, we need to get the actual server config
			// The MCPProxyValue references a server in the manager
			// We'll create a minimal MCPServer struct for storage
			result.Config.MCPServers[name] = &mcp.MCPServer{
				Name: v.ServerName,
			}
		}
	}
}

// mergeResults merges two LoadResult objects, with the second result taking precedence.
// This is used to merge default config with user config.
func (l *Loader) mergeResults(base, override *LoadResult) *LoadResult {
	result := &LoadResult{
		Config:      base.Config.Clone(),
		Interpreter: override.Interpreter, // Use the most recent interpreter
		Errors:      append([]error{}, base.Errors...),
	}
	result.Errors = append(result.Errors, override.Errors...)

	// Merge configuration fields (override takes precedence for non-empty values)
	if override.Config.Prompt != "" && override.Config.Prompt != "gsh> " {
		result.Config.Prompt = override.Config.Prompt
	}
	if override.Config.LogLevel != "" && override.Config.LogLevel != "info" {
		result.Config.LogLevel = override.Config.LogLevel
	}
	if override.Config.PredictModel != "" {
		result.Config.PredictModel = override.Config.PredictModel
	}
	if override.Config.DefaultAgentModel != "" {
		result.Config.DefaultAgentModel = override.Config.DefaultAgentModel
	}

	// Merge maps (override takes precedence)
	for name, model := range override.Config.Models {
		result.Config.Models[name] = model
	}
	for name, agent := range override.Config.Agents {
		result.Config.Agents[name] = agent
	}
	for name, tool := range override.Config.Tools {
		result.Config.Tools[name] = tool
	}
	for name, server := range override.Config.MCPServers {
		result.Config.MCPServers[name] = server
	}

	return result
}

// extractGSHConfig extracts values from the GSH_CONFIG object.
func (l *Loader) extractGSHConfig(value interpreter.Value, result *LoadResult) {
	obj, ok := value.(*interpreter.ObjectValue)
	if !ok {
		result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG must be an object, got %s", value.Type()))
		return
	}

	// Extract prompt
	if prompt, ok := obj.Properties["prompt"]; ok {
		if strVal, ok := prompt.(*interpreter.StringValue); ok {
			result.Config.Prompt = strVal.Value
		} else {
			result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG.prompt must be a string"))
		}
	}

	// Extract logLevel
	if logLevel, ok := obj.Properties["logLevel"]; ok {
		if strVal, ok := logLevel.(*interpreter.StringValue); ok {
			result.Config.LogLevel = strVal.Value
		} else {
			result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG.logLevel must be a string"))
		}
	}

	// Extract predictModel
	if predictModel, ok := obj.Properties["predictModel"]; ok {
		if modelVal, ok := predictModel.(*interpreter.ModelValue); ok {
			// Only accept model reference, use its name
			result.Config.PredictModel = modelVal.Name
		} else {
			result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG.predictModel must be a model reference, got %s", predictModel.Type()))
		}
	}

	// Extract defaultAgentModel
	if defaultAgentModel, ok := obj.Properties["defaultAgentModel"]; ok {
		if modelVal, ok := defaultAgentModel.(*interpreter.ModelValue); ok {
			// Only accept model reference, use its name
			result.Config.DefaultAgentModel = modelVal.Name
		} else {
			result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG.defaultAgentModel must be a model reference, got %s", defaultAgentModel.Type()))
		}
	}

	// Extract starshipIntegration
	if starshipIntegration, ok := obj.Properties["starshipIntegration"]; ok {
		if boolVal, ok := starshipIntegration.(*interpreter.BoolValue); ok {
			result.Config.StarshipIntegration = &boolVal.Value
		} else {
			result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG.starshipIntegration must be a boolean"))
		}
	}
}
