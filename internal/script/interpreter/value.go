package interpreter

import (
	"fmt"
	"strconv"
	"strings"
)

// ValueType represents the type of a value
type ValueType int

const (
	// ValueTypeNull represents a null value
	ValueTypeNull ValueType = iota
	// ValueTypeNumber represents a number value
	ValueTypeNumber
	// ValueTypeString represents a string value
	ValueTypeString
	// ValueTypeBool represents a boolean value
	ValueTypeBool
	// ValueTypeArray represents an array value
	ValueTypeArray
	// ValueTypeObject represents an object value
	ValueTypeObject
	// ValueTypeTool represents a tool/function value
	ValueTypeTool
	// ValueTypeError represents an error value
	ValueTypeError
	// ValueTypeModel represents a model configuration
	ValueTypeModel
	// ValueTypeAgent represents an agent configuration
	ValueTypeAgent
	// ValueTypeConversation represents a conversation state
	ValueTypeConversation
	// ValueTypeMap represents a map value
	ValueTypeMap
	// ValueTypeSet represents a set value
	ValueTypeSet
)

// String returns the string representation of the value type
func (vt ValueType) String() string {
	switch vt {
	case ValueTypeNull:
		return "null"
	case ValueTypeNumber:
		return "number"
	case ValueTypeString:
		return "string"
	case ValueTypeBool:
		return "boolean"
	case ValueTypeArray:
		return "array"
	case ValueTypeObject:
		return "object"
	case ValueTypeTool:
		return "tool"
	case ValueTypeError:
		return "error"
	case ValueTypeModel:
		return "model"
	case ValueTypeAgent:
		return "agent"
	case ValueTypeConversation:
		return "conversation"
	case ValueTypeMap:
		return "map"
	case ValueTypeSet:
		return "set"
	default:
		return "unknown"
	}
}

// Value represents a runtime value in the interpreter
type Value interface {
	Type() ValueType
	String() string
	IsTruthy() bool
	Equals(other Value) bool
}

// NullValue represents a null value
type NullValue struct{}

func (n *NullValue) Type() ValueType { return ValueTypeNull }
func (n *NullValue) String() string  { return "null" }
func (n *NullValue) IsTruthy() bool  { return false }
func (n *NullValue) Equals(other Value) bool {
	_, ok := other.(*NullValue)
	return ok
}

// NumberValue represents a number value
type NumberValue struct {
	Value float64
}

func (n *NumberValue) Type() ValueType { return ValueTypeNumber }
func (n *NumberValue) String() string {
	// Format number intelligently
	if n.Value == float64(int64(n.Value)) {
		return strconv.FormatInt(int64(n.Value), 10)
	}
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}
func (n *NumberValue) IsTruthy() bool { return n.Value != 0 }
func (n *NumberValue) Equals(other Value) bool {
	if otherNum, ok := other.(*NumberValue); ok {
		return n.Value == otherNum.Value
	}
	return false
}

// StringValue represents a string value
type StringValue struct {
	Value string
}

func (s *StringValue) Type() ValueType { return ValueTypeString }
func (s *StringValue) String() string  { return s.Value }
func (s *StringValue) IsTruthy() bool  { return s.Value != "" }
func (s *StringValue) Equals(other Value) bool {
	if otherStr, ok := other.(*StringValue); ok {
		return s.Value == otherStr.Value
	}
	return false
}

// BoolValue represents a boolean value
type BoolValue struct {
	Value bool
}

func (b *BoolValue) Type() ValueType { return ValueTypeBool }
func (b *BoolValue) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}
func (b *BoolValue) IsTruthy() bool { return b.Value }
func (b *BoolValue) Equals(other Value) bool {
	if otherBool, ok := other.(*BoolValue); ok {
		return b.Value == otherBool.Value
	}
	return false
}

// ArrayValue represents an array value
type ArrayValue struct {
	Elements []Value
}

