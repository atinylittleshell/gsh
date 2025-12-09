// Package completion provides tab completion functionality for the gsh REPL.
// It manages completion specifications, integrates with bash's compgen,
// and provides file and word completion capabilities.
package completion

import (
	"strings"
	"unicode"
)

// splitPreservingQuotes splits a command line into words while preserving quotes.
func splitPreservingQuotes(line string) []string {
	var words []string
	var currentWord strings.Builder
	inQuote := false
	quoteChar := rune(0)
	lastWasSpace := true // Start with true to handle leading spaces

	for _, r := range line {
		switch {
		case r == '\'' || r == '"':
			if inQuote {
				if r == quoteChar {
					// End of quote
					inQuote = false
					quoteChar = 0
				}
			} else {
				// Start of quote
				inQuote = true
				quoteChar = r
			}
			currentWord.WriteRune(r)
			lastWasSpace = false
		case unicode.IsSpace(r):
			if inQuote {
				currentWord.WriteRune(r)
			} else if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
				lastWasSpace = true
			}
		default:
			if lastWasSpace && currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
			currentWord.WriteRune(r)
			lastWasSpace = false
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}
