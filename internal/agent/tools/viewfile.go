package tools

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/utils"
	"github.com/atinylittleshell/gsh/pkg/gline"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

const LINES_TO_READ = 100
const MAX_VIEW_SIZE = 16 * 1024 * 4 // roughly 16k tokens

var ViewFileToolDefinition = openai.Tool{
	Type: "function",
	Function: &openai.FunctionDefinition{
		Name:        "view_file",
		Description: fmt.Sprintf(`View the content of a text file, at most %d lines at a time. If the content is too large, tail will be truncated and replaced with <gsh:truncated />.`, LINES_TO_READ),
		Parameters: utils.GenerateJsonSchema(struct {
			Path      string `json:"path" description:"Absolute path to the file" required:"true"`
			StartLine int    `json:"start_line" description:"Optional. Line number to start viewing. The first line in the file has line number 1. If not provided, we will read from the beginning of the file." required:"false"`
		}{}),
	},
}

func ViewFileTool(runner *interp.Runner, logger *zap.Logger, params map[string]any) string {
	path, ok := params["path"].(string)
	if !ok {
		logger.Error("The view_file tool failed to parse parameter 'path'")
		return failedToolResponse("The view_file tool failed to parse parameter 'path'")
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(environment.GetPwd(runner), path)
	}

	startLine := 1
	startLineVal, startLineExists := params["start_line"]
	if startLineExists {
		startLineFloat, ok := startLineVal.(float64)
		if !ok {
			logger.Error("The view_file tool failed to parse parameter 'start_line'")
			return failedToolResponse("The view_file tool failed to parse parameter 'start_line'")
		}
		startLine = int(startLineFloat)
	}

	endLine := startLine + LINES_TO_READ

	file, err := os.Open(path)
	if err != nil {
		logger.Error("view_file tool received invalid path", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error opening file: %s", err))
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to close file", zap.Error(err))
		}
	}()

	agentName := environment.GetAgentName(runner)
	printToolMessage(fmt.Sprintf("%s: I'm reading the following file:", agentName))
	fmt.Print(gline.RESET_CURSOR_COLUMN + utils.HideHomeDirPath(runner, path) + "\n")

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		logger.Error("view_file tool received invalid path", zap.Error(err))
		return failedToolResponse(fmt.Sprintf("Error reading file: %s", err))
	}

	lines := strings.Split(buf.String(), "\n")
	if startLine < 1 {
		return failedToolResponse("start_line must be greater than or equal to 1")
	}
	if startLine > len(lines) {
		return failedToolResponse("start_line is greater than the number of lines in the file")
	}
	if endLine > len(lines)+1 {
		endLine = len(lines) + 1
	}

	// convert 1-based line numbers to 0-based indexes
	result := strings.Join(lines[startLine-1:endLine-1], "\n")
	if len(result) > MAX_VIEW_SIZE {
		return result[:MAX_VIEW_SIZE] + "\n<gsh:truncated />"
	}

	return result
}
