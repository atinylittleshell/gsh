package interpreter

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
)

// OriginType represents the origin of a script
type OriginType string

const (
	// OriginEmbed indicates a script embedded in the binary
	OriginEmbed OriginType = "embed"
	// OriginFilesystem indicates a script on the filesystem
	OriginFilesystem OriginType = "filesystem"
)

// ScriptOrigin represents the origin of a script being executed
type ScriptOrigin struct {
	Type     OriginType // "embed" or "filesystem"
	BasePath string     // Directory containing the script
	EmbedFS  fs.FS      // Embedded filesystem (only used when Type is OriginEmbed)
}

// resolveImportPath resolves an import path relative to the current script origin
func (i *Interpreter) resolveImportPath(importPath string) (*ScriptOrigin, string, error) {
	if i.currentOrigin == nil {
		// No origin set, assume filesystem relative to current directory
		if filepath.IsAbs(importPath) {
			return &ScriptOrigin{Type: OriginFilesystem, BasePath: filepath.Dir(importPath)}, importPath, nil
		}
		// Relative to current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get current directory: %w", err)
		}
		resolvedPath := filepath.Join(cwd, importPath)
		return &ScriptOrigin{Type: OriginFilesystem, BasePath: filepath.Dir(resolvedPath)}, resolvedPath, nil
	}

	// Handle relative paths
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		var resolvedPath, baseDir string

		if i.currentOrigin.Type == OriginEmbed {
			// For embedded files, use path package (forward slashes)
			resolvedPath = path.Join(i.currentOrigin.BasePath, importPath)
			resolvedPath = path.Clean(resolvedPath)
			baseDir = path.Dir(resolvedPath)
		} else {
			// For filesystem, use filepath package (OS-specific)
			resolvedPath = filepath.Join(i.currentOrigin.BasePath, importPath)
			resolvedPath = filepath.Clean(resolvedPath)
			baseDir = filepath.Dir(resolvedPath)
		}

		// Copy EmbedFS from current origin to resolved origin
		return &ScriptOrigin{Type: i.currentOrigin.Type, BasePath: baseDir, EmbedFS: i.currentOrigin.EmbedFS}, resolvedPath, nil
	}

	// Handle absolute paths (filesystem only)
	if filepath.IsAbs(importPath) {
		return &ScriptOrigin{Type: OriginFilesystem, BasePath: filepath.Dir(importPath)}, importPath, nil
	}

	return nil, "", fmt.Errorf("invalid import path: %s (must be relative starting with ./ or ../, or absolute)", importPath)
}

// readImportedFile reads a file from either the embedded filesystem or the real filesystem
func (i *Interpreter) readImportedFile(origin *ScriptOrigin, filePath string) (string, error) {
	if origin.Type == OriginEmbed {
		// Check if embedFS is configured
		if origin.EmbedFS == nil {
			return "", fmt.Errorf("import from embedded file %s failed: embedded filesystem not configured", filePath)
		}
		// Read from embedded filesystem
		content, err := fs.ReadFile(origin.EmbedFS, filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded file %s: %w", filePath, err)
		}
		return string(content), nil
	}

	// Read from real filesystem
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return string(content), nil
}

// normalizeImportKey creates a unique key for tracking imports
func normalizeImportKey(origin *ScriptOrigin, path string) string {
	return fmt.Sprintf("%s:%s", origin.Type, path)
}

// evalImportStatement evaluates an import statement
func (i *Interpreter) evalImportStatement(node *parser.ImportStatement) (Value, error) {
	importPath := node.Path.Value

	// Resolve the import path
	origin, resolvedPath, err := i.resolveImportPath(importPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve import path %q: %w", importPath, err)
	}

	// Check for circular imports
	importKey := normalizeImportKey(origin, resolvedPath)
	if i.importedFiles[importKey] {
		// Already imported, check if it's a circular import or just a repeat
		if exports, ok := i.moduleExports[importKey]; ok {
			// Module already processed, import the requested symbols
			if len(node.Symbols) > 0 {
				for _, sym := range node.Symbols {
					if val, ok := exports[sym]; ok {
						i.env.Set(sym, val)
					} else {
						return nil, fmt.Errorf("symbol %q is not exported from %q", sym, importPath)
					}
				}
			}
			return &NullValue{}, nil
		}
		// Currently being imported (circular)
		return nil, fmt.Errorf("circular import detected: %s", importPath)
	}

	// Mark as being imported
	i.importedFiles[importKey] = true

	// Read the file content
	content, err := i.readImportedFile(origin, resolvedPath)
	if err != nil {
		return nil, err
	}

	// Skip shebang line if present
	if strings.HasPrefix(content, "#!") {
		if idx := strings.Index(content, "\n"); idx >= 0 {
			content = content[idx+1:]
		}
	}

	// Parse the module
	l := lexer.New(content)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s: %s", importPath, strings.Join(p.Errors(), "; "))
	}

	// Save current state
	prevOrigin := i.currentOrigin
	prevEnv := i.env
	prevExportedNames := i.exportedNames

	// Set up module execution environment
	i.currentOrigin = origin
	i.env = NewEnclosedEnvironment(prevEnv) // Module has its own scope but can access outer
	i.exportedNames = make(map[string]bool)

	// Execute the module
	var lastResult Value = &NullValue{}
	for _, stmt := range program.Statements {
		val, err := i.evalStatement(stmt)
		if err != nil {
			// Restore state before returning error
			i.currentOrigin = prevOrigin
			i.env = prevEnv
			i.exportedNames = prevExportedNames
			return nil, fmt.Errorf("error in %s: %w", importPath, err)
		}
		if val != nil {
			lastResult = val
		}
	}

	// Collect exports from the module
	exports := make(map[string]Value)
	for name := range i.exportedNames {
		if val, ok := i.env.Get(name); ok {
			exports[name] = val
		}
	}
	i.moduleExports[importKey] = exports

	// Restore previous state
	i.currentOrigin = prevOrigin
	i.env = prevEnv
	i.exportedNames = prevExportedNames

	// Import requested symbols into current scope
	if len(node.Symbols) > 0 {
		for _, sym := range node.Symbols {
			if val, ok := exports[sym]; ok {
				i.env.Set(sym, val)
			} else {
				return nil, fmt.Errorf("symbol %q is not exported from %q", sym, importPath)
			}
		}
	}

	return lastResult, nil
}

// evalExportStatement evaluates an export statement
func (i *Interpreter) evalExportStatement(node *parser.ExportStatement) (Value, error) {
	// Evaluate the declaration
	val, err := i.evalStatement(node.Declaration)
	if err != nil {
		return nil, err
	}

	// Mark the symbol as exported
	if node.Name != "" {
		i.exportedNames[node.Name] = true
	}

	return val, nil
}
