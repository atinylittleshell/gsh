package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/analytics"
	"github.com/atinylittleshell/gsh/internal/appupdate"
	"github.com/atinylittleshell/gsh/internal/bash"
	"github.com/atinylittleshell/gsh/internal/completion"
	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/evaluate"
	"github.com/atinylittleshell/gsh/internal/filesystem"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"go.uber.org/zap"
	"golang.org/x/term"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

var BUILD_VERSION = "dev"

//go:embed .gshrc.default
var DEFAULT_VARS []byte

var command = flag.String("c", "", "run a command")
var loginShell = flag.Bool("l", false, "run as a login shell")

var helpFlag = flag.Bool("h", false, "display help information")
var versionFlag = flag.Bool("ver", false, "display build version")

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(BUILD_VERSION)
		return
	}

	if *helpFlag {
		fmt.Println("Usage of gsh:")
		flag.PrintDefaults()
		return
	}

	// Initialize the history manager
	historyManager, err := initializeHistoryManager()
	if err != nil {
		panic("failed to initialize history manager")
	}

	// Initialize the analytics manager
	analyticsManager, err := initializeAnalyticsManager()
	if err != nil {
		panic("failed to initialize analytics manager")
	}

	// Initialize the completion manager
	completionManager := initializeCompletionManager()

	// Initialize the shell interpreter
	runner, err := initializeRunner(analyticsManager, historyManager, completionManager)
	if err != nil {
		panic(err)
	}

	// Initialize the logger
	logger, err := initializeLogger(runner)
	if err != nil {
		panic(err)
	}
	defer logger.Sync() // Flush any buffered log entries

	analyticsManager.Logger = logger

	logger.Info("-------- new gsh session --------", zap.Any("args", os.Args))

	appupdate.HandleSelfUpdate(
		BUILD_VERSION,
		logger,
		filesystem.DefaultFileSystem{},
		core.DefaultUserPrompter{},
		appupdate.DefaultUpdater{},
	)

	// Start running
	err = run(runner, historyManager, analyticsManager, completionManager, logger)

	// Handle exit status
	if code, ok := interp.IsExitStatus(err); ok {
		os.Exit(int(code))
	}

	if err != nil {
		logger.Error("unhandled error", zap.Error(err))
		os.Exit(1)
	}
}

func run(
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	analyticsManager *analytics.AnalyticsManager,
	completionManager *completion.CompletionManager,
	logger *zap.Logger,
) error {
	ctx := context.Background()

	// gsh -c "echo hello"
	if *command != "" {
		return bash.RunBashScriptFromReader(ctx, runner, strings.NewReader(*command), "gsh")
	}

	// gsh
	if flag.NArg() == 0 {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return core.RunInteractiveShell(ctx, runner, historyManager, analyticsManager, completionManager, logger)
		}

		return bash.RunBashScriptFromReader(ctx, runner, os.Stdin, "gsh")
	}

	// gsh script.sh or gsh script.gsh
	for _, filePath := range flag.Args() {
		if isGshScript(filePath) {
			if err := runGshScript(ctx, filePath, logger); err != nil {
				return err
			}
		} else {
			if err := bash.RunBashScriptFromFile(ctx, runner, filePath); err != nil {
				return err
			}
		}
	}

	return nil
}

// isGshScript checks if a file is a .gsh script
func isGshScript(filePath string) bool {
	return strings.HasSuffix(filePath, ".gsh")
}

// runGshScript executes a .gsh script file
func runGshScript(ctx context.Context, filePath string, logger *zap.Logger) error {
	// Read the script file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// Skip shebang line if present
	script := string(content)
	if strings.HasPrefix(script, "#!") {
		if idx := strings.Index(script, "\n"); idx >= 0 {
			script = script[idx+1:]
		}
	}

	// Lex the script
	l := lexer.New(script)

	// Parse the script
	p := parser.New(l)
	program := p.ParseProgram()

	// Check for parsing errors
	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			fmt.Fprintf(os.Stderr, "Parse error: %s\n", err)
		}
		return fmt.Errorf("failed to parse script")
	}

	// Create interpreter
	interp := interpreter.New()
	defer interp.Close()

	// Execute the script
	_, err = interp.Eval(program)
	if err != nil {
		// Print the error to stderr for better user experience
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return fmt.Errorf("runtime error: %w", err)
	}

	return nil
}

func initializeLogger(runner *interp.Runner) (*zap.Logger, error) {
	logLevel := environment.GetLogLevel(runner)
	if BUILD_VERSION == "dev" {
		logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	if environment.ShouldCleanLogFile(runner) {
		os.Remove(core.LogFile())
	}

	// Initialize the logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = logLevel
	loggerConfig.OutputPaths = []string{
		core.LogFile(),
	}
	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func initializeHistoryManager() (*history.HistoryManager, error) {
	historyManager, err := history.NewHistoryManager(core.HistoryFile())
	if err != nil {
		return nil, err
	}

	return historyManager, nil
}

func initializeAnalyticsManager() (*analytics.AnalyticsManager, error) {
	analyticsManager, err := analytics.NewAnalyticsManager(core.AnalyticsFile())
	if err != nil {
		return nil, err
	}

	return analyticsManager, nil
}

func initializeCompletionManager() *completion.CompletionManager {
	return completion.NewCompletionManager()
}

// initializeRunner loads the shell configuration files and sets up the interpreter.
func initializeRunner(analyticsManager *analytics.AnalyticsManager, historyManager *history.HistoryManager, completionManager *completion.CompletionManager) (*interp.Runner, error) {
	shellPath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	env := expand.ListEnviron(append(
		os.Environ(),
		fmt.Sprintf("SHELL=%s", shellPath),
		fmt.Sprintf("GSH_BUILD_VERSION=%s", BUILD_VERSION),
	)...)

	runner, err := interp.New(
		interp.Interactive(true),
		interp.Env(env),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandlers(
			analytics.NewAnalyticsCommandHandler(analyticsManager),
			evaluate.NewEvaluateCommandHandler(analyticsManager),
			history.NewHistoryCommandHandler(historyManager),
			completion.NewCompleteCommandHandler(completionManager),
		),
	)
	if err != nil {
		panic(err)
	}

	// load default vars
	if err := bash.RunBashScriptFromReader(
		context.Background(),
		runner,
		bytes.NewReader(DEFAULT_VARS),
		"gsh",
	); err != nil {
		panic(err)
	}

	configFiles := []string{
		filepath.Join(core.HomeDir(), ".gshrc"),
		filepath.Join(core.HomeDir(), ".gshenv"),
	}

	// Check if this is a login shell
	if *loginShell || strings.HasPrefix(os.Args[0], "-") {
		// Prepend .gsh_profile to the list of config files
		configFiles = append(
			[]string{
				"/etc/profile",
				filepath.Join(core.HomeDir(), ".gsh_profile"),
			},
			configFiles...,
		)
	}

	for _, configFile := range configFiles {
		if stat, err := os.Stat(configFile); err == nil && stat.Size() > 0 {
			if err := bash.RunBashScriptFromFile(context.Background(), runner, configFile); err != nil {
				fmt.Fprintf(os.Stderr, "failed to load %s: %v\n", configFile, err)
			}
		}
	}

	analyticsManager.Runner = runner

	return runner, nil
}
