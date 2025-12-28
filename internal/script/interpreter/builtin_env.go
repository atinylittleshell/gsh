package interpreter

import "fmt"

// EnvValue represents the env object for environment variable access
// It holds a reference to the interpreter to access the shared sh runner
type EnvValue struct {
	interp *Interpreter
}

func (e *EnvValue) Type() ValueType { return ValueTypeObject }
func (e *EnvValue) String() string  { return "<env>" }
func (e *EnvValue) IsTruthy() bool  { return true }
func (e *EnvValue) Equals(other Value) bool {
	_, ok := other.(*EnvValue)
	return ok
}

// GetProperty gets an environment variable from the sh runner
func (e *EnvValue) GetProperty(name string) Value {
	value := e.interp.GetEnv(name)
	if value == "" {
		// Return null if environment variable is not set
		return &NullValue{}
	}
	return &StringValue{Value: value}
}

// SetProperty sets an environment variable in the sh runner
func (e *EnvValue) SetProperty(name string, value Value) error {
	// Convert value to string
	var strValue string
	switch v := value.(type) {
	case *StringValue:
		strValue = v.Value
	case *NumberValue:
		strValue = fmt.Sprintf("%v", v.Value)
	case *BoolValue:
		if v.Value {
			strValue = "true"
		} else {
			strValue = "false"
		}
	case *NullValue:
		// Setting to null unsets the variable
		e.interp.UnsetEnv(name)
		return nil
	default:
		strValue = v.String()
	}

	e.interp.SetEnv(name, strValue)
	return nil
}
