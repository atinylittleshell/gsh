package gline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/internal/git"
	"github.com/atinylittleshell/gsh/internal/system"
	"github.com/charmbracelet/lipgloss"
)

// Command Classification
type CommandKind int

const (
	KindRawShell CommandKind = iota
	KindAgentChat
	KindAgentControl
	KindSubagent
	KindUnknown
)

// Execution Risk States
type RiskLevel int

const (
	RiskCalm RiskLevel = iota
	RiskWarning
	RiskAlert
)

// BorderStatusModel manages the state and rendering of border status elements
type BorderStatusModel struct {
	width int

	// Input State
	commandBuffer string
	kind          CommandKind
	riskScore     int
	riskLevel     RiskLevel

	// Context State
	user      string
	host      string
	cwd       string
	gitStatus *git.RepoStatus

	// Resource State
	resources *system.Resources

	// Styles
	styles BorderStyles
}

type BorderStyles struct {
	BadgeRaw     lipgloss.Style
	BadgeAgent   lipgloss.Style
	BadgeControl lipgloss.Style
	BadgeSub     lipgloss.Style

	RiskCalm    lipgloss.Style
	RiskWarning lipgloss.Style
	RiskAlert   lipgloss.Style

	ContextUser lipgloss.Style
	ContextDir  lipgloss.Style
	ContextGit  lipgloss.Style
	Divider     lipgloss.Style

	ResCool  lipgloss.Style
	ResWarm  lipgloss.Style
	ResHot   lipgloss.Style
	ResLabel lipgloss.Style
}

func NewBorderStatusModel() BorderStatusModel {
	s := BorderStyles{
		BadgeRaw:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")), // gray
		BadgeAgent:   lipgloss.NewStyle().Foreground(lipgloss.Color("75")),  // blue
		BadgeControl: lipgloss.NewStyle().Foreground(lipgloss.Color("208")), // orange
		BadgeSub:     lipgloss.NewStyle().Foreground(lipgloss.Color("141")), // purple

		RiskCalm:    lipgloss.NewStyle().Foreground(lipgloss.Color("77")),  // green
		RiskWarning: lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // amber
		RiskAlert:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")), // red

		ContextUser: lipgloss.NewStyle().Foreground(lipgloss.Color("62")),  // match border color
		ContextDir:  lipgloss.NewStyle().Foreground(lipgloss.Color("62")),  // match border color
		ContextGit:  lipgloss.NewStyle().Foreground(lipgloss.Color("246")), // gray default
		Divider:     lipgloss.NewStyle().Foreground(lipgloss.Color("62")),  // match border color

		ResCool:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),  // green
		ResWarm:  lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // amber
		ResHot:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")), // red
		ResLabel: lipgloss.NewStyle().Foreground(lipgloss.Color("62")),  // match border color
	}

	return BorderStatusModel{
		styles: s,
	}
}

func (m *BorderStatusModel) SetWidth(w int) {
	m.width = w
}

func (m *BorderStatusModel) UpdateInput(input string) {
	if m.commandBuffer == input {
		return
	}
	m.commandBuffer = input
	m.classifyCommand()
	m.computeRisk()
}

func (m *BorderStatusModel) UpdateContext(user, host, cwd string) {
	m.user = user
	m.host = host
	m.cwd = cwd
}

func (m *BorderStatusModel) UpdateGit(status *git.RepoStatus) {
	m.gitStatus = status
}

func (m *BorderStatusModel) UpdateResources(res *system.Resources) {
	m.resources = res
}

