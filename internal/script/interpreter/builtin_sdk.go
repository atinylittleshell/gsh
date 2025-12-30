package interpreter

import (
	"fmt"
	"math"
	"math/rand"
)

// registerGshSDK registers the gsh SDK object with all its properties
func (i *Interpreter) registerGshSDK() {
	// Create gsh.terminal object (read-only properties)
	terminalObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"width":  {Value: &DynamicValue{Get: func() Value { return &NumberValue{Value: float64(i.sdkConfig.GetTermWidth())} }}, ReadOnly: true},
			"height": {Value: &DynamicValue{Get: func() Value { return &NumberValue{Value: float64(i.sdkConfig.GetTermHeight())} }}, ReadOnly: true},
			"isTTY":  {Value: &BoolValue{Value: i.sdkConfig.IsTTY()}, ReadOnly: true},
		},
	}

	// Create gsh.logging object (read/write)
	loggingObj := &LoggingObjectValue{interp: i}

	// Create gsh.integrations object
	integrationsObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"starship": {Value: &DynamicValue{
				Get: func() Value { return &BoolValue{Value: i.sdkConfig.GetStarshipEnabled()} },
				Set: func(v Value) error {
					if b, ok := v.(*BoolValue); ok {
						i.sdkConfig.SetStarshipEnabled(b.Value)
						return nil
					}
					return fmt.Errorf("gsh.integrations.starship must be a boolean")
				},
			}},
		},
	}

	// Create gsh.lastAgentRequest object (read-only, but properties updated by system)
	lastAgentRequestObj := &DynamicValue{
		Get: func() Value { return i.sdkConfig.GetLastAgentRequest() },
	}

	// Create gsh.repl object (null in script mode, set when running in REPL)
	replObj := &DynamicValue{
		Get: func() Value {
			replCtx := i.sdkConfig.GetREPLContext()
			if replCtx == nil {
				return &NullValue{}
			}
			// Return the gsh.repl object with models and lastCommand
			return &REPLObjectValue{
				context: replCtx,
			}
		},
	}

	// Create gsh.tools object with native tool implementations
	toolsObj := i.createNativeToolsObject()

	// Create gsh.ui object for UI control (spinner, styles, cursor)
	uiObj := i.createUIObject()

	// Create Math object with common methods and constants
	mathObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			// Methods
			"random": {Value: &BuiltinValue{
				Name: "Math.random",
				Fn:   builtinMathRandom,
			}, ReadOnly: true},
			"floor": {Value: &BuiltinValue{
				Name: "Math.floor",
				Fn:   builtinMathFloor,
			}, ReadOnly: true},
			"ceil": {Value: &BuiltinValue{
				Name: "Math.ceil",
				Fn:   builtinMathCeil,
			}, ReadOnly: true},
			"round": {Value: &BuiltinValue{
				Name: "Math.round",
				Fn:   builtinMathRound,
			}, ReadOnly: true},
			"abs": {Value: &BuiltinValue{
				Name: "Math.abs",
				Fn:   builtinMathAbs,
			}, ReadOnly: true},
			"min": {Value: &BuiltinValue{
				Name: "Math.min",
				Fn:   builtinMathMin,
			}, ReadOnly: true},
			"max": {Value: &BuiltinValue{
				Name: "Math.max",
				Fn:   builtinMathMax,
			}, ReadOnly: true},
			"pow": {Value: &BuiltinValue{
				Name: "Math.pow",
				Fn:   builtinMathPow,
			}, ReadOnly: true},
			"sqrt": {Value: &BuiltinValue{
				Name: "Math.sqrt",
				Fn:   builtinMathSqrt,
			}, ReadOnly: true},
			"sin": {Value: &BuiltinValue{
				Name: "Math.sin",
				Fn:   builtinMathSin,
			}, ReadOnly: true},
			"cos": {Value: &BuiltinValue{
				Name: "Math.cos",
				Fn:   builtinMathCos,
			}, ReadOnly: true},
			"tan": {Value: &BuiltinValue{
				Name: "Math.tan",
				Fn:   builtinMathTan,
			}, ReadOnly: true},
			"log": {Value: &BuiltinValue{
				Name: "Math.log",
				Fn:   builtinMathLog,
			}, ReadOnly: true},
			"log10": {Value: &BuiltinValue{
				Name: "Math.log10",
				Fn:   builtinMathLog10,
			}, ReadOnly: true},
			"log2": {Value: &BuiltinValue{
				Name: "Math.log2",
				Fn:   builtinMathLog2,
			}, ReadOnly: true},
			"exp": {Value: &BuiltinValue{
				Name: "Math.exp",
				Fn:   builtinMathExp,
			}, ReadOnly: true},
			// Constants
			"PI": {Value: &NumberValue{Value: math.Pi}, ReadOnly: true},
			"E":  {Value: &NumberValue{Value: math.E}, ReadOnly: true},
		},
	}

	// Create gsh object
	gshObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"version":          {Value: &StringValue{Value: i.version}, ReadOnly: true},
			"terminal":         {Value: terminalObj, ReadOnly: true},
			"logging":          {Value: loggingObj},
			"integrations":     {Value: integrationsObj},
			"lastAgentRequest": {Value: lastAgentRequestObj, ReadOnly: true},
			"repl":             {Value: replObj, ReadOnly: true},
			"tools":            {Value: toolsObj, ReadOnly: true},
			"ui":               {Value: uiObj, ReadOnly: true},
			"on": {Value: &BuiltinValue{
				Name: "gsh.on",
				Fn:   i.builtinGshOn,
			}, ReadOnly: true},
			"off": {Value: &BuiltinValue{
				Name: "gsh.off",
				Fn:   i.builtinGshOff,
			}, ReadOnly: true},
		},
	}

	// Register Math as a global object (not under gsh)
	i.env.Set("Math", mathObj)

	i.env.Set("gsh", gshObj)
}

