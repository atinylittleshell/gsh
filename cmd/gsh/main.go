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
	"github.com/atinylittleshell/gsh/internal/bash"
	"github.com/atinylittleshell/gsh/internal/coach"
	"github.com/atinylittleshell/gsh/internal/completion"
	"github.com/atinylittleshell/gsh/internal/config"
	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/evaluate"
	"github.com/atinylittleshell/gsh/internal/history"
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
var rcFile = flag.String("rcfile", "", "use a custom rc file instead of ~/.gshrc")
var strictConfig = flag.Bool("strict-config", false, "fail fast if configuration files contain errors (like bash 'set -e')")

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

	// Initialize the stderr capturer
	stderrCapturer := core.NewStderrCapturer(os.Stderr)

	// Initialize the shell interpreter
	runner, err := initializeRunner(analyticsManager, historyManager, completionManager, stderrCapturer)
	if err != nil {
		panic(err)
	}

	// Register session config override getter so environment package can access config overrides
	environment.SetSessionConfigOverrideGetter(config.GetSessionOverride)

	// Initialize the logger
	logger, err := initializeLogger(runner)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync() // Flush any buffered log entries
	}()

	analyticsManager.Logger = logger

	logger.Info("-------- new gsh session --------", zap.Any("args", os.Args))

	// Initialize the coach manager (uses same database as history)
	coachManager, err := coach.NewCoachManager(historyManager.GetDB(), historyManager, runner, logger)
	if err != nil {
		logger.Warn("failed to initialize coach manager", zap.Error(err))
		// Coach is optional, continue without it
		coachManager = nil
	}

	// Start running
	err = run(runner, historyManager, analyticsManager, completionManager, coachManager, logger, stderrCapturer)

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
	coachManager *coach.CoachManager,
	logger *zap.Logger,
	stderrCapturer *core.StderrCapturer,
) error {
	ctx := context.Background()

	// gsh -c "echo hello"
	if *command != "" {
		return bash.RunBashScriptFromReader(ctx, runner, strings.NewReader(*command), "gsh")
	}

	// gsh
	if flag.NArg() == 0 {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return core.RunInteractiveShell(ctx, runner, historyManager, analyticsManager, completionManager, coachManager, logger, stderrCapturer)
		}

		return bash.RunBashScriptFromReader(ctx, runner, os.Stdin, "gsh")
	}

	// gsh script.sh
	for _, filePath := range flag.Args() {
		if err := bash.RunBashScriptFromFile(ctx, runner, filePath); err != nil {
			return err
		}
	}

	return nil
}

func initializeLogger(runner *interp.Runner) (*zap.Logger, error) {
	logLevel := environment.GetLogLevel(runner)
	if BUILD_VERSION == "dev" {
		logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	if environment.ShouldCleanLogFile(runner) {
		_ = os.Remove(core.LogFile())
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
func initializeRunner(analyticsManager *analytics.AnalyticsManager, historyManager *history.HistoryManager, completionManager *completion.CompletionManager, stderrCapturer *core.StderrCapturer) (*interp.Runner, error) {
	shellPath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	// Create a dynamic environment that can include GSH variables
	dynamicEnv := environment.NewDynamicEnviron()
	// Set initial system environment variables
	dynamicEnv.UpdateSystemEnv()
	// Add GSH-specific environment variables
	dynamicEnv.UpdateGSHVar("SHELL", shellPath)
	dynamicEnv.UpdateGSHVar("GSH_BUILD_VERSION", BUILD_VERSION)
	env := expand.Environ(dynamicEnv)

	var runner *interp.Runner

	// Create interpreter with all necessary configuration in a single call
	runner, err = interp.New(
		interp.Interactive(true),
		interp.Env(env),
		interp.StdIO(os.Stdin, os.Stdout, stderrCapturer),
		interp.ExecHandlers(
			bash.NewTypesetCommandHandler(),
			bash.SetBuiltinHandler(),
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

	var configFiles []string

	// If custom rcfile is provided, use it instead of the default ones
	if *rcFile != "" {
		configFiles = []string{*rcFile}
	} else {
		configFiles = []string{
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
	}

	for _, configFile := range configFiles {
		if stat, err := os.Stat(configFile); err == nil && stat.Size() > 0 {
			if err := bash.RunBashScriptFromFile(context.Background(), runner, configFile); err != nil {
				// Enhanced error reporting with context
				fmt.Fprintf(os.Stderr, "Configuration file %s contains errors: %v\n", configFile, err)

				if *strictConfig {
					// In strict mode (like bash 'set -e'), fail fast on configuration errors
					return nil, fmt.Errorf("aborting due to configuration error in %s: %w", configFile, err)
				}
				// In permissive mode (default), continue despite configuration errors
				// This maintains backward compatibility while providing better visibility
			}
			// Configuration loaded successfully in permissive mode
		}
		// File not found or empty - this is normal behavior, not an error
	}

	// Sync gsh variables to system environment so they're visible to 'env' command
	environment.SyncVariablesToEnv(runner)

	analyticsManager.Runner = runner

	// Set the global runner for the typeset command handler
	bash.SetTypesetRunner(runner)

	return runner, nil
}
