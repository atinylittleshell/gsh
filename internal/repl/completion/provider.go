package completion

import (
	"context"
	"strings"
	"unicode"

	"github.com/atinylittleshell/gsh/internal/repl/completion/completers"
	"mvdan.cc/sh/v3/interp"
)

// RunnerProvider is an interface for getting the shell runner and current directory.
// This allows the completion provider to work with different executor implementations.
type RunnerProvider interface {
	// Runner returns the underlying mvdan/sh runner.
	Runner() *interp.Runner
	// GetPwd returns the current working directory.
	GetPwd() string
}

// Provider implements the CompletionProvider interface for the REPL input.
// It parses user input and routes completion requests to the appropriate source
// (specs, files, commands, macros, etc.)
type Provider struct {
	specRegistry     *SpecRegistry
	runnerProvider   RunnerProvider
	macroCompleter   *completers.MacroCompleter
	builtinCompleter *completers.BuiltinCompleter
	commandCompleter *completers.CommandCompleter
}

// NewProvider creates a new completion Provider.
func NewProvider(runnerProvider RunnerProvider) *Provider {
	var runner *interp.Runner
	if runnerProvider != nil {
		runner = runnerProvider.Runner()
	}

	return &Provider{
		specRegistry:     NewSpecRegistry(),
		runnerProvider:   runnerProvider,
		macroCompleter:   completers.NewMacroCompleter(runner),
		builtinCompleter: completers.NewBuiltinCompleter(),
		commandCompleter: completers.NewCommandCompleter(runner, runnerProvider.GetPwd, GetFileCompletions),
	}
}

// RegisterSpec adds or updates a completion specification for a command.
func (p *Provider) RegisterSpec(spec CompletionSpec) {
	p.specRegistry.AddSpec(spec)
}

// UnregisterSpec removes a completion specification for a command.
func (p *Provider) UnregisterSpec(command string) {
	p.specRegistry.RemoveSpec(command)
}

// GetCompletions returns completion suggestions for the current input line.
func (p *Provider) GetCompletions(line string, pos int) []string {
	// First check for special prefixes (#/ and #!)
	if completion := p.checkSpecialPrefixes(line, pos); completion != nil {
		return completion
	}

	// Split the line into words, preserving quotes
	line = line[:pos]
	words := SplitPreservingQuotes(line)
	if len(words) == 0 {
		return make([]string, 0)
	}

	// Get the command (first word)
	command := words[0]

	// Look up completion spec for this command
	spec, ok := p.specRegistry.GetSpec(command)
	if ok {
		// If line ends with space, we're completing a new argument
		// Add empty string to indicate this
		if strings.HasSuffix(line, " ") {
			words = append(words, "")
		}

		// Execute the completion
		suggestions, err := p.specRegistry.ExecuteCompletion(context.Background(), p.runnerProvider.Runner(), spec, words)
		if err != nil {
			return make([]string, 0)
		}

		if suggestions == nil {
			return make([]string, 0)
		}
		return suggestions
	}

	// No specific completion spec, check if we should complete command names
	if len(words) == 1 && !strings.HasSuffix(line, " ") {
		// Single word that doesn't end with space
		// Check if this looks like a path-based command
		if p.commandCompleter.IsPathBasedCommand(command) {
			// For path-based commands, complete with executable files in that path
			executableCompletions := p.commandCompleter.GetExecutableCompletions(command)
			if len(executableCompletions) > 0 {
				return executableCompletions
			}
		} else {
			// Regular command name completion
			commandCompletions := p.commandCompleter.GetAvailableCommands(command)
			if len(commandCompletions) > 0 {
				return commandCompletions
			}
		}
	}

	// No command matches or multiple words, try file path completion
	var prefix string
	if strings.HasSuffix(line, " ") {
		// If line ends with space, use empty prefix to list all files
		prefix = ""
	} else if len(words) > 1 {
		// Get the last word as the prefix for file completion
		prefix = words[len(words)-1]
	} else {
		return make([]string, 0)
	}

	completions := GetFileCompletions(prefix, p.runnerProvider.GetPwd())

	// Quote completions that contain spaces, but don't add command prefix
	// The completion handler will replace only the current word (file path)
	for i, completion := range completions {
		if strings.Contains(completion, " ") {
			// Quote completions that contain spaces
			completions[i] = "\"" + completion + "\""
		}
	}
	return completions
}

