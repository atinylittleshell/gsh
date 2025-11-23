package evaluate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/atinylittleshell/gsh/internal/analytics"
	"mvdan.cc/sh/v3/interp"
)

func NewEvaluateCommandHandler(analyticsManager *analytics.AnalyticsManager) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			if args[0] != "gsh_evaluate" {
				return next(ctx, args)
			}

			// Default values
			limit := 100
			modelId := "" // empty string means use default from GetLLMClient
			iterations := 3

			// Parse flags and arguments
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "-h", "--help":
					printEvaluateHelp()
					return nil
				case "-l", "--limit":
					if i+1 < len(args) {
						if val, err := strconv.Atoi(args[i+1]); err == nil {
							limit = val
							i++ // skip the next argument since we consumed it
						}
					}
				case "-m", "--model":
					if i+1 < len(args) {
						modelId = args[i+1]
						i++ // skip the next argument since we consumed it
					}
				case "-i", "--iterations":
					if i+1 < len(args) {
						if val, err := strconv.Atoi(args[i+1]); err == nil {
							iterations = val
							i++ // skip the next argument since we consumed it
						}
					}
				}
			}

			return RunEvaluation(analyticsManager, limit, modelId, iterations)
		}
	}
}

func printEvaluateHelp() {
	help := []string{
		"Usage: gsh_evaluate [options]",
		"Evaluate how well the configured models work for you.",
		"",
		"Options:",
		"  -h, --help               display this help message",
		"  -l, --limit <number>     limit the number of entries to evaluate (default: 100)",
		"  -m, --model <model-id>   specify the model to use (default: use the default fast model)",
		"  -i, --iterations <number> number of times to repeat the evaluation (default: 3)",
	}
	fmt.Println(strings.Join(help, "\n"))
}
