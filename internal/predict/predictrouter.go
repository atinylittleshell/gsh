package predict

import "strings"

type PredictRouter struct {
	PrefixPredictor    *LLMPrefixPredictor
	NullStatePredictor *LLMNullStatePredictor
}

func (p *PredictRouter) UpdateContext(context *map[string]string) {
	if p.PrefixPredictor != nil {
		p.PrefixPredictor.UpdateContext(context)
	}

	if p.NullStatePredictor != nil {
		p.NullStatePredictor.UpdateContext(context)
	}
}

func (p *PredictRouter) Predict(input string) (string, string, error) {
	// Skip LLM prediction when input is blank (empty or whitespace only)
	if strings.TrimSpace(input) == "" {
		return "", "", nil
	}
	return p.PrefixPredictor.Predict(input)
}
