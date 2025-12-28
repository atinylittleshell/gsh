package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"go.uber.org/zap"
	shinterp "mvdan.cc/sh/v3/interp"
)

// newTestExecutor creates a REPLExecutor for testing with a fresh interpreter.
func newTestExecutor(t *testing.T, logger *zap.Logger, handlers ...ExecMiddleware) *REPLExecutor {
	t.Helper()
	interp := interpreter.New()
	exec, err := NewREPLExecutor(interp, logger, handlers...)
	if err != nil {
		t.Fatalf("NewREPLExecutor() error = %v", err)
	}
	return exec
}

func TestNewREPLExecutor(t *testing.T) {
	t.Run("creates executor with defaults", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		if exec.Runner() == nil {
			t.Error("expected runner to be initialized")
		}
		if exec.Interpreter() == nil {
			t.Error("expected interpreter to be initialized")
		}
	})

	t.Run("creates executor with logger", func(t *testing.T) {
		logger := zap.NewNop()

		exec := newTestExecutor(t, logger)
		defer exec.Close()

		if exec.logger != logger {
			t.Error("expected logger to be set")
		}
	})

	t.Run("creates executor with exec handlers", func(t *testing.T) {
		handlerCalled := false
		handler := func(next shinterp.ExecHandlerFunc) shinterp.ExecHandlerFunc {
			return func(ctx context.Context, args []string) error {
				if len(args) > 0 && args[0] == "testcmd" {
					handlerCalled = true
					return nil
				}
				return next(ctx, args)
			}
		}

		exec := newTestExecutor(t, nil, handler)
		defer exec.Close()

		ctx := context.Background()
		_, err := exec.ExecuteBash(ctx, "testcmd")
		if err != nil {
			t.Fatalf("ExecuteBash() error = %v", err)
		}

		if !handlerCalled {
			t.Error("exec handler was not called")
		}
	})

	t.Run("returns error for nil interpreter", func(t *testing.T) {
		_, err := NewREPLExecutor(nil, nil)
		if err == nil {
			t.Error("expected error for nil interpreter")
		}
	})
}

func TestREPLExecutor_ExecuteBash(t *testing.T) {
	t.Run("executes simple echo command", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		exitCode, err := exec.ExecuteBash(ctx, "true")
		if err != nil {
			t.Fatalf("ExecuteBash() error = %v", err)
		}
		if exitCode != 0 {
			t.Errorf("ExecuteBash() exitCode = %d, want 0", exitCode)
		}
	})

	t.Run("returns exit code for failed command", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		exitCode, err := exec.ExecuteBash(ctx, "exit 42")
		if err != nil {
			t.Fatalf("ExecuteBash() error = %v", err)
		}
		if exitCode != 42 {
			t.Errorf("ExecuteBash() exitCode = %d, want 42", exitCode)
		}
	})

	t.Run("returns error for invalid syntax", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		_, err := exec.ExecuteBash(ctx, "if then else")
		if err == nil {
			t.Error("expected error for invalid syntax")
		}
	})
}

