package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderWelcome(t *testing.T) {
	tests := []struct {
		name      string
		info      WelcomeInfo
		termWidth int
		wantLogo  bool
		wantTexts []string
	}{
		{
			name: "full info with wide terminal",
			info: WelcomeInfo{
				PredictModel: "gemma3:1b",
				AgentModel:   "devstral-small-2",
				Version:      "1.0.0",
			},
			termWidth: 80,
			wantLogo:  true,
			wantTexts: []string{
				"The G Shell",
				"version: 1.0.0",
				"predict: gemma3:1b",
				"agent:   devstral-small-2",
				"tip:", // Dynamic tip of the day
			},
		},
		{
			name: "dev version",
			info: WelcomeInfo{
				PredictModel: "test-model",
				AgentModel:   "test-agent",
				Version:      "dev",
			},
			termWidth: 80,
			wantLogo:  true,
			wantTexts: []string{
				"The G Shell",
				"development",
				"predict: test-model",
				"agent:   test-agent",
			},
		},
		{
			name: "no models configured",
			info: WelcomeInfo{
				Version: "1.0.0",
			},
			termWidth: 80,
			wantLogo:  true,
			wantTexts: []string{
				"The G Shell",
				"predict: not configured",
				"agent:   not configured",
			},
		},
		{
			name: "narrow terminal - no logo",
			info: WelcomeInfo{
				PredictModel: "gemma3:1b",
				AgentModel:   "devstral-small-2",
				Version:      "1.0.0",
			},
			termWidth: 30,
			wantLogo:  false,
			wantTexts: []string{
				"The G Shell",
				"predict: gemma3:1b",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderWelcome(&buf, tt.info, tt.termWidth)
			output := buf.String()

			// Check for expected texts
			for _, text := range tt.wantTexts {
				assert.Contains(t, output, text, "output should contain %q", text)
			}

			// Check for logo presence (ASCII art contains "__ _" pattern)
			if tt.wantLogo {
				assert.Contains(t, output, "__ _", "output should contain logo")
			}
		})
	}
}

func TestRenderWelcome_LogoPresent(t *testing.T) {
	var buf bytes.Buffer
	info := WelcomeInfo{
		PredictModel: "test-model",
		AgentModel:   "test-agent",
		Version:      "1.0.0",
	}
	RenderWelcome(&buf, info, 80)
	output := buf.String()

	// The ASCII logo should contain these patterns from the gsh ASCII art
	logoPatterns := []string{
		"__ _",    // Part of the 'g'
		"|___/",   // Bottom of the 'g'
		"| '_ \\", // Part of 's' and 'h'
	}

	for _, pattern := range logoPatterns {
		assert.True(t, strings.Contains(output, pattern),
			"logo should contain %q", pattern)
	}
}

func TestRenderWelcome_TwoColumnLayout(t *testing.T) {
	var buf bytes.Buffer
	info := WelcomeInfo{
		PredictModel: "gemma3:1b",
		AgentModel:   "devstral-small-2",
		Version:      "1.0.0",
	}
	RenderWelcome(&buf, info, 80)
	output := buf.String()

	// In two-column layout, logo and info should be on the same lines
	lines := strings.Split(output, "\n")

	// Find a line that has both logo characters and info text
	foundTwoColumn := false
	for _, line := range lines {
		// Logo uses special characters like | and _
		// Info uses text like "version" or "predict"
		hasLogoChars := strings.Contains(line, "|") || strings.Contains(line, "_")
		hasInfoText := strings.Contains(line, "version") || strings.Contains(line, "predict") || strings.Contains(line, "generative")
		if hasLogoChars && hasInfoText {
			foundTwoColumn = true
			break
		}
	}

	assert.True(t, foundTwoColumn, "should have two-column layout with logo and info on same line")
}

func TestGetTipOfTheDay(t *testing.T) {
	t.Run("returns a non-empty tip", func(t *testing.T) {
		tip := getTipOfTheDay()
		assert.NotEmpty(t, tip, "tip should not be empty")
	})

	t.Run("returns different tips on different calls", func(t *testing.T) {
		// Call many times to ensure we see variation
		// With len(tips) calls, we should definitely see different tips
		numCalls := len(tips)
		tips := make(map[string]bool)
		for i := 0; i < numCalls; i++ {
			tips[getTipOfTheDay()] = true
		}
		// With len(tips) random selections, we should have multiple unique tips
		assert.Greater(t, len(tips), 1, "should return different tips across multiple calls")
	})

	t.Run("tip is from the tips list", func(t *testing.T) {
		tip := getTipOfTheDay()
		found := false
		for _, t := range tips {
			if t == tip {
				found = true
				break
			}
		}
		assert.True(t, found, "tip should be from the tips list")
	})
}