// builtinGshOn implements gsh.on(event, handler)
func (i *Interpreter) builtinGshOn(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("gsh.on() takes 2 arguments (event: string, handler: tool), got %d", len(args))
	}

	eventName, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("gsh.on() first argument must be a string (event name), got %s", args[0].Type())
	}

	handler, ok := args[1].(*ToolValue)
	if !ok {
		return nil, fmt.Errorf("gsh.on() second argument must be a tool, got %s", args[1].Type())
	}

	// Register the handler and return the handler ID
	handlerID := i.eventManager.On(eventName.Value, handler)
	return &StringValue{Value: handlerID}, nil
}

// builtinGshOff implements gsh.off(event, handlerID?)
func (i *Interpreter) builtinGshOff(args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("gsh.off() takes 1 or 2 arguments (event: string, handlerID?: string), got %d", len(args))
	}

	eventName, ok := args[0].(*StringValue)
	if !ok {
		return nil, fmt.Errorf("gsh.off() first argument must be a string (event name), got %s", args[0].Type())
	}

	handlerID := ""
	if len(args) == 2 {
		if args[1].Type() != ValueTypeNull {
			handlerIDVal, ok := args[1].(*StringValue)
			if !ok {
				return nil, fmt.Errorf("gsh.off() second argument must be a string (handler ID) or null, got %s", args[1].Type())
			}
			handlerID = handlerIDVal.Value
		}
	}

	i.eventManager.Off(eventName.Value, handlerID)
	return &NullValue{}, nil
}

// builtinMathRandom implements Math.random()
// Returns a random number between 0 (inclusive) and 1 (exclusive)
func builtinMathRandom(args []Value) (Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("Math.random() takes no arguments, got %d", len(args))
	}
	return &NumberValue{Value: rand.Float64()}, nil
}

// builtinMathFloor implements Math.floor()
// Returns the largest integer less than or equal to a given number
func builtinMathFloor(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.floor() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.floor() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Floor(numVal.Value)}, nil
}

// builtinMathCeil implements Math.ceil()
// Returns the smallest integer greater than or equal to a given number
func builtinMathCeil(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.ceil() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.ceil() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Ceil(numVal.Value)}, nil
}

// builtinMathRound implements Math.round()
// Returns the value of a number rounded to the nearest integer
func builtinMathRound(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.round() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.round() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Round(numVal.Value)}, nil
}

// builtinMathAbs implements Math.abs()
// Returns the absolute value of a number
func builtinMathAbs(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.abs() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.abs() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Abs(numVal.Value)}, nil
}

