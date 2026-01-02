package interpreter

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
	"os"
)

// EventManager manages event handlers for the SDK event system
type EventManager struct {
	mu       sync.RWMutex
	handlers map[string]map[string]*ToolValue // event -> handlerID -> handler
	nextID   int
}

// NewEventManager creates a new event manager
func NewEventManager() *EventManager {
	return &EventManager{
		handlers: make(map[string]map[string]*ToolValue),
		nextID:   0,
	}
}

// On registers an event handler and returns a unique handler ID
func (em *EventManager) On(eventName string, handler *ToolValue) string {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Create event map if it doesn't exist
	if em.handlers[eventName] == nil {
		em.handlers[eventName] = make(map[string]*ToolValue)
	}

	// Generate unique handler ID
	em.nextID++
	handlerID := fmt.Sprintf("handler_%d", em.nextID)

	// Register the handler
	em.handlers[eventName][handlerID] = handler

	return handlerID
}

// Off removes an event handler. If handlerID is empty, removes all handlers for the event.
func (em *EventManager) Off(eventName string, handlerID string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if handlerID == "" {
		// Remove all handlers for this event
		delete(em.handlers, eventName)
	} else {
		// Remove specific handler
		if handlers, exists := em.handlers[eventName]; exists {
			delete(handlers, handlerID)
			// Clean up empty event maps
			if len(handlers) == 0 {
				delete(em.handlers, eventName)
			}
		}
	}
}

// GetHandlers returns all handlers for a given event
func (em *EventManager) GetHandlers(eventName string) []*ToolValue {
	em.mu.RLock()
	defer em.mu.RUnlock()

	handlers := em.handlers[eventName]
	if handlers == nil {
		return nil
	}

	// Return a copy of the handlers slice
	result := make([]*ToolValue, 0, len(handlers))
	for _, handler := range handlers {
		result = append(result, handler)
	}
	return result
}

// SDKConfig manages runtime configuration for the SDK
type SDKConfig struct {
	mu               sync.RWMutex
	logger           *zap.Logger
	atomicLevel      zap.AtomicLevel
	logFile          string // read-only, set at initialization
	lastAgentRequest Value
	// Models holds the model tier definitions (available in both REPL and script mode)
	models *Models
	// REPL context (nil in script mode)
	replContext *REPLContext
}

// REPLContext holds REPL-specific state that's available in the SDK
type REPLContext struct {
	LastCommand       *REPLLastCommand
	PromptValue       Value              // Prompt string set by event handlers (read/write via gsh.prompt)
	MiddlewareManager *MiddlewareManager // Middleware manager for input processing
	Interpreter       *Interpreter       // Reference to interpreter for middleware execution
}

// Models holds the model tier definitions (available in both REPL and script mode)
type Models struct {
	Lite      *ModelValue
	Workhorse *ModelValue
	Premium   *ModelValue
}

// REPLLastCommand holds information about the last executed command
type REPLLastCommand struct {
	ExitCode   int
	DurationMs int64
}

// NewSDKConfig creates a new SDK configuration
// The logger should have been created with an AtomicLevel for dynamic level changes to work
func NewSDKConfig(logger *zap.Logger, atomicLevel zap.AtomicLevel) *SDKConfig {
	// Extract log file from logger if available
	logFile := ""
	// Note: zap doesn't expose output paths directly, so logFile stays empty
	// It would need to be passed separately if needed in the future

	return &SDKConfig{
		logger:           logger,
		atomicLevel:      atomicLevel,
		logFile:          logFile,
		lastAgentRequest: &NullValue{},
		models:           &Models{}, // Initialize empty models (available in both REPL and script mode)
	}
}

// GetTermWidth returns the terminal width
func (sc *SDKConfig) GetTermWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // Default fallback
	}
	return width
}

// GetTermHeight returns the terminal height
func (sc *SDKConfig) GetTermHeight() int {
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 24 // Default fallback
	}
	return height
}

// IsTTY returns whether stdout is a TTY
func (sc *SDKConfig) IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// GetLogLevel returns the current log level
func (sc *SDKConfig) GetLogLevel() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.atomicLevel.Level().String()
}

// SetLogLevel sets the log level dynamically
func (sc *SDKConfig) SetLogLevel(level string) error {
	// Parse the level string to zapcore.Level
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		return fmt.Errorf("invalid log level '%s', must be one of: debug, info, warn, error", level)
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.atomicLevel.SetLevel(zapLevel)
	return nil
}

// GetLogFile returns the current log file path (read-only)
func (sc *SDKConfig) GetLogFile() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.logFile
}

// GetLastAgentRequest returns the last agent request data
func (sc *SDKConfig) GetLastAgentRequest() Value {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.lastAgentRequest
}

// SetLastAgentRequest sets the last agent request data
func (sc *SDKConfig) SetLastAgentRequest(value Value) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.lastAgentRequest = value
}

// SetREPLContext sets the REPL context (called from REPL initialization)
func (sc *SDKConfig) SetREPLContext(ctx *REPLContext) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.replContext = ctx
}

// GetREPLContext returns the REPL context (nil in script mode)
func (sc *SDKConfig) GetREPLContext() *REPLContext {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.replContext
}

// UpdateLastCommand updates the last command's exit code and duration
func (sc *SDKConfig) UpdateLastCommand(exitCode int, durationMs int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.replContext != nil && sc.replContext.LastCommand != nil {
		sc.replContext.LastCommand.ExitCode = exitCode
		sc.replContext.LastCommand.DurationMs = durationMs
	}
}

// GetModels returns the models configuration (available in both REPL and script mode)
func (sc *SDKConfig) GetModels() *Models {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.models
}
