package interpreter

import (
	"testing"
)

func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Register a provider
	openai := NewOpenAIProvider()
	registry.Register(openai)

	// Test Get
	provider, ok := registry.Get("openai")
	if !ok {
		t.Fatal("expected to find 'openai' provider")
	}

	if provider.Name() != "openai" {
		t.Errorf("expected provider name 'openai', got %q", provider.Name())
	}

	// Test Get with non-existent provider
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("expected not to find 'nonexistent' provider")
	}
}

func TestOpenAIProviderName(t *testing.T) {
	provider := NewOpenAIProvider()
	if provider.Name() != "openai" {
		t.Errorf("expected provider name 'openai', got %q", provider.Name())
	}
}
