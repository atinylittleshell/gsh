package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/analytics"
	"github.com/atinylittleshell/gsh/internal/analytics/telemetry"
	"github.com/atinylittleshell/gsh/internal/appupdate"
	"github.com/atinylittleshell/gsh/internal/bash"
	"github.com/atinylittleshell/gsh/internal/completion"
	"github.com/atinylittleshell/gsh/internal/core"
	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/evaluate"
	"github.com/atinylittleshell/gsh/internal/filesystem"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/repl"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"go.uber.org/zap"
	"golang.org/x/term"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

var BUILD_VERSION = "dev"

//go:embed defaults/*
var defaultConfigFS embed.FS

const defaultConfigPath = "defaults/init.gsh"

// Help text for the main command
const mainHelpText = `gsh - A battery-included, POSIX-compatible, generative shell.

USAGE:
  gsh [options]
  gsh <command> [options] [args...]

COMMANDS:
  run <script> [args...]        Execute a script file (.gsh or .sh)
  telemetry [status|on|off]     Manage anonymous usage telemetry

OPTIONS:
  -h, --help                    Display help information
  -v, --version                 Display version
  -l, --login                   Run as a login shell
      --repl-config <path>      Use custom REPL config (default: ~/.gsh/repl.gsh)

EXAMPLES:
  gsh                           Start interactive shell
  gsh --login                   Start as login shell
  gsh run script.gsh            Execute a gsh script
  gsh run deploy.sh             Execute a bash script
  gsh telemetry status          Check telemetry status
`

// Help text for the run subcommand
const runHelpText = `Execute a script file (.gsh or .sh)

USAGE:
  gsh run [options] <script> [args...]

OPTIONS:
  -h, --help                    Display help information

ARGUMENTS:
  <script>                      Path to the script file (.gsh or .sh)
  [args...]                     Arguments passed to the script

EXAMPLES:
  gsh run script.gsh            Execute a gsh script
  gsh run deploy.sh             Execute a bash script
  gsh run agent.gsh --verbose   Execute with arguments

SCRIPTING:
  Files with .gsh extension use the gsh scripting language for agentic
  workflows with MCP servers, AI models, and agents.

  For documentation and examples, see: https://github.com/atinylittleshell/gsh
`

// Help text for the telemetry subcommand
const telemetryHelpText = `Manage anonymous usage telemetry for gsh.

USAGE:
  gsh telemetry [command]

COMMANDS:
  status                        Show current telemetry status (default)
  on                            Enable telemetry
  off                           Disable telemetry

OPTIONS:
  -h, --help                    Display help information

ENVIRONMENT VARIABLES:
  GSH_NO_TELEMETRY=1            Disable telemetry via environment

WHAT WE COLLECT:
  - gsh version, OS, CPU architecture
  - Session duration
  - Feature usage counts
  - Error categories (not error messages)

WHAT WE NEVER COLLECT:
  - Commands, prompts, or any user input
  - File paths or filenames
  - API keys or environment variables
  - Error messages or stack traces
  - Any personally identifiable information

Learn more: https://github.com/atinylittleshell/gsh#telemetry
`

// CLI options for REPL mode
type replOptions struct {
	login      bool
	replConfig string
}

func main() {
	startTime := time.Now()

	// Parse command line
	args := os.Args[1:]

	// Check for help/version flags first (can appear anywhere)
	if containsHelpFlag(args) {
		// Determine which help to show based on subcommand
		printContextualHelp(args)
		return
	}

	if containsVersionFlag(args) {
		fmt.Println(BUILD_VERSION)
		return
	}

	// No args or only flags = REPL mode
	if len(args) == 0 || (len(args) > 0 && strings.HasPrefix(args[0], "-")) {
		opts := parseREPLOptions(args)
		runREPLMode(startTime, opts)
		return
	}

	// First non-flag arg is the subcommand
	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "run":
		runRunCommand(startTime, subargs)
	case "telemetry":
		runTelemetryCommand(subargs)
	default:
		fmt.Fprintf(os.Stderr, "gsh: unknown command: %s\n", subcommand)
		fmt.Fprintf(os.Stderr, "Run 'gsh --help' for usage.\n")
		os.Exit(1)
	}
}

// containsHelpFlag checks if args contain -h or --help (case insensitive)
func containsHelpFlag(args []string) bool {
	for _, arg := range args {
		lower := strings.ToLower(arg)
		if lower == "-h" || lower == "--help" {
			return true
		}
	}
	return false
}

// containsVersionFlag checks if args contain -V or --version (case insensitive)
func containsVersionFlag(args []string) bool {
	for _, arg := range args {
		lower := strings.ToLower(arg)
		if lower == "-v" || lower == "--version" {
			return true
		}
	}
	return false
}

// printContextualHelp prints help based on the subcommand context
func printContextualHelp(args []string) {
	// Find the subcommand (first non-flag arg)
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		switch arg {
		case "run":
			fmt.Print(runHelpText)
			return
		case "telemetry":
			fmt.Print(telemetryHelpText)
			return
		}
	}
	// No subcommand found, show main help
	fmt.Print(mainHelpText)
}

