package completion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
	"mvdan.cc/sh/v3/interp"
)

// CompletionType represents the type of completion
type CompletionType string

const (
	// WordListCompletion represents word list based completion (-W option)
	WordListCompletion CompletionType = "W"
	// FunctionCompletion represents function based completion (-F option)
	FunctionCompletion CompletionType = "F"
	// CommandCompletion represents command based completion (-C option)
	CommandCompletion CompletionType = "C"
)

// CompletionSpec represents a completion specification for a command
type CompletionSpec struct {
	Command string
	Type    CompletionType
	Value   string   // function name, wordlist, or command
	Options []string // additional options like -o dirname
}

// CompletionManager manages command completion specifications
type CompletionManager struct {
	specs map[string]CompletionSpec
}

// NewCompletionManager creates a new CompletionManager
func NewCompletionManager() *CompletionManager {
	return &CompletionManager{
		specs: make(map[string]CompletionSpec),
	}
}

// AddSpec adds or updates a completion specification
func (m *CompletionManager) AddSpec(spec CompletionSpec) {
	m.specs[spec.Command] = spec
}

// RemoveSpec removes a completion specification
func (m *CompletionManager) RemoveSpec(command string) {
	delete(m.specs, command)
}

// GetSpec retrieves a completion specification
func (m *CompletionManager) GetSpec(command string) (CompletionSpec, bool) {
	spec, ok := m.specs[command]
	return spec, ok
}

// ListSpecs returns all completion specifications
func (m *CompletionManager) ListSpecs() []CompletionSpec {
	specs := make([]CompletionSpec, 0, len(m.specs))
	for _, spec := range m.specs {
		specs = append(specs, spec)
	}
	return specs
}

// ExecuteCompletion executes a completion specification for a given command line
// and returns the list of possible completions
func (m *CompletionManager) ExecuteCompletion(ctx context.Context, runner *interp.Runner, spec CompletionSpec, args []string, line string, pos int) ([]shellinput.CompletionCandidate, error) {
	switch spec.Type {
	case WordListCompletion:
		words := strings.Fields(spec.Value)
		completions := make([]shellinput.CompletionCandidate, 0)
		word := ""
		if len(args) > 0 {
			word = args[len(args)-1]
		}
		for _, w := range words {
			if word == "" || strings.HasPrefix(w, word) {
				completions = append(completions, shellinput.CompletionCandidate{Value: w})
			}
		}
		return completions, nil

	case FunctionCompletion:
		fn := NewCompletionFunction(spec.Value, runner)
		strs, err := fn.Execute(ctx, args)
		if err != nil {
			return nil, err
		}
		completions := make([]shellinput.CompletionCandidate, len(strs))
		for i, s := range strs {
			completions[i] = shellinput.CompletionCandidate{Value: s}
		}
		return completions, nil

	case CommandCompletion:
		return m.RunExternalCompleter(ctx, spec.Value, args, line, pos)

	default:
		return nil, fmt.Errorf("unsupported completion type: %s", spec.Type)
	}
}