func (m *BorderStatusModel) classifyCommand() {
	input := strings.TrimSpace(m.commandBuffer)
	if strings.HasPrefix(input, "@!") {
		m.kind = KindAgentControl
	} else if strings.HasPrefix(input, "@") {
		// Check for subagent?
		// For now simple heuristic: @name ...
		parts := strings.Fields(input)
		if len(parts) > 0 && len(parts[0]) > 1 {
			// Assume subagent if it looks like one, or agent chat if just @
			// Spec says: starts with @ followed by text but not @! is agent chat.
			// starts with @name where name matches subagent is Subagent.
			// We don't have subagent list here easily.
			// We'll treat all @name as subagent/agent potentially.
			// Let's stick to simple: @ is chat, @name is subagent?
			// Spec: "Agent chat: starts with @ followed by text... (e.g. @explain)"
			// Wait, "@explain" is a command to the default agent? Or is "explain" a subagent?
			// In gsh, "@" invokes default agent. "@@" selects subagent?
			// Spec says: "Subagent: starts with @name where name matches a discovered subagent."
			// Since we don't know discovered subagents here, we might need to default to Agent Chat
			// unless we can verify.
			// Let's simplify: @! -> Control. @... -> Chat (default).
			// If we want to distinguish Subagent, we'd need injection of known subagents.
			// For now, map all @... to KindAgentChat or KindSubagent based on simple rule?
			// Let's map @ (alone) or @ followed by space to Chat.
			// @word to Subagent?
			if input == "@" || strings.HasPrefix(input, "@ ") {
				m.kind = KindAgentChat
			} else {
				// @something
				m.kind = KindSubagent
			}
		} else {
			m.kind = KindAgentChat
		}
	} else {
		m.kind = KindRawShell
	}
}

func (m *BorderStatusModel) computeRisk() {
	// Simple heuristic implementation
	score := 0
	input := m.commandBuffer

	// Destructive flags
	if strings.Contains(input, "rm -rf") || strings.Contains(input, ":(){") || strings.Contains(input, "mkfs") {
		score += 5
	}
	if strings.Contains(input, "dd if=") || strings.Contains(input, "chmod -R") || strings.Contains(input, "chown -R") {
		score += 3
	}

	// Privilege
	if strings.HasPrefix(input, "sudo") || strings.HasPrefix(input, "doas") || strings.HasPrefix(input, "su -") {
		score += 2
	}

	// Network
	if strings.Contains(input, "curl") || strings.Contains(input, "wget") || strings.Contains(input, "ssh") || strings.Contains(input, "scp") {
		score += 1
	}

	// Safe signals
	if strings.HasPrefix(input, "echo") || strings.HasPrefix(input, "ls") || strings.HasPrefix(input, "pwd") {
		score -= 1
	}

	if score < 0 {
		score = 0
	}
	m.riskScore = score

	if score <= 2 {
		m.riskLevel = RiskCalm
	} else if score <= 5 {
		m.riskLevel = RiskWarning
	} else {
		m.riskLevel = RiskAlert
	}
}

// Rendering

func (m BorderStatusModel) RenderTopLeft() string {
	// Badge
	var badge string
	var style lipgloss.Style
	switch m.kind {
	case KindRawShell:
		badge = "$"
		style = m.styles.BadgeRaw
	case KindAgentChat:
		badge = "🤖" // or @
		style = m.styles.BadgeAgent
	case KindAgentControl:
		badge = "!"
		style = m.styles.BadgeControl
	case KindSubagent:
		badge = "◇"
		style = m.styles.BadgeSub
	default:
		badge = "?"
		style = m.styles.BadgeRaw
	}

	// Risk Meter
	// ▂▆█
	// 3-5 cells.
	// Calm: muted/green thin bar
	// Warning: amber bar 2-3 ticks
	// Alert: red bar full

	var riskBar string
	var riskStyle lipgloss.Style

	switch m.riskLevel {
	case RiskCalm:
		riskBar = "▂"
		riskStyle = m.styles.RiskCalm
	case RiskWarning:
		riskBar = "▂▆"
		riskStyle = m.styles.RiskWarning
	case RiskAlert:
		riskBar = "▂▆█"
		riskStyle = m.styles.RiskAlert
	}

	// Add space to the left of the badge for consistent spacing
	return " " + style.Render(badge) + " " + riskStyle.Render(riskBar) + " "
}

// TopLeftWidth returns the actual display width of the top-left section.
// This accounts for terminal-specific rendering of emoji characters like 🤖.
func (m BorderStatusModel) TopLeftWidth() int {
	// Calculate width based on badge type
	var badgeWidth int
	switch m.kind {
	case KindAgentChat:
		// Robot emoji has ambiguous width - use terminal probing
		badgeWidth = GetRobotWidth()
	default:
		// Other badges are single-width ASCII characters
		badgeWidth = 1
	}

	// Risk bar width based on level
	var riskBarWidth int
	switch m.riskLevel {
	case RiskCalm:
		riskBarWidth = 1 // "▂"
	case RiskWarning:
		riskBarWidth = 2 // "▂▆"
	case RiskAlert:
		riskBarWidth = 3 // "▂▆█"
	}

	// Total: leading space + badge + space + riskBar + trailing space
	return 1 + badgeWidth + 1 + riskBarWidth + 1
}

