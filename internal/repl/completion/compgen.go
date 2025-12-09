package completion

import (
	"context"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// CompletionFunction represents a bash completion function.
type CompletionFunction struct {
	Name   string
	Runner *interp.Runner
}

// NewCompletionFunction creates a new CompletionFunction.
func NewCompletionFunction(name string, runner *interp.Runner) *CompletionFunction {
	return &CompletionFunction{
		Name:   name,
		Runner: runner,
	}
}

// Execute runs the completion function with the given arguments.
func (f *CompletionFunction) Execute(ctx context.Context, args []string) ([]string, error) {
	script := fmt.Sprintf(`
		# Set up completion environment
		COMP_LINE=%q
		COMP_POINT=%d
		COMP_WORDS=(%s)
		COMP_CWORD=%d

		# Initialize empty COMPREPLY
		COMPREPLY=()

		# Call the completion function
		%s
	`,
		strings.Join(args, " "),
		len(strings.Join(args, " ")),
		strings.Join(args, " "),
		len(args)-1,
		f.Name,
	)

	// Parse and execute the script
	file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse completion script: %w", err)
	}

	if err := f.Runner.Run(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to execute completion function: %w", err)
	}

	// Get COMPREPLY from the runner's variables
	compreply, ok := f.Runner.Vars["COMPREPLY"]
	if !ok {
		return []string{}, nil
	}

	if compreply.Kind != expand.Indexed {
		return []string{}, nil
	}

	// Get all elements of the array
	results := compreply.List
	return results, nil
}

// NewCompgenCommandHandler creates a new ExecHandler for the compgen command.
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
		wordList     string
		functionName string
		word         string // The word to generate completions for
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
			fmt.Printf("%s\n", w)
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
			fmt.Printf("%s\n", completion)
		}
	}
	return nil
}
