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
	fmt.Print(gline.RESET_CURSOR_COLUMN + environment.GetAgentPrompt(runner, logger) + command + "\n")

	// Check if the command matches any pre-approved patterns using secure compound command validation
	approvedPatterns := environment.GetApprovedBashCommandRegex(runner, logger)
	isPreApproved := false
	for _, pattern := range approvedPatterns {
		matched, err := regexp.MatchString(pattern, command)
		if err == nil && matched {
			isPreApproved = true
			break
		}
	}

	var confirmResponse string
	if isPreApproved {
		confirmResponse = "y"
	} else {
		confirmResponse = userConfirmation(
			logger,
			"gsh: Do I have your permission to run this command?",
			reason, // Only pass reason, not command (already displayed)
		)
	}
	if confirmResponse == "n" {
		return failedToolResponse("User declined this request")
	} else if confirmResponse != "y" {
		return failedToolResponse(fmt.Sprintf("User declined this request: %s", confirmResponse))
	}

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
