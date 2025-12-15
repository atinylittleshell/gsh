package shellinput

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/muesli/ansi"
	"github.com/sahilm/fuzzy"
)

// HistoryItem represents a single command history entry with metadata
type HistoryItem struct {
	Command   string
	Directory string
	Timestamp time.Time
}

// HistoryFilterMode defines the scope of history search
type HistoryFilterMode int

const (
	HistoryFilterAll HistoryFilterMode = iota
	HistoryFilterDirectory
	HistoryFilterSession
)

func (m HistoryFilterMode) String() string {
	switch m {
	case HistoryFilterDirectory:
		return "Directory"
	case HistoryFilterSession:
		return "Session"
	default:
		return "All"
	}
}

// HistorySortMode defines the sort order of history search results
type HistorySortMode int

const (
	HistorySortRecent HistorySortMode = iota
	HistorySortRelevance
	HistorySortAlphabetical
)

func (m HistorySortMode) String() string {
	switch m {
	case HistorySortRelevance:
		return "Relevance"
	case HistorySortAlphabetical:
		return "Alphabetical"
	default:
		return "Recent"
	}
}

// historySearchState tracks the state of the rich history search
type historySearchState struct {
	filteredIndices []int // indices into Model.historyItems
	selected        int   // index into filteredIndices
	filterMode      HistoryFilterMode
	sortMode        HistorySortMode
	currentDir      string // used for filtering by directory
}

// SetRichHistory sets the history items for the rich search
func (m *Model) SetRichHistory(items []HistoryItem) {
	m.historyItems = items
}

// SetCurrentDirectory sets the current directory for filtering history
func (m *Model) SetCurrentDirectory(dir string) {
	m.historySearchState.currentDir = dir
}

