package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/styles"
	"github.com/atinylittleshell/gsh/internal/utils"
	"github.com/atinylittleshell/gsh/pkg/gline"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var BashToolDefinition = openai.Tool{
	Type: "function",
	Function: &openai.FunctionDefinition{
		Name: "bash",
		Description: `Run a single-line command in a bash shell.
* When invoking this tool, the contents of the "command" parameter does NOT need to be XML-escaped.
* Avoid combining multiple bash commands into one using "&&", ";" or multiple lines. Instead, run each command separately.
* State is persistent across command calls and discussions with the user.`,
		Parameters: utils.GenerateJsonSchema(struct {
			Reason  string `json:"reason" description:"A concise reason for why you need to run this command" required:"true"`
			Command string `json:"command" description:"The bash command to run" required:"true"`
		}{}),
	},
}

// GenerateCommandRegex generates a regex pattern from a bash command
// The pattern is specific enough to match similar commands but general enough to be useful
// For example:
// - Command: "ls -la /tmp" → Regex: "^ls.*"
// - Command: "git status" → Regex: "^git status.*"
// - Command: "npm install package" → Regex: "^npm install.*"
func GenerateCommandRegex(command string) string {
	// Split the command into parts
	parts := strings.Fields(command)

	// If we have no parts, return a pattern that won't match anything
	if len(parts) == 0 {
		return "^$"
	}

	baseCommand := parts[0]

	// If there's only one part, just use the base command
	if len(parts) == 1 {
		return "^" + regexp.QuoteMeta(baseCommand) + ".*"
	}

	// Check if the second part looks like a subcommand (not a flag)
	// Subcommands typically:
	// - Don't start with a dash (-)
	// - Are alphabetic (not paths, numbers, or special chars)
	// - Are relatively short (< 20 chars to avoid matching full arguments)
	secondPart := parts[1]
	if hasSubcommand(baseCommand, secondPart) {
		// Include the subcommand in the pattern
		return "^" + regexp.QuoteMeta(baseCommand+" "+secondPart) + ".*"
	}

	// For regular commands with flags/args, just use the base command
	return "^" + regexp.QuoteMeta(baseCommand) + ".*"
}

// hasSubcommand determines if a string looks like a subcommand or argument that should be included in the regex pattern.
// For security purposes, we want to include the first non-flag argument in the pattern to make it more specific.
// This applies to all commands, whether they have traditional "subcommands" (like git, npm) or just arguments (like echo, grep).
func hasSubcommand(baseCommand, s string) bool {
	// Empty string is not a subcommand
	if s == "" {
		return false
	}

	// Flags start with - or --
	if strings.HasPrefix(s, "-") {
		return false
	}

	// Arguments should be reasonably short (avoid matching full file paths or long arguments)
	if len(s) > 20 {
		return false
	}

	// Arguments should be primarily alphabetic (allow hyphens for commands like "cache-clean")
	// but not contain special shell characters or path separators
	for _, ch := range s {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch == '-', ch == '_':
		case ch >= '0' && ch <= '9':
		default:
			return false
		}
	}

	// Must start with a letter (not a number)
	firstChar := rune(s[0])
	return (firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z')
}

// GenerateSpecificCommandRegex creates a more specific regex pattern for a given command prefix.
// This is used for pre-selection in the permissions menu to ensure only exact matches are pre-selected.
// Unlike GenerateCommandRegex, this creates unique patterns for each specific prefix.
//
// For example:
// - Command: "awk" → Regex: "^awk$"
// - Command: "awk -F'|'" → Regex: "^awk -F'|'.*"
// - Command: "awk -F'|' '{...}'" → Regex: "^awk -F'|' '{...}'.*"
func GenerateSpecificCommandRegex(command string) string {
	// Split the command into parts
	parts := strings.Fields(command)

	// If we have no parts, return a pattern that won't match anything
	if len(parts) == 0 {
		return "^$"
	}

	if len(parts) == 1 {
		// For single commands, match exactly
		return "^" + regexp.QuoteMeta(parts[0]) + "$"
	} else {
		// For multi-part commands, match the exact prefix followed by anything
		return "^" + regexp.QuoteMeta(command) + ".*"
	}
}

