package tools

import (
	"fmt"
	"regexp"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ExtractCommands parses a shell command and extracts all individual commands
// from compound statements (semicolons, &&, ||, pipes, subshells, etc.)
func ExtractCommands(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return []string{}, nil
	}

	parser := syntax.NewParser()

	var commands []string

	// Parse the command using Stmts to handle multiple statements
	err := parser.Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		stmtCommands := extractFromStatement(stmt)
		commands = append(commands, stmtCommands...)

		// Also walk the AST to find command substitutions
		syntax.Walk(stmt, func(node syntax.Node) bool {
			if cmdSubst, ok := node.(*syntax.CmdSubst); ok {
				substCommands := extractCommandsFromCmdSubst(cmdSubst)
				commands = append(commands, substCommands...)
			}
			return true
		})

		return true // Continue parsing more statements
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	// Remove duplicates and empty commands
	result := deduplicateCommands(commands)
	if result == nil {
		return []string{}, nil
	}
	return result, nil
}

// extractFromStatement extracts commands from a single statement
func extractFromStatement(stmt *syntax.Stmt) []string {
	var commands []string

	if stmt.Cmd != nil {
		commands = append(commands, extractFromCommand(stmt.Cmd)...)
	}

	return commands
}

// extractFromCommand extracts commands from a command node
func extractFromCommand(cmd syntax.Command) []string {
	var commands []string

	switch c := cmd.(type) {
	case *syntax.CallExpr:
		if cmdStr := extractCallCommand(c); cmdStr != "" {
			commands = append(commands, cmdStr)
		}
	case *syntax.BinaryCmd:
		// Handle binary commands (&&, ||, |)
		// For BinaryCmd, X and Y are *syntax.Stmt, so we extract from them directly
		commands = append(commands, extractFromStatement(c.X)...)
		commands = append(commands, extractFromStatement(c.Y)...)
	case *syntax.Subshell:
		// Handle subshells - extract from the statements within
		if c.Stmts != nil {
			for _, stmt := range c.Stmts {
				commands = append(commands, extractFromStatement(stmt)...)
			}
		}
	}

	return commands
}

// extractCallCommand extracts the command string from a CallExpr
func extractCallCommand(call *syntax.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}

	var parts []string
	for _, arg := range call.Args {
		if part := extractWordString(arg); part != "" {
			parts = append(parts, part)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

// extractWordString extracts string content from a Word node
func extractWordString(word *syntax.Word) string {
	if word == nil || len(word.Parts) == 0 {
		return ""
	}

	var result strings.Builder
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			result.WriteString(p.Value)
		case *syntax.SglQuoted:
			result.WriteString("'" + p.Value + "'")
		case *syntax.DblQuoted:
			result.WriteString("\"")
			for _, qpart := range p.Parts {
				if lit, ok := qpart.(*syntax.Lit); ok {
					result.WriteString(lit.Value)
				}
			}
			result.WriteString("\"")
		case *syntax.CmdSubst:
			// For command substitution, we'll represent it as $(...)
			result.WriteString("$(")
			if p.Stmts != nil {
				for i, stmt := range p.Stmts {
					if i > 0 {
						result.WriteString("; ")
					}
					subCommands := extractFromStatement(stmt)
					if len(subCommands) > 0 {
						result.WriteString(subCommands[0])
					}
				}
			}
			result.WriteString(")")
		case *syntax.ParamExp:
			// Handle parameter expansion like $VAR
			result.WriteString("$")
			if p.Param != nil {
				result.WriteString(p.Param.Value)
			}
		}
	}

	return result.String()
}

// extractCommandsFromCmdSubst extracts commands from command substitution
func extractCommandsFromCmdSubst(cmdSubst *syntax.CmdSubst) []string {
	var commands []string
	if cmdSubst.Stmts != nil {
		for _, stmt := range cmdSubst.Stmts {
			commands = append(commands, extractFromStatement(stmt)...)
		}
	}
	return commands
}

// deduplicateCommands removes duplicate commands while preserving order
func deduplicateCommands(commands []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" && !seen[cmd] {
			seen[cmd] = true
			result = append(result, cmd)
		}
	}

	return result
}

// ValidateCompoundCommand checks if all individual commands in a compound command
// are approved by the given regex patterns
func ValidateCompoundCommand(command string, patterns []string) (bool, error) {
	// Extract all individual commands
	commands, err := ExtractCommands(command)
	if err != nil {
		return false, fmt.Errorf("failed to extract commands: %w", err)
	}

	// If no commands were extracted, treat as not approved
	if len(commands) == 0 {
		return false, nil
	}

	// Check each command against all patterns
	for _, cmd := range commands {
		approved := false
		for _, pattern := range patterns {
			matched, err := regexp.MatchString(pattern, cmd)
			if err == nil && matched {
				approved = true
				break
			}
		}

		// If any command is not approved, the entire compound command is not approved
		if !approved {
			return false, nil
		}
	}

	// All commands are approved
	return true, nil
}

// GenerateCompoundCommandRegex generates regex patterns for all commands in a compound command
// This is used when the user chooses "always allow" for a compound command
func GenerateCompoundCommandRegex(command string) ([]string, error) {
	commands, err := ExtractCommands(command)
	if err != nil {
		return nil, fmt.Errorf("failed to extract commands: %w", err)
	}

	var patterns []string
	for _, cmd := range commands {
		pattern := GenerateCommandRegex(cmd)
		if pattern != "" {
			patterns = append(patterns, pattern)
		}
	}

	result := deduplicateCommands(patterns)
	if result == nil {
		return []string{}, nil
	}
	return result, nil
}
