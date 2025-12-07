package context

import (
	"fmt"
	"runtime"
)

// SystemInfoRetriever retrieves system information context.
type SystemInfoRetriever struct{}

// NewSystemInfoRetriever creates a new SystemInfoRetriever.
func NewSystemInfoRetriever() *SystemInfoRetriever {
	return &SystemInfoRetriever{}
}

// Name returns the retriever name.
func (r *SystemInfoRetriever) Name() string {
	return "system_info"
}

// GetContext returns system information formatted for LLM context.
func (r *SystemInfoRetriever) GetContext() (string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	return fmt.Sprintf("<system_info>OS: %s, Arch: %s</system_info>", osName, arch), nil
}
