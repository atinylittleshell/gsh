package completion

import (
	"os/exec"
	"strings"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
)

// GitCompleter handles built-in completion for git
type GitCompleter struct{}

func (g *GitCompleter) GetCompletions(args []string, line string) []shellinput.CompletionCandidate {
	if len(args) == 0 {
		// Complete git subcommands
		commands := []struct {
			val  string
			desc string
		}{
			{"add", "Add file contents to the index"},
			{"branch", "List, create, or delete branches"},
			{"checkout", "Switch branches or restore working tree files"},
			{"clone", "Clone a repository into a new directory"},
			{"commit", "Record changes to the repository"},
			{"diff", "Show changes between commits, commit and working tree, etc"},
			{"fetch", "Download objects and refs from another repository"},
			{"init", "Create an empty Git repository or reinitialize an existing one"},
			{"log", "Show commit logs"},
			{"merge", "Join two or more development histories together"},
			{"pull", "Fetch from and integrate with another repository or a local branch"},
			{"push", "Update remote refs along with associated objects"},
			{"rebase", "Reapply commits on top of another base tip"},
			{"remote", "Manage set of tracked repositories"},
			{"reset", "Reset current HEAD to the specified state"},
			{"restore", "Restore working tree files"},
			{"show", "Show various types of objects"},
			{"status", "Show the working tree status"},
			{"switch", "Switch branches"},
			{"tag", "Create, list, delete or verify a tag object signed with GPG"},
		}

		var candidates []shellinput.CompletionCandidate
		for _, cmd := range commands {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       cmd.val,
				Description: cmd.desc,
			})
		}
		return candidates
	}

	subcommand := args[0]
	// args[1:] are arguments to the subcommand
	// current word being completed is the last one in args
	// BUT if line ends with space, we're completing a new empty word
	currentWord := ""
	if len(args) > 1 {
		currentWord = args[len(args)-1]
	}
	// If line ends with space, we're starting a new word (empty prefix)
	if len(line) > 0 && line[len(line)-1] == ' ' {
		currentWord = ""
	}

	switch subcommand {
	case "checkout", "switch", "merge", "rebase":
		return g.completeBranches(currentWord)
	case "add", "rm", "restore":
		return g.completeFiles(currentWord)
	}

	return nil
}

func (g *GitCompleter) completeBranches(prefix string) []shellinput.CompletionCandidate {
	// Run git branch with format to get branch names and their latest commit messages
	// Format: branch_name|commit_subject
	cmd := exec.Command("git", "branch", "--format=%(refname:short)|%(contents:subject)")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var candidates []shellinput.CompletionCandidate
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by the delimiter to get branch name and commit message
		parts := strings.SplitN(line, "|", 2)
		branchName := parts[0]
		commitMsg := ""
		if len(parts) > 1 {
			commitMsg = parts[1]
			// Truncate long commit messages
			if len(commitMsg) > 80 {
				commitMsg = commitMsg[:77] + "..."
			}
		}

		if strings.HasPrefix(branchName, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       branchName,
				Description: commitMsg,
			})
		}
	}
	return candidates
}

func (g *GitCompleter) completeFiles(prefix string) []shellinput.CompletionCandidate {
	// For 'add', 'rm', etc., we usually want modified files or all files.
	// 'git status --porcelain' gives status of files.
	// Or just rely on file completion fallback if prefix looks like path.
	// Let's try git status for modified files which are most likely targets.

	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		// Fallback to simple file completion from disk?
		// Actually, let's return nothing and let the shell fall back to standard file completion
		// if we can't find specific git files.
		// BUT, if we return non-nil empty list, it might stop fallback.
		// If we return nil, it falls back.
		return nil
	}

	var candidates []shellinput.CompletionCandidate
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if len(line) > 3 {
			// XY PATH
			path := line[3:]
			if strings.HasPrefix(path, prefix) {
				candidates = append(candidates, shellinput.CompletionCandidate{
					Value:       path,
					Description: "Modified file",
				})
			}
		}
	}

	// Also allow standard files if we have few candidates?
	// The user might want to add a new untracked file (which shows up in porcelain with ??)
	// So porcelain covers untracked files too.

	return candidates
}
