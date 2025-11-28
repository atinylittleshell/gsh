package completion

import (
	"context"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// NewCompgenCommandHandler creates a new ExecHandler for the compgen command
func NewCompgenCommandHandler(runner *interp.Runner) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 || args[0] != "compgen" {
				return next(ctx, args)
			}

			// Handle the compgen command
			return handleCompgenCommand(ctx, runner, args[1:])
		}
	}
}

func handleCompgenCommand(ctx context.Context, runner *interp.Runner, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("compgen: no options specified")
	}

	// Parse options
	var (
		wordList    string
		functionName string
		word        string // The word to generate completions for
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-W":
			if i+1 >= len(args) {
				return fmt.Errorf("option -W requires a word list")
			}
			i++
			wordList = args[i]
		case "-F":
			if i+1 >= len(args) {
				return fmt.Errorf("option -F requires a function name")
			}
			i++
			functionName = args[i]
		default:
			if !strings.HasPrefix(arg, "-") {
				word = arg
				break
			}
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	// Generate completions based on the options
	if wordList != "" {
		return generateWordListCompletions(word, wordList)
	}

	if functionName != "" {
		return generateFunctionCompletions(ctx, runner, functionName, word)
	}

	return fmt.Errorf("compgen: no completion type specified")
}

func generateWordListCompletions(word string, wordList string) error {
	words := strings.Fields(wordList)
	for _, w := range words {
		if word == "" || strings.HasPrefix(w, word) {
			_, _ = printf("%s\n", w)
		}
	}
	return nil
}

func generateFunctionCompletions(ctx context.Context, runner *interp.Runner, functionName string, word string) error {
	// Create a completion function
	fn := NewCompletionFunction(functionName, runner)

	// Execute the function with the word as argument
	completions, err := fn.Execute(ctx, []string{word})
	if err != nil {
		return fmt.Errorf("failed to execute completion function: %w", err)
	}

	// Print the completions
	for _, completion := range completions {
		if word == "" || strings.HasPrefix(completion, word) {
			_, _ = printf("%s\n", completion)
		}
	}
	return nil
}