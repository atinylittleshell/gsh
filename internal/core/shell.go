package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/agent"
	"github.com/atinylittleshell/gsh/internal/analytics"
	"github.com/atinylittleshell/gsh/internal/bash"
	"github.com/atinylittleshell/gsh/internal/coach"
	"github.com/atinylittleshell/gsh/internal/completion"
	"github.com/atinylittleshell/gsh/internal/config"
	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/idle"
	"github.com/atinylittleshell/gsh/internal/predict"
	"github.com/atinylittleshell/gsh/internal/rag"
	"github.com/atinylittleshell/gsh/internal/rag/retrievers"
	"github.com/atinylittleshell/gsh/internal/styles"
	"github.com/atinylittleshell/gsh/internal/subagent"
	"github.com/atinylittleshell/gsh/internal/termtitle"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"go.uber.org/zap"
	"golang.org/x/term"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func RunInteractiveShell(
	ctx context.Context,
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	analyticsManager *analytics.AnalyticsManager,
	completionManager *completion.CompletionManager,
	coachManager *coach.CoachManager,
	logger *zap.Logger,
	stderrCapturer *StderrCapturer,
) error {
	state := &ShellState{}
	contextProvider := &rag.ContextProvider{
		Logger: logger,
		Retrievers: []rag.ContextRetriever{
			retrievers.SystemInfoContextRetriever{Runner: runner},
			retrievers.WorkingDirectoryContextRetriever{Runner: runner},
			retrievers.GitStatusContextRetriever{Runner: runner, Logger: logger},
			retrievers.ConciseHistoryContextRetriever{Runner: runner, Logger: logger, HistoryManager: historyManager},
			retrievers.VerboseHistoryContextRetriever{Runner: runner, Logger: logger, HistoryManager: historyManager},
		},
	}
	predictor := &predict.PredictRouter{
		PrefixPredictor:    predict.NewLLMPrefixPredictor(runner, historyManager, logger),
		NullStatePredictor: predict.NewLLMNullStatePredictor(runner, logger),
	}
	explainer := predict.NewLLMExplainer(runner, logger)
	agent := agent.NewAgent(runner, historyManager, logger)

	// Set up subagent integration
	subagentIntegration := subagent.NewSubagentIntegration(runner, historyManager, logger)

	// Set up completion
	completionProvider := completion.NewShellCompletionProvider(completionManager, runner)
	completionProvider.SetSubagentProvider(subagentIntegration.GetCompletionProvider())

	// Set up idle summary generator
	idleSummaryGenerator := idle.NewSummaryGenerator(runner, historyManager, logger)

	// Set up terminal title manager
	termTitleManager := termtitle.NewManager(runner, logger)

	chanSIGINT := make(chan os.Signal, 1)
	signal.Notify(chanSIGINT, os.Interrupt)

	go func() {
		for {
			// ignore SIGINT
			<-chanSIGINT
		}
	}()

	for {
		prompt := environment.GetPrompt(runner, logger)
		logger.Debug("prompt updated", zap.String("prompt", prompt))

		ragContext := contextProvider.GetContext()
		logger.Debug("context updated", zap.Any("context", ragContext))

		predictor.UpdateContext(ragContext)
		explainer.UpdateContext(ragContext)
		agent.UpdateContext(ragContext)

		// Fetch recent entries for standard history (Up/Down) - scoped to current directory for now, or generally recent
		// Note: GetRecentEntries reverses the list (oldest first) so standard history navigation works correctly
		historyEntries, err := historyManager.GetRecentEntries(environment.GetPwd(runner), 1024)
		if err != nil {
			logger.Warn("error getting recent history entries", zap.Error(err))
			historyEntries = []history.HistoryEntry{}
		}

		historyCommands := make([]string, len(historyEntries))
		for i := len(historyEntries) - 1; i >= 0; i-- {
			historyCommands[len(historyEntries)-1-i] = historyEntries[i].Command
		}

		// Fetch all entries for rich search (Ctrl+R)
		allHistoryEntries, err := historyManager.GetAllEntries()
		if err != nil {
			logger.Warn("error getting all history entries", zap.Error(err))
			allHistoryEntries = []history.HistoryEntry{}
		}

		richHistory := make([]shellinput.HistoryItem, len(allHistoryEntries))
		for i, entry := range allHistoryEntries {
			richHistory[i] = shellinput.HistoryItem{
				Command:   entry.Command,
				Directory: entry.Directory,
				Timestamp: entry.CreatedAt,
			}
		}

		// Read input
		options := gline.NewOptions()
		options.AssistantHeight = environment.GetAssistantHeight(runner, logger)
		options.CompletionProvider = completionProvider
		options.RichHistory = richHistory
		options.CurrentDirectory = environment.GetPwd(runner)

		// Populate context for border status
		options.CurrentDirectory = environment.GetPwd(runner)
		options.User = environment.GetUser(runner)
		options.Host, _ = os.Hostname()

		// Configure idle summary
		idleTimeout := environment.GetIdleSummaryTimeout(runner, logger)
		options.IdleSummaryTimeout = idleTimeout
		if idleTimeout > 0 {
			options.IdleSummaryGenerator = idleSummaryGenerator.GenerateSummary
		}

		// Get coach startup content for the Assistant Box
		var coachContent string
		if coachManager != nil {
			if content := coachManager.GetDisplayContent(); content != nil {
				coachContent = content.Icon + " " + content.Title
				if content.Content != "" {
					coachContent += "\n" + content.Content
				}
				if content.Action != "" {
					coachContent += "\n" + content.Action
				}
			}
		}

		line, err := gline.Gline(prompt, historyCommands, coachContent, predictor, explainer, analyticsManager, logger, options)

		logger.Debug("received command", zap.String("line", line))

		if err != nil {
			if err == gline.ErrInterrupted {
				// User pressed Ctrl+C, restart loop with fresh prompt
				logger.Debug("input interrupted by user")
				continue
			}
			logger.Error("error reading input through gline", zap.Error(err))
			return err
		}

		// Handle agent chat and macros
		if strings.HasPrefix(line, "@") {
			chatMessage := strings.TrimSpace(line[1:])

			// Handle agent controls
			if strings.HasPrefix(chatMessage, "!") {
				control := strings.TrimSpace(strings.TrimPrefix(chatMessage, "!"))

				// Try subagent controls first
				if subagentIntegration.HandleAgentControl(control) {
					continue
				}

				// Handle built-in agent controls
				switch control {
				case "new":
					agent.ResetChat()
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: Chat session reset.\n") + gline.RESET_CURSOR_COLUMN)
					continue
				case "tokens":
					agent.PrintTokenStats()
					continue
				case "config":
					if err := config.RunConfigUI(runner); err != nil {
						logger.Error("error running config UI", zap.Error(err))
						fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("gsh: Error running config: "+err.Error()+"\n") + gline.RESET_CURSOR_COLUMN)
					}
					// Sync any gsh variables that were changed in the config UI
					environment.SyncVariablesToEnv(runner)
					continue
				default:
					// Handle coach command with subcommands
					if strings.HasPrefix(control, "coach") {
						if coachManager == nil {
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("gsh: Coach not initialized\n") + gline.RESET_CURSOR_COLUMN)
							continue
						}

						// Parse subcommand (e.g., "coach tips" -> "tips")
						coachArgs := strings.TrimSpace(strings.TrimPrefix(control, "coach"))

						switch coachArgs {
						case "", "dashboard":
							fmt.Print(coachManager.RenderDashboard())
						case "stats":
							fmt.Print(coachManager.RenderStats())
						case "achievements":
							fmt.Print(coachManager.RenderAchievements())
						case "challenges":
							fmt.Print(coachManager.RenderChallenges())
						case "tips":
							fmt.Print(coachManager.RenderAllTips())
						case "reset-tips":
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("Resetting tips and generating new ones from your history...\nThis may take a moment.\n\n") + gline.RESET_CURSOR_COLUMN)
							result := coachManager.ResetAndRegenerateTips()
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(result+"\n") + gline.RESET_CURSOR_COLUMN)
						default:
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: Unknown coach command: "+coachArgs+"\n") + gline.RESET_CURSOR_COLUMN)
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("Available: @!coach [stats|achievements|challenges|tips|reset-tips]\n") + gline.RESET_CURSOR_COLUMN)
						}
						continue
					}
					logger.Warn("unknown agent control", zap.String("control", control))
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: Unknown agent control: "+control+"\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}
			}

			// Handle magic fix
			if chatMessage == "?" {
				if state.LastExitCode == 0 {
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: Last command succeeded.\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}

				prompt := fmt.Sprintf("The command `%s` failed with exit code %d.\nThe stderr output was:\n%s\n\nExplain why it failed and suggest a fix. Do not execute the fix yet. Provide the fixed command in a markdown code block.", state.LastCommand, state.LastExitCode, state.LastStderr)

				chatChannel, err := agent.Chat(prompt)
				if err != nil {
					logger.Error("error chatting with agent", zap.Error(err))
					continue
				}

				var fullResponse strings.Builder
				for message := range chatChannel {
					fullResponse.WriteString(message)
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: "+message+"\n") + gline.RESET_CURSOR_COLUMN)
				}

				// Extract code block
				responseStr := fullResponse.String()
				codeBlockRegex := regexp.MustCompile("(?s)```(?:bash|sh|zsh)?\\s+(.*?)\\s+```")
				matches := codeBlockRegex.FindAllStringSubmatch(responseStr, -1)

				var fixedCmd string
				if len(matches) > 0 {
					fixedCmd = strings.TrimSpace(matches[len(matches)-1][1])
				}

				if fixedCmd != "" {
					defaultToYes := environment.GetDefaultToYes(runner)
					promptText := "Run this fix? [y/N] "
					if defaultToYes {
						promptText = "Run this fix? [Y/n] "
					}

					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("\nCommand: "+fixedCmd+"\n") + gline.RESET_CURSOR_COLUMN)
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(promptText) + gline.RESET_CURSOR_COLUMN)

					// Read single key in raw mode
					fd := int(os.Stdin.Fd())
					oldState, err := term.MakeRaw(fd)
					if err != nil {
						logger.Error("failed to set raw mode", zap.Error(err))
						continue
					}
					var buf [1]byte
					_, _ = os.Stdin.Read(buf[:])
					_ = term.Restore(fd, oldState)

					char := buf[0]
					// Echo the character and newline
					if char == '\r' || char == '\n' {
						fmt.Println()
					} else {
						fmt.Printf("%c\n", char)
					}

					// Determine if confirmed based on default setting
					confirmed := char == 'y' || char == 'Y'
					if defaultToYes && (char == '\r' || char == '\n') {
						confirmed = true
					}

					if confirmed {
						fmt.Println()
						shouldExit, err := executeCommand(ctx, fixedCmd, historyManager, coachManager, runner, logger, state, stderrCapturer)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
						}

						// Record command for terminal title updates
						termTitleManager.RecordCommand(fixedCmd)

						// Sync any gsh variables that might have been changed during command execution
						environment.SyncVariablesToEnv(runner)

						if shouldExit {
							logger.Debug("exiting...")
							break
						}
					}
				}
				continue
			}

			// Handle macros
			if strings.HasPrefix(chatMessage, "/") {
				macroName := strings.TrimSpace(strings.TrimPrefix(chatMessage, "/"))
				macros := environment.GetAgentMacros(runner, logger)
				if message, ok := macros[macroName]; ok {
					chatMessage = message
				} else {
					logger.Warn("macro not found", zap.String("macro", macroName))
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: Macro not found: "+macroName+"\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}
			}

			// Check for subagent commands first
			handled, chatChannel, subagent, err := subagentIntegration.HandleCommand(chatMessage)
			if handled {
				if err != nil {
					logger.Error("error with subagent command", zap.Error(err))
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("gsh: "+err.Error()+"\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}

				// Print notification: "Asking <subagent_name> to assist with this task"
				fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(fmt.Sprintf("Asking %s to assist with this task\n", subagent.Name)) + gline.RESET_CURSOR_COLUMN)

				// Handle subagent response with subagent identification
				for message := range chatChannel {
					prefix := fmt.Sprintf("gsh [%s]: ", subagent.Name)
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(prefix+message+"\n") + gline.RESET_CURSOR_COLUMN)
				}
				continue
			}

			// Fall back to regular agent chat
			chatChannel, err = agent.Chat(chatMessage)
			if err != nil {
				logger.Error("error chatting with agent", zap.Error(err))
				continue
			}

			for message := range chatChannel {
				fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: "+message+"\n") + gline.RESET_CURSOR_COLUMN)
			}

			continue
		}

		// Handle empty input
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Execute the command
		shouldExit, err := executeCommand(ctx, line, historyManager, coachManager, runner, logger, state, stderrCapturer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		}

		// Record command for terminal title updates
		termTitleManager.RecordCommand(line)

		// Sync any gsh variables that might have been changed during command execution
		environment.SyncVariablesToEnv(runner)

		if shouldExit {
			logger.Debug("exiting...")
			break
		}
	}

	return nil
}

