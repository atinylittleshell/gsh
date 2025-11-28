package bash

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// For testing purposes
var typesetPrintf = func(format string, a ...any) (int, error) {
	return fmt.Printf(format, a...)
}

// Global runner reference that can be set after initialization
// NOTE: This global variable pattern is intentionally used here due to the constraints
// of the interp.ExecHandlerFunc signature, which doesn't allow passing additional context.
// The runner must be available to the handler closure, and since handlers are registered
// before the runner is created, we use this global state approach.
//
// Testability is maintained through SetTypesetRunner() which allows tests to inject
// mock runners. The tight coupling is a necessary trade-off for the framework integration.
var globalRunner *interp.Runner

// SetTypesetRunner sets the global runner reference for the typeset command handler
// This function enables dependency injection for testing purposes, allowing tests
// to provide their own runner instances without modifying global application state.
func SetTypesetRunner(runner *interp.Runner) {
	globalRunner = runner
}

// NewTypesetCommandHandler creates a new ExecHandler for the typeset and declare commands
func NewTypesetCommandHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Handle both typeset and declare commands
			if args[0] != "typeset" && args[0] != "declare" && args[0] != "gsh_typeset" {
				return next(ctx, args)
			}

			// Use the global runner reference
			if globalRunner == nil {
				return fmt.Errorf("typeset: runner not initialized")
			}

			// Now we have access to the runner, so we can implement the real functionality
			return handleTypesetCommand(globalRunner, args)
		}
	}
}

func handleTypesetCommand(runner *interp.Runner, args []string) error {
	// Parse options - skip the command name (args[0])
	var (
		listFunctions     bool // -f: list function definitions
		listFunctionNames bool // -F: list function names only
		listVariables     bool // -p: list variables with attributes
	)

	// If no options provided, default to listing variables
	if len(args) <= 1 {
		listVariables = true
	}

	// Parse command-line options - start from args[1] to skip command name
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			// Non-option argument, stop parsing options
			break
		}

		// Handle combined options like -fp
		for _, ch := range arg[1:] {
			switch ch {
			case 'f':
				listFunctions = true
			case 'F':
				listFunctionNames = true
			case 'p':
				listVariables = true
			default:
				return fmt.Errorf("typeset: -%c: invalid option", ch)
			}
		}
	}

	// If no specific option was set, default to listing variables
	if !listFunctions && !listFunctionNames && !listVariables {
		listVariables = true
	}

	// Handle function listing
	if listFunctions {
		return printFunctionDefinitions(runner)
	}

	if listFunctionNames {
		return printFunctionNames(runner)
	}

	// Handle variable listing
	if listVariables {
		return printVariables(runner)
	}

	return nil
}

// printFunctionDefinitions prints all function definitions in bash-compatible format
func printFunctionDefinitions(runner *interp.Runner) error {
	if runner.Funcs == nil {
		return nil
	}

	// Get all function names and sort them
	names := make([]string, 0, len(runner.Funcs))
	for name := range runner.Funcs {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print each function definition
	for _, name := range names {
		fn := runner.Funcs[name]
		if fn == nil {
			continue
		}

		// Format: function_name () { body }
		_, _ = typesetPrintf("%s () {\n", name)

		// Print the function body
		printFunctionBody(fn)

		_, _ = typesetPrintf("}\n")
	}

	return nil
}

// printFunctionBody prints the statements in a function body
func printFunctionBody(fn *syntax.Stmt) {
	if fn == nil {
		return
	}

	// Use the syntax printer to format the function body
	// This will give us a bash-compatible representation
	var buf strings.Builder
	_ = syntax.NewPrinter().Print(&buf, fn)

	// Indent each line of the body
	lines := strings.Split(buf.String(), "\n")
	for _, line := range lines {
		if line != "" {
			_, _ = typesetPrintf("    %s\n", line)
		}
	}
}

// printFunctionNames prints just the function names (one per line)
func printFunctionNames(runner *interp.Runner) error {
	if runner.Funcs == nil {
		return nil
	}

	// Get all function names and sort them
	names := make([]string, 0, len(runner.Funcs))
	for name := range runner.Funcs {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print each function name
	for _, name := range names {
		_, _ = typesetPrintf("declare -f %s\n", name)
	}

	return nil
}

// printVariables prints all variables with their values in bash-compatible format
func printVariables(runner *interp.Runner) error {
	if runner.Vars == nil {
		return nil
	}

	// Collect all variable names
	var names []string
	for name := range runner.Vars {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print each variable
	for _, name := range names {
		vr, ok := runner.Vars[name]
		if !ok {
			continue
		}

		value := vr.String()

		// Determine if the variable is exported
		exported := vr.Exported

		// Format the output
		if exported {
			_, _ = typesetPrintf("declare -x %s=%q\n", name, value)
		} else {
			_, _ = typesetPrintf("declare -- %s=%q\n", name, value)
		}
	}

	return nil
}
