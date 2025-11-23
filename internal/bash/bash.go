package bash

import (
	"context"
	"io"
	"os"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func RunBashScriptFromReader(ctx context.Context, runner *interp.Runner, reader io.Reader, name string) error {
	prog, err := syntax.NewParser().Parse(reader, name)
	if err != nil {
		return err
	}
	return runner.Run(ctx, prog)
}

func RunBashScriptFromFile(ctx context.Context, runner *interp.Runner, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return RunBashScriptFromReader(ctx, runner, f, filePath)
}

func RunBashCommandInSubShell(ctx context.Context, runner *interp.Runner, command string) (string, string, error) {
	subShell := runner.Subshell()

	outBuf := &threadSafeBuffer{}
	outWriter := io.Writer(outBuf)
	errBuf := &threadSafeBuffer{}
	errWriter := io.Writer(errBuf)
	interp.StdIO(nil, outWriter, errWriter)(subShell) //nolint:all

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		return "", "", err
	}

	err = subShell.Run(ctx, prog)
	if err != nil {
		return "", "", err
	}

	return outBuf.String(), errBuf.String(), nil
}

func RunBashCommand(ctx context.Context, runner *interp.Runner, command string) (string, string, error) {
	outBuf := &threadSafeBuffer{}
	outWriter := io.Writer(outBuf)
	errBuf := &threadSafeBuffer{}
	errWriter := io.Writer(errBuf)
	interp.StdIO(nil, outWriter, errWriter)(runner)            //nolint:all
	defer interp.StdIO(os.Stdin, os.Stdout, os.Stderr)(runner) //nolint:all

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		return "", "", err
	}

	err = runner.Run(ctx, prog)
	if err != nil {
		return "", "", err
	}

	return outBuf.String(), errBuf.String(), nil
}
