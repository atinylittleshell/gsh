package telemetry

import (
	"fmt"
	"path/filepath"
	"strings"
)

// HandleTelemetryCommand processes the "gsh telemetry" CLI command
// Returns true if the command was handled, false if it should be passed through
func HandleTelemetryCommand(args []string) (handled bool, err error) {
	if len(args) < 2 {
		return false, nil
	}

	// Check if the first arg is "gsh" (handles both "gsh" and "./bin/gsh" etc.)
	baseName := filepath.Base(args[0])
	if baseName != "gsh" || args[1] != "telemetry" {
		return false, nil
	}

	// gsh telemetry (no subcommand) - show status
	if len(args) == 2 {
		fmt.Printf("Telemetry: %s\n", GetTelemetryStatus())
		return true, nil
	}

	subcommand := args[2]
	switch subcommand {
	case "status":
		fmt.Printf("Telemetry: %s\n", GetTelemetryStatus())
		return true, nil

	case "off":
		if err := SetTelemetryEnabled(false); err != nil {
			return true, fmt.Errorf("failed to disable telemetry: %w", err)
		}
		fmt.Println("Telemetry disabled. No data will be sent.")
		return true, nil

	case "on":
		if err := SetTelemetryEnabled(true); err != nil {
			return true, fmt.Errorf("failed to enable telemetry: %w", err)
		}
		fmt.Println("Telemetry enabled. Thank you for helping improve gsh!")
		return true, nil

	case "-h":
		printTelemetryHelp()
		return true, nil

	default:
		return true, fmt.Errorf("unknown telemetry subcommand: %s\nRun 'gsh telemetry -h' for usage", subcommand)
	}
}

func printTelemetryHelp() {
	help := []string{
		"Usage: gsh telemetry [command]",
		"",
		"Manage anonymous usage telemetry for gsh.",
		"",
		"Commands:",
		"  status    Show current telemetry status (default)",
		"  on        Enable telemetry",
		"  off       Disable telemetry",
		"  -h        Show this help message",
		"",
		"Environment Variables:",
		"  GSH_NO_TELEMETRY=1     Disable telemetry via environment",
		"  GSH_TELEMETRY_DEBUG=1  Show what would be sent (doesn't actually send)",
		"",
		"What we collect:",
		"  - gsh version, OS, CPU architecture",
		"  - Session duration",
		"  - Feature usage counts",
		"  - Error categories (not error messages)",
		"",
		"What we NEVER collect:",
		"  - Commands, prompts, or any user input",
		"  - File paths or filenames",
		"  - API keys or environment variables",
		"  - Error messages or stack traces",
		"  - Any personally identifiable information",
		"",
		"Learn more: https://github.com/atinylittleshell/gsh#telemetry",
	}
	fmt.Println(strings.Join(help, "\n"))
}
