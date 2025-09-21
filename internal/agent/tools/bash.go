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

	// For most commands, we'll use the first part as the base
	// For certain commands like git, we might want to include the first two parts
	baseCommand := parts[0]

	// Special handling for commands that have meaningful subcommands
	// like git, npm, etc.
	specialCommands := map[string]bool{
		"git":     true,
		"npm":     true,
		"yarn":    true,
		"docker":  true,
		"kubectl": true,
	}

	if specialCommands[baseCommand] && len(parts) > 1 {
		// For special commands, include the first two parts
		return "^" + regexp.QuoteMeta(baseCommand+" "+parts[1]) + ".*"
	} else {
		// For regular commands, just use the base command
		return "^" + regexp.QuoteMeta(baseCommand) + ".*"
	}
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

	// Special handling for commands that have meaningful subcommands
	specialCommands := map[string]bool{
		"git":     true,
		"npm":     true,
		"yarn":    true,
		"docker":  true,
		"kubectl": true,
	}

	if specialCommands[baseCommand] && len(parts) > 1 {
		// For special commands, include the subcommand in the pattern
		return "^" + regexp.QuoteMeta(baseCommand+" "+parts[1]) + ".*"
	} else if len(parts) == 1 {
		// For single-word commands, use the base pattern
		return "^" + regexp.QuoteMeta(baseCommand) + ".*"
	} else {
		// For multi-word regular commands, use the full prefix
		// This ensures "awk -F'|'" generates a different pattern than "awk"
		return "^" + regexp.QuoteMeta(prefix) + ".*"
	}
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
			"gsh: Do I have your permission to run the following command?",
			fmt.Sprintf("%s\n\n%s", command, reason),
		)
	}
	if confirmResponse == "n" {
		return failedToolResponse("User declined this request")
	} else if confirmResponse == "manage" {
		// User chose "manage" - show permissions menu for command prefixes
		menuResponse, err := ShowPermissionsMenu(logger, command)
		if err != nil {
			logger.Error("Failed to show permissions menu", zap.Error(err))
			return failedToolResponse("Failed to show permissions menu")
		}

		// Process the menu response
		if menuResponse == "n" {
			return failedToolResponse("User declined this request")
		} else if menuResponse == "manage" {
			// User selected specific permissions - the permissions menu has already saved
			// the enabled permissions to authorized_commands, so we just continue
			logger.Info("Permissions have been saved by the permissions menu")
		} else if menuResponse != "y" {
			return failedToolResponse(fmt.Sprintf("User declined this request: %s", menuResponse))
		}
		// If menuResponse == "y", continue with execution
	} else if confirmResponse == "always" {
		// Legacy support for "always" - treat as "manage" for backward compatibility
		regexPatterns, err := GenerateCompoundCommandRegex(command)
		if err != nil {
			logger.Error("Failed to generate regex patterns for compound command", zap.Error(err))
			// Fall back to single command pattern generation
			regexPattern := GenerateCommandRegex(command)
			err = environment.AppendToAuthorizedCommands(regexPattern)
			if err != nil {
				logger.Error("Failed to append command to authorized_commands file", zap.Error(err))
			}
		} else {
			// Save all generated patterns
			for _, pattern := range regexPatterns {
				err = environment.AppendToAuthorizedCommands(pattern)
				if err != nil {
					logger.Error("Failed to append command pattern to authorized_commands file",
						zap.String("pattern", pattern), zap.Error(err))
					// Continue with other patterns even if one fails
				}
			}
		}
	} else if confirmResponse != "y" {
		return failedToolResponse(fmt.Sprintf("User declined this request: %s", confirmResponse))
	}

	fmt.Print(gline.RESET_CURSOR_COLUMN + environment.GetPrompt(runner, logger) + command + "\n")

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	multiOut := io.MultiWriter(os.Stdout, outBuf)
	multiErr := io.MultiWriter(os.Stderr, errBuf)

	interp.StdIO(os.Stdin, multiOut, multiErr)(runner)
	defer interp.StdIO(os.Stdin, os.Stdout, os.Stderr)(runner)

	historyEntry, _ := historyManager.StartCommand(command, environment.GetPwd(runner))

	err = runner.Run(context.Background(), prog)

	exitCode := -1
	if err != nil {
		status, ok := interp.IsExitStatus(err)
		if ok {
			exitCode = int(status)
		} else {
			return failedToolResponse(fmt.Sprintf("Error running command: %s", err))
		}
	} else {
		exitCode = 0
	}
	stdout := outBuf.String()
	stderr := errBuf.String()

	historyManager.FinishCommand(historyEntry, exitCode)

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
