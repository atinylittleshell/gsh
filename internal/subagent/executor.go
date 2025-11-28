package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/atinylittleshell/gsh/internal/agent/tools"
	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/styles"
	"github.com/atinylittleshell/gsh/internal/utils"
	"github.com/atinylittleshell/gsh/pkg/gline"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// SubagentExecutor handles the execution of individual subagents
type SubagentExecutor struct {
	runner         *interp.Runner
	historyManager *history.HistoryManager
	logger         *zap.Logger
	subagent       *Subagent

	// LLM client and configuration (can be overridden per subagent)
	llmClient      *openai.Client
	llmModelConfig utils.LLMModelConfig

	// Chat session state
	messages []openai.ChatCompletionMessage
}

// NewSubagentExecutor creates a new executor for a specific subagent
func NewSubagentExecutor(
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	logger *zap.Logger,
	subagent *Subagent,
) *SubagentExecutor {
	// Get LLM client configuration
	llmClient, modelConfig := utils.GetLLMClient(runner, utils.SlowModel)

	// Override model if subagent specifies one
	if subagent.Model != "" && subagent.Model != "inherit" {
		modelConfig.ModelId = subagent.Model
	}

	executor := &SubagentExecutor{
		runner:         runner,
		historyManager: historyManager,
		logger:         logger,
		subagent:       subagent,
		llmClient:      llmClient,
		llmModelConfig: modelConfig,
	}

	// Initialize chat session with subagent's system prompt
	executor.resetChatSession()

	return executor
}

// resetChatSession initializes or resets the chat session with the subagent's system prompt
func (e *SubagentExecutor) resetChatSession() {
	systemPrompt := fmt.Sprintf(`You are %s, a specialized AI assistant.

%s

# Available Tools

You have access to the following tools: %v

# Tool Restrictions

%s

# Instructions

* Whenever possible, prefer using tools to complete tasks rather than just telling the user how to do them.
* You can run multiple commands in sequence if needed.
* The user can see the output of any tool you run, so there's no need to repeat that in your response.
* If you see a tool call response enclosed in <gsh_tool_call_error> tags, that means the tool call failed.
* Never call multiple tools in parallel. Always call at most one tool at a time.
`,
		e.subagent.Name,
		e.subagent.SystemPrompt,
		e.subagent.AllowedTools,
		e.getToolRestrictionText(),
	)

	e.messages = []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}
}

// getToolRestrictionText generates text describing tool restrictions for the system prompt
func (e *SubagentExecutor) getToolRestrictionText() string {
	var restrictions []string

	if e.subagent.FileRegex != "" {
		restrictions = append(restrictions,
			fmt.Sprintf("- File access is restricted to files matching pattern: %s", e.subagent.FileRegex))
	}

	if !e.hasToolAccess("bash") {
		restrictions = append(restrictions, "- Command execution (bash) is not available")
	}

	if !e.hasToolAccess("create_file") && !e.hasToolAccess("edit_file") {
		restrictions = append(restrictions, "- File creation and editing are not available")
	}

	if len(restrictions) == 0 {
		return "No specific tool restrictions apply."
	}

	return strings.Join(restrictions, "\n")
}

// hasToolAccess checks if the subagent has access to a specific tool
func (e *SubagentExecutor) hasToolAccess(toolName string) bool {
	for _, allowedTool := range e.subagent.AllowedTools {
		if allowedTool == toolName {
			return true
		}
	}
	return false
}

