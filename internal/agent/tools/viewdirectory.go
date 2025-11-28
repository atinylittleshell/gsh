package tools

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/utils"
	"github.com/atinylittleshell/gsh/pkg/gline"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

const (
	MAX_DEPTH = 2
)

var ViewDirectoryToolDefinition = openai.Tool{
	Type: "function",
	Function: &openai.FunctionDefinition{
		Name:        "view_directory",
		Description: `View the content in a directory up to 2 levels deep.`,
		Parameters: utils.GenerateJsonSchema(struct {
			Path string `json:"path" description:"Absolute path to the directory" required:"true"`
		}{}),
	},
}

func ViewDirectoryTool(runner *interp.Runner, logger *zap.Logger, params map[string]any) string {
	path, ok := params["path"].(string)
	if !ok {
		logger.Error("The view_directory tool failed to parse parameter 'path'")
		return failedToolResponse("The view_directory tool failed to parse parameter 'path'")
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(environment.GetPwd(runner), path)
	}

	var buf bytes.Buffer
	writer := io.StringWriter(&buf)

	agentName := environment.GetAgentName(runner)
	printToolMessage(fmt.Sprintf("%s: I'm viewing the following directory:", agentName))
	fmt.Print(gline.RESET_CURSOR_COLUMN + utils.HideHomeDirPath(runner, path) + "\n")

	err := walkDir(logger, writer, path, 1)
	if err != nil {
		return failedToolResponse(fmt.Sprintf("Error reading directory: %s", err))
	}

	return buf.String()
}

func walkDir(logger *zap.Logger, writer io.StringWriter, dir string, depth int) error {
	if depth > MAX_DEPTH {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Error("Error reading directory", zap.String("dir", dir), zap.Error(err))
		return err
	}

	// Print each entry, and if it's a directory, recurse into it (unless at max depth).
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			_, _ = writer.WriteString(fullPath + string(os.PathSeparator) + "\n")

			if depth < MAX_DEPTH {
				_ = walkDir(logger, writer, fullPath, depth+1)
			}
		} else {
			_, _ = writer.WriteString(fullPath + "\n")
		}
	}
	return nil
}