func TestREPLExecutor_ExecuteBashInSubshell(t *testing.T) {
	t.Run("captures stdout", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		stdout, stderr, exitCode, err := exec.ExecuteBashInSubshell(ctx, "echo hello")
		if err != nil {
			t.Fatalf("ExecuteBashInSubshell() error = %v", err)
		}
		if exitCode != 0 {
			t.Errorf("ExecuteBashInSubshell() exitCode = %d, want 0", exitCode)
		}
		if strings.TrimSpace(stdout) != "hello" {
			t.Errorf("ExecuteBashInSubshell() stdout = %q, want %q", stdout, "hello\n")
		}
		if stderr != "" {
			t.Errorf("ExecuteBashInSubshell() stderr = %q, want empty", stderr)
		}
	})

	t.Run("captures stderr", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		stdout, stderr, exitCode, err := exec.ExecuteBashInSubshell(ctx, "echo error >&2")
		if err != nil {
			t.Fatalf("ExecuteBashInSubshell() error = %v", err)
		}
		if exitCode != 0 {
			t.Errorf("ExecuteBashInSubshell() exitCode = %d, want 0", exitCode)
		}
		if stdout != "" {
			t.Errorf("ExecuteBashInSubshell() stdout = %q, want empty", stdout)
		}
		if strings.TrimSpace(stderr) != "error" {
			t.Errorf("ExecuteBashInSubshell() stderr = %q, want %q", stderr, "error\n")
		}
	})

	t.Run("handles empty command", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		stdout, stderr, exitCode, err := exec.ExecuteBashInSubshell(ctx, "")
		if err != nil {
			t.Fatalf("ExecuteBashInSubshell() error = %v", err)
		}
		if exitCode != 0 {
			t.Errorf("ExecuteBashInSubshell() exitCode = %d, want 0", exitCode)
		}
		if stdout != "" || stderr != "" {
			t.Errorf("ExecuteBashInSubshell() = (%q, %q), want empty", stdout, stderr)
		}
	})

	t.Run("does not affect parent shell variables", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		// Set a variable via bash command (which populates runner.Vars)
		ctx := context.Background()
		_, err := exec.ExecuteBash(ctx, "TEST_VAR_PARENT=original")
		if err != nil {
			t.Fatalf("ExecuteBash() error = %v", err)
		}

		// Run a subshell command that tries to modify it
		_, _, _, err = exec.ExecuteBashInSubshell(ctx, "TEST_VAR_PARENT=modified")
		if err != nil {
			t.Fatalf("ExecuteBashInSubshell() error = %v", err)
		}

		// Parent shell variable should be unchanged
		if got := exec.GetEnv("TEST_VAR_PARENT"); got != "original" {
			t.Errorf("GetEnv(TEST_VAR_PARENT) = %q, want %q", got, "original")
		}
	})
}

func TestREPLExecutor_ExecuteGsh(t *testing.T) {
	t.Run("executes simple gsh script", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		err := exec.ExecuteGsh(ctx, `x = 1 + 2`)
		if err != nil {
			t.Fatalf("ExecuteGsh() error = %v", err)
		}
	})

	t.Run("returns error for parse errors", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		err := exec.ExecuteGsh(ctx, `if { }`)
		if err == nil {
			t.Error("expected error for invalid gsh syntax")
		}
		if !strings.Contains(err.Error(), "parse error") {
			t.Errorf("error = %v, want to contain 'parse error'", err)
		}
	})

	t.Run("returns error for runtime errors", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		err := exec.ExecuteGsh(ctx, `undefinedVariable + 1`)
		if err == nil {
			t.Error("expected error for undefined variable")
		}
		if !strings.Contains(err.Error(), "execution error") {
			t.Errorf("error = %v, want to contain 'execution error'", err)
		}
	})
}

func TestREPLExecutor_Environment(t *testing.T) {
	t.Run("GetEnv returns empty for undefined variable", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		got := exec.GetEnv("UNDEFINED_VAR_12345")
		if got != "" {
			t.Errorf("GetEnv() = %q, want empty", got)
		}
	})

	t.Run("SetEnv and GetEnv work together", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		exec.SetEnv("MY_TEST_VAR", "test_value")
		got := exec.GetEnv("MY_TEST_VAR")
		if got != "test_value" {
			t.Errorf("GetEnv() = %q, want %q", got, "test_value")
		}
	})

	t.Run("SetEnv overwrites existing value", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		exec.SetEnv("MY_TEST_VAR", "first")
		exec.SetEnv("MY_TEST_VAR", "second")
		got := exec.GetEnv("MY_TEST_VAR")
		if got != "second" {
			t.Errorf("GetEnv() = %q, want %q", got, "second")
		}
	})

	t.Run("bash commands can set environment variables", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		_, err := exec.ExecuteBash(ctx, "MY_BASH_VAR=from_bash")
		if err != nil {
			t.Fatalf("ExecuteBash() error = %v", err)
		}

		got := exec.GetEnv("MY_BASH_VAR")
		if got != "from_bash" {
			t.Errorf("GetEnv(MY_BASH_VAR) = %q, want %q", got, "from_bash")
		}
	})
}

