package predict

import (
	"context"
	"testing"

	"github.com/atinylittleshell/gsh/internal/history"
	"github.com/atinylittleshell/gsh/internal/repl/input"
	"github.com/atinylittleshell/gsh/internal/script/interpreter"
	"github.com/atinylittleshell/gsh/internal/script/lexer"
	"github.com/atinylittleshell/gsh/internal/script/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func evalScript(t *testing.T, interp *interpreter.Interpreter, script string) {
	t.Helper()

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	_, err := interp.Eval(program)
	require.NoError(t, err)
}

func TestEventPredictionProvider_UsesMiddleware(t *testing.T) {
	interp := interpreter.New(nil)
	script := `
tool predict(ctx, next) {
    return { prediction: "ls -la" }
}
gsh.use("repl.predict", predict)
`
	evalScript(t, interp, script)

	provider := NewEventPredictionProvider(interp, zap.NewNop())
	pred, err := provider.Predict(context.Background(), input.PredictionRequest{Input: "ls"})

	require.NoError(t, err)
	assert.Equal(t, "ls -la", pred.Prediction)
}

func TestEventPredictionProvider_ReturnsEmptyOnNull(t *testing.T) {
	interp := interpreter.New(nil)
	script := `
tool predict(ctx, next) {
    return null
}
gsh.use("repl.predict", predict)
`
	evalScript(t, interp, script)

	provider := NewEventPredictionProvider(interp, zap.NewNop())

	pred, err := provider.Predict(context.Background(), input.PredictionRequest{Input: "echo"})
	require.NoError(t, err)
	assert.Equal(t, "", pred.Prediction)
}

func TestEventPredictionProvider_ReturnsEmptyOnError(t *testing.T) {
	interp := interpreter.New(nil)
	script := `
tool predict(ctx, next) {
    return { error: "boom" }
}
gsh.use("repl.predict", predict)
`
	evalScript(t, interp, script)

	provider := NewEventPredictionProvider(interp, zap.NewNop())

	pred, err := provider.Predict(context.Background(), input.PredictionRequest{Input: "git"})
	require.NoError(t, err)
	assert.Equal(t, "", pred.Prediction)
}

func TestEventPredictionProvider_HistoryContext(t *testing.T) {
	interp := interpreter.New(nil)
	script := `
tool predict(ctx, next) {
    if (ctx.source == "history" && ctx.history != null && ctx.history.length > 0) {
        return { prediction: ctx.history[0].command, source: "history" }
    }
    return null
}
gsh.use("repl.predict", predict)
`
	evalScript(t, interp, script)

	provider := NewEventPredictionProvider(interp, zap.NewNop())

	request := input.PredictionRequest{
		Input: "git",
		History: []history.HistoryEntry{
			{Command: "git status"},
		},
		Source: input.PredictionSourceHistory,
	}

	pred, err := provider.Predict(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, "git status", pred.Prediction)
	assert.Equal(t, input.PredictionSourceHistory, pred.Source)
}
