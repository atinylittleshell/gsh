package repl

import (
	"testing"

	// Import all subpackages to verify the directory structure is correct
	_ "github.com/atinylittleshell/gsh/internal/repl/agent"
	_ "github.com/atinylittleshell/gsh/internal/repl/completion"
	_ "github.com/atinylittleshell/gsh/internal/repl/config"
	_ "github.com/atinylittleshell/gsh/internal/repl/context"
	_ "github.com/atinylittleshell/gsh/internal/repl/executor"
	_ "github.com/atinylittleshell/gsh/internal/repl/input"
	_ "github.com/atinylittleshell/gsh/internal/repl/predict"
	_ "github.com/atinylittleshell/gsh/internal/repl/util"
)

func TestDirectoryStructure(t *testing.T) {
	// This test verifies that all subpackages in internal/repl/ can be imported.
	// The imports above will fail at compile time if any package is missing
	// or has incorrect package declarations.
	t.Log("All internal/repl subpackages are correctly structured and importable")
}
