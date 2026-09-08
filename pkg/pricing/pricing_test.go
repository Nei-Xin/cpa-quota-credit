package pricing

import (
	"testing"
)

func TestPricingService_GetModelPricing(t *testing.T) {
	svc := NewService(Config{})

	tests := []struct {
		model    string
		wantProv string
	}{
		{"claude-3-5-sonnet-20241022", "anthropic"},
		{"claude-3-7-sonnet", "anthropic"},
		{"claude-opus-5", "anthropic"},
		{"gpt-4o", "openai"},
		{"gpt-5.1-codex", "openai"},
		{"models/gemini-2.5-pro", "vertex_ai"},
		{"gemini-3.6-flash-high", "vertex_ai"},
		{"deepseek-reasoner", "deepseek"},
	}

	for _, tt := range tests {
		p := svc.GetModelPricing(tt.model)
		if p == nil {
			t.Errorf("GetModelPricing(%q) returned nil, expected valid pricing", tt.model)
			continue
		}
		if p.LiteLLMProvider != tt.wantProv {
			t.Errorf("GetModelPricing(%q) provider = %q, want %q", tt.model, p.LiteLLMProvider, tt.wantProv)
		}
	}
}
