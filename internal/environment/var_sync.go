package environment

import (
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// gshVariableNames contains the list of GSH-specific environment variables
// This variable is used to maintain consistency across the codebase
var gshVariableNames = []string{
	"GSH_PROMPT", "GSH_APROMPT", "GSH_BUILD_VERSION", "GSH_LOG_LEVEL",
	"GSH_CLEAN_LOG_FILE", "GSH_MINIMUM_HEIGHT", "GSH_ASSISTANT_HEIGHT",
	"GSH_AGENT_NAME", "GSH_FAST_MODEL_API_KEY", "GSH_FAST_MODEL_BASE_URL",
	"GSH_FAST_MODEL_ID", "GSH_SLOW_MODEL_API_KEY", "GSH_SLOW_MODEL_BASE_URL",
	"GSH_SLOW_MODEL_ID", "GSH_AGENT_CONTEXT_WINDOW_TOKENS", "GSH_PAST_COMMANDS_CONTEXT_LIMIT",
	"GSH_CONTEXT_TYPES_FOR_AGENT", "GSH_CONTEXT_TYPES_FOR_PREDICTION_WITH_PREFIX",
	"GSH_CONTEXT_TYPES_FOR_PREDICTION_WITHOUT_PREFIX", "GSH_CONTEXT_TYPES_FOR_EXPLANATION",
	"GSH_CONTEXT_NUM_HISTORY_CONCISE", "GSH_CONTEXT_NUM_HISTORY_VERBOSE",
	"GSH_AGENT_APPROVED_BASH_COMMAND_REGEX", "GSH_AGENT_MACROS", "GSH_DEFAULT_TO_YES",
}

// DynamicEnviron implements expand.Environ to provide a dynamic environment
// that includes both system environment variables and GSH-specific variables
type DynamicEnviron struct {
	systemEnv expand.Environ
	gshVars   map[string]string
}

// NewDynamicEnviron creates a new DynamicEnviron that wraps the system environment
// and adds GSH-specific variables
func NewDynamicEnviron() *DynamicEnviron {
	return &DynamicEnviron{
		systemEnv: expand.ListEnviron(os.Environ()...),
		gshVars:   make(map[string]string),
	}
}

// Get retrieves a variable by name, checking GSH variables first, then system environment
func (de *DynamicEnviron) Get(name string) expand.Variable {
	// Check GSH variables first
	if value, exists := de.gshVars[name]; exists {
		return expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		}
	}

	// Fall back to system environment
	return de.systemEnv.Get(name)
}

// Each iterates over all variables, including both GSH and system variables
func (de *DynamicEnviron) Each(fn func(name string, vr expand.Variable) bool) {
	// First, iterate over GSH variables
	for name, value := range de.gshVars {
		if !fn(name, expand.Variable{
			Exported: true,
			Kind:     expand.String,
			Str:      value,
		}) {
			return
		}
	}

	// Then iterate over system environment, skipping GSH variables we already added
	de.systemEnv.Each(func(name string, vr expand.Variable) bool {
		if _, isGSH := de.gshVars[name]; !isGSH {
			return fn(name, vr)
		}
		return true
	})
}

// UpdateGSHVar updates a GSH variable in the dynamic environment
func (de *DynamicEnviron) UpdateGSHVar(name, value string) {
	de.gshVars[name] = value
}

// UpdateSystemEnv updates the system environment wrapper
func (de *DynamicEnviron) UpdateSystemEnv() {
	de.systemEnv = expand.ListEnviron(os.Environ()...)
}

// SyncVariablesToEnv syncs gsh's internal variables to system environment variables
// This makes variables like GSH_PROMPT visible to external commands like 'env'
func SyncVariablesToEnv(runner *interp.Runner) {
	// Check if we already have a DynamicEnviron, if not create one
	var dynamicEnv *DynamicEnviron
	if existingDynamicEnv, ok := runner.Env.(*DynamicEnviron); ok {
		dynamicEnv = existingDynamicEnv
	} else {
		dynamicEnv = NewDynamicEnviron()
	}

	for _, varName := range gshVariableNames {
		if varValue, exists := runner.Vars[varName]; exists {
			value := varValue.String()
			if err := os.Setenv(varName, value); err != nil {
				return
			}
			dynamicEnv.UpdateGSHVar(varName, value)
			continue
		}

		_ = os.Unsetenv(varName)
		delete(dynamicEnv.gshVars, varName)
	}

	// Update the system environment in the dynamic environment
	dynamicEnv.UpdateSystemEnv()

	// Set the runner's environment to our dynamic environment
	runner.Env = dynamicEnv
}

// SyncVariableToEnv syncs a single gsh variable to system environment
func SyncVariableToEnv(runner *interp.Runner, varName string) {
	if varValue, exists := runner.Vars[varName]; exists {
		value := varValue.String()
		if err := os.Setenv(varName, value); err != nil {
			return
		}

		// Update in the dynamic environment
		if dynamicEnv, ok := runner.Env.(*DynamicEnviron); ok {
			dynamicEnv.UpdateGSHVar(varName, value)
		}
		return
	}

	if err := os.Unsetenv(varName); err != nil {
		return
	}
	if dynamicEnv, ok := runner.Env.(*DynamicEnviron); ok {
		delete(dynamicEnv.gshVars, varName)
	}
}

// IsGSHVariable checks if a variable name is a gsh-specific variable that should be synced
func IsGSHVariable(name string) bool {
	for _, gshVar := range gshVariableNames {
		if name == gshVar {
			return true
		}
	}
	return strings.HasPrefix(name, "GSH_")
}