func executeCommand(ctx context.Context, input string, historyManager *history.HistoryManager, coachManager *coach.CoachManager, runner *interp.Runner, logger *zap.Logger, state *ShellState, stderrCapturer *StderrCapturer) (bool, error) {
	// Pre-process input to transform typeset/declare -f/-F/-p commands to gsh_typeset
	logger.Debug("preprocessing input", zap.String("original_input", input), zap.Int("input_length", len(input)))

	// Validate input before preprocessing
	if input == "" {
		logger.Warn("empty input received for preprocessing")
		return false, nil
	}

	// Add timeout protection for preprocessing
	preprocessStart := time.Now()
	logger.Debug("calling bash.PreprocessTypesetCommands", zap.String("input", input))
	processedInput := bash.PreprocessTypesetCommands(input)
	logger.Debug("bash.PreprocessTypesetCommands completed", zap.String("output", processedInput))
	preprocessDuration := time.Since(preprocessStart)

	logger.Debug("preprocessing completed",
		zap.String("processed_input", processedInput),
		zap.Int("processed_length", len(processedInput)),
		zap.Duration("preprocess_duration", preprocessDuration),
		zap.Bool("input_changed", input != processedInput))

	// Check if preprocessing took too long (potential resource exhaustion)
	if preprocessDuration > 100*time.Millisecond {
		logger.Warn("preprocessing took unusually long",
			zap.Duration("duration", preprocessDuration),
			zap.Int("input_length", len(input)))
	}

	input = processedInput

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(input), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if prog == nil {
		logger.Error("invalid command", zap.String("command", input))
		return false, nil
	}
	if err != nil {
		logger.Error("error parsing command", zap.String("command", input), zap.Error(err))
		return false, err
	}

	historyEntry, _ := historyManager.StartCommand(input, environment.GetPwd(runner))

	state.LastCommand = input
	if stderrCapturer != nil {
		stderrCapturer.StartCapture()
	}

	startTime := time.Now()
	err = runner.Run(ctx, prog)
	exited := runner.Exited()

	if stderrCapturer != nil {
		state.LastStderr = stderrCapturer.StopCapture()
	}

	endTime := time.Now()

	durationMs := endTime.Sub(startTime).Milliseconds()
	_, _, _ = bash.RunBashCommand(ctx, runner, fmt.Sprintf("GSH_LAST_COMMAND_DURATION_MS=%d", durationMs))

	var exitCode int
	if err != nil {
		status, ok := interp.IsExitStatus(err)
		if !ok {
			exitCode = -1
		} else {
			exitCode = int(status)
		}
	} else {
		exitCode = 0
	}

	state.LastExitCode = exitCode

	_, _ = historyManager.FinishCommand(historyEntry, exitCode)
	_, _, _ = bash.RunBashCommand(ctx, runner, fmt.Sprintf("GSH_LAST_COMMAND_EXIT_CODE=%d", exitCode))

	// Record command for coach gamification
	if coachManager != nil {
		coachManager.RecordCommand(input, exitCode, durationMs)
	}

	return exited, nil
}
