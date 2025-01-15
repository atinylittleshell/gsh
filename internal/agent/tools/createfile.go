package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/utils"
	"github.com/atinylittleshell/gsh/pkg/gline"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

var CreateFileToolDefinition = openai.Tool{
	Type: "function",
	Function: &openai.FunctionDefinition{
		Name:        "create_file",
		Description: `Create a file with the specified content.`,
		Parameters: utils.GenerateJsonSchema(struct {
			Path    string `json:"path" description:"Absolute path to the file" required:"true"`
			Content string `json:"content" description:"The content to write to the file" required:"true"`
		}{}),
	},
}

func CreateFileTool(runner *interp.Runner, logger *zap.Logger, params map[string]any) string {
	path, ok := params["path"].(string)
	if !ok {
		logger.Error("The create_file tool failed to parse parameter 'path'")
		return failedToolResponse("The create_file tool failed to parse parameter 'path'")
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(environment.GetPwd(runner), path)
	}

	content, ok := params["content"].(string)
	if !ok {
		logger.Error("The create_file tool failed to parse parameter 'content'")
		return failedToolResponse("The create_file tool failed to parse parameter 'content'")
	}

	privacySafePath := utils.HideHomeDirPath(runner, path)

	fmt.Print(gline.RESET_CURSOR_COLUMN + fmt.Sprintf("%s\n\n%s\n", privacySafePath, content) + gline.RESET_CURSOR_COLUMN)

	confirmResponse := userConfirmation(
		logger,
		"gsh: Do I have your permission to create the file with the content shown above?",
		"",
	)
	if confirmResponse == "n" {
		return failedToolResponse("User declined this request")
	} else if confirmResponse != "y" {
		return failedToolResponse(fmt.Sprintf("User declined this request: %s", confirmResponse))
	}



	file, err := os.Create(path)
	defer file.Close()

	if err != nil {
		logger.Error("create_file tool failed to create file", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error creating file: %s", err))
	}

	_, err = file.WriteString(content)
	if err != nil {
		logger.Error("create_file tool received invalid content", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error writing to file: %s", err))
	}

	return fmt.Sprintf("File successfully created at %s", path)
}
