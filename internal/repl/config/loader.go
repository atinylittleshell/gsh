// Package config provides configuration management for the gsh REPL.
package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
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

	// Parse the source
	lex := lexer.New(source)
	p := parser.New(lex)
	program := p.ParseProgram()

	// Check for parser errors
	if len(p.Errors()) > 0 {
		for _, errMsg := range p.Errors() {
			result.Errors = append(result.Errors, fmt.Errorf("parse error: %s", errMsg))
		}
		// Continue with defaults on parse errors
		return result, nil
	}

	// Create interpreter and evaluate
	interp := interpreter.NewWithLogger(l.logger)
	evalResult, err := interp.Eval(program)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("eval error: %w", err))
		// Continue with defaults on eval errors
		return result, nil
	}

	result.Interpreter = interp

	// Extract configuration from the environment
	l.extractConfig(evalResult, result)

	return result, nil
}

// LoadDefaultConfigPath loads configuration from the default path (~/.gshrc.gsh).
// It first loads .gshrc.default.gsh (system defaults), then merges user's ~/.gshrc.gsh.
// defaultContent is the embedded content of .gshrc.default.gsh (can be empty).
func (l *Loader) LoadDefaultConfigPath(defaultContent string) (*LoadResult, error) {
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

	// Load system default config from embedded content if provided
	if defaultContent != "" {
		defaultResult, err := l.LoadFromString(defaultContent)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to load default config: %w", err))
		} else {
			// Merge default config into result
			result = l.mergeResults(result, defaultResult)
			l.logger.Debug("loaded embedded default configuration")
		}
	}

	// Load user configuration (~/.gshrc.gsh)
	userConfigPath := filepath.Join(homeDir, ".gshrc.gsh")
	userResult, err := l.LoadFromFile(userConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	// Merge user config into result (user config takes precedence)
	result = l.mergeResults(result, userResult)

	return result, nil
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

// extractConfig extracts configuration values from the evaluation result.
func (l *Loader) extractConfig(evalResult *interpreter.EvalResult, result *LoadResult) {
	vars := evalResult.Variables()

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

}
