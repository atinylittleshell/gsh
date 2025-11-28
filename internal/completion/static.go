package completion

import (
	"github.com/atinylittleshell/gsh/pkg/shellinput"
)

// StaticCompleter handles static word lists for common commands
type StaticCompleter struct {
	completions map[string][]shellinput.CompletionCandidate
}

func NewStaticCompleter() *StaticCompleter {
	sc := &StaticCompleter{
		completions: make(map[string][]shellinput.CompletionCandidate),
	}
	sc.registerDefaults()
	return sc
}

func (s *StaticCompleter) registerDefaults() {
	s.register("docker", []string{
		"attach", "build", "commit", "cp", "create", "diff", "events", "exec",
		"export", "history", "images", "import", "info", "inspect", "kill",
		"load", "login", "logout", "logs", "pause", "port", "ps", "pull",
		"push", "rename", "restart", "rm", "rmi", "run", "save", "search",
		"start", "stats", "stop", "tag", "top", "unpause", "update", "version", "wait",
	})

	s.register("kubectl", []string{
		"apply", "get", "describe", "delete", "logs", "exec", "port-forward",
		"config", "cluster-info", "top", "explain", "run", "create", "edit",
		"scale", "autoscale", "rollout", "cordon", "drain", "taint", "label",
		"annotate", "completion", "api-resources", "api-versions", "version",
	})

	s.register("npm", []string{
		"install", "start", "test", "run", "build", "publish", "update",
		"uninstall", "init", "version", "config", "list", "audit", "outdated",
		"ci", "cache", "doctor", "login", "logout", "link", "unlink",
	})

	s.register("yarn", []string{
		"add", "install", "remove", "run", "test", "build", "start", "publish",
		"init", "list", "global", "upgrade", "why", "cache", "create",
	})

	s.register("pnpm", []string{
		"add", "install", "remove", "run", "test", "build", "start", "publish",
		"init", "list", "store", "update", "why", "prune",
	})

	s.register("go", []string{
		"build", "run", "test", "get", "mod", "install", "list", "vet", "fmt",
		"doc", "env", "bug", "clean", "fix", "generate", "tool", "version", "work",
	})
}

func (s *StaticCompleter) register(command string, subcommands []string) {
	var candidates []shellinput.CompletionCandidate
	for _, sub := range subcommands {
		candidates = append(candidates, shellinput.CompletionCandidate{Value: sub})
	}
	s.completions[command] = candidates
}

func (s *StaticCompleter) GetCompletions(command string, args []string) []shellinput.CompletionCandidate {
	// Only provide completion for the first argument (subcommand)
	if len(args) == 0 {
		if candidates, ok := s.completions[command]; ok {
			return candidates
		}
	}
	// Filter by prefix
	if len(args) == 1 {
		prefix := args[0]
		if candidates, ok := s.completions[command]; ok {
			var filtered []shellinput.CompletionCandidate
			for _, c := range candidates {
				if len(c.Value) >= len(prefix) && c.Value[:len(prefix)] == prefix {
					filtered = append(filtered, c)
				}
			}
			return filtered
		}
	}
	return nil
}
