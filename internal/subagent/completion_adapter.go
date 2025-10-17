package subagent

import (
	"github.com/atinylittleshell/gsh/internal/completion"
)

// CompletionAdapter adapts SubagentManager to the completion.SubagentProvider interface
// This avoids import cycles by keeping the completion package independent
type CompletionAdapter struct {
	manager        *SubagentManager
	ensureUpToDate func() // Function to ensure subagents are up-to-date
}

// NewCompletionAdapter creates a new adapter for the completion system
func NewCompletionAdapter(manager *SubagentManager, ensureUpToDate func()) *CompletionAdapter {
	return &CompletionAdapter{
		manager:        manager,
		ensureUpToDate: ensureUpToDate,
	}
}

// GetAllSubagents returns all subagents as completion.SubagentInfo
func (ca *CompletionAdapter) GetAllSubagents() map[string]*completion.SubagentInfo {
	// Ensure subagents are up-to-date before returning them
	if ca.ensureUpToDate != nil {
		ca.ensureUpToDate()
	}

	subagents := ca.manager.GetAllSubagents()
	result := make(map[string]*completion.SubagentInfo, len(subagents))

	for id, subagent := range subagents {
		result[id] = &completion.SubagentInfo{
			ID:           subagent.ID,
			Name:         subagent.Name,
			Description:  subagent.Description,
			AllowedTools: subagent.AllowedTools,
			FileRegex:    subagent.FileRegex,
			Model:        subagent.Model,
		}
	}

	return result
}

// GetSubagent returns a specific subagent as completion.SubagentInfo
func (ca *CompletionAdapter) GetSubagent(id string) (*completion.SubagentInfo, bool) {
	// Ensure subagents are up-to-date before returning them
	if ca.ensureUpToDate != nil {
		ca.ensureUpToDate()
	}

	subagent, exists := ca.manager.GetSubagent(id)
	if !exists {
		return nil, false
	}

	return &completion.SubagentInfo{
		ID:           subagent.ID,
		Name:         subagent.Name,
		Description:  subagent.Description,
		AllowedTools: subagent.AllowedTools,
		FileRegex:    subagent.FileRegex,
		Model:        subagent.Model,
	}, true
}