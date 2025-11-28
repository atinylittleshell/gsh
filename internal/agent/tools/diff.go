package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/atinylittleshell/gsh/internal/utils"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func getDiff(runner *interp.Runner, logger *zap.Logger, file1, file2 string) (string, error) {
	command := fmt.Sprintf("git diff --color=always --no-index %s %s", file1, file2)

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(command), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if err != nil {
		logger.Error("Failed to preview code edits", zap.Error(err))
		return "", err
	}

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	outWriter := io.Writer(outBuf)
	errWriter := io.Writer(errBuf)

	subShell := runner.Subshell()
	_ = interp.StdIO(nil, outWriter, errWriter)(subShell)

	err = subShell.Run(context.Background(), prog)

	exitCode := -1
	if err != nil {
		status, ok := interp.IsExitStatus(err)
		if ok {
			exitCode = int(status)
		}
	} else {
		exitCode = 0
	}

	if exitCode == 128 {
		return "", fmt.Errorf("error running git diff command: %s", errBuf.String())
	}

	result := strings.ReplaceAll(outBuf.String(), "b"+file2, "")
	result = strings.ReplaceAll(result, " a"+file1, " "+utils.HideHomeDirPath(runner, file1))
	result = strings.ReplaceAll(result, file1, " "+utils.HideHomeDirPath(runner, file1))

	return result, nil
}

