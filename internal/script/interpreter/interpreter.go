package interpreter

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/atinylittleshell/gsh/internal/acp"
	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/mcp"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// acpClientEntry holds an ACP client and its active sessions
type acpClientEntry struct {
	client   *acp.Client
	sessions map[string]acp.ACPSession // Sessions keyed by session ID
}

// ACPClientFactory is a function type for creating ACP clients.
// This can be overridden in tests to inject mock clients.
type ACPClientFactory func(config acp.ClientConfig) (*acp.Client, error)

// Interpreter represents the gsh script interpreter
type Interpreter struct {
	env              *Environment
	mcpManager       *mcp.Manager
	providerRegistry *ProviderRegistry
	callStacks       *goroutineCallStacks // Per-goroutine call stacks for error reporting
	logger           *zap.Logger          // Optional zap logger for log.* functions
	stdin            io.Reader            // Reader for input() function, defaults to os.Stdin
	runner           *interp.Runner       // Shared sh runner for env vars, working dir, and exec()
	runnerMu         sync.RWMutex         // Protects runner access

	// Context for cancellation (e.g., Ctrl+C handling)
	ctx   context.Context // Current execution context
	ctxMu sync.RWMutex    // Protects ctx access

	// SDK infrastructure
	eventManager *EventManager
	sdkConfig    *SDKConfig
	version      string // gsh version

	// Module/import infrastructure
	currentOrigin *ScriptOrigin               // Origin of currently executing script
	importedFiles map[string]bool             // Track imported files (prevent circular imports)
	moduleExports map[string]map[string]Value // Cache of exported symbols per module
	exportedNames map[string]bool             // Names exported by the current module

	// ACP client management
	acpClients       map[string]*acpClientEntry // ACP clients keyed by agent name
	acpClientsMu     sync.RWMutex               // Protects acpClients access
	acpClientFactory ACPClientFactory           // Factory for creating ACP clients (can be overridden for testing)
}

// EvalResult represents the result of evaluating a program
type EvalResult struct {
	FinalResult Value        // The value of the last statement executed
	Env         *Environment // The environment after execution (contains all variables)
}

// Value returns the final result value for convenient access
func (r *EvalResult) Value() Value {
	return r.FinalResult
}

// Variables returns all top-level variables as a map (excluding built-ins)
func (r *EvalResult) Variables() map[string]Value {
	if r.Env == nil {
		return make(map[string]Value)
	}
	// Return a copy to prevent external modification, excluding built-ins
	vars := make(map[string]Value)
	for k, v := range r.Env.store {
		// Skip built-in functions and objects
		if isBuiltin(k) {
			continue
		}
		vars[k] = v
	}
	return vars
}

// Options configures the interpreter.
// All fields are optional - nil/zero values use sensible defaults.
type Options struct {
	// Env is a custom gsh environment. If nil, a new one is created.
	Env *Environment
	// Logger is a zap logger for log.* functions. If nil, log.* writes to stderr.
	Logger *zap.Logger
	// LogLevel is an AtomicLevel for dynamic log level changes.
	// If nil/zero value, a default InfoLevel is created.
	LogLevel zap.AtomicLevel
	// Runner is a sh runner for bash execution. If nil, a new one is created.
	// Sharing a runner allows the interpreter to inherit env vars and working directory.
	Runner *interp.Runner
	// Version is the gsh version string. If empty, "unknown" is used.
	Version string
}