// builtinMathMin implements Math.min()
// Returns the smallest of zero or more numbers
func builtinMathMin(args []Value) (Value, error) {
	if len(args) == 0 {
		return &NumberValue{Value: math.Inf(1)}, nil
	}

	min := math.Inf(1)
	for _, arg := range args {
		numVal, ok := arg.(*NumberValue)
		if !ok {
			return nil, fmt.Errorf("Math.min() arguments must be numbers, got %s", arg.Type())
		}
		if numVal.Value < min {
			min = numVal.Value
		}
	}

	return &NumberValue{Value: min}, nil
}

// builtinMathMax implements Math.max()
// Returns the largest of zero or more numbers
func builtinMathMax(args []Value) (Value, error) {
	if len(args) == 0 {
		return &NumberValue{Value: math.Inf(-1)}, nil
	}

	max := math.Inf(-1)
	for _, arg := range args {
		numVal, ok := arg.(*NumberValue)
		if !ok {
			return nil, fmt.Errorf("Math.max() arguments must be numbers, got %s", arg.Type())
		}
		if numVal.Value > max {
			max = numVal.Value
		}
	}

	return &NumberValue{Value: max}, nil
}

// builtinMathPow implements Math.pow()
// Returns the base to the exponent power
func builtinMathPow(args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Math.pow() takes exactly 2 arguments, got %d", len(args))
	}

	base, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.pow() first argument must be a number, got %s", args[0].Type())
	}

	exponent, ok := args[1].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.pow() second argument must be a number, got %s", args[1].Type())
	}

	return &NumberValue{Value: math.Pow(base.Value, exponent.Value)}, nil
}

// builtinMathSqrt implements Math.sqrt()
// Returns the square root of a number
func builtinMathSqrt(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.sqrt() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.sqrt() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Sqrt(numVal.Value)}, nil
}

// builtinMathSin implements Math.sin()
// Returns the sine of a number (in radians)
func builtinMathSin(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.sin() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.sin() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Sin(numVal.Value)}, nil
}

// builtinMathCos implements Math.cos()
// Returns the cosine of a number (in radians)
func builtinMathCos(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.cos() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.cos() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Cos(numVal.Value)}, nil
}

// builtinMathTan implements Math.tan()
// Returns the tangent of a number (in radians)
func builtinMathTan(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.tan() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.tan() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Tan(numVal.Value)}, nil
}

// builtinMathLog implements Math.log()
// Returns the natural logarithm (base e) of a number
func builtinMathLog(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.log() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.log() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Log(numVal.Value)}, nil
}

// builtinMathLog10 implements Math.log10()
// Returns the base-10 logarithm of a number
func builtinMathLog10(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.log10() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.log10() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Log10(numVal.Value)}, nil
}

// builtinMathLog2 implements Math.log2()
// Returns the base-2 logarithm of a number
func builtinMathLog2(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.log2() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.log2() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Log2(numVal.Value)}, nil
}

// builtinMathExp implements Math.exp()
// Returns e raised to the power of a number
func builtinMathExp(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Math.exp() takes exactly 1 argument, got %d", len(args))
	}

	numVal, ok := args[0].(*NumberValue)
	if !ok {
		return nil, fmt.Errorf("Math.exp() argument must be a number, got %s", args[0].Type())
	}

	return &NumberValue{Value: math.Exp(numVal.Value)}, nil
}

// LoggingObjectValue represents the gsh.logging object with dynamic properties
type LoggingObjectValue struct {
	interp *Interpreter
}

func (l *LoggingObjectValue) Type() ValueType { return ValueTypeObject }
func (l *LoggingObjectValue) String() string  { return "<gsh.logging>" }
func (l *LoggingObjectValue) IsTruthy() bool  { return true }
func (l *LoggingObjectValue) Equals(other Value) bool {
	_, ok := other.(*LoggingObjectValue)
	return ok
}

func (l *LoggingObjectValue) GetProperty(name string) Value {
	switch name {
	case "level":
		return &StringValue{Value: l.interp.sdkConfig.GetLogLevel()}
	case "file":
		file := l.interp.sdkConfig.GetLogFile()
		if file == "" {
			return &NullValue{}
		}
		return &StringValue{Value: file}
	default:
		return &NullValue{}
	}
}

