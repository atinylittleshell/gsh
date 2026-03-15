package utils

import (
	"fmt"
	"strings"

	"github.com/kunchenguid/gsh/internal/environment"
	"github.com/sashabaranov/go-openai/jsonschema"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

func GenerateJsonSchema(value any) *jsonschema.Definition {
	result, err := jsonschema.GenerateSchemaForType(value)
	if err != nil {
		panic(err)
	}
	return result
}

func ComposeContextText(context *map[string]string, contextTypes []string, logger *zap.Logger) string {
	contextText := ""
	if context == nil {
		return contextText
	}

	if len(contextTypes) == 0 {
		return contextText
	}

	for _, contextType := range contextTypes {
		text, ok := (*context)[contextType]
		if !ok {
			logger.Warn("context type not found", zap.String("context_type", contextType))
			continue
		}

		contextText += "\n" + text + "\n"
	}

	return contextText
}

func HideHomeDirPath(runner *interp.Runner, path string) string {
	homeDir := environment.GetHomeDir(runner)
	if homeDir == "" {
		return path
	}

	if strings.HasPrefix(path, homeDir) {
		return fmt.Sprintf("~%s", path[len(homeDir):])
	}

	return path
}