// RunExternalCompleter executes an external command to generate completions
func (m *CompletionManager) RunExternalCompleter(ctx context.Context, command string, args []string, line string, pos int) ([]shellinput.CompletionCandidate, error) {
	// Prepare arguments for the external command
	// $1 is the command name being completed
	// $2 is the word being completed
	// $3 is the word preceding that
	var arg1, arg2, arg3 string
	if len(args) > 0 {
		arg1 = args[0]
		arg2 = args[len(args)-1]
		if len(args) > 1 {
			arg3 = args[len(args)-2]
		}
	}

	// Prepare environment variables
	env := os.Environ()
	// Bash variables
	env = append(env, fmt.Sprintf("COMP_LINE=%s", line))
	env = append(env, fmt.Sprintf("COMP_POINT=%d", pos))
	env = append(env, "COMP_KEY=9")  // 9 is TAB
	env = append(env, "COMP_TYPE=9") // 9 is TAB

	// Zsh variables for compatibility
	// BUFFER: The entire command line
	env = append(env, fmt.Sprintf("BUFFER=%s", line))
	// CURSOR: The cursor position
	env = append(env, fmt.Sprintf("CURSOR=%d", pos))
	// LBUFFER: The part of the line before the cursor
	if pos <= len(line) {
		env = append(env, fmt.Sprintf("LBUFFER=%s", line[:pos]))
	}
	// RBUFFER: The part of the line after the cursor
	if pos < len(line) {
		env = append(env, fmt.Sprintf("RBUFFER=%s", line[pos:]))
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("%s \"$@\"", command), "--", arg1, arg2, arg3)
	cmd.Env = env

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// If command fails, we return empty completions rather than error
		return []shellinput.CompletionCandidate{}, nil
	}

	output := out.String()
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		return []shellinput.CompletionCandidate{}, nil
	}

	// Try to parse as JSON first (Carapace style)
	// Check if it starts with [ or {
	if strings.HasPrefix(trimmedOutput, "[") || strings.HasPrefix(trimmedOutput, "{") {
		var candidates []shellinput.CompletionCandidate
		// Try parsing as simple list of strings
		var stringList []string
		if err := json.Unmarshal([]byte(trimmedOutput), &stringList); err == nil {
			for _, s := range stringList {
				candidates = append(candidates, shellinput.CompletionCandidate{Value: s})
			}
			return candidates, nil
		}

		// Try parsing as list of objects with Value/Display/Description
		type JsonCandidate struct {
			Value       string `json:"Value"`
			Display     string `json:"Display"`
			Description string `json:"Description"`
		}
		var objList []JsonCandidate
		if err := json.Unmarshal([]byte(trimmedOutput), &objList); err == nil {
			for _, o := range objList {
				candidates = append(candidates, shellinput.CompletionCandidate{
					Value:       o.Value,
					Display:     o.Display,
					Description: o.Description,
				})
			}
			return candidates, nil
		}
	}

	// Parse line-by-line (Bash/Zsh style)
	lines := strings.Split(output, "\n")
	completions := make([]shellinput.CompletionCandidate, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}

		var candidate shellinput.CompletionCandidate

		// Try to parse as JSON (single object or array)
		if strings.HasPrefix(l, "{") || strings.HasPrefix(l, "[") {
			// Try parsing as a single JSON object with Value/Display/Description
			type JsonCandidate struct {
				Value       string `json:"Value"`
				Display     string `json:"Display"`
				Description string `json:"Description"`
			}
			var obj JsonCandidate
			if err := json.Unmarshal([]byte(l), &obj); err == nil && obj.Value != "" {
				candidate.Value = obj.Value
				candidate.Display = obj.Display
				candidate.Description = obj.Description
				completions = append(completions, candidate)
				continue
			}
			// Try parsing as an array of JSON objects
			var objList []JsonCandidate
			if err := json.Unmarshal([]byte(l), &objList); err == nil && len(objList) > 0 {
				for _, o := range objList {
					completions = append(completions, shellinput.CompletionCandidate{
						Value:       o.Value,
						Display:     o.Display,
						Description: o.Description,
					})
				}
				continue
			}
			// Try parsing as a simple list of strings
			var stringList []string
			if err := json.Unmarshal([]byte(l), &stringList); err == nil && len(stringList) > 0 {
				for _, s := range stringList {
					completions = append(completions, shellinput.CompletionCandidate{Value: s})
				}
				continue
			}
			// If JSON parsing fails, fall through to regular parsing
		}

		// Check for tab delimiter (Value\tDescription)
		if strings.Contains(l, "\t") {
			parts := strings.SplitN(l, "\t", 2)
			candidate.Value = parts[0]
			if len(parts) > 1 {
				candidate.Description = parts[1]
			}
		} else if strings.Contains(l, ":") {
			// Check for colon delimiter (Value:Description) - Zsh style
			// Be careful not to split if colon is part of the value (heuristics might be needed)
			// For now, we assume simple Zsh style "value:description"
			parts := strings.SplitN(l, ":", 2)
			candidate.Value = parts[0]
			if len(parts) > 1 {
				candidate.Description = parts[1]
			}
		} else {
			// Plain value
			candidate.Value = l
		}

		completions = append(completions, candidate)
	}

	return completions, nil
}