func (l *LoggingObjectValue) SetProperty(name string, value Value) error {
	switch name {
	case "level":
		str, ok := value.(*StringValue)
		if !ok {
			return fmt.Errorf("gsh.logging.level must be a string, got %s", value.Type())
		}
		return l.interp.sdkConfig.SetLogLevel(str.Value)
	case "file":
		return fmt.Errorf("gsh.logging.file is read-only")
	default:
		return fmt.Errorf("cannot set property '%s' on gsh.logging", name)
	}
}

// DynamicValue represents a value with custom get/set behavior
type DynamicValue struct {
	Get func() Value
	Set func(Value) error
}

func (d *DynamicValue) Type() ValueType { return ValueTypeObject }
func (d *DynamicValue) String() string {
	if d.Get != nil {
		return d.Get().String()
	}
	return "<dynamic>"
}
func (d *DynamicValue) IsTruthy() bool {
	if d.Get != nil {
		return d.Get().IsTruthy()
	}
	return false
}
func (d *DynamicValue) Equals(other Value) bool {
	if d.Get != nil {
		return d.Get().Equals(other)
	}
	return false
}
func (d *DynamicValue) GetProperty(name string) Value {
	if d.Get != nil {
		innerVal := d.Get()
		if obj, ok := innerVal.(interface{ GetProperty(string) Value }); ok {
			return obj.GetProperty(name)
		}
	}
	return &NullValue{}
}
func (d *DynamicValue) SetProperty(name string, value Value) error {
	if d.Get != nil {
		innerVal := d.Get()
		if obj, ok := innerVal.(interface{ SetProperty(string, Value) error }); ok {
			return obj.SetProperty(name, value)
		}
	}
	return fmt.Errorf("cannot set property on dynamic value")
}

// REPLObjectValue represents the gsh.repl object with models and lastCommand properties
type REPLObjectValue struct {
	context *REPLContext
}

func (r *REPLObjectValue) Type() ValueType { return ValueTypeObject }
func (r *REPLObjectValue) String() string  { return "<gsh.repl>" }
func (r *REPLObjectValue) IsTruthy() bool  { return true }
func (r *REPLObjectValue) Equals(other Value) bool {
	_, ok := other.(*REPLObjectValue)
	return ok
}

func (r *REPLObjectValue) GetProperty(name string) Value {
	if r.context == nil {
		return &NullValue{}
	}
	switch name {
	case "models":
		return &REPLModelsObjectValue{models: r.context.Models}
	case "lastCommand":
		return &REPLLastCommandObjectValue{lastCommand: r.context.LastCommand}
	case "agents":
		return &REPLAgentsArrayValue{context: r.context}
	case "currentAgent":
		if r.context.CurrentAgent == nil {
			return &NullValue{}
		}
		return &REPLAgentObjectValue{agent: r.context.CurrentAgent, context: r.context, isDefault: r.context.CurrentAgent.Name == "default"}
	case "prompt":
		if r.context.PromptValue == nil {
			return &StringValue{Value: ""}
		}
		return r.context.PromptValue
	default:
		return &NullValue{}
	}
}

func (r *REPLObjectValue) SetProperty(name string, value Value) error {
	switch name {
	case "currentAgent":
		// Allow setting currentAgent to switch agents
		agentObj, ok := value.(*REPLAgentObjectValue)
		if !ok {
			return fmt.Errorf("gsh.repl.currentAgent must be set to an agent from gsh.repl.agents")
		}
		r.context.CurrentAgent = agentObj.agent
		// Notify REPL of agent switch
		if r.context.OnAgentSwitch != nil {
			r.context.OnAgentSwitch(agentObj.agent)
		}
		return nil
	case "prompt":
		// Allow setting the prompt string
		promptStr, ok := value.(*StringValue)
		if !ok {
			return fmt.Errorf("gsh.repl.prompt must be a string, got %s", value.Type())
		}
		if r.context != nil {
			r.context.PromptValue = promptStr
		}
		return nil
	default:
		return fmt.Errorf("cannot set property '%s' on gsh.repl", name)
	}
}

// REPLModelsObjectValue represents the gsh.repl.models object
type REPLModelsObjectValue struct {
	models *REPLModels
}