// parseREPLOptions parses flags for REPL mode
func parseREPLOptions(args []string) replOptions {
	opts := replOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "-l" || strings.ToLower(arg) == "--login":
			opts.login = true
		case strings.ToLower(arg) == "--repl-config":
			if i+1 < len(args) {
				i++
				opts.replConfig = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "gsh: --repl-config requires a path argument\n")
				os.Exit(1)
			}
		case strings.HasPrefix(strings.ToLower(arg), "--repl-config="):
			opts.replConfig = strings.SplitN(arg, "=", 2)[1]
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "gsh: unknown option: %s\n", arg)
				fmt.Fprintf(os.Stderr, "Run 'gsh --help' for usage.\n")
				os.Exit(1)
			}
		}
		i++
	}
	return opts
}

// runREPLMode starts the interactive REPL
func runREPLMode(startTime time.Time, opts replOptions) {
	// Initialize telemetry client
	telemetryClient, err := telemetry.NewClient(telemetry.Config{
		Version: BUILD_VERSION,
	})
	if err != nil {
		telemetryClient = nil
	}
	if telemetryClient != nil {
		defer telemetryClient.Close()
	}

	// Show migration message if upgrading from v0.x
	if appupdate.IsUpgradeFromV0() {
		fmt.Print(appupdate.GetMigrationMessage())
		fmt.Println()
	}

	// Update version marker to current version
	_ = appupdate.UpdateVersionMarker(BUILD_VERSION)

	// Show first-run notification if this is the first time (for telemetry)
	if telemetry.IsFirstRun() {
		fmt.Print(telemetry.GetFirstRunNotification())
		fmt.Println()
		_ = telemetry.MarkFirstRunComplete()
	}

	// Initialize managers
	historyManager, err := initializeHistoryManager()
	if err != nil {
		panic("failed to initialize history manager")
	}

	analyticsManager, err := initializeAnalyticsManager()
	if err != nil {
		panic("failed to initialize analytics manager")
	}

	completionManager := initializeCompletionManager()

	runner, err := initializeRunner(analyticsManager, historyManager, completionManager, opts.login)
	if err != nil {
		panic(err)
	}

	logger, _, err := initializeLogger(runner)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	analyticsManager.Logger = logger
	logger.Info("-------- new gsh session --------", zap.Any("args", os.Args))

	// Check for updates in background
	appupdate.HandleSelfUpdate(
		BUILD_VERSION,
		logger,
		filesystem.DefaultFileSystem{},
		appupdate.DefaultUpdater{},
	)

	ctx := context.Background()

	// Handle piped input (non-terminal stdin)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		err = bash.RunBashScriptFromReader(ctx, runner, os.Stdin, "gsh")
		handleExitError(err, logger)
		return
	}

	// Start interactive REPL
	if telemetryClient != nil {
		telemetryClient.TrackSessionStart("repl")
	}

	err = runInteractiveShell(ctx, logger, runner, startTime, telemetryClient, opts.replConfig)
	handleExitError(err, logger)
}

// runRunCommand handles the "run" subcommand
func runRunCommand(startTime time.Time, args []string) {
	// Check for help flag in subcommand args
	if containsHelpFlag(args) {
		fmt.Print(runHelpText)
		return
	}

	// Find the script file (first non-flag argument)
	var scriptPath string
	var scriptArgs []string
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			scriptPath = arg
			scriptArgs = args[i+1:]
			break
		}
	}

	if scriptPath == "" {
		fmt.Fprintf(os.Stderr, "gsh run: missing script path\n")
		fmt.Fprintf(os.Stderr, "Run 'gsh run --help' for usage.\n")
		os.Exit(1)
	}

	// Initialize telemetry
	telemetryClient, err := telemetry.NewClient(telemetry.Config{
		Version: BUILD_VERSION,
	})
	if err != nil {
		telemetryClient = nil
	}
	if telemetryClient != nil {
		defer telemetryClient.Close()
	}

	// Initialize managers (minimal for script execution)
	historyManager, _ := initializeHistoryManager()
	analyticsManager, _ := initializeAnalyticsManager()
	completionManager := initializeCompletionManager()

	runner, err := initializeRunner(analyticsManager, historyManager, completionManager, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gsh: failed to initialize: %v\n", err)
		os.Exit(1)
	}

	logger, logLevel, err := initializeLogger(runner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gsh: failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("-------- new gsh run session --------", zap.String("script", scriptPath), zap.Any("args", scriptArgs))

	ctx := context.Background()

	// Execute the script
	if isGshScript(scriptPath) {
		if telemetryClient != nil {
			telemetryClient.TrackSessionStart("script")
			telemetryClient.TrackScriptExecution()
			startupMs := time.Since(startTime).Milliseconds()
			telemetryClient.TrackStartupTime(startupMs)
		}
		if err := runGshScript(ctx, scriptPath, logger, logLevel, runner); err != nil {
			if telemetryClient != nil {
				telemetryClient.TrackError(telemetry.ErrorCategoryScript)
			}
			handleExitError(err, logger)
		}
	} else {
		if err := bash.RunBashScriptFromFile(ctx, runner, scriptPath); err != nil {
			handleExitError(err, logger)
		}
	}
}

