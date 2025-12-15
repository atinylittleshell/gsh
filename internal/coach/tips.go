package coach

import (
	"math/rand"
	"time"
)

// TipCategory represents categories for static tips
type TipCategory string

const (
	TipCategoryProductivity TipCategory = "productivity"
	TipCategoryShortcut     TipCategory = "shortcut"
	TipCategoryCommand      TipCategory = "command"
	TipCategoryGit          TipCategory = "git"
	TipCategoryFunFact      TipCategory = "fun_fact"
	TipCategoryMotivation   TipCategory = "motivation"
)

// StaticTip represents a predefined tip
type StaticTip struct {
	ID       string
	Category TipCategory
	Icon     string
	Title    string
	Content  string
	Priority int // Higher = more likely to show
}

// StaticTips contains all static tip definitions
var StaticTips = []StaticTip{
	// Productivity Tips
	{ID: "prod_ctrl_r", Category: TipCategoryProductivity, Icon: "💡", Title: "Search History", Content: "Use Ctrl+R to search your command history", Priority: 10},
	{ID: "prod_bang_bang", Category: TipCategoryProductivity, Icon: "💡", Title: "Repeat Command", Content: "Use !! to repeat the last command", Priority: 9},
	{ID: "prod_bang_dollar", Category: TipCategoryProductivity, Icon: "💡", Title: "Last Argument", Content: "Use !$ to reuse the last argument from the previous command", Priority: 9},
	{ID: "prod_chain_and", Category: TipCategoryProductivity, Icon: "💡", Title: "Chain Commands", Content: "Use && to chain commands that depend on each other", Priority: 8},
	{ID: "prod_pipe", Category: TipCategoryProductivity, Icon: "💡", Title: "Pipe Output", Content: "Use | (pipe) to pass output between commands", Priority: 8},
	{ID: "prod_ctrl_a", Category: TipCategoryProductivity, Icon: "💡", Title: "Jump to Start", Content: "Use Ctrl+A to jump to the beginning of the line", Priority: 7},
	{ID: "prod_ctrl_e", Category: TipCategoryProductivity, Icon: "💡", Title: "Jump to End", Content: "Use Ctrl+E to jump to the end of the line", Priority: 7},
	{ID: "prod_ctrl_w", Category: TipCategoryProductivity, Icon: "💡", Title: "Delete Word", Content: "Use Ctrl+W to delete the word before the cursor", Priority: 7},
	{ID: "prod_ctrl_u", Category: TipCategoryProductivity, Icon: "💡", Title: "Clear Line", Content: "Use Ctrl+U to clear the line before the cursor", Priority: 6},
	{ID: "prod_ctrl_k", Category: TipCategoryProductivity, Icon: "💡", Title: "Clear After", Content: "Use Ctrl+K to clear the line after the cursor", Priority: 6},
	{ID: "prod_alt_dot", Category: TipCategoryProductivity, Icon: "💡", Title: "Insert Last Arg", Content: "Use Alt+. to insert the last argument of the previous command", Priority: 6},
	{ID: "prod_pushd", Category: TipCategoryProductivity, Icon: "💡", Title: "Directory Stack", Content: "Use pushd/popd to manage a directory stack for quick navigation", Priority: 5},
	{ID: "prod_ctrl_z", Category: TipCategoryProductivity, Icon: "💡", Title: "Suspend Process", Content: "Use Ctrl+Z to suspend and 'fg' to resume a process", Priority: 5},

	// Shortcut Tips
	{ID: "short_ctrl_l", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Clear Screen", Content: "Ctrl+L clears the screen (same as 'clear')", Priority: 8},
	{ID: "short_tab", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Tab Complete", Content: "Tab completes commands and file paths automatically", Priority: 10},
	{ID: "short_double_tab", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Show All", Content: "Double-Tab shows all possible completions", Priority: 7},
	{ID: "short_ctrl_c", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Cancel Command", Content: "Ctrl+C cancels the current command", Priority: 9},
	{ID: "short_ctrl_d", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Exit Shell", Content: "Ctrl+D exits the shell (or sends EOF)", Priority: 6},
	{ID: "short_alt_bf", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Word Movement", Content: "Alt+B/F moves backward/forward word by word", Priority: 6},
	{ID: "short_ctrl_t", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Swap Chars", Content: "Ctrl+T swaps the last two characters", Priority: 4},
	{ID: "short_alt_t", Category: TipCategoryShortcut, Icon: "⌨️", Title: "Swap Words", Content: "Alt+T swaps the last two words", Priority: 4},

	// Command Tips
	{ID: "cmd_cd_dash", Category: TipCategoryCommand, Icon: "📚", Title: "Previous Directory", Content: "'cd -' takes you to your previous directory", Priority: 9},
	{ID: "cmd_mkdir_p", Category: TipCategoryCommand, Icon: "📚", Title: "Nested Dirs", Content: "'mkdir -p' creates nested directories in one command", Priority: 8},
	{ID: "cmd_cp_r", Category: TipCategoryCommand, Icon: "📚", Title: "Copy Recursive", Content: "'cp -r' copies directories recursively", Priority: 7},
	{ID: "cmd_rm_i", Category: TipCategoryCommand, Icon: "📚", Title: "Safe Delete", Content: "'rm -i' asks before deleting each file", Priority: 7},
	{ID: "cmd_less", Category: TipCategoryCommand, Icon: "📚", Title: "Better Pager", Content: "'less' is better than 'more' for viewing files (q to quit)", Priority: 6},
	{ID: "cmd_tail_f", Category: TipCategoryCommand, Icon: "📚", Title: "Follow Logs", Content: "'tail -f' follows file updates in real-time (great for logs)", Priority: 8},
	{ID: "cmd_head_n", Category: TipCategoryCommand, Icon: "📚", Title: "First Lines", Content: "'head -n 5' shows only the first 5 lines of a file", Priority: 5},
	{ID: "cmd_wc_l", Category: TipCategoryCommand, Icon: "📚", Title: "Count Lines", Content: "'wc -l' counts lines in a file", Priority: 5},
	{ID: "cmd_sort_u", Category: TipCategoryCommand, Icon: "📚", Title: "Sort Unique", Content: "'sort -u' sorts and removes duplicate lines", Priority: 5},
	{ID: "cmd_xargs", Category: TipCategoryCommand, Icon: "📚", Title: "Build Commands", Content: "'xargs' converts input into arguments for another command", Priority: 6},
	{ID: "cmd_tee", Category: TipCategoryCommand, Icon: "📚", Title: "Split Output", Content: "'tee' writes output to both file and stdout simultaneously", Priority: 5},
	{ID: "cmd_watch", Category: TipCategoryCommand, Icon: "📚", Title: "Repeat Command", Content: "'watch' runs a command repeatedly and shows the output", Priority: 5},

	// Git Tips
	{ID: "git_oneline", Category: TipCategoryGit, Icon: "🌿", Title: "Compact Log", Content: "'git log --oneline' shows compact history", Priority: 8},
	{ID: "git_staged", Category: TipCategoryGit, Icon: "🌿", Title: "Staged Changes", Content: "'git diff --staged' shows staged changes", Priority: 8},
	{ID: "git_stash", Category: TipCategoryGit, Icon: "🌿", Title: "Save Work", Content: "'git stash' saves your work without committing", Priority: 7},
	{ID: "git_amend", Category: TipCategoryGit, Icon: "🌿", Title: "Fix Last Commit", Content: "'git commit --amend' modifies the last commit", Priority: 7},
	{ID: "git_cherry", Category: TipCategoryGit, Icon: "🌿", Title: "Cherry Pick", Content: "'git cherry-pick' applies specific commits to current branch", Priority: 5},
	{ID: "git_bisect", Category: TipCategoryGit, Icon: "🌿", Title: "Find Bug", Content: "'git bisect' helps find the commit that introduced a bug", Priority: 5},
	{ID: "git_reflog", Category: TipCategoryGit, Icon: "🌿", Title: "Undo History", Content: "'git reflog' shows all recent actions (even undone ones)", Priority: 6},
	{ID: "git_blame", Category: TipCategoryGit, Icon: "🌿", Title: "Line History", Content: "'git blame' shows who changed each line and when", Priority: 6},
	{ID: "git_worktree", Category: TipCategoryGit, Icon: "🌿", Title: "Multiple Branches", Content: "'git worktree' lets you work on multiple branches at once", Priority: 4},
	{ID: "git_rebase_i", Category: TipCategoryGit, Icon: "🌿", Title: "Edit History", Content: "'git rebase -i' for interactive history editing", Priority: 5},

	// Fun Facts
	{ID: "fun_unix_1971", Category: TipCategoryFunFact, Icon: "🎲", Title: "Unix History", Content: "The first Unix shell was written in 1971", Priority: 3},
	{ID: "fun_grep", Category: TipCategoryFunFact, Icon: "🎲", Title: "grep Origin", Content: "'grep' stands for 'Global Regular Expression Print'", Priority: 4},
	{ID: "fun_awk", Category: TipCategoryFunFact, Icon: "🎲", Title: "awk Origin", Content: "'awk' is named after its creators: Aho, Weinberger, Kernighan", Priority: 3},
	{ID: "fun_sudo", Category: TipCategoryFunFact, Icon: "🎲", Title: "sudo Origin", Content: "'sudo' stands for 'superuser do'", Priority: 4},
	{ID: "fun_root", Category: TipCategoryFunFact, Icon: "🎲", Title: "Root Directory", Content: "The '/' root directory is called 'slash'", Priority: 3},
	{ID: "fun_daemon", Category: TipCategoryFunFact, Icon: "🎲", Title: "Daemon Origin", Content: "'daemon' processes are named after Greek spirits that work in the background", Priority: 3},
	{ID: "fun_bash", Category: TipCategoryFunFact, Icon: "🎲", Title: "Bash Origin", Content: "Bash stands for 'Bourne Again SHell' (a pun on Bourne shell)", Priority: 4},
	{ID: "fun_tty", Category: TipCategoryFunFact, Icon: "🎲", Title: "tty Origin", Content: "The 'tty' command comes from 'teletypewriter'", Priority: 3},
	{ID: "fun_null", Category: TipCategoryFunFact, Icon: "🎲", Title: "Bit Bucket", Content: "'/dev/null' is called the 'bit bucket' - data goes in, nothing comes out", Priority: 4},
	{ID: "fun_pipe", Category: TipCategoryFunFact, Icon: "🎲", Title: "Pipe History", Content: "The pipe | was invented by Doug McIlroy in 1973", Priority: 4},

	// Motivation
	{ID: "mot_got_this", Category: TipCategoryMotivation, Icon: "🚀", Title: "Keep Going!", Content: "You've got this! Every command makes you better.", Priority: 2},
	{ID: "mot_practice", Category: TipCategoryMotivation, Icon: "💪", Title: "Practice", Content: "Practice makes perfect. Keep typing!", Priority: 2},
	{ID: "mot_improve", Category: TipCategoryMotivation, Icon: "🌟", Title: "Progress", Content: "Small improvements add up to big gains.", Priority: 2},
	{ID: "mot_pro", Category: TipCategoryMotivation, Icon: "🏆", Title: "Terminal Pro", Content: "You're on your way to becoming a terminal pro!", Priority: 2},
	{ID: "mot_beginner", Category: TipCategoryMotivation, Icon: "✨", Title: "Everyone Starts", Content: "Every expert was once a beginner.", Priority: 2},
	{ID: "mot_focus", Category: TipCategoryMotivation, Icon: "🎯", Title: "Focus", Content: "Focus on progress, not perfection.", Priority: 2},
	{ID: "mot_streak", Category: TipCategoryMotivation, Icon: "🔥", Title: "Streak Power", Content: "Your streak is building! Don't break the chain!", Priority: 3},
	{ID: "mot_speed", Category: TipCategoryMotivation, Icon: "⚡", Title: "Getting Faster", Content: "Speed comes with practice. You're getting faster!", Priority: 2},
}

// GetRandomStaticTip returns a random static tip
func GetRandomStaticTip() *StaticTip {
	if len(StaticTips) == 0 {
		return nil
	}

	// Weight by priority
	totalWeight := 0
	for _, tip := range StaticTips {
		totalWeight += tip.Priority
	}

	r := rand.Intn(totalWeight)
	cumulative := 0
	for i := range StaticTips {
		cumulative += StaticTips[i].Priority
		if r < cumulative {
			return &StaticTips[i]
		}
	}

	return &StaticTips[len(StaticTips)-1]
}

// GetStaticTipByCategory returns a random tip from a specific category
func GetStaticTipByCategory(category TipCategory) *StaticTip {
	var categoryTips []StaticTip
	for _, tip := range StaticTips {
		if tip.Category == category {
			categoryTips = append(categoryTips, tip)
		}
	}

	if len(categoryTips) == 0 {
		return nil
	}

	return &categoryTips[rand.Intn(len(categoryTips))]
}

// GetStaticTipByID returns a specific tip by ID
func GetStaticTipByID(id string) *StaticTip {
	for i := range StaticTips {
		if StaticTips[i].ID == id {
			return &StaticTips[i]
		}
	}
	return nil
}

// GetAllStaticTipsByCategory returns all tips in a category
func GetAllStaticTipsByCategory(category TipCategory) []StaticTip {
	var result []StaticTip
	for _, tip := range StaticTips {
		if tip.Category == category {
			result = append(result, tip)
		}
	}
	return result
}

// ConvertStaticTipToDisplay converts a StaticTip to CoachDisplayContent
func ConvertStaticTipToDisplay(tip *StaticTip) *CoachDisplayContent {
	if tip == nil {
		return nil
	}
	return &CoachDisplayContent{
		Type:     "tip",
		Icon:     tip.Icon,
		Title:    tip.Title,
		Content:  tip.Content,
		Priority: tip.Priority,
	}
}

// GetTimeBasedGreeting returns a greeting based on time of day
func GetTimeBasedGreeting() string {
	hour := time.Now().Hour()
	switch {
	case hour < 6:
		return "Burning the midnight oil"
	case hour < 12:
		return "Good morning"
	case hour < 17:
		return "Good afternoon"
	case hour < 21:
		return "Good evening"
	default:
		return "Working late"
	}
}

// GetTimeBasedIcon returns an icon based on time of day
func GetTimeBasedIcon() string {
	hour := time.Now().Hour()
	switch {
	case hour < 6:
		return "🌙"
	case hour < 12:
		return "🌅"
	case hour < 17:
		return "☀️"
	case hour < 21:
		return "🌆"
	default:
		return "🌃"
	}
}
