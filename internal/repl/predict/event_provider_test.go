package predict

import (
	"context"
	"testing"

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
	pred, err := provider.Predict(context.Background(), "ls", interpreter.PredictTriggerDebounced)

	require.NoError(t, err)
	assert.Equal(t, "ls -la", pred)
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

	pred, err := provider.Predict(context.Background(), "echo", interpreter.PredictTriggerDebounced)
	require.NoError(t, err)
	assert.Equal(t, "", pred)
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

	pred, err := provider.Predict(context.Background(), "git", interpreter.PredictTriggerDebounced)
	require.NoError(t, err)
	assert.Equal(t, "", pred)
}