func (m BorderStatusModel) RenderTopContext(maxWidth int) string {
	// Items to display: [Dir with Git icons], [optional: User@Host if space allows]
	// Git status is now appended directly to the directory display
	// Strategy: Keep directory with git status, optionally show user@host if space allows

	// Prepare raw strings
	var items []string
	var styles []lipgloss.Style

	// Dir with Git Status appended
	dir := m.cwd
	if len(dir) > 20 {
		// Resolve home directory safely - first try HOME env var, then os.UserHomeDir()
		var homeDir string
		homeEnv := os.Getenv("HOME")
		if homeEnv != "" {
			homeDir = homeEnv
		} else {
			// HOME is empty, try os.UserHomeDir() as fallback
			var err error
			homeDir, err = os.UserHomeDir()
			if err != nil {
				homeDir = "" // Failed to resolve home, will use basename logic
			}
		}

		// Only perform "~" substitution if we have a valid home directory and dir starts with it
		if homeDir != "" && strings.HasPrefix(dir, homeDir) {
			dir = "~" + dir[len(homeDir):]
		} else {
			// Fall back to basename logic for long paths
			dir = filepath.Base(dir)
			if dir == "/" {
				dir = "/"
			} else {
				dir = ".../" + dir
			}
		}
	}

	// If we have a valid dir, prepare it with git status icons appended.
	// This is now the main display element instead of a separate git section.
	if dir != "" {
		displayStr := " " + m.styles.ContextDir.Render(dir)

		if m.gitStatus != nil {
			var symbol string
			var gitStyle lipgloss.Style

			if !m.gitStatus.Clean {
				if m.gitStatus.Conflict {
					symbol = "!"
					gitStyle = m.styles.RiskAlert
				} else {
					symbol = "●"
					gitStyle = m.styles.RiskWarning
				}
			} else {
				symbol = "✓"
				gitStyle = m.styles.RiskCalm
			}

			// Add space + symbol
			displayStr += " " + gitStyle.Render(symbol)

			// Arrows - use the same gitStyle for consistency
			if m.gitStatus.Ahead > 0 {
				displayStr += gitStyle.Render(fmt.Sprintf(" ⬆%d", m.gitStatus.Ahead))
			}
			if m.gitStatus.Behind > 0 {
				displayStr += gitStyle.Render(fmt.Sprintf(" ⬇%d", m.gitStatus.Behind))
			}

			// Add space to the right of git status
			displayStr += " "
		}

		items = append(items, displayStr)
		styles = append(styles, lipgloss.NewStyle()) // Style embedded in string
	}

	if len(items) == 0 {
		// Just fill
		if maxWidth > 0 {
			return m.styles.Divider.Render(strings.Repeat("─", maxWidth))
		}
		return ""
	}

	// Calculate widths
	totalContentWidth := 0
	for _, item := range items {
		totalContentWidth += lipgloss.Width(item)
	}

	// Git status expansion logic removed - git status is now shown as icons with directory
	// No need for separate git branch expansion since we only show status icons

	// If it doesn't fit, drop items
	// Prioritize: Directory (with git icons) > everything else
	// Git status is now shown as icons with directory, so no separate git section
	// Items list is now typically: [Dir with git icons] only

	// Try to fit all
	// We need spacing gaps. At least 1 char per gap?
	// If we use "spread evenly", we have gaps before each item.
	// Minimum width = content + len(items).

	// Reduce strategy simplified:
	// 1. Keep Directory with git icons (primary)
	// 2. Drop everything else if needed (rare case)

	activeIndices := []int{}
	// Initially all
	for i := range items {
		activeIndices = append(activeIndices, i)
	}

	checkFit := func(indices []int) bool {
		w := 0
		for _, i := range indices {
			w += lipgloss.Width(items[i])
		}
		// Need gaps.
		// Distribute logic: LeftAnchor --[gap]-- Item1 --[gap]-- Item2 ...
		// We have `LeftAnchor` already rendered. So we are filling `maxWidth`.
		// We need `len(indices)` gaps.
		// Min gap size = 1?
		minWidth := w + len(indices)
		return minWidth <= maxWidth
	}

	if !checkFit(activeIndices) {
		// Since we now prioritize directory with git icons above all,
		// and typically only have one item, this should rarely trigger
		// But if it does, ensure we keep at least the directory
		if len(items) > 1 {
			// Keep only the first item (which should be the directory with git icons)
			activeIndices = []int{0}
		} else {
			// 1 item, if it doesn't fit, maybe truncate it?
			// For now, if even 1 item doesn't fit + 1 gap, we might return empty or truncated.
			// Dir is already truncated.
			if totalContentWidth > maxWidth {
				// Fallback to just lines
				if maxWidth > 0 {
					return m.styles.Divider.Render(strings.Repeat("─", maxWidth))
				}
				return ""
			}
		}
	}

	// Now we have indices that fit.
	// Distribute them with intelligent centering.
	// Space = maxWidth - contentWidth.
	// Gaps = len(activeIndices).
	// We put a gap BEFORE each item.

	contentWidth := 0
	for _, i := range activeIndices {
		contentWidth += lipgloss.Width(items[i])
	}

	totalSpace := maxWidth - contentWidth
	numGaps := len(activeIndices)

	if numGaps == 0 {
		if maxWidth > 0 {
			return m.styles.Divider.Render(strings.Repeat("─", maxWidth))
		}
		return ""
	}

	// Handle edge case: insufficient space for content + gaps
	// Ensure we have enough space for minimum gap requirements
	minRequiredWidth := contentWidth + numGaps // 1 char per gap minimum
	if maxWidth < minRequiredWidth {
		// Not enough space for content + gaps, just render a simple divider
		if maxWidth > 0 {
			return m.styles.Divider.Render(strings.Repeat("─", maxWidth))
		}
		return ""
	}

	baseGap := totalSpace / numGaps
	extraGap := totalSpace % numGaps

	var sb strings.Builder

	for i, idx := range activeIndices {
		gapSize := baseGap
		if i < extraGap {
			gapSize++
		}

		// Ensure gapSize is never negative (defensive programming)
		if gapSize < 0 {
			gapSize = 0
		}

		// Render Gap
		sb.WriteString(m.styles.Divider.Render(strings.Repeat("─", gapSize)))

		// Render Item
		sb.WriteString(styles[idx].Render(items[idx]))
	}

	return sb.String()
}