func (m *REPLModelsObjectValue) Type() ValueType { return ValueTypeObject }
func (m *REPLModelsObjectValue) String() string  { return "<gsh.repl.models>" }
func (m *REPLModelsObjectValue) IsTruthy() bool  { return true }
func (m *REPLModelsObjectValue) Equals(other Value) bool {
	_, ok := other.(*REPLModelsObjectValue)
	return ok
}

func (m *REPLModelsObjectValue) GetProperty(name string) Value {
	if m.models == nil {
		return &NullValue{}
	}
	switch name {
	case "lite":
		if m.models.Lite == nil {
			return &NullValue{}
		}
		return m.models.Lite
	case "workhorse":
		if m.models.Workhorse == nil {
			return &NullValue{}
		}
		return m.models.Workhorse
	case "premium":
		if m.models.Premium == nil {
			return &NullValue{}
		}
		return m.models.Premium
	default:
		return &NullValue{}
	}
}

func (m *REPLModelsObjectValue) SetProperty(name string, value Value) error {
	if m.models == nil {
		return fmt.Errorf("gsh.repl.models is not initialized")
	}

	// Validate that the value is a ModelValue
	modelVal, ok := value.(*ModelValue)
	if !ok {
		return fmt.Errorf("gsh.repl.models.%s must be a model, got %s", name, value.Type())
	}

	switch name {
	case "lite":
		m.models.Lite = modelVal
		return nil
	case "workhorse":
		m.models.Workhorse = modelVal
		return nil
	case "premium":
		m.models.Premium = modelVal
		return nil
	default:
		return fmt.Errorf("unknown property '%s' on gsh.repl.models", name)
	}
}

// REPLLastCommandObjectValue represents the gsh.repl.lastCommand object
type REPLLastCommandObjectValue struct {
	lastCommand *REPLLastCommand
}

func (c *REPLLastCommandObjectValue) Type() ValueType { return ValueTypeObject }
func (c *REPLLastCommandObjectValue) String() string  { return "<gsh.repl.lastCommand>" }
func (c *REPLLastCommandObjectValue) IsTruthy() bool  { return true }
func (c *REPLLastCommandObjectValue) Equals(other Value) bool {
	_, ok := other.(*REPLLastCommandObjectValue)
	return ok
}

func (c *REPLLastCommandObjectValue) GetProperty(name string) Value {
	if c.lastCommand == nil {
		return &NullValue{}
	}
	switch name {
	case "exitCode":
		return &NumberValue{Value: float64(c.lastCommand.ExitCode)}
	case "durationMs":
		return &NumberValue{Value: float64(c.lastCommand.DurationMs)}
	default:
		return &NullValue{}
	}
}

func (c *REPLLastCommandObjectValue) SetProperty(name string, value Value) error {
	return fmt.Errorf("cannot set property '%s' on gsh.repl.lastCommand", name)
}

// createNativeToolsObject creates the gsh.tools object with all native tool implementations.
// These tools use a single implementation shared between the SDK and the REPL agent.
// The tool definitions come from native_tools.go to avoid duplication.
func (i *Interpreter) createNativeToolsObject() *ObjectValue {
	return &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"exec":      {Value: CreateExecNativeTool(), ReadOnly: true},
			"grep":      {Value: CreateGrepNativeTool(), ReadOnly: true},
			"view_file": {Value: CreateViewFileNativeTool(), ReadOnly: true},
			"edit_file": {Value: CreateEditFileNativeTool(), ReadOnly: true},
		},
	}
}

// REPLAgentsArrayValue represents the gsh.repl.agents array
type REPLAgentsArrayValue struct {
	context *REPLContext
}

func (a *REPLAgentsArrayValue) Type() ValueType { return ValueTypeArray }
func (a *REPLAgentsArrayValue) String() string  { return "<gsh.repl.agents>" }
func (a *REPLAgentsArrayValue) IsTruthy() bool  { return a.context != nil && len(a.context.Agents) > 0 }
func (a *REPLAgentsArrayValue) Equals(other Value) bool {
	_, ok := other.(*REPLAgentsArrayValue)
	return ok
}