// New creates a new interpreter with the given options.
// Pass nil for default options.
func New(opts *Options) *Interpreter {
	if opts == nil {
		opts = &Options{}
	}
	registry := NewProviderRegistry()
	registry.Register(NewOpenAIProvider())

	// Create sh runner if not provided
	runner := opts.Runner
	if runner == nil {
		shEnv := expand.ListEnviron(os.Environ()...)
		var err error
		runner, err = interp.New(
			interp.Env(shEnv),
			interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		)
		if err != nil {
			// This should never fail with basic options
			panic(fmt.Sprintf("failed to create sh runner: %v", err))
		}
	}

	// Create gsh environment if not provided
	gshEnv := opts.Env
	if gshEnv == nil {
		gshEnv = NewEnvironment()
	}

	// Determine version
	version := opts.Version
	if version == "" {
		version = "unknown"
	}

	// Use provided AtomicLevel or create a default one
	atomicLevel := opts.LogLevel
	// Check if it's a zero value by trying to use it
	if atomicLevel == (zap.AtomicLevel{}) {
		// Zero value means it wasn't set, create a default InfoLevel
		// In production, this will be overridden by the caller if needed
		atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	i := &Interpreter{
		env:              gshEnv,
		mcpManager:       mcp.NewManager(),
		providerRegistry: registry,
		logger:           opts.Logger,
		stdin:            os.Stdin,
		runner:           runner,
		callStacks:       newGoroutineCallStacks(),
		eventManager:     NewEventManager(),
		sdkConfig:        NewSDKConfig(opts.Logger, atomicLevel),
		version:          version,
		importedFiles:    make(map[string]bool),
		moduleExports:    make(map[string]map[string]Value),
		exportedNames:    make(map[string]bool),
		acpClients:       make(map[string]*acpClientEntry),
		acpClientFactory: defaultACPClientFactory,
	}
	i.registerBuiltins()
	i.registerGshSDK()
	return i
}

// defaultACPClientFactory is the default factory for creating ACP clients.
func defaultACPClientFactory(config acp.ClientConfig) (*acp.Client, error) {
	client := acp.NewClient(config)
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// SetACPClientFactory sets a custom factory for creating ACP clients.
// This is primarily used for testing with mock clients.
func (i *Interpreter) SetACPClientFactory(factory ACPClientFactory) {
	i.acpClientFactory = factory
}

// InjectACPSession injects a mock ACP session for testing.
// This bypasses the normal client creation and allows direct session injection.
func (i *Interpreter) InjectACPSession(agentName, sessionID string, session acp.ACPSession) {
	i.acpClientsMu.Lock()
	defer i.acpClientsMu.Unlock()

	entry, ok := i.acpClients[agentName]
	if !ok {
		entry = &acpClientEntry{
			client:   nil, // No actual client needed for testing
			sessions: make(map[string]acp.ACPSession),
		}
		i.acpClients[agentName] = entry
	}
	entry.sessions[sessionID] = session
}

// SetStdin sets the stdin reader for the input() function
// This is useful for testing or for providing custom input sources
func (i *Interpreter) SetStdin(r io.Reader) {
	i.stdin = r
}

// SetContext sets the execution context for the interpreter.
// This context is used for cancellation (e.g., when Ctrl+C is pressed).
// The REPL sets this before executing commands so that long-running operations
// (like agent execution or shell commands) can be cancelled.
func (i *Interpreter) SetContext(ctx context.Context) {
	i.ctxMu.Lock()
	defer i.ctxMu.Unlock()
	i.ctx = ctx
}

// Context returns the current execution context.
// If no context has been set, returns context.Background().
// This should be used by operations that support cancellation.
func (i *Interpreter) Context() context.Context {
	i.ctxMu.RLock()
	defer i.ctxMu.RUnlock()
	if i.ctx == nil {
		return context.Background()
	}
	return i.ctx
}

// Runner returns the underlying sh runner
// This is used by the REPL executor to share the same runner for bash commands
func (i *Interpreter) Runner() *interp.Runner {
	return i.runner
}

// RunnerMutex returns the mutex protecting the runner
// The REPL executor should hold this lock when accessing the runner
func (i *Interpreter) RunnerMutex() *sync.RWMutex {
	return &i.runnerMu
}

// GetWorkingDir returns the current working directory from the sh runner
func (i *Interpreter) GetWorkingDir() string {
	i.runnerMu.RLock()
	defer i.runnerMu.RUnlock()
	return i.runner.Dir
}

// GetEnv returns an environment variable
// It first checks runner.Vars (for variables set during the session),
// then falls back to os.Getenv (for inherited environment variables)
func (i *Interpreter) GetEnv(name string) string {
	i.runnerMu.RLock()
	defer i.runnerMu.RUnlock()
	// First check runner.Vars for variables set during the session
	if i.runner.Vars != nil {
		if v, ok := i.runner.Vars[name]; ok {
			return v.String()
		}
	}
	// Fall back to OS environment for inherited variables
	return os.Getenv(name)
}

// SetEnv sets an environment variable by running an export command through the runner
// This ensures the variable is properly inherited by subshells
func (i *Interpreter) SetEnv(name, value string) {
	i.runnerMu.Lock()
	defer i.runnerMu.Unlock()

	// Escape the value for shell (simple escaping - wrap in single quotes and escape single quotes)
	escapedValue := strings.ReplaceAll(value, "'", "'\"'\"'")
	cmd := fmt.Sprintf("export %s='%s'", name, escapedValue)

	prog, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		// Fallback to direct assignment if parsing fails
		if i.runner.Vars == nil {
			i.runner.Vars = make(map[string]expand.Variable)
		}
		i.runner.Vars[name] = expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		}
		return
	}

	// Run the export command - ignore errors since this is best-effort
	_ = i.runner.Run(context.Background(), prog)
}