// Chat handles a chat interaction with the subagent
func (e *SubagentExecutor) Chat(prompt string) (<-chan string, error) {
	e.logger.Debug("Starting subagent chat",
		zap.String("subagent", e.subagent.Name),
		zap.String("prompt", prompt))

	// Add user message
	userMessage := openai.ChatCompletionMessage{
		Role:    "user",
		Content: prompt,
	}
	e.messages = append(e.messages, userMessage)

	responseChannel := make(chan string)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		select {
		case <-signalChan:
			cancel()
			signal.Stop(signalChan)
		case <-ctx.Done():
			signal.Stop(signalChan)
		}
	}()

	go func() {
		defer close(responseChannel)
		defer cancel()
		defer signal.Stop(signalChan)

		continueSession := true

		// Set prompt for subagent execution
		// We must use the runner to set the variable so that it persists correctly
		// across tool execution calls (which use runner.Run and might overwrite Vars)
		originalPromptVar := e.runner.Vars["GSH_APROMPT"]
		originalPrompt := ""
		if originalPromptVar.IsSet() {
			originalPrompt = originalPromptVar.String()
		}

		// Determine subagent prompt (icon or name)
		subagentPrompt := fmt.Sprintf("%s > ", e.subagent.Name) // Default: "Name > "

		// Let's try to be smart about extracting an emoji/icon
		parts := strings.Fields(e.subagent.Name)
		if len(parts) > 0 {
			// Check if first part contains non-alphanumeric chars
			firstPart := parts[0]
			matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, firstPart)
			if !matched && len(firstPart) < 10 {
				// Assume it's an icon if short and has special chars
				// For icon, use "Icon> " (no space before >)
				subagentPrompt = firstPart + "> "
			}
		}

		// Helper to set variable via runner execution
		setVar := func(name, value string) {
			quotedValue, err := syntax.Quote(value, syntax.LangBash)
			if err != nil {
				// Fallback to simple quoting if syntax.Quote fails
				quotedValue = fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "'\\''"))
			}
			cmd := fmt.Sprintf("%s=%s", name, quotedValue)
			p, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
			if err == nil {
				if err := e.runner.Run(ctx, p); err != nil {
					e.logger.Warn("Failed to set subagent prompt variable", zap.String("variable", name), zap.Error(err))
				}
			} else {
				// Fallback to direct assignment if parsing fails (unlikely)
				e.runner.Vars[name] = expand.Variable{Kind: expand.String, Str: value}
			}
		}

		// Set the prompt
		setVar("GSH_APROMPT", subagentPrompt)

		// Restore prompt when done
		defer func() {
			setVar("GSH_APROMPT", originalPrompt)
		}()

		for continueSession {
			continueSession = false

			// Build available tools based on subagent configuration
			var availableTools []openai.Tool
			for _, toolName := range e.subagent.AllowedTools {
				if tool := e.getToolDefinition(toolName); tool != nil {
					availableTools = append(availableTools, *tool)
				}
			}

			request := openai.ChatCompletionRequest{
				Model:    e.llmModelConfig.ModelId,
				Messages: e.messages,
				Tools:    availableTools,
			}

			if e.llmModelConfig.Temperature != nil {
				request.Temperature = float32(*e.llmModelConfig.Temperature)
			}
			if e.llmModelConfig.ParallelToolCalls != nil {
				request.ParallelToolCalls = *e.llmModelConfig.ParallelToolCalls
			}

			response, err := e.llmClient.CreateChatCompletion(ctx, request)
			if err != nil {
				if ctx.Err() == context.Canceled {
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("Subagent chat interrupted by user") + "\n")
					e.logger.Info("Subagent chat interrupted by user", zap.String("subagent", e.subagent.Name))
					return
				}
				fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR(fmt.Sprintf("Error communicating with LLM: %s", err)) + "\n")
				e.logger.Error("Error in subagent chat", zap.String("subagent", e.subagent.Name), zap.Error(err))
				return
			}

			if len(response.Choices) == 0 {
				fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("LLM responded with empty response") + "\n")
				e.logger.Error("Empty LLM response", zap.String("subagent", e.subagent.Name))
				return
			}

			msg := response.Choices[0]
			e.logger.Debug("Subagent LLM response",
				zap.String("subagent", e.subagent.Name),
				zap.Any("response", msg))
			e.messages = append(e.messages, msg.Message)

			if msg.FinishReason == "stop" || msg.FinishReason == "end_turn" || msg.FinishReason == "tool_calls" || msg.FinishReason == "function_call" {
				if msg.Message.Content != "" {
					responseChannel <- strings.TrimSpace(msg.Message.Content)
				}

				if len(msg.Message.ToolCalls) > 0 {
					allToolCallsSucceeded := true
					for _, toolCall := range msg.Message.ToolCalls {
						if !e.handleToolCall(toolCall) {
							allToolCallsSucceeded = false
						}
					}

					if allToolCallsSucceeded {
						continueSession = true
					}
				}
			} else if msg.FinishReason != "" {
				e.logger.Warn("LLM finished for unexpected reason",
					zap.String("subagent", e.subagent.Name),
					zap.String("reason", string(msg.FinishReason)))
			}
		}
	}()

	return responseChannel, nil
}

