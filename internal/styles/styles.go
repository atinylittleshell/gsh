package styles

import (
	"os"

	"github.com/muesli/termenv"
)

var (
	stdout = termenv.NewOutput(os.Stdout)
	stderr = termenv.NewOutput(os.Stderr)

	ERROR = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("9")).
			String()
	}
	AGENT_MESSAGE = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("12")).
			String()
	}
	AGENT_QUESTION = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("11")).
			Bold().
			String()
	}
	LOG = func(s string) string {
		return stderr.String(s).
			Foreground(stderr.Color("8")).
			String()
	}
)
