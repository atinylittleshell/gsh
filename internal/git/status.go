package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RepoStatus struct {
	RepoName string
	Branch   string
	Clean    bool
	Staged   int
	Unstaged int
	Ahead    int
	Behind   int
	Conflict bool
}

func GetStatus(dir string) *RepoStatus {
	return GetStatusWithContext(context.Background(), dir)
}

func GetStatusWithContext(ctx context.Context, dir string) *RepoStatus {
	// check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}

	// check if inside a git repo
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	repoPath := strings.TrimSpace(string(out))
	repoName := filepath.Base(repoPath)

	// Get status --porcelain=v2 --branch
	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain=v2", "--branch")
	cmd.Dir = dir
	out, err = cmd.Output()
	if err != nil {
		return nil
	}

	status := &RepoStatus{
		RepoName: repoName,
		Clean:    true,
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		switch parts[0] {
		case "#":
			switch parts[1] {
			case "branch.head":
				if len(parts) > 2 {
					status.Branch = parts[2]
				} else {
					status.Branch = "detached"
				}
			case "branch.ab":
				// branch.ab +ahead -behind
				if len(parts) >= 4 {
					// parse +ahead
					status.Ahead = parseInt(strings.TrimPrefix(parts[2], "+"))
					status.Behind = parseInt(strings.TrimPrefix(parts[3], "-"))
				}
			}
		case "1", "2":
			// 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
			// 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
			xy := parts[1]
			if xy[0] != '.' {
				status.Staged++
				status.Clean = false
			}
			if xy[1] != '.' {
				status.Unstaged++
				status.Clean = false
			}
		case "u":
			// u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
			status.Conflict = true
			status.Clean = false
			// Treat conflicts as both staged and unstaged usually, or just conflict.
			// The spec says: red for conflicts
		case "?":
			status.Unstaged++
			status.Clean = false
		}
	}

	if status.Conflict {
		status.Clean = false
	}

	return status
}

func parseInt(s string) int {
	var res int
	for _, r := range s {
		if r < '0' || r > '9' {
			continue
		}
		res = res*10 + int(r-'0')
	}
	return res
}

// Timeout helper could be added if needed, but for now we rely on git being reasonably fast on local repos
func GetStatusWithTimeout(dir string, timeout time.Duration) *RepoStatus {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return GetStatusWithContext(ctx, dir)
}
