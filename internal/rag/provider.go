package rag

import (
	"strings"

	"go.uber.org/zap"
)

type ContextProvider struct {
	Logger     *zap.Logger
	Retrievers []ContextRetriever
}

func (p *ContextProvider) GetContext() *map[string]string {
	result := make(map[string]string)

	for _, retriever := range p.Retrievers {
		output, err := retriever.GetContext()
		if err != nil {
			p.Logger.Warn("error getting context from retriever", zap.String("retriever", retriever.Name()), zap.Error(err))
			continue
		}

		result[retriever.Name()] = strings.TrimSpace(output)
	}

	return &result
}