// UnsetEnv removes an environment variable by running an unset command through the runner
// This ensures the variable is properly removed from subshells
func (i *Interpreter) UnsetEnv(name string) {
	i.runnerMu.Lock()
	defer i.runnerMu.Unlock()

	cmd := fmt.Sprintf("unset %s", name)
	prog, err := syntax.NewParser().Parse(strings.NewReader(cmd), "")
	if err != nil {
		// Fallback to direct deletion if parsing fails
		if i.runner.Vars != nil {
			delete(i.runner.Vars, name)
		}
		return
	}

	// Run the unset command - ignore errors since this is best-effort
	_ = i.runner.Run(context.Background(), prog)
}

// EvalString parses and evaluates a source string in the interpreter.
// Pass nil for origin if no import resolution is needed (e.g., REPL input).
// For filesystem scripts, pass a ScriptOrigin with Type=OriginFilesystem.
// For embedded scripts, pass a ScriptOrigin with Type=OriginEmbed and EmbedFS set.
func (i *Interpreter) EvalString(source string, origin *ScriptOrigin) (*EvalResult, error) {
	// Set up origin if provided
	var prevOrigin *ScriptOrigin
	if origin != nil {
		prevOrigin = i.currentOrigin
		i.currentOrigin = origin
	}

	// Ensure we restore state on exit
	defer func() {
		if origin != nil {
			i.currentOrigin = prevOrigin
		}
	}()

	lex := lexer.New(source)
	p := parser.New(lex)
	program := p.ParseProgram()

	// Check for parser errors
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors: %s", strings.Join(p.Errors(), "; "))
	}

	return i.Eval(program)
}