// checkSpecialPrefixes checks for #/ and #! prefixes and returns appropriate completions.
func (p *Provider) checkSpecialPrefixes(line string, pos int) []string {
	// Get the current word being completed
	start, end := p.getCurrentWordBoundary(line, pos)
	if start < 0 || end < 0 {
		return nil
	}

	currentWord := line[start:end]

	// Check if the current word starts with #/ or #!
	if strings.HasPrefix(currentWord, "#/") {
		completions := p.macroCompleter.GetCompletions(currentWord)
		if len(completions) == 0 {
			// No macro matches found, fall back to path completion
			pathPrefix := strings.TrimPrefix(currentWord, "#/")
			completions := GetFileCompletions(pathPrefix, p.runnerProvider.GetPwd())

			// Build the proper prefix for the current line context
			var linePrefix string
			if start > 0 {
				linePrefix = line[:start]
			}

			// Add completions with proper prefix
			for i, completion := range completions {
				completions[i] = linePrefix + completion
			}
			return completions
		}
		return completions
	} else if strings.HasPrefix(currentWord, "#!") {
		completions := p.builtinCompleter.GetCompletions(currentWord)
		if len(completions) == 0 {
			// No builtin command matches found, fall back to path completion
			pathPrefix := strings.TrimPrefix(currentWord, "#!")
			completions := GetFileCompletions(pathPrefix, p.runnerProvider.GetPwd())

			// Build the proper prefix for the current line context
			var linePrefix string
			if start > 0 {
				linePrefix = line[:start]
			}

			// Add completions with proper prefix
			for i, completion := range completions {
				completions[i] = linePrefix + completion
			}
			return completions
		}
		return completions
	}

	// Also check if we're at the beginning of a potential prefix
	// Look backwards to see if there's a #/ or #! that we should complete
	if start > 0 {
		// Find the start of the word that might contain our prefix
		wordStart := start
		for wordStart > 0 && !unicode.IsSpace(rune(line[wordStart-1])) {
			wordStart--
		}

		potentialWord := line[wordStart:end]
		if strings.HasPrefix(potentialWord, "#/") {
			completions := p.macroCompleter.GetCompletions(potentialWord)
			if len(completions) == 0 {
				// No macro matches found, fall back to path completion
				pathPrefix := strings.TrimPrefix(potentialWord, "#/")
				completions := GetFileCompletions(pathPrefix, p.runnerProvider.GetPwd())

				// Build the proper prefix for the current line context
				var linePrefix string
				if wordStart > 0 {
					linePrefix = line[:wordStart]
				}

				// Add completions with proper prefix
				for i, completion := range completions {
					completions[i] = linePrefix + completion
				}
				return completions
			}
			return completions
		} else if strings.HasPrefix(potentialWord, "#!") {
			completions := p.builtinCompleter.GetCompletions(potentialWord)
			if len(completions) == 0 {
				// No builtin command matches found, fall back to path completion
				pathPrefix := strings.TrimPrefix(potentialWord, "#!")
				completions := GetFileCompletions(pathPrefix, p.runnerProvider.GetPwd())

				// Build the proper prefix for the current line context
				var linePrefix string
				if wordStart > 0 {
					linePrefix = line[:wordStart]
				}

				// Add completions with proper prefix
				for i, completion := range completions {
					completions[i] = linePrefix + completion
				}
				return completions
			}
			return completions
		}
	}

	return nil
}

// getCurrentWordBoundary finds the start and end of the current word at cursor position.
func (p *Provider) getCurrentWordBoundary(line string, pos int) (int, int) {
	if len(line) == 0 || pos > len(line) {
		return -1, -1
	}

	// Find start of word
	start := pos
	for start > 0 && !unicode.IsSpace(rune(line[start-1])) {
		start--
	}

	// Find end of word
	end := pos
	for end < len(line) && !unicode.IsSpace(rune(line[end])) {
		end++
	}

	return start, end
}

// GetHelpInfo returns help information for special commands like #! and #/.
func (p *Provider) GetHelpInfo(line string, pos int) string {
	// Get the current word being completed
	start, end := p.getCurrentWordBoundary(line, pos)
	if start < 0 || end < 0 {
		return ""
	}

	currentWord := line[start:end]

	// Check if the current word starts with #! (agent controls)
	if strings.HasPrefix(currentWord, "#!") {
		command := strings.TrimPrefix(currentWord, "#!")
		return p.builtinCompleter.GetHelp(command)
	}

	// Check if the current word starts with #/ (macros)
	if strings.HasPrefix(currentWord, "#/") {
		macroName := strings.TrimPrefix(currentWord, "#/")
		return p.macroCompleter.GetHelp(macroName)
	}

	// Also check if we're at the beginning of a potential prefix
	if start > 0 {
		// Find the start of the word that might contain our prefix
		wordStart := start
		for wordStart > 0 && !unicode.IsSpace(rune(line[wordStart-1])) {
			wordStart--
		}

		potentialWord := line[wordStart:end]
		if strings.HasPrefix(potentialWord, "#!") {
			command := strings.TrimPrefix(potentialWord, "#!")
			return p.builtinCompleter.GetHelp(command)
		} else if strings.HasPrefix(potentialWord, "#/") {
			macroName := strings.TrimPrefix(potentialWord, "#/")
			return p.macroCompleter.GetHelp(macroName)
		}
	}

	return ""
}