// GeneratePreselectionPattern generates the pattern that should be checked for pre-selection.
// This function determines what pattern in the authorized_commands file would correspond
// to this specific prefix being authorized.
//
// The key insight is that we want literal matching: only the prefix that would generate
// the exact same pattern should be pre-selected.
//
// For example:
// - Prefix "awk" should only be pre-selected if "^awk.*" is in the file
// - Prefix "awk -F'|'" should only be pre-selected if "^awk -F'|'.*" is in the file
// - Prefix "git status" should only be pre-selected if "^git status.*" is in the file
func GeneratePreselectionPattern(prefix string) string {
	// For pre-selection, we want to check for the specific pattern that would be saved
	// when THIS exact prefix is selected in the menu

	// Split the prefix into parts
	parts := strings.Fields(prefix)
	if len(parts) == 0 {
		return "^$"
	}

	baseCommand := parts[0]

	// Single-word commands use the base pattern
	if len(parts) == 1 {
		return "^" + regexp.QuoteMeta(baseCommand) + ".*"
	}

	// Check if the second part looks like a subcommand
	secondPart := parts[1]
	if hasSubcommand(baseCommand, secondPart) {
		// For commands with subcommands, include the subcommand in the pattern
		return "^" + regexp.QuoteMeta(baseCommand+" "+secondPart) + ".*"
	}

	// For multi-word regular commands (with flags/args), use the full prefix
	// This ensures "awk -F'|'" generates a different pattern than "awk"
	return "^" + regexp.QuoteMeta(prefix) + ".*"
}

func BashTool(runner *interp.Runner, historyManager *history.HistoryManager, logger *zap.Logger, params map[string]any) string {
	reason, ok := params["reason"].(string)
	if !ok {
		logger.Error("The bash tool failed to parse parameter 'reason'")
		return failedToolResponse("The bash tool failed to parse parameter 'reason'")
	}
	command, ok := params["command"].(string)
	if !ok {
		logger.Error("The bash tool failed to parse parameter 'command'")
		return failedToolResponse("The bash tool failed to parse parameter 'command'")
	}

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		logger.Error("LLM bash tool received invalid command", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("`%s` is not a valid bash command: %s", command, err))
	}

	// Always display the command first for consistent behavior
	fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(environment.GetAgentPrompt(runner)+command) + "\n")

	// Check if the command matches any pre-approved patterns using secure compound command validation
	approvedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)
	isPreApproved, err := ValidateCompoundCommand(command, approvedPatterns)
	if err != nil {
		logger.Debug("Failed to validate compound command", zap.Error(err))
		isPreApproved = false
	}

	var confirmResponse string
	if isPreApproved {
		confirmResponse = "y"
	} else {
		confirmResponse = userConfirmation(
			logger,
			runner,
			"gsh: Do I have your permission to run this command?",
			reason, // Only pass reason, not command (already displayed)
		)
	}
	if confirmResponse == "n" {
		return failedToolResponse("User declined this request")
	} else if confirmResponse == "m" {
		// User chose "m" (manage) - show permissions menu for command prefixes
		menuResponse, err := ShowPermissionsMenu(logger, command)
		if err != nil {
			logger.Error("Failed to show permissions menu", zap.Error(err))
			return failedToolResponse("Failed to show permissions menu")
		}

		// Process the menu response
		if strings.ToLower(menuResponse) == "n" {
			return failedToolResponse("User declined this request")
		} else if strings.ToLower(menuResponse) == "m" || strings.ToLower(menuResponse) == "manage" {
			// User selected specific permissions - the permissions menu has already saved
			// the enabled permissions to authorized_commands, so we just continue
			logger.Info("Permissions have been saved by the permissions menu")
		} else if strings.ToLower(menuResponse) != "y" {
			return failedToolResponse(fmt.Sprintf("User declined this request: %s", menuResponse))
		}
		// If menuResponse == "y", continue with execution
	} else if confirmResponse != "y" {
		return failedToolResponse(fmt.Sprintf("User declined this request: %s", confirmResponse))
	}

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	multiOut := io.MultiWriter(os.Stdout, outBuf)
	multiErr := io.MultiWriter(os.Stderr, errBuf)

	_ = interp.StdIO(os.Stdin, multiOut, multiErr)(runner)
	defer func() {
		_ = interp.StdIO(os.Stdin, os.Stdout, os.Stderr)(runner)
	}()

	historyEntry, _ := historyManager.StartCommand(command, environment.GetPwd(runner))

	err = runner.Run(context.Background(), prog)

	exitCode := 0
	if err != nil {
		status, ok := interp.IsExitStatus(err)
		if ok {
			exitCode = int(status)
		} else {
			return failedToolResponse(fmt.Sprintf("Error running command: %s", err))
		}
	}
	stdout := outBuf.String()
	stderr := errBuf.String()

	_, _ = historyManager.FinishCommand(historyEntry, exitCode)

	jsonBuffer, err := json.Marshal(map[string]any{
		"stdout":   stdout,
		"stderr":   stderr,
		"exitCode": exitCode,
	})
	if err != nil {
		logger.Error("Failed to marshal tool response", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Failed to marshal tool response: %s", err))
	}

	return string(jsonBuffer)
}
