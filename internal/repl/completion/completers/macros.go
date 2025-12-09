// Package completers provides specialized completion sources for different types of input.
package completers

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// MacroCompleter provides completions for macros (prefixed with #/).
type MacroCompleter struct {
	runner *interp.Runner
}

// NewMacroCompleter creates a new MacroCompleter.
func NewMacroCompleter(runner *interp.Runner) *MacroCompleter {
	return &MacroCompleter{runner: runner}
}

// GetCompletions returns macro completions for the given prefix.
func (c *MacroCompleter) GetCompletions(prefix string) []string {
	var macrosStr string
	if c.runner != nil {
		macrosStr = c.runner.Vars["GSH_AGENT_MACROS"].String()
	} else {
		// Fallback to environment variable for testing
		macrosStr = os.Getenv("GSH_AGENT_MACROS")
	}

	if macrosStr == "" {
		return []string{}
	}

	var macros map[string]interface{}
	if err := json.Unmarshal([]byte(macrosStr), &macros); err != nil {
		return []string{}
	}

	var completions []string
	prefixAfterSlash := strings.TrimPrefix(prefix, "#/")

	for macroName := range macros {
		if strings.HasPrefix(macroName, prefixAfterSlash) {
			completions = append(completions, "#/"+macroName)
		}
	}

	// Sort alphabetically for consistent ordering
	sort.Strings(completions)
	return completions
}

// GetHelp returns help information for a macro.
func (c *MacroCompleter) GetHelp(macroName string) string {
	var macrosStr string
	if c.runner != nil {
		macrosStr = c.runner.Vars["GSH_AGENT_MACROS"].String()
	} else {
		// Fallback to environment variable for testing
		macrosStr = os.Getenv("GSH_AGENT_MACROS")
	}

	if macrosStr == "" {
		if macroName == "" {
			return "**Chat Macros** - Quick shortcuts for common agent messages\n\nNo macros are currently configured."
		}
		return ""
	}

	var macros map[string]interface{}
	if err := json.Unmarshal([]byte(macrosStr), &macros); err != nil {
		return ""
	}

	if macroName == "" {
		// Show general macro help
		var macroList []string
		for name := range macros {
			macroList = append(macroList, "• **#/"+name+"**")
		}
		sort.Strings(macroList)

		if len(macroList) == 0 {
			return "**Chat Macros** - Quick shortcuts for common agent messages\n\nNo macros are currently configured."
		}

		return "**Chat Macros** - Quick shortcuts for common agent messages\n\nAvailable macros:\n" + strings.Join(macroList, "\n")
	}

	// Check for exact match first
	if message, ok := macros[macroName]; ok {
		if msgStr, ok := message.(string); ok {
			return "**#/" + macroName + "** - Chat macro\n\n**Expands to:**\n" + msgStr
		}
	}

	// Check for partial matches
	var matches []string
	for name, message := range macros {
		if strings.HasPrefix(name, macroName) {
			if msgStr, ok := message.(string); ok {
				matches = append(matches, "• **#/"+name+"** - "+msgStr)
			}
		}
	}

	if len(matches) > 0 {
		sort.Strings(matches)
		return "**Chat Macros** - Matching macros:\n\n" + strings.Join(matches, "\n")
	}

	return ""
}
