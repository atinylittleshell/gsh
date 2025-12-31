// Package config provides configuration management for the gsh REPL.
package config

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"go.uber.org/zap"
)

// Loader handles loading and parsing of ~/.gsh/repl.gsh configuration files.
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

// LoadFromFile loads configuration from a repl.gsh file.
// Returns the configuration and any non-fatal errors encountered.
// If the file doesn't exist, returns default configuration with no error.
func (l *Loader) LoadFromFile(path string) (*LoadResult, error) {
	result := &LoadResult{
		Config: DefaultConfig(),
		Errors: []error{},
	}

	// Get absolute path for proper import resolution
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	// Check if file exists
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return defaults
			return result, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return l.LoadFromFileContent(absPath, string(content))
}

// LoadFromFileContent loads configuration from a gsh script string with a known file path.
// This sets up proper import resolution relative to the file's directory.
func (l *Loader) LoadFromFileContent(filePath string, source string) (*LoadResult, error) {
	interp := interpreter.New(&interpreter.Options{Logger: l.logger})

	result := &LoadResult{
		Config:      DefaultConfig(),
		Interpreter: interp,
		Errors:      []error{},
	}

	// Evaluate with filesystem origin for import resolution
	_, err := interp.EvalString(source, &interpreter.ScriptOrigin{
		Type:     interpreter.OriginFilesystem,
		BasePath: filepath.Dir(filePath),
	})
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, nil
	}

	// Extract configuration from the interpreter
	l.ExtractConfigFromInterpreter(interp, result)

	return result, nil
}

// LoadFromString loads configuration from a gsh script string.
// Creates a new interpreter internally. For loading into an existing interpreter,
// use LoadFromStringInto.
func (l *Loader) LoadFromString(source string) (*LoadResult, error) {
	interp := interpreter.New(&interpreter.Options{Logger: l.logger})
	return l.LoadFromStringInto(interp, source)
}

// LoadFromStringInto loads configuration from a gsh script string into an existing interpreter.
func (l *Loader) LoadFromStringInto(interp *interpreter.Interpreter, source string) (*LoadResult, error) {
	result := &LoadResult{
		Config:      DefaultConfig(),
		Interpreter: interp,
		Errors:      []error{},
	}

	_, err := interp.EvalString(source, nil)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result, nil
	}

	// Extract configuration from the interpreter
	l.ExtractConfigFromInterpreter(interp, result)

	return result, nil
}

// EmbeddedDefaults contains the embedded default configuration.
// If EmbedFS is provided, imports from the default config will resolve against it.
type EmbeddedDefaults struct {
	Content  string // The content of the main default config file
	FS       fs.FS  // The embedded filesystem (optional, enables imports)
	BasePath string // The base path within the embedded FS (e.g., "defaults")
}

// LoadDefaultConfigPathInto loads configuration from the default path (~/.gsh/repl.gsh)
// into an existing interpreter.
//
// Loading order:
//  1. defaults/init.gsh (system defaults) - sets up SDK properties and event handlers
//  2. ~/.gsh/repl.gsh (user config) - can override SDK properties and add custom handlers
//
// Configuration is now managed via the SDK (gsh.* properties and gsh.on() event handlers)
// rather than the legacy GSH_CONFIG object.
//
// defaults contains the embedded default configuration (can have empty Content to skip).
func (l *Loader) LoadDefaultConfigPathInto(interp *interpreter.Interpreter, defaults EmbeddedDefaults) (*LoadResult, error) {
	// Start with default configuration
	result := &LoadResult{
		Config:      DefaultConfig(),
		Interpreter: interp,
		Errors:      []error{},
	}

	// 1. Load default config first (sets up SDK properties and event handlers)
	if defaults.Content != "" {
		var origin *interpreter.ScriptOrigin
		if defaults.FS != nil {
			origin = &interpreter.ScriptOrigin{
				Type:     interpreter.OriginEmbed,
				BasePath: defaults.BasePath,
				EmbedFS:  defaults.FS,
			}
		}

		_, err := interp.EvalString(defaults.Content, origin)
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

	// 2. Load user config into SAME interpreter (can override SDK properties)
	gshDir := core.DataDir()
	userConfigPath := filepath.Join(gshDir, "repl.gsh")
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
		// Evaluate with filesystem origin for import resolution
		// Imports in ~/.gsh/repl.gsh will resolve relative to the .gsh directory
		_, err := interp.EvalString(string(userContent), &interpreter.ScriptOrigin{
			Type:     interpreter.OriginFilesystem,
			BasePath: gshDir,
		})
		if err != nil {
			result.Errors = append(result.Errors, err)
			if l.logger != nil {
				l.logger.Warn("errors loading user config", zap.Error(err))
			}
		} else if l.logger != nil {
			l.logger.Debug("loaded user configuration", zap.String("path", userConfigPath))
		}
	}

	// 3. Extract declarations (models, agents, tools, MCP servers)
	l.ExtractConfigFromInterpreter(interp, result)

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

// ExtractConfigFromInterpreter extracts declarations from the interpreter's environment.
// This extracts models, agents, tools, and MCP servers defined in the config files.
func (l *Loader) ExtractConfigFromInterpreter(interp *interpreter.Interpreter, result *LoadResult) {
	// Get all variables from the interpreter's environment
	vars := interp.GetVariables()

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
