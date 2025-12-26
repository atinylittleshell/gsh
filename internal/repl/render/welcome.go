// Package render provides agent output rendering functionality for the REPL.
package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// WelcomeInfo contains information to display in the welcome screen.
type WelcomeInfo struct {
	// PredictModel is the name/identifier of the prediction model (empty if not configured)
	PredictModel string
	// AgentModel is the name/identifier of the agent model (empty if not configured)
	AgentModel string
	// Version is the gsh version string
	Version string
}

// tips is the list of tips to display in the welcome screen.
// A "tip of the day" is selected based on the current date.
var tips = []string{
	// Agent basics
	"use # to chat with the agent",
	"use # /clear to reset the conversation",
	"use # /agents to list available agents",
	"use # /agent <name> to switch agents",
	"agents remember context across messages in a session",

	// Navigation and history
	"press Tab to autocomplete commands and file paths",
	"press Up/Down to navigate command history",
	"press Ctrl+A to jump to start of line",
	"press Ctrl+E to jump to end of line",

	// Configuration
	"you can customize your shell prompt in ~/.gshrc.gsh",
	"starship integration is automatic if starship is in PATH",
	"set logLevel: \"debug\" in GSH_CONFIG for troubleshooting",
	"you can define bash aliases in ~/.gshrc",

	// Predictions
	"press Ctrl+F to accept a command prediction",
	"command predictions use your command history for context",
	"use a small fast model like gemma3:1b for predictions",

	// Agents and models
	"define custom agents with specialized system prompts and tools",
	"agents in the REPL can execute shell commands and access files",

	// MCP and tools
	"connect to MCP servers to give agents more capabilities",
	"define custom agents, tools, and MCP servers in ~/.gshrc.gsh",

	// Scripts
	"run gsh scripts with: gsh script.gsh",
	"use exec() in scripts to run bash commands",

	// General tips
	"press Ctrl+D on an empty line to exit",
}

// ASCII art logo for GSH - compact version that fits well in terminals
var gshLogo = []string{
	"  __ _ ___| |__  ",
	" / _` / __| '_ \\ ",
	"| (_| \\__ \\ | | |",
	" \\__, |___/_| |_|",
	" |___/           ",
}

// getTipOfTheDay returns a tip based on the current date.
// The same tip is shown for the entire day, changing at midnight.
func getTipOfTheDay() string {
	if len(tips) == 0 {
		return ""
	}
	// Use the current date as seed to get consistent tip for the day
	now := time.Now()
	// Create a simple hash from year, month, day. The formula is wrong but good enough for this purpose.
	daysSinceEpoch := now.Year()*365 + int(now.Month())*31 + now.Day()
	index := daysSinceEpoch % len(tips)
	return tips[index]
}

// RenderWelcome renders the welcome screen to the given writer.
// The welcome screen displays the GSH logo on the left and configuration info on the right.
func RenderWelcome(w io.Writer, info WelcomeInfo, termWidth int) {
	// Define styles using the primary yellow color
	titleStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
	logoStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	labelStyle := lipgloss.NewStyle().Foreground(ColorGray)
	valueStyle := lipgloss.NewStyle().Foreground(ColorYellow)
	dimStyle := lipgloss.NewStyle().Foreground(ColorGray).Italic(true)

	// Calculate layout dimensions
	logoWidth := 18 // Width of the logo
	minGap := 4     // Minimum gap between logo and info
	maxInfoWidth := 40

	// Build info lines
	var infoLines []string

	// Title
	infoLines = append(infoLines, titleStyle.Render("The Generative Shell"))
	infoLines = append(infoLines, "")

	// Version
	if info.Version != "" && info.Version != "dev" {
		infoLines = append(infoLines, labelStyle.Render("version: ")+valueStyle.Render(info.Version))
	} else if info.Version == "dev" {
		infoLines = append(infoLines, labelStyle.Render("version: ")+dimStyle.Render("development"))
	}

	// Predict model
	if info.PredictModel != "" {
		infoLines = append(infoLines, labelStyle.Render("predict: ")+valueStyle.Render(info.PredictModel))
	} else {
		infoLines = append(infoLines, labelStyle.Render("predict: ")+dimStyle.Render("not configured"))
	}

	// Agent model
	if info.AgentModel != "" {
		infoLines = append(infoLines, labelStyle.Render("agent:   ")+valueStyle.Render(info.AgentModel))
	} else {
		infoLines = append(infoLines, labelStyle.Render("agent:   ")+dimStyle.Render("not configured"))
	}

	// Calculate the number of lines we need (max of logo or info)
	numLines := len(gshLogo)
	if len(infoLines) > numLines {
		numLines = len(infoLines)
	}

	// Calculate actual info width based on terminal width
	infoWidth := termWidth - logoWidth - minGap
	if infoWidth > maxInfoWidth {
		infoWidth = maxInfoWidth
	}
	// Get tip of the day
	tip := getTipOfTheDay()

	if infoWidth < 20 {
		// Terminal too narrow, just show info without logo
		for _, line := range infoLines {
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
		if tip != "" {
			fmt.Fprintln(w, dimStyle.Render("tip: "+tip))
		}
		fmt.Fprintln(w)
		return
	}

	// Build the two-column layout
	var output strings.Builder

	// Add a blank line before
	output.WriteString("\n")

	for i := 0; i < numLines; i++ {
		// Get logo line (or empty if past logo)
		var logoLine string
		if i < len(gshLogo) {
			logoLine = logoStyle.Render(gshLogo[i])
		} else {
			logoLine = strings.Repeat(" ", logoWidth)
		}

		// Get info line (or empty if past info)
		var infoLine string
		if i < len(infoLines) {
			infoLine = infoLines[i]
		}

		// Combine with gap
		gap := strings.Repeat(" ", minGap)
		output.WriteString(logoLine + gap + infoLine + "\n")
	}

	// Add tip spanning full width (below the two-column layout)
	output.WriteString("\n")
	if tip != "" {
		output.WriteString(dimStyle.Render("tip: "+tip) + "\n")
	}

	// Add a blank line after
	output.WriteString("\n")

	fmt.Fprint(w, output.String())
}