func TestREPLExecutor_GetPwd(t *testing.T) {
	t.Run("returns current working directory", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		got := exec.GetPwd()
		// Should return a valid directory path
		if got == "" {
			t.Error("GetPwd() returned empty string")
		}
	})

	t.Run("updates after cd command", func(t *testing.T) {
		// Create a temporary directory with subdirectory
		tmpDir, err := os.MkdirTemp("", "executor-test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		exec := newTestExecutor(t, nil)
		defer exec.Close()

		// Change to temp directory first, then to subdirectory
		ctx := context.Background()
		_, err = exec.ExecuteBash(ctx, "cd "+tmpDir)
		if err != nil {
			t.Fatalf("ExecuteBash(cd tmpDir) error = %v", err)
		}

		_, err = exec.ExecuteBash(ctx, "cd subdir")
		if err != nil {
			t.Fatalf("ExecuteBash(cd subdir) error = %v", err)
		}

		got := exec.GetPwd()
		// Resolve symlinks for comparison (macOS /var -> /private/var)
		wantResolved, _ := filepath.EvalSymlinks(subDir)
		gotResolved, _ := filepath.EvalSymlinks(got)
		if gotResolved != wantResolved {
			t.Errorf("GetPwd() = %q, want %q", got, subDir)
		}
	})
}

func TestREPLExecutor_Close(t *testing.T) {
	t.Run("close is idempotent", func(t *testing.T) {
		exec := newTestExecutor(t, nil)

		// Close multiple times should not panic
		if err := exec.Close(); err != nil {
			t.Errorf("first Close() error = %v", err)
		}
		if err := exec.Close(); err != nil {
			t.Errorf("second Close() error = %v", err)
		}
	})
}

func TestREPLExecutor_Runner(t *testing.T) {
	t.Run("returns the underlying runner", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		if exec.Runner() == nil {
			t.Error("Runner() returned nil")
		}
	})
}

func TestREPLExecutor_Interpreter(t *testing.T) {
	t.Run("returns the underlying interpreter", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		if exec.Interpreter() == nil {
			t.Error("Interpreter() returned nil")
		}
	})
}

func TestREPLExecutor_RunBashScriptFromReader(t *testing.T) {
	t.Run("runs script from reader", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		reader := strings.NewReader("SCRIPT_RAN_VAR=yes")
		err := exec.RunBashScriptFromReader(ctx, reader, "test.sh")
		if err != nil {
			t.Fatalf("RunBashScriptFromReader() error = %v", err)
		}

		if got := exec.GetEnv("SCRIPT_RAN_VAR"); got != "yes" {
			t.Errorf("GetEnv(SCRIPT_RAN_VAR) = %q, want %q", got, "yes")
		}
	})

	t.Run("returns error for invalid script", func(t *testing.T) {
		exec := newTestExecutor(t, nil)
		defer exec.Close()

		ctx := context.Background()
		reader := strings.NewReader("if then else")
		err := exec.RunBashScriptFromReader(ctx, reader, "test.sh")
		if err == nil {
			t.Error("expected error for invalid script")
		}
	})
}

func TestThreadSafeBuffer(t *testing.T) {
	t.Run("write and string are thread-safe", func(t *testing.T) {
		buf := &threadSafeBuffer{}

		// Write from multiple goroutines
		done := make(chan bool)
		for i := range 10 {
			go func(n int) {
				for range 100 {
					buf.Write([]byte("x"))
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for range 10 {
			<-done
		}

		// Should have 1000 characters
		if len(buf.String()) != 1000 {
			t.Errorf("buffer length = %d, want 1000", len(buf.String()))
		}
	})
}
