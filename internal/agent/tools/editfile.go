package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/filesystem"
	"github.com/atinylittleshell/gsh/internal/utils"
	"github.com/atinylittleshell/gsh/pkg/gline"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

var EditFileToolDefinition = openai.Tool{
	Type: "function",
	Function: &openai.FunctionDefinition{
		Name:        "edit_file",
		Description: `Edit the content of a file.`,
		Parameters: utils.GenerateJsonSchema(struct {
			Path   string `json:"path" description:"Absolute path to the file" required:"true"`
			OldStr string `json:"old_str" description:"The old string in the file to be replaced. This must be a unique chunk in the file, ideally complete lines." required:"true"`
			NewStr string `json:"new_str" description:"The new string that will replace the old one" required:"true"`
		}{}),
	},
}

type editFileParams struct {
	path   string
	oldStr string
	newStr string
}

func validateAndExtractParams(runner *interp.Runner, logger *zap.Logger, params map[string]any) (*editFileParams, string) {
	path, ok := params["path"].(string)
	if !ok {
		logger.Error("The edit_file tool failed to parse parameter 'path'")
		return nil, "The edit_file tool failed to parse parameter 'path'"
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(environment.GetPwd(runner), path)
	}

	oldStr, ok := params["old_str"].(string)
	if !ok {
		logger.Error("The edit_file tool failed to parse parameter 'old_str'")
		return nil, "The edit_file tool failed to parse parameter 'old_str'"
	}

	newStr, ok := params["new_str"].(string)
	if !ok {
		logger.Error("The edit_file tool failed to parse parameter 'new_str'")
		return nil, "The edit_file tool failed to parse parameter 'new_str'"
	}

	return &editFileParams{
		path:   path,
		oldStr: oldStr,
		newStr: newStr,
	}, ""
}

func readFileContents(logger *zap.Logger, fs filesystem.FileSystem, path string) (string, string) {
	content, err := fs.ReadFile(path)
	if err != nil {
		logger.Error("edit_file tool failed to read file", zap.Error(err))
		return "", fmt.Sprintf("Error reading file: %s", err)
	}

	return content, ""
}

func validateAndReplaceContent(content, oldStr, newStr string) (string, string) {
	if strings.Count(content, oldStr) != 1 {
		return "", "The old string must be unique in the file"
	}

	return strings.ReplaceAll(content, oldStr, newStr), ""
}

func previewAndConfirm(runner *interp.Runner, logger *zap.Logger, path string, newContent string) string {
	tmpFile, err := os.CreateTemp("", "gsh_edit_file_preview")
	if err != nil {
		logger.Error("edit_file tool failed to create temporary file", zap.Error(err))
		return fmt.Sprintf("Error creating temporary file: %s", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			logger.Warn("failed to clean up temp file", zap.Error(err))
		}
	}()

	_, err = tmpFile.WriteString(newContent)
	if err != nil {
		logger.Error("edit_file tool failed to write to temporary file", zap.Error(err))
		return fmt.Sprintf("Error writing to temporary file: %s", err)
	}

	diff, err := getDiff(runner, logger, path, tmpFile.Name())
	if err != nil {
		return fmt.Sprintf("Error generating diff: %s", err)
	}

	fmt.Print(gline.RESET_CURSOR_COLUMN + diff + "\n" + gline.RESET_CURSOR_COLUMN)

	agentName := environment.GetAgentName(runner)
	confirmResponse := userConfirmation(logger, runner, fmt.Sprintf("%s: Do I have your permission to make the edit proposed above?", agentName), "")
	if confirmResponse == "n" {
		return "User declined this request"
	} else if confirmResponse != "y" {
		return fmt.Sprintf("User declined this request: %s", confirmResponse)
	}

	return ""
}

func writeFile(logger *zap.Logger, fs filesystem.FileSystem, path string, content string) string {
	err := fs.WriteFile(path, content)
	if err != nil {
		logger.Error("edit_file tool failed to write file", zap.Error(err))
		return fmt.Sprintf("Error writing to file: %s", err)
	}

	return ""
}

func EditFileTool(runner *interp.Runner, logger *zap.Logger, params map[string]any) string {
	fs := filesystem.DefaultFileSystem{}

	fileParams, errMsg := validateAndExtractParams(runner, logger, params)
	if errMsg != "" {
		return failedToolResponse(errMsg)
	}

	content, errMsg := readFileContents(logger, fs, fileParams.path)
	if errMsg != "" {
		return failedToolResponse(errMsg)
	}

	newContent, errMsg := validateAndReplaceContent(content, fileParams.oldStr, fileParams.newStr)
	if errMsg != "" {
		return failedToolResponse(errMsg)
	}

	if errMsg = previewAndConfirm(runner, logger, fileParams.path, newContent); errMsg != "" {
		return failedToolResponse(errMsg)
	}

	fmt.Print(gline.RESET_CURSOR_COLUMN + utils.HideHomeDirPath(runner, fileParams.path) + "\n")

	if errMsg = writeFile(logger, fs, fileParams.path, newContent); errMsg != "" {
		return failedToolResponse(errMsg)
	}

	return fmt.Sprintf("File successfully edited at %s", fileParams.path)
}
