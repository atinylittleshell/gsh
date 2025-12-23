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
func (l *Loader) LoadDefaultConfigPath() (*LoadResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &LoadResult{
			Config: DefaultConfig(),
			Errors: []error{fmt.Errorf("failed to get home directory: %w", err)},
		}, nil
	}

	configPath := filepath.Join(homeDir, ".gshrc.gsh")
	return l.LoadFromFile(configPath)
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
		if strVal, ok := predictModel.(*interpreter.StringValue); ok {
			result.Config.PredictModel = strVal.Value
		} else {
			result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG.predictModel must be a string"))
		}
	}

	// Extract defaultAgent
	if defaultAgent, ok := obj.Properties["defaultAgent"]; ok {
		if strVal, ok := defaultAgent.(*interpreter.StringValue); ok {
			result.Config.DefaultAgent = strVal.Value
		} else {
			result.Errors = append(result.Errors, fmt.Errorf("GSH_CONFIG.defaultAgent must be a string"))
		}
	}

}