// runTelemetryCommand handles the "telemetry" subcommand
func runTelemetryCommand(args []string) {
	// Check for help flag
	if containsHelpFlag(args) {
		fmt.Print(telemetryHelpText)
		return
	}

	// Default to "status" if no subcommand
	subcommand := "status"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		subcommand = args[0]
	}

	switch subcommand {
	case "status":
		fmt.Printf("Telemetry: %s\n", telemetry.GetTelemetryStatus())
	case "on":
		if err := telemetry.SetTelemetryEnabled(true); err != nil {
			fmt.Fprintf(os.Stderr, "gsh: failed to enable telemetry: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Telemetry enabled. Thank you for helping improve gsh!")
	case "off":
		if err := telemetry.SetTelemetryEnabled(false); err != nil {
			fmt.Fprintf(os.Stderr, "gsh: failed to disable telemetry: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Telemetry disabled. No data will be sent.")
	default:
		fmt.Fprintf(os.Stderr, "gsh telemetry: unknown command: %s\n", subcommand)
		fmt.Fprintf(os.Stderr, "Run 'gsh telemetry --help' for usage.\n")
		os.Exit(1)
	}
}

// handleExitError handles exit status and errors
func handleExitError(err error, logger *zap.Logger) {
	if err == nil {
		return
	}

	var exitStatus interp.ExitStatus
	if errors.As(err, &exitStatus) {
		os.Exit(int(exitStatus))
	}

	if logger != nil {
		logger.Error("unhandled error", zap.Error(err))
	}
	os.Exit(1)
}

// runInteractiveShell starts the new REPL implementation.
func runInteractiveShell(ctx context.Context, logger *zap.Logger, runner *interp.Runner, startTime time.Time, startupTracker repl.StartupTimeTracker, replConfigPath string) error {
	// Read default config content from embedded FS
	defaultContent, err := defaultConfigFS.ReadFile(defaultConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded default config: %w", err)
	}

	r, err := repl.NewREPL(repl.Options{
		Logger:                logger,
		ConfigPath:            replConfigPath, // Custom config path (empty = default ~/.gsh/repl.gsh)
		DefaultConfigContent:  string(defaultContent),
		DefaultConfigFS:       defaultConfigFS,
		DefaultConfigBasePath: "defaults", // defaults/init.gsh imports from this directory
		BuildVersion:          BUILD_VERSION,
		Runner:                runner,
		StartTime:             startTime,
		StartupTracker:        startupTracker,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize REPL: %w", err)
	}
	defer r.Close()

	return r.Run(ctx)
}

// isGshScript checks if a file is a .gsh script
func isGshScript(filePath string) bool {
	return strings.HasSuffix(filePath, ".gsh")
}

// runGshScript executes a .gsh script file
func runGshScript(ctx context.Context, filePath string, logger *zap.Logger, logLevel zap.AtomicLevel, runner *interp.Runner) error {
	// Get absolute path for proper import resolution
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve script path: %w", err)
	}

	// Read the script file
	content, err := os.ReadFile(absPath)
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

	// Create interpreter with the shared runner, logger, and log level
	// This allows gsh scripts to share environment, logger, and log level with bash execution
	gshInterp := interpreter.New(&interpreter.Options{
		Logger:   logger,
		Runner:   runner,
		Version:  BUILD_VERSION,
		LogLevel: logLevel,
	})
	defer gshInterp.Close()

	// Execute the script with filesystem origin for import resolution
	_, err = gshInterp.EvalString(script, &interpreter.ScriptOrigin{
		Type:     interpreter.OriginFilesystem,
		BasePath: filepath.Dir(absPath),
	})
	if err != nil {
		// Print the error to stderr for better user experience
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return fmt.Errorf("runtime error: %w", err)
	}

	return nil
}

func initializeLogger(runner *interp.Runner) (*zap.Logger, zap.AtomicLevel, error) {
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

	// In dev builds, logs only go to file to avoid interfering with Bubble Tea UI
	// Use `tail -f ~/.gsh/gsh.log` to monitor logs in real-time

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, zap.AtomicLevel{}, err
	}

	return logger, logLevel, nil
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
func initializeRunner(analyticsManager *analytics.AnalyticsManager, historyManager *history.HistoryManager, completionManager *completion.CompletionManager, loginShell bool) (*interp.Runner, error) {
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

	configFiles := []string{
		filepath.Join(core.HomeDir(), ".gshrc"),
		filepath.Join(core.HomeDir(), ".gshenv"),
	}

	// Check if this is a login shell
	if loginShell || strings.HasPrefix(os.Args[0], "-") {
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