func (a *REPLAgentsArrayValue) GetProperty(name string) Value {
	if a.context == nil {
		return &NullValue{}
	}

	switch name {
	case "length":
		return &NumberValue{Value: float64(len(a.context.Agents))}
	case "push":
		// Return a builtin function for push
		return &BuiltinValue{
			Name: "gsh.repl.agents.push",
			Fn:   a.pushMethod,
		}
	default:
		return &NullValue{}
	}
}

func (a *REPLAgentsArrayValue) SetProperty(name string, value Value) error {
	return fmt.Errorf("cannot set property '%s' on gsh.repl.agents", name)
}

// GetIndex implements array indexing for gsh.repl.agents[i]
func (a *REPLAgentsArrayValue) GetIndex(index int) Value {
	if a.context == nil || index < 0 || index >= len(a.context.Agents) {
		return &NullValue{}
	}
	agent := a.context.Agents[index]
	return &REPLAgentObjectValue{
		agent:     agent,
		context:   a.context,
		isDefault: index == 0,
	}
}

// SetIndex implements array index assignment for gsh.repl.agents[i] = value
func (a *REPLAgentsArrayValue) SetIndex(index int, value Value) error {
	if index == 0 {
		return fmt.Errorf("cannot replace agents[0] (the default agent)")
	}
	return fmt.Errorf("cannot set agent at index %d; use agents.push() to add new agents", index)
}

// Len returns the length of the agents array
func (a *REPLAgentsArrayValue) Len() int {
	if a.context == nil {
		return 0
	}
	return len(a.context.Agents)
}

// pushMethod implements gsh.repl.agents.push(agentConfig)
func (a *REPLAgentsArrayValue) pushMethod(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("agents.push() takes exactly 1 argument (agent config object), got %d", len(args))
	}

	// Accept either an AgentValue directly or an object with name, model, systemPrompt, tools
	if agentVal, ok := args[0].(*AgentValue); ok {
		// Direct AgentValue - validate and add
		if agentVal.Name == "" {
			return nil, fmt.Errorf("agent must have a non-empty name")
		}
		if agentVal.Name == "default" {
			return nil, fmt.Errorf("agent name 'default' is reserved for the built-in agent")
		}
		// Check for duplicate names
		for _, existing := range a.context.Agents {
			if existing.Name == agentVal.Name {
				return nil, fmt.Errorf("agent with name '%s' already exists", agentVal.Name)
			}
		}
		// Validate model is present
		if _, hasModel := agentVal.Config["model"]; !hasModel {
			return nil, fmt.Errorf("agent '%s' must have a 'model' in config", agentVal.Name)
		}

		// Add to agents array
		a.context.Agents = append(a.context.Agents, agentVal)

		// Notify REPL of new agent
		if a.context.OnAgentAdded != nil {
			a.context.OnAgentAdded(agentVal)
		}

		// Return the new length (like JavaScript Array.push)
		return &NumberValue{Value: float64(len(a.context.Agents))}, nil
	}

	// Expect an object with name, model, systemPrompt, tools
	obj, ok := args[0].(*ObjectValue)
	if !ok {
		return nil, fmt.Errorf("agents.push() argument must be an agent or an object with name, model, systemPrompt, and tools")
	}

	// Extract name (required)
	nameVal := obj.GetPropertyValue("name")
	nameStr, ok := nameVal.(*StringValue)
	if !ok || nameStr.Value == "" {
		return nil, fmt.Errorf("agent config must have a non-empty 'name' property")
	}
	if nameStr.Value == "default" {
		return nil, fmt.Errorf("agent name 'default' is reserved for the built-in agent")
	}

	// Check for duplicate names
	for _, existing := range a.context.Agents {
		if existing.Name == nameStr.Value {
			return nil, fmt.Errorf("agent with name '%s' already exists", nameStr.Value)
		}
	}

	// Extract model (required)
	modelVal := obj.GetPropertyValue("model")
	model, ok := modelVal.(*ModelValue)
	if !ok {
		return nil, fmt.Errorf("agent config must have a 'model' property referencing a model declaration")
	}

	// Build the config map for AgentValue
	config := make(map[string]Value)
	config["model"] = model

	// Extract systemPrompt (optional)
	if promptVal := obj.GetPropertyValue("systemPrompt"); promptVal.Type() != ValueTypeNull {
		config["systemPrompt"] = promptVal
	}

	// Extract tools (optional)
	if toolsVal := obj.GetPropertyValue("tools"); toolsVal.Type() != ValueTypeNull {
		config["tools"] = toolsVal
	}

	// Create the new agent using AgentValue
	newAgent := &AgentValue{
		Name:   nameStr.Value,
		Config: config,
	}

	// Add to agents array
	a.context.Agents = append(a.context.Agents, newAgent)

	// Notify REPL of new agent
	if a.context.OnAgentAdded != nil {
		a.context.OnAgentAdded(newAgent)
	}

	// Return the new length (like JavaScript Array.push)
	return &NumberValue{Value: float64(len(a.context.Agents))}, nil
}

