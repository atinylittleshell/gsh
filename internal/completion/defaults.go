package completion

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/atinylittleshell/gsh/pkg/shellinput"
)

// DefaultCompleter handles built-in default completions for common commands
type DefaultCompleter struct{}

// GetCompletions tries to provide completions for the given command and context
func (d *DefaultCompleter) GetCompletions(command string, args []string, line string, pos int) ([]shellinput.CompletionCandidate, bool) {
	switch command {
	case "cd":
		return d.completeDirectories(args), true
	case "export", "unset":
		return d.completeEnvVars(args), true
	case "ssh", "scp", "sftp":
		return d.completeSSHHosts(args), true
	case "make":
		return d.completeMakeTargets(args), true
	case "kill":
		return d.completeKillSignals(args), true
	case "man", "help":
		// For now, just return nil to let it fall back or implementation TODO
		// Implementing full man page scanning is expensive for a default
		return nil, false
	}
	return nil, false
}

func (d *DefaultCompleter) completeDirectories(args []string) []shellinput.CompletionCandidate {
	// Use the last arg as prefix, or empty if none
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	// Re-use getFileCompletions but filter for directories only
	// We need to access the current directory. For now, we assume current process CWD.
	// ideally this should come from context/runner.
	cwd, _ := os.Getwd()

	// We can use the existing getFileCompletions helper if we can filter its output
	// But getFileCompletions returns strings. We can parse them.
	// Or we implement a specific directory walker.
	// Let's reuse getFileCompletions for consistency and filter.
	candidates := getFileCompletions(prefix, cwd)

	var dirs []shellinput.CompletionCandidate
	for _, c := range candidates {
		// Check if it's a directory by looking at the Suffix field
		if c.Suffix == string(os.PathSeparator) {
			c.Description = "Directory"
			dirs = append(dirs, c)
		}
	}
	return dirs
}

func (d *DefaultCompleter) completeEnvVars(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	var candidates []shellinput.CompletionCandidate
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		key := parts[0]
		if strings.HasPrefix(key, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       key,
				Description: "Environment Variable",
			})
		}
	}
	return candidates
}

func (d *DefaultCompleter) completeSSHHosts(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	hosts := make(map[string]bool)

	// Parse ~/.ssh/config
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".ssh", "config")
		if file, err := os.Open(configPath); err == nil {
			defer func() {
				_ = file.Close()
			}()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "Host ") {
					// Host can have multiple aliases: "Host foo bar"
					parts := strings.Fields(line)
					if len(parts) > 1 {
						for _, host := range parts[1:] {
							if host != "*" { // Skip wildcards
								hosts[host] = true
							}
						}
					}
				}
			}
		}
	}

	// We could also parse known_hosts, but that often contains hashed entries or IPs.
	// ~/.ssh/config is the most useful source for "smart" completion.

	var candidates []shellinput.CompletionCandidate
	for host := range hosts {
		if strings.HasPrefix(host, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       host,
				Description: "SSH Host",
			})
		}
	}
	return candidates
}

func (d *DefaultCompleter) completeMakeTargets(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	// Look for Makefile in current directory
	cwd, _ := os.Getwd()
	makefiles := []string{"Makefile", "makefile", "GNUmakefile"}

	var candidates []shellinput.CompletionCandidate

	for _, mk := range makefiles {
		path := filepath.Join(cwd, mk)
		if file, err := os.Open(path); err == nil {
			defer func() {
				_ = file.Close()
			}()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// Simple regex for targets: starts with word characters, ends with colon
				// Exclude .PHONY etc.
				if strings.Contains(line, ":") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ".") {
					parts := strings.SplitN(line, ":", 2)
					target := strings.TrimSpace(parts[0])
					// Handle multiple targets "clean install:"
					targets := strings.Fields(target)
					for _, t := range targets {
						if strings.HasPrefix(t, prefix) {
							candidates = append(candidates, shellinput.CompletionCandidate{
								Value:       t,
								Description: "Make target",
							})
						}
					}
				}
			}
			break // Only parse the first found makefile
		}
	}
	return candidates
}

func (d *DefaultCompleter) completeKillSignals(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	// If prefix doesn't start with -, maybe they are typing a PID.
	// If it starts with -, they want a signal.
	if strings.HasPrefix(prefix, "-") {
		signals := []string{
			"-HUP", "-INT", "-QUIT", "-ILL", "-TRAP", "-ABRT", "-BUS", "-FPE",
			"-KILL", "-USR1", "-SEGV", "-USR2", "-PIPE", "-ALRM", "-TERM",
			"-STKFLT", "-CHLD", "-CONT", "-STOP", "-TSTP", "-TTIN", "-TTOU",
			"-URG", "-XCPU", "-XFSZ", "-VTALRM", "-PROF", "-WINCH", "-IO",
			"-PWR", "-SYS",
		}

		var candidates []shellinput.CompletionCandidate
		for _, sig := range signals {
			if strings.HasPrefix(sig, prefix) {
				candidates = append(candidates, shellinput.CompletionCandidate{
					Value:       sig,
					Description: "Signal",
				})
			}
		}
		return candidates
	}
	return nil
}
