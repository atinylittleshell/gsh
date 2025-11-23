package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestComposeContextText(t *testing.T) {
	// Mock logger
	logger, _ := zap.NewDevelopment(zap.IncreaseLevel(zapcore.WarnLevel))

	context := map[string]string{
		"type1": "This is type 1",
		"type2": "This is type 2",
	}

	// Test with valid keys
	result := ComposeContextText(&context, []string{"type1", "type2"}, logger)
	assert.Equal(t, "\nThis is type 1\n\nThis is type 2\n", result, "Should concatenate values for valid keys")

	// Test with a missing key
	result = ComposeContextText(&context, []string{"type1", "type3"}, logger)
	assert.Equal(t, "\nThis is type 1\n", result, "Should skip missing keys and log a warning")

	// Test with empty contextTypes
	result = ComposeContextText(&context, []string{}, logger)
	assert.Equal(t, "", result, "Should return empty string for empty contextTypes")

	// Test with nil context
	result = ComposeContextText(nil, []string{"type1"}, logger)
	assert.Equal(t, "", result, "Should return empty string for nil context")
}