// getToolDefinition returns the OpenAI tool definition for a given tool name
func (e *SubagentExecutor) getToolDefinition(toolName string) *openai.Tool {
	switch toolName {
	case "bash":
		return &tools.BashToolDefinition
	case "view_file":
		return &tools.ViewFileToolDefinition
	case "view_directory":
		return &tools.ViewDirectoryToolDefinition
	case "create_file":
		return &tools.CreateFileToolDefinition
	case "edit_file":
		return &tools.EditFileToolDefinition
	default:
		e.logger.Warn("Unknown tool requested", zap.String("tool", toolName), zap.String("subagent", e.subagent.Name))
		return nil
	}
}

// handleToolCall executes a tool call with appropriate restrictions
func (e *SubagentExecutor) handleToolCall(toolCall openai.ToolCall) bool {
	var params map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		e.logger.Error("Failed to parse tool call arguments",
			zap.String("subagent", e.subagent.Name),
			zap.String("arguments", toolCall.Function.Arguments),
			zap.Error(err))
		fmt.Print(gline.RESET_CURSOR_COLUMN +
			styles.ERROR("Subagent provided invalid tool call arguments") + "\n")
		return false
	}

	e.logger.Debug("Handling subagent tool call",
		zap.String("subagent", e.subagent.Name),
		zap.String("tool", toolCall.Function.Name),
		zap.Any("params", params))

	// Check if tool is allowed
	if !e.hasToolAccess(toolCall.Function.Name) {
		toolResponse := fmt.Sprintf("<gsh_tool_call_error>Tool '%s' is not available for this subagent</gsh_tool_call_error>", toolCall.Function.Name)
		e.messages = append(e.messages, openai.ChatCompletionMessage{
			Role:       "tool",
			ToolCallID: toolCall.ID,
			Content:    toolResponse,
		})
		return true
	}

	// Apply file access restrictions
	if e.subagent.FileRegex != "" && (toolCall.Function.Name == "view_file" || toolCall.Function.Name == "create_file" || toolCall.Function.Name == "edit_file") {
		if filePath, ok := params["path"].(string); ok {
			if matched, err := regexp.MatchString(e.subagent.FileRegex, filePath); err != nil || !matched {
				toolResponse := fmt.Sprintf("<gsh_tool_call_error>File access denied: '%s' does not match allowed pattern '%s'</gsh_tool_call_error>", filePath, e.subagent.FileRegex)
				e.messages = append(e.messages, openai.ChatCompletionMessage{
					Role:       "tool",
					ToolCallID: toolCall.ID,
					Content:    toolResponse,
				})
				return true
			}
		}
	}

	// Execute the tool call
	toolResponse := e.executeToolCall(toolCall.Function.Name, params)

	e.messages = append(e.messages, openai.ChatCompletionMessage{
		Role:       "tool",
		ToolCallID: toolCall.ID,
		Content:    toolResponse,
	})
	return true
}

// executeToolCall executes the actual tool with the given parameters
func (e *SubagentExecutor) executeToolCall(toolName string, params map[string]any) string {
	switch toolName {
	case "bash":
		return tools.BashTool(e.runner, e.historyManager, e.logger, params)
	case "view_file":
		return tools.ViewFileTool(e.runner, e.logger, params)
	case "view_directory":
		return tools.ViewDirectoryTool(e.runner, e.logger, params)
	case "create_file":
		return tools.CreateFileTool(e.runner, e.logger, params)
	case "edit_file":
		return tools.EditFileTool(e.runner, e.logger, params)
	default:
		return fmt.Sprintf("<gsh_tool_call_error>Unknown tool: %s</gsh_tool_call_error>", toolName)
	}
}

// ResetChat resets the chat session for this subagent
func (e *SubagentExecutor) ResetChat() {
	e.resetChatSession()
	e.logger.Debug("Reset chat session", zap.String("subagent", e.subagent.Name))
}