// REPLAgentObjectValue represents an individual agent in gsh.repl.agents
// It wraps an AgentValue and provides property access consistent with the SDK.
// When properties are modified, the OnAgentModified callback is invoked to sync
// changes to the REPL's agent manager.
type REPLAgentObjectValue struct {
	agent     *AgentValue
	context   *REPLContext
	isDefault bool // True if this is agents[0]
}

func (o *REPLAgentObjectValue) Type() ValueType { return ValueTypeObject }
func (o *REPLAgentObjectValue) String() string {
	if o.agent == nil {
		return "<agent null>"
	}
	return fmt.Sprintf("<agent %s>", o.agent.Name)
}
func (o *REPLAgentObjectValue) IsTruthy() bool { return o.agent != nil }
func (o *REPLAgentObjectValue) Equals(other Value) bool {
	if otherAgent, ok := other.(*REPLAgentObjectValue); ok {
		return o.agent == otherAgent.agent
	}
	return false
}

func (o *REPLAgentObjectValue) GetProperty(name string) Value {
	if o.agent == nil {
		return &NullValue{}
	}

	switch name {
	case "name":
		return &StringValue{Value: o.agent.Name}
	case "model":
		if model, ok := o.agent.Config["model"]; ok {
			return model
		}
		return &NullValue{}
	case "systemPrompt":
		if prompt, ok := o.agent.Config["systemPrompt"]; ok {
			return prompt
		}
		return &StringValue{Value: ""}
	case "tools":
		if tools, ok := o.agent.Config["tools"]; ok {
			return tools
		}
		return &ArrayValue{Elements: []Value{}}
	default:
		// Allow access to any other config properties
		if val, ok := o.agent.Config[name]; ok {
			return val
		}
		return &NullValue{}
	}
}

func (o *REPLAgentObjectValue) SetProperty(name string, value Value) error {
	if o.agent == nil {
		return fmt.Errorf("cannot set property on null agent")
	}

	switch name {
	case "name":
		// Agent names are immutable - they are used as keys in the agent manager
		return fmt.Errorf("cannot change agent name; create a new agent instead")

	case "model":
		model, ok := value.(*ModelValue)
		if !ok {
			return fmt.Errorf("agent model must be a model reference")
		}
		o.agent.Config["model"] = model
		o.notifyModified()
		return nil

	case "systemPrompt":
		promptStr, ok := value.(*StringValue)
		if !ok {
			return fmt.Errorf("agent systemPrompt must be a string")
		}
		o.agent.Config["systemPrompt"] = promptStr
		o.notifyModified()
		return nil

	case "tools":
		toolsArr, ok := value.(*ArrayValue)
		if !ok {
			return fmt.Errorf("agent tools must be an array")
		}
		// Validate that all elements are tools
		for i, elem := range toolsArr.Elements {
			if _, isScript := elem.(*ToolValue); !isScript {
				if _, isNative := elem.(*NativeToolValue); !isNative {
					return fmt.Errorf("agent tools[%d] must be a tool, got %s", i, elem.Type())
				}
			}
		}
		o.agent.Config["tools"] = toolsArr
		o.notifyModified()
		return nil

	default:
		return fmt.Errorf("cannot set property '%s' on agent", name)
	}
}

// notifyModified calls the OnAgentModified callback if set
func (o *REPLAgentObjectValue) notifyModified() {
	if o.context != nil && o.context.OnAgentModified != nil {
		o.context.OnAgentModified(o.agent)
	}
}
