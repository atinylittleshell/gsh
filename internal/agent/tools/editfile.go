package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atinylittleshell/gsh/internal/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var EditFileToolDefinition = openai.Tool{
	Type: "function",
	Function: &openai.FunctionDefinition{
		Name:        "edit_file",
		Description: `Edit the content of a file.`,
		Parameters: utils.GenerateJsonSchema(struct {
			Path   string `json:"path" jsonschema_description:"Absolute path to the file" jsonschema_required:"true"`
			OldStr string `json:"old_str" jsonschema_description:"The old string in the file to be replaced. This must be a unique chunk in the file, ideally complete lines." jsonschema_required:"true"`
			NewStr string `json:"new_str" jsonschema_description:"The new string that will replace the old one" jsonschema_required:"true"`
		}{}),
	},
}

func EditFileTool(runner *interp.Runner, logger *zap.Logger, params map[string]any) string {
	path, ok := params["path"].(string)
	if !ok {
		logger.Error("The create_file tool failed to parse parameter 'path'")
		return failedToolResponse("The create_file tool failed to parse parameter 'path'")
	}

	oldStr, ok := params["old_str"].(string)
	if !ok {
		logger.Error("The create_file tool failed to parse parameter 'old_str'")
		return failedToolResponse("The create_file tool failed to parse parameter 'old_str'")
	}

	newStr, ok := params["new_str"].(string)
	if !ok {
		logger.Error("The create_file tool failed to parse parameter 'new_str'")
		return failedToolResponse("The create_file tool failed to parse parameter 'new_str'")
	}

	file, err := os.Open(path)
	if err != nil {
		logger.Error("edit_file tool received invalid path", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error opening file: %s", err))
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		logger.Error("edit_file tool failed to read file", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error reading file: %s", err))
	}

	contents := buf.String()

	if strings.Count(contents, oldStr) != 1 {
		return failedToolResponse("The old string must be unique in the file")
	}

	newContents := strings.ReplaceAll(contents, oldStr, newStr)

	tmpFile, err := os.CreateTemp("", "gsh_edit_file_preview")
	if err != nil {
		logger.Error("edit_file tool failed to create temporary file", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error creating temporary file: %s", err))
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(newContents)
	if err != nil {
		logger.Error("edit_file tool failed to write to temporary file", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error writing to temporary file: %s", err))
	}

	diff, err := getDiff(runner, logger, path, tmpFile.Name())
	if err != nil {
		return failedToolResponse(fmt.Sprintf("Error generating diff: %s", err))
	}

	confirmResponse := userConfirmation(logger, "gsh: Do I have your permission to edit the following file?", diff)
	if confirmResponse == "n" {
		return failedToolResponse("User declined this request")
	} else if confirmResponse != "y" {
		return failedToolResponse(fmt.Sprintf("User declined this request: %s", confirmResponse))
	}

	fmt.Println(path)

	file, err = os.Create(path)
	if err != nil {
		logger.Error("edit_file tool failed to create file", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error creating file: %s", err))
	}

	_, err = file.WriteString(newContents)
	if err != nil {
		logger.Error("edit_file tool received invalid content", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error writing to file: %s", err))
	}

	return fmt.Sprintf("File successfully edited at %s", path)
}

func getDiff(runner *interp.Runner, logger *zap.Logger, file1, file2 string) (string, error) {
	command := fmt.Sprintf("git diff --color=always --no-index %s %s", file1, file2)

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		logger.Error("Failed to preview code edits", zap.Error(err))
		return "", err
	}

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	outWriter := io.Writer(outBuf)
	errWriter := io.Writer(errBuf)

	subShell := runner.Subshell()
	interp.StdIO(nil, outWriter, errWriter)(subShell)

	err = subShell.Run(context.Background(), prog)

	exitCode := -1
	if err != nil {
		status, ok := interp.IsExitStatus(err)
		if ok {
			exitCode = int(status)
		}
	} else {
		exitCode = 0
	}

	if exitCode == 128 {
		return "", fmt.Errorf("Error running git diff command: %s", errBuf.String())
	}

	result := strings.ReplaceAll(outBuf.String(), "b"+file2, "")
	return result, nil
}
