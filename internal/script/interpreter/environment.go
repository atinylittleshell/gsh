package interpreter

import "fmt"

// Environment represents a scope for variable bindings
type Environment struct {
	store map[string]Value
	outer *Environment // parent scope for nested scopes
}

// NewEnvironment creates a new environment
func NewEnvironment() *Environment {
	return &Environment{
		store: make(map[string]Value),
		outer: nil,
	}
}

// NewEnclosedEnvironment creates a new environment enclosed by an outer environment
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get retrieves a value from the environment by name
// It searches the current scope and all parent scopes
func (e *Environment) Get(name string) (Value, bool) {
	value, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return value, ok
}

// Set assigns a value to a variable in the environment
// It always sets the variable in the current scope, potentially shadowing parent variables
func (e *Environment) Set(name string, value Value) {
	e.store[name] = value
}

// Define creates a new variable in the current scope
// It returns an error if the variable already exists in the current scope
func (e *Environment) Define(name string, value Value) error {
	if _, ok := e.store[name]; ok {
		return fmt.Errorf("variable '%s' already defined in current scope", name)
	}
	e.store[name] = value
	return nil
}

// Update updates an existing variable's value
// It returns an error if the variable doesn't exist
func (e *Environment) Update(name string, value Value) error {
	// Check current scope
	if _, ok := e.store[name]; ok {
		e.store[name] = value
		return nil
	}

	// Check parent scopes
	if e.outer != nil {
		return e.outer.Update(name, value)
	}

	return fmt.Errorf("undefined variable '%s'", name)
}

// Has checks if a variable exists in the current scope or any parent scope
func (e *Environment) Has(name string) bool {
	_, ok := e.Get(name)
	return ok
}

// Delete removes a variable from the current scope
// It returns true if the variable was found and deleted, false otherwise
func (e *Environment) Delete(name string) bool {
	if _, ok := e.store[name]; ok {
		delete(e.store, name)
		return true
	}
	return false
}

// Keys returns all variable names in the current scope (not including parent scopes)
func (e *Environment) Keys() []string {
	keys := make([]string, 0, len(e.store))
	for k := range e.store {
		keys = append(keys, k)
	}
	return keys
}

// AllKeys returns all variable names in the current scope and all parent scopes
func (e *Environment) AllKeys() []string {
	keys := make(map[string]bool)

	// Add keys from current scope
	for k := range e.store {
		keys[k] = true
	}

	// Add keys from parent scopes
	if e.outer != nil {
		for _, k := range e.outer.AllKeys() {
			keys[k] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}

// Clone creates a shallow copy of the environment (current scope only)
func (e *Environment) Clone() *Environment {
	newEnv := &Environment{
		store: make(map[string]Value, len(e.store)),
		outer: e.outer,
	}
	for k, v := range e.store {
		newEnv.store[k] = v
	}
	return newEnv
}