func (a *ArrayValue) Type() ValueType { return ValueTypeArray }
func (a *ArrayValue) String() string {
	var out strings.Builder
	out.WriteString("[")
	for i, elem := range a.Elements {
		if i > 0 {
			out.WriteString(", ")
		}
		// For strings, add quotes in the array representation
		if elem.Type() == ValueTypeString {
			out.WriteString(`"`)
			out.WriteString(elem.String())
			out.WriteString(`"`)
		} else {
			out.WriteString(elem.String())
		}
	}
	out.WriteString("]")
	return out.String()
}
func (a *ArrayValue) IsTruthy() bool { return len(a.Elements) > 0 }
func (a *ArrayValue) Equals(other Value) bool {
	if otherArr, ok := other.(*ArrayValue); ok {
		if len(a.Elements) != len(otherArr.Elements) {
			return false
		}
		for i := range a.Elements {
			if !a.Elements[i].Equals(otherArr.Elements[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// PropertyDescriptor represents metadata about an object property
type PropertyDescriptor struct {
	Value    Value             // Static value (used if Getter is nil)
	ReadOnly bool              // Whether the property is read-only
	Getter   func() Value      // Dynamic getter (takes precedence over Value)
	Setter   func(Value) error // Custom setter (for validation/side effects)
}

// ObjectValue represents an object value
type ObjectValue struct {
	Properties map[string]*PropertyDescriptor
}

func (o *ObjectValue) Type() ValueType { return ValueTypeObject }
func (o *ObjectValue) String() string {
	var out strings.Builder
	out.WriteString("{")
	first := true
	for key := range o.Properties {
		if !first {
			out.WriteString(", ")
		}
		first = false
		out.WriteString(key)
		out.WriteString(": ")

		// Get the actual value
		value := o.GetPropertyValue(key)

		// For strings, add quotes in the object representation
		if value.Type() == ValueTypeString {
			out.WriteString(`"`)
			out.WriteString(value.String())
			out.WriteString(`"`)
		} else {
			out.WriteString(value.String())
		}
	}
	out.WriteString("}")
	return out.String()
}
func (o *ObjectValue) IsTruthy() bool { return len(o.Properties) > 0 }
func (o *ObjectValue) Equals(other Value) bool {
	if otherObj, ok := other.(*ObjectValue); ok {
		if len(o.Properties) != len(otherObj.Properties) {
			return false
		}
		for key := range o.Properties {
			thisVal := o.GetPropertyValue(key)
			otherVal := otherObj.GetPropertyValue(key)
			if !thisVal.Equals(otherVal) {
				return false
			}
		}
		return true
	}
	return false
}

// GetPropertyValue gets the actual value of a property, handling getters
func (o *ObjectValue) GetPropertyValue(key string) Value {
	desc, exists := o.Properties[key]
	if !exists {
		return &NullValue{}
	}
	if desc.Getter != nil {
		return desc.Getter()
	}
	return desc.Value
}

// SetPropertyValue sets a property value, respecting read-only and custom setters
func (o *ObjectValue) SetPropertyValue(key string, value Value) error {
	desc, exists := o.Properties[key]
	if !exists {
		// Property doesn't exist, create new descriptor
		o.Properties[key] = &PropertyDescriptor{
			Value:    value,
			ReadOnly: false,
		}
		return nil
	}

	// Check if read-only
	if desc.ReadOnly {
		return fmt.Errorf("cannot set read-only property '%s'", key)
	}

	// Use custom setter if available
	if desc.Setter != nil {
		return desc.Setter(value)
	}

	// Update the value
	desc.Value = value
	return nil
}

// DeepMerge creates a new ObjectValue by deeply merging this object with another.
// Properties from the override object take precedence over properties in this object.
// When both objects have a property with the same key and both values are ObjectValues,
// those nested objects are recursively merged. Otherwise, the override value replaces
// the base value entirely.
//
// The returned ObjectValue is completely independent - modifying it will not affect
// either the receiver or the override object.
func (o *ObjectValue) DeepMerge(override *ObjectValue) *ObjectValue {
	if override == nil {
		// Return a deep copy if no override
		return o.DeepCopy()
	}

	merged := &ObjectValue{
		Properties: make(map[string]*PropertyDescriptor),
	}

	// Deep copy all properties from base (this object)
	for key, desc := range o.Properties {
		merged.Properties[key] = deepCopyDescriptor(desc)
	}

	// Merge/override with properties from override object
	for key, overrideDesc := range override.Properties {
		baseDesc, exists := merged.Properties[key]
		if exists {
			// Both have this key - check if we need to deep merge
			baseVal := baseDesc.Value
			overrideVal := overrideDesc.Value

			baseObj, baseIsObj := baseVal.(*ObjectValue)
			overrideObj, overrideIsObj := overrideVal.(*ObjectValue)

			if baseIsObj && overrideIsObj {
				// Both are objects - recursively merge
				merged.Properties[key] = &PropertyDescriptor{
					Value:    baseObj.DeepMerge(overrideObj),
					ReadOnly: overrideDesc.ReadOnly, // Use override's metadata
					Getter:   overrideDesc.Getter,
					Setter:   overrideDesc.Setter,
				}
			} else {
				// Override replaces base (different types or non-objects)
				merged.Properties[key] = deepCopyDescriptor(overrideDesc)
			}
		} else {
			// Key only in override - add it (deep copy to ensure independence)
			merged.Properties[key] = deepCopyDescriptor(overrideDesc)
		}
	}

	return merged
}

// DeepCopy creates a completely independent deep copy of the ObjectValue.
// Modifying the copy will not affect the original object.
func (o *ObjectValue) DeepCopy() *ObjectValue {
	if o == nil {
		return nil
	}
	copied := &ObjectValue{
		Properties: make(map[string]*PropertyDescriptor, len(o.Properties)),
	}
	for key, desc := range o.Properties {
		copied.Properties[key] = deepCopyDescriptor(desc)
	}
	return copied
}

// deepCopyDescriptor creates a deep copy of a PropertyDescriptor
func deepCopyDescriptor(desc *PropertyDescriptor) *PropertyDescriptor {
	if desc == nil {
		return nil
	}
	return &PropertyDescriptor{
		Value:    deepCopyValue(desc.Value),
		ReadOnly: desc.ReadOnly,
		Getter:   desc.Getter, // Functions are immutable, safe to share
		Setter:   desc.Setter,
	}
}

// deepCopyValue creates a deep copy of a Value.
// For ObjectValue and ArrayValue, this creates independent copies.
// For primitive values (string, number, bool, null), the same reference is returned
// since these are immutable.
func deepCopyValue(v Value) Value {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case *ObjectValue:
		return val.DeepCopy()
	case *ArrayValue:
		return val.DeepCopy()
	default:
		// Primitive values (StringValue, NumberValue, BoolValue, NullValue, etc.)
		// are immutable, so returning the same reference is safe
		return v
	}
}

// DeepCopy creates a completely independent deep copy of the ArrayValue.
// Modifying the copy will not affect the original array.
func (a *ArrayValue) DeepCopy() *ArrayValue {
	if a == nil {
		return nil
	}
	copied := &ArrayValue{
		Elements: make([]Value, len(a.Elements)),
	}
	for i, elem := range a.Elements {
		copied.Elements[i] = deepCopyValue(elem)
	}
	return copied
}

// ErrorValue represents an error value
type ErrorValue struct {
	Message string
}

func (e *ErrorValue) Type() ValueType { return ValueTypeError }
func (e *ErrorValue) String() string  { return fmt.Sprintf("Error: %s", e.Message) }
func (e *ErrorValue) IsTruthy() bool  { return false }
func (e *ErrorValue) Equals(other Value) bool {
	if otherErr, ok := other.(*ErrorValue); ok {
		return e.Message == otherErr.Message
	}
	return false
}

// NewError creates a new error value
func NewError(format string, args ...interface{}) *ErrorValue {
	return &ErrorValue{Message: fmt.Sprintf(format, args...)}
}

// ToolValue represents a tool/function value
type ToolValue struct {
	Name       string
	Parameters []string
	ParamTypes map[string]string // parameter name -> type annotation (optional)
	ReturnType string            // return type annotation (optional)
	Body       interface{}       // *parser.BlockStatement for user-defined tools
	Env        *Environment      // closure environment
}

func (t *ToolValue) Type() ValueType { return ValueTypeTool }
func (t *ToolValue) String() string {
	return fmt.Sprintf("<tool %s>", t.Name)
}
func (t *ToolValue) IsTruthy() bool { return true }
func (t *ToolValue) Equals(other Value) bool {
	if otherTool, ok := other.(*ToolValue); ok {
		return t.Name == otherTool.Name
	}
	return false
}

// ModelValue represents a model configuration
type ModelValue struct {
	Name     string
	Config   map[string]Value
	Provider ModelProvider
}

func (m *ModelValue) Type() ValueType { return ValueTypeModel }
func (m *ModelValue) String() string {
	return fmt.Sprintf("<model %s>", m.Name)
}
func (m *ModelValue) IsTruthy() bool { return true }
func (m *ModelValue) Equals(other Value) bool {
	if otherModel, ok := other.(*ModelValue); ok {
		return m.Name == otherModel.Name
	}
	return false
}

// GetProperty returns a property of the model
func (m *ModelValue) GetProperty(name string) Value {
	switch name {
	case "name":
		return &StringValue{Value: m.Name}
	default:
		return &NullValue{}
	}
}

// SetProperty sets a property of the model (read-only, so always errors)
func (m *ModelValue) SetProperty(name string, value Value) error {
	return fmt.Errorf("cannot set property '%s' on model", name)
}

// ChatCompletion performs a chat completion using this model's provider.
// This is a convenience method that delegates to the model's provider.
func (m *ModelValue) ChatCompletion(request ChatRequest) (*ChatResponse, error) {
	if m.Provider == nil {
		return nil, fmt.Errorf("model '%s' has no provider configured", m.Name)
	}
	// Ensure the request uses this model
	request.Model = m
	return m.Provider.ChatCompletion(request)
}

// StreamingChatCompletion performs a streaming chat completion using this model's provider.
// This is a convenience method that delegates to the model's provider.
func (m *ModelValue) StreamingChatCompletion(request ChatRequest, callbacks *StreamCallbacks) (*ChatResponse, error) {
	if m.Provider == nil {
		return nil, fmt.Errorf("model '%s' has no provider configured", m.Name)
	}
	// Ensure the request uses this model
	request.Model = m
	return m.Provider.StreamingChatCompletion(request, callbacks)
}

// AgentValue represents an agent configuration
type AgentValue struct {
	Name   string
	Config map[string]Value
}

func (a *AgentValue) Type() ValueType { return ValueTypeAgent }
func (a *AgentValue) String() string {
	return fmt.Sprintf("<agent %s>", a.Name)
}
func (a *AgentValue) IsTruthy() bool { return true }
func (a *AgentValue) Equals(other Value) bool {
	if otherAgent, ok := other.(*AgentValue); ok {
		return a.Name == otherAgent.Name
	}
	return false
}

// ConversationValue represents a conversation state
type ConversationValue struct {
	// Messages in the conversation history
	Messages []ChatMessage
}

func (c *ConversationValue) Type() ValueType { return ValueTypeConversation }
func (c *ConversationValue) String() string {
	return fmt.Sprintf("<conversation with %d messages>", len(c.Messages))
}
func (c *ConversationValue) IsTruthy() bool { return len(c.Messages) > 0 }
func (c *ConversationValue) Equals(other Value) bool {
	if otherConv, ok := other.(*ConversationValue); ok {
		return len(c.Messages) == len(otherConv.Messages)
	}
	return false
}

// MapValue represents a map value (key-value pairs)
type MapValue struct {
	Entries map[string]Value
}

func (m *MapValue) Type() ValueType { return ValueTypeMap }
func (m *MapValue) String() string {
	var out strings.Builder
	out.WriteString("Map({")
	first := true
	for key, value := range m.Entries {
		if !first {
			out.WriteString(", ")
		}
		first = false
		out.WriteString(key)
		out.WriteString(" => ")
		if value.Type() == ValueTypeString {
			out.WriteString(`"`)
			out.WriteString(value.String())
			out.WriteString(`"`)
		} else {
			out.WriteString(value.String())
		}
	}
	out.WriteString("})")
	return out.String()
}
func (m *MapValue) IsTruthy() bool { return len(m.Entries) > 0 }
func (m *MapValue) Equals(other Value) bool {
	if otherMap, ok := other.(*MapValue); ok {
		if len(m.Entries) != len(otherMap.Entries) {
			return false
		}
		for key, value := range m.Entries {
			otherValue, exists := otherMap.Entries[key]
			if !exists || !value.Equals(otherValue) {
				return false
			}
		}
		return true
	}
	return false
}

// SetValue represents a set value (unique values)
type SetValue struct {
	Elements map[string]Value // Using map for uniqueness, key is String() representation
}

func (s *SetValue) Type() ValueType { return ValueTypeSet }
func (s *SetValue) String() string {
	var out strings.Builder
	out.WriteString("Set({")
	first := true
	for _, value := range s.Elements {
		if !first {
			out.WriteString(", ")
		}
		first = false
		if value.Type() == ValueTypeString {
			out.WriteString(`"`)
			out.WriteString(value.String())
			out.WriteString(`"`)
		} else {
			out.WriteString(value.String())
		}
	}
	out.WriteString("})")
	return out.String()
}
func (s *SetValue) IsTruthy() bool { return len(s.Elements) > 0 }
func (s *SetValue) Equals(other Value) bool {
	if otherSet, ok := other.(*SetValue); ok {
		if len(s.Elements) != len(otherSet.Elements) {
			return false
		}
		for key := range s.Elements {
			if _, exists := otherSet.Elements[key]; !exists {
				return false
			}
		}
		return true
	}
	return false
}