// HistorySearchBoxView renders the history search box
func (m Model) HistorySearchBoxView(height, width int) string {
	if !m.inReverseSearch {
		return ""
	}

	if height <= 0 {
		height = 5 // default fallback
	}

	// Calculate header and footer height
	headerHeight := 1
	footerHeight := 1
	listHeight := height - headerHeight - footerHeight
	if listHeight < 1 {
		listHeight = 1
	}

	var content strings.Builder

	// Define styles
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // Cyan for selected
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // White/Light Gray for normal
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))    // Dim gray for metadata
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))   // Slightly brighter for help

	// Render Header
	// e.g. "Filter: All | Sort: Recent | 35 matches"
	filterText := fmt.Sprintf("Filter: %s", m.historySearchState.filterMode.String())
	sortText := fmt.Sprintf("Sort: %s", m.historySearchState.sortMode.String())
	matchCount := len(m.historySearchState.filteredIndices)
	header := headerStyle.Render(fmt.Sprintf("%s | %s | %d matches",
		filterStyle.Render(filterText),
		filterStyle.Render(sortText),
		matchCount))
	content.WriteString(header + "\n")

	if matchCount == 0 {
		content.WriteString(lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("240")).Render("No history matches found"))
		content.WriteString("\n")
		helpText := "Ctrl+F: Filter | Ctrl+O: Sort | Enter: Select | Esc: Cancel"
		content.WriteString(helpStyle.Render(helpText))
		return content.String()
	}

	totalItems := matchCount

	// Calculate pagination
	selectedIdx := m.historySearchState.selected
	if selectedIdx < 0 {
		selectedIdx = 0
	}

	// Ensure selected index is within bounds of filtered items
	if selectedIdx >= totalItems {
		selectedIdx = totalItems - 1
	}

	startIdx := 0
	endIdx := totalItems

	if totalItems > listHeight {
		// Center the selected item if possible
		halfHeight := listHeight / 2
		startIdx = selectedIdx - halfHeight
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + listHeight
		if endIdx > totalItems {
			endIdx = totalItems
			startIdx = endIdx - listHeight
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Columns widths
	// Timestamp: ~15 chars ("2 hours ago")
	timeWidth := 15

	// Render rows
	for i := startIdx; i < endIdx; i++ {
		if i >= len(m.historySearchState.filteredIndices) {
			break
		}

		originalIdx := m.historySearchState.filteredIndices[i]
		item := m.historyItems[originalIdx]

		isRowSelected := i == selectedIdx

		// Prefix
		prefix := "  "
		if isRowSelected {
			prefix = "> "
		}

		// Timestamp
		timeStr := humanize.Time(item.Timestamp)
		if len(timeStr) > timeWidth {
			timeStr = timeStr[:timeWidth]
		}
		// Pad timestamp
		timeStr = fmt.Sprintf("%-*s", timeWidth, timeStr)

		// Command
		// Calculate available width for command
		// width - prefix(2) - timestamp(timeWidth) - spacing(2)
		cmdWidth := width - 2 - timeWidth - 2
		if cmdWidth < 10 {
			cmdWidth = 10 // Minimum width
		}

		cmdStr := item.Command
		// Simple truncation
		if ansi.PrintableRuneWidth(cmdStr) > cmdWidth {
			// truncate
			runes := []rune(cmdStr)
			if len(runes) > cmdWidth-1 {
				cmdStr = string(runes[:cmdWidth-1]) + "…"
			}
		} else {
			// pad
			cmdStr = fmt.Sprintf("%-*s", cmdWidth, cmdStr)
		}

		line := ""
		if isRowSelected {
			line = selectedStyle.Render(prefix + cmdStr) + "  " + dimStyle.Render(timeStr)
		} else {
			line = normalStyle.Render(prefix + cmdStr) + "  " + dimStyle.Render(timeStr)
		}

		content.WriteString(line)
		if i < endIdx-1 {
			content.WriteString("\n")
		}
	}

	// Add help footer
	content.WriteString("\n")
	helpText := "Ctrl+F: Filter | Ctrl+O: Sort | Enter: Select | Esc: Cancel"
	content.WriteString(helpStyle.Render(helpText))

	return content.String()
}

// updateHistorySearch updates the filtered list based on the query and filter mode
func (m *Model) updateHistorySearch() {
	query := m.reverseSearchQuery

	// Create a subset of items based on filter mode first, deduplicating by command
	// We keep track of seen commands to only include the first (most recent) occurrence
	seen := make(map[string]bool)
	var candidates []int // indices into historyItems

	for i, item := range m.historyItems {
		// Skip duplicates - keep only the first (most recent) occurrence of each command
		if seen[item.Command] {
			continue
		}

		match := true
		switch m.historySearchState.filterMode {
		case HistoryFilterDirectory:
			if m.historySearchState.currentDir != "" && item.Directory != m.historySearchState.currentDir {
				match = false
			}
		case HistoryFilterSession:
			// Session filtering requires SessionID, which we don't track yet.
			// Falling back to All for now or implementing if requested.
			// Assuming "Session: Current" implies items created in this session.
			// But we load from DB. We'd need to know which entries are from this session.
			// For now, treat as All or TODO.
		}

		if match {
			seen[item.Command] = true
			candidates = append(candidates, i)
		}
	}

	if query == "" {
		// Sort candidates if needed
		switch m.historySearchState.sortMode {
		case HistorySortRecent:
			// Candidates are already sorted by recent (index order in historyItems)
		case HistorySortAlphabetical:
			sort.SliceStable(candidates, func(i, j int) bool {
				return m.historyItems[candidates[i]].Command < m.historyItems[candidates[j]].Command
			})
		case HistorySortRelevance:
			// Relevance implies query relevance, but with empty query, fallback to Recent
		}

		m.historySearchState.filteredIndices = candidates
		m.historySearchState.selected = 0
		return
	}

	// Fuzzy search on candidates
	// We need to create a source that maps candidates back to historyItems
	source := historySourceSubset{
		indices: candidates,
		items:   m.historyItems,
	}

	matches := fuzzy.FindFrom(query, source)

	// Sort matches based on sort mode
	switch m.historySearchState.sortMode {
	case HistorySortRecent:
		// Sort by index in candidates (which preserves original time-descending order, i.e. Newest First)
		// Assuming historyItems are ordered Newest First (index 0 is newest),
		// then lower index means more recent.
		sort.SliceStable(matches, func(i, j int) bool {
			return matches[i].Index < matches[j].Index
		})
	case HistorySortAlphabetical:
		sort.SliceStable(matches, func(i, j int) bool {
			return matches[i].Str < matches[j].Str
		})
	case HistorySortRelevance:
		// Already sorted by fuzzy score
	}

	m.historySearchState.filteredIndices = make([]int, len(matches))
	for i, match := range matches {
		// match.Index is index into 'candidates', so we need candidates[match.Index]
		m.historySearchState.filteredIndices[i] = candidates[match.Index]
	}
	m.historySearchState.selected = 0
}

// historySourceSubset adapts a subset of HistoryItems for fuzzy matching
type historySourceSubset struct {
	indices []int
	items   []HistoryItem
}

func (h historySourceSubset) String(i int) string {
	return h.items[h.indices[i]].Command
}

func (h historySourceSubset) Len() int {
	return len(h.indices)
}

// historySearchUp moves selection up (older/previous in list)
func (m *Model) historySearchUp() {
	if m.historySearchState.selected > 0 {
		m.historySearchState.selected--
	}
}

// historySearchDown moves selection down (newer/next in list)
func (m *Model) historySearchDown() {
	if m.historySearchState.selected < len(m.historySearchState.filteredIndices)-1 {
		m.historySearchState.selected++
	}
}

// toggleHistoryFilter cycles through filter modes
func (m *Model) toggleHistoryFilter() {
	switch m.historySearchState.filterMode {
	case HistoryFilterAll:
		m.historySearchState.filterMode = HistoryFilterDirectory
	case HistoryFilterDirectory:
		m.historySearchState.filterMode = HistoryFilterAll // Skip session for now as we don't track it
	default:
		m.historySearchState.filterMode = HistoryFilterAll
	}
	m.updateHistorySearch()
}

// toggleHistorySort cycles through sort modes
func (m *Model) toggleHistorySort() {
	switch m.historySearchState.sortMode {
	case HistorySortRecent:
		m.historySearchState.sortMode = HistorySortRelevance
	case HistorySortRelevance:
		m.historySearchState.sortMode = HistorySortAlphabetical
	case HistorySortAlphabetical:
		m.historySearchState.sortMode = HistorySortRecent
	default:
		m.historySearchState.sortMode = HistorySortRecent
	}
	m.updateHistorySearch()
}
