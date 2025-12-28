package interpreter

import "fmt"

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

	// Create gsh object
	gshObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"version":          {Value: &StringValue{Value: i.version}, ReadOnly: true},
			"terminal":         {Value: terminalObj, ReadOnly: true},
			"logging":          {Value: loggingObj},
			"integrations":     {Value: integrationsObj},
			"lastAgentRequest": {Value: lastAgentRequestObj, ReadOnly: true},
			"repl":             {Value: replObj, ReadOnly: true},
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
	if d.Set != nil {
		return d.Set(value)
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
	default:
		return &NullValue{}
	}
}

func (r *REPLObjectValue) SetProperty(name string, value Value) error {
	return fmt.Errorf("cannot set property '%s' on gsh.repl", name)
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
	return fmt.Errorf("cannot set property '%s' on gsh.repl.models", name)
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