// Close cleans up resources used by the interpreter
func (i *Interpreter) Close() error {
	var lastErr error

	// Close all ACP clients
	i.acpClientsMu.Lock()
	for _, entry := range i.acpClients {
		if entry.client != nil {
			if err := entry.client.Close(); err != nil {
				lastErr = err
			}
		}
	}
	i.acpClients = make(map[string]*acpClientEntry)
	i.acpClientsMu.Unlock()

	// Close MCP manager
	if i.mcpManager != nil {
		if err := i.mcpManager.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// SetVariable defines or updates a variable in the interpreter's environment
func (i *Interpreter) SetVariable(name string, value Value) {
	i.env.Set(name, value)
}

// GetVariables returns all top-level variables from the interpreter's environment (excluding built-ins)
func (i *Interpreter) GetVariables() map[string]Value {
	vars := make(map[string]Value)
	for k, v := range i.env.store {
		// Skip built-in functions and objects
		if isBuiltin(k) {
			continue
		}
		vars[k] = v
	}
	return vars
}

// GetEventHandlers returns all registered handlers for a given event name
func (i *Interpreter) GetEventHandlers(eventName string) []*ToolValue {
	return i.eventManager.GetHandlers(eventName)
}

// EmitEvent emits an event by executing the middleware chain.
// Each middleware handler receives (ctx, next) where:
//   - ctx: event-specific context object
//   - next: function to call the next middleware in chain
//
// Middleware can:
//   - Pass through: return next(ctx) - continues to next middleware
//   - Stop chain and override: return { ... } without calling next
//   - Transform context: modify ctx, then return next(ctx)
//
// The return value from the chain (if any non-null value) can be used
// to override default behavior. The interpretation is event-specific.
func (i *Interpreter) EmitEvent(eventName string, ctx Value) Value {
	handlers := i.eventManager.GetHandlers(eventName)
	if len(handlers) == 0 {
		return nil
	}

	// Execute the middleware chain starting from the first handler
	result, err := i.executeMiddlewareChain(eventName, handlers, 0, ctx)
	if err != nil {
		// Log the error but don't fail - middleware errors shouldn't crash the system
		if i.logger != nil {
			i.logger.Warn("error in middleware chain",
				zap.String("event", eventName),
				zap.Error(err))
		} else {
			fmt.Fprintf(os.Stderr, "gsh: error in middleware chain for event '%s': %v\n", eventName, err)
		}
		return nil
	}

	return result
}

// executeMiddlewareChain executes middleware handlers recursively
func (i *Interpreter) executeMiddlewareChain(eventName string, handlers []*ToolValue, index int, ctx Value) (Value, error) {
	// If we've exhausted all middleware, return nil (no override)
	if index >= len(handlers) {
		return nil, nil
	}

	handler := handlers[index]

	// Create the next() function that continues the chain
	nextFn := &BuiltinValue{
		Name: "next",
		Fn: func(args []Value) (Value, error) {
			// Get ctx from args (middleware may have modified it)
			nextCtx := ctx
			if len(args) > 0 && args[0] != nil {
				nextCtx = args[0]
			}

			// Execute next middleware in chain
			return i.executeMiddlewareChain(eventName, handlers, index+1, nextCtx)
		},
	}

	// Call the middleware with (ctx, next)
	result, err := i.CallTool(handler, []Value{ctx, nextFn})
	if err != nil {
		// Log errors at Warn level so they're visible by default
		if i.logger != nil {
			i.logger.Warn("error in middleware handler",
				zap.String("event", eventName),
				zap.String("handler", handler.Name),
				zap.Error(err))
		} else {
			fmt.Fprintf(os.Stderr, "gsh: error in middleware handler '%s' for event '%s': %v\n", handler.Name, eventName, err)
		}
		// Continue with the rest of the chain despite the error
		return i.executeMiddlewareChain(eventName, handlers, index+1, ctx)
	}

	// Return the result (could be nil, null, or an override value)
	// If middleware didn't call next() and returned something, that's the final result
	// If middleware called next(), its return value propagates back through the chain
	if result != nil && result.Type() != ValueTypeNull {
		return result, nil
	}

	return nil, nil
}

// SDKConfig returns the SDK configuration
func (i *Interpreter) SDKConfig() *SDKConfig {
	return i.sdkConfig
}

// Eval evaluates a program and returns the result.
func (i *Interpreter) Eval(program *parser.Program) (*EvalResult, error) {
	var finalResult Value = &NullValue{}

	for _, stmt := range program.Statements {
		val, err := i.evalStatement(stmt)
		if err != nil {
			return nil, i.wrapError(err, stmt)
		}
		finalResult = val
	}

	return &EvalResult{
		FinalResult: finalResult,
		Env:         i.env,
	}, nil
}

// wrapError wraps an error with stack trace information
func (i *Interpreter) wrapError(err error, node parser.Node) error {
	// Don't wrap control flow errors
	if _, isControlFlow := err.(*ControlFlowError); isControlFlow {
		return err
	}

	// Get the current goroutine's call stack
	callStack := i.callStacks.get()

	// If it's already a RuntimeError, add current stack
	if rte, ok := err.(*RuntimeError); ok {
		// Add all frames from the call stack
		for _, frame := range callStack {
			rte.AddStackFrame(frame.FunctionName, frame.Location)
		}
		return rte
	}

	// Otherwise, create a new RuntimeError
	rte := &RuntimeError{
		Message:    err.Error(),
		StackTrace: make([]StackFrame, len(callStack)),
	}
	copy(rte.StackTrace, callStack)

	return rte
}

// pushStackFrame adds a frame to the current goroutine's call stack
func (i *Interpreter) pushStackFrame(functionName, location string) {
	i.callStacks.push(functionName, location)
}

// popStackFrame removes the top frame from the current goroutine's call stack
func (i *Interpreter) popStackFrame() {
	i.callStacks.pop()
}
