package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/atinylittleshell/gsh/internal/agent"
	"github.com/atinylittleshell/gsh/internal/analytics"
	"github.com/atinylittleshell/gsh/internal/bash"
	"github.com/atinylittleshell/gsh/internal/completion"
	"github.com/atinylittleshell/gsh/internal/environment"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/predict"
	"github.com/atinylittleshell/gsh/internal/rag"
	"github.com/atinylittleshell/gsh/internal/rag/retrievers"
	"github.com/atinylittleshell/gsh/internal/styles"
	"github.com/atinylittleshell/gsh/internal/subagent"
	"github.com/atinylittleshell/gsh/pkg/gline"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func RunInteractiveShell(
	ctx context.Context,
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	analyticsManager *analytics.AnalyticsManager,
	completionManager *completion.CompletionManager,
	logger *zap.Logger,
) error {
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

		historyEntries, err := historyManager.GetRecentEntries(environment.GetPwd(runner), 1024)
		if err != nil {
			logger.Warn("error getting recent history entries", zap.Error(err))
			historyEntries = []history.HistoryEntry{}
		}

		historyCommands := make([]string, len(historyEntries))
		for i := len(historyEntries) - 1; i >= 0; i-- {
			historyCommands[len(historyEntries)-1-i] = historyEntries[i].Command
		}

		// Read input
		options := gline.NewOptions()
		options.MinHeight = environment.GetMinimumLines(runner, logger)
		options.CompletionProvider = completionProvider

		line, err := gline.Gline(prompt, historyCommands, "", predictor, explainer, analyticsManager, logger, options)

		logger.Debug("received command", zap.String("line", line))

		if err != nil {
			logger.Error("error reading input through gline", zap.Error(err))
			return err
		}

		// Handle agent chat and macros
		if strings.HasPrefix(line, "#") {
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
				default:
					logger.Warn("unknown agent control", zap.String("control", control))
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("gsh: Unknown agent control: "+control+"\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}
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
		shouldExit, err := executeCommand(ctx, line, historyManager, runner, logger)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		}

		if shouldExit {
			logger.Debug("exiting...")
			break
		}
	}

	return nil
}

func executeCommand(ctx context.Context, input string, historyManager *history.HistoryManager, runner *interp.Runner, logger *zap.Logger) (bool, error) {
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

	startTime := time.Now()
	err = runner.Run(ctx, prog)
	exited := runner.Exited()
	endTime := time.Now()

	durationMs := endTime.Sub(startTime).Milliseconds()
	bash.RunBashCommand(ctx, runner, fmt.Sprintf("GSH_LAST_COMMAND_DURATION_MS=%d", durationMs))

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

	historyManager.FinishCommand(historyEntry, exitCode)
	bash.RunBashCommand(ctx, runner, fmt.Sprintf("GSH_LAST_COMMAND_EXIT_CODE=%d", exitCode))

	return exited, nil
}
