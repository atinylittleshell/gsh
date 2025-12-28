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

	// Create gsh object
	gshObj := &ObjectValue{
		Properties: map[string]*PropertyDescriptor{
			"version":          {Value: &StringValue{Value: i.version}, ReadOnly: true},
			"terminal":         {Value: terminalObj, ReadOnly: true},
			"logging":          {Value: loggingObj},
			"integrations":     {Value: integrationsObj},
			"lastAgentRequest": {Value: lastAgentRequestObj, ReadOnly: true},
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