func (m BorderStatusModel) RenderBottomLeft() string {
	if m.resources == nil {
		return m.styles.ResLabel.Render("C: --% R: --%")
	}

	// CPU
	cpu := m.resources.CPUPercent
	cpuStr := m.styles.ResLabel.Render("C:") + m.formatPercentage(cpu/100.0)

	// RAM
	ramRatio := 0.0
	if m.resources.RAMTotal > 0 {
		ramRatio = float64(m.resources.RAMUsed) / float64(m.resources.RAMTotal)
	}
	ramStr := m.styles.ResLabel.Render("R:") + m.formatPercentage(ramRatio)

	// Add spaces around the resource display to match lightning bolt formatting
	return " " + cpuStr + " " + ramStr + " "
}

func (m BorderStatusModel) RenderBottomCenter() string {
	// User@Host - centered at bottom
	host := m.host
	if len(host) > 16 {
		host = host[:16]
	}
	if m.user != "" {
		// Add spaces around user@host to match lightning bolt formatting
		return " " + m.styles.ContextUser.Render(fmt.Sprintf("%s@%s", m.user, host)) + " "
	}
	return ""
}

func (m BorderStatusModel) formatPercentage(ratio float64) string {
	// Color:
	// < 0.5: Cool (Green)
	// < 0.8: Warm (Amber)
	// > 0.8: Hot (Red)

	var style lipgloss.Style
	if ratio < 0.5 {
		style = m.styles.ResCool
	} else if ratio < 0.8 {
		style = m.styles.ResWarm
	} else {
		style = m.styles.ResHot
	}

	pct := int(ratio * 100)
	text := fmt.Sprintf("%d%%", pct)

	return style.Render(text)
}
