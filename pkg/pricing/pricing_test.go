package pricing

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOverrideSurvivesRemoteSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"gpt-5.6-sol":{"input_cost_per_token":0.000005,"output_cost_per_token":0.00003}}`))
	}))
	defer server.Close()
	want := DefaultFallbackPricing["gpt-5.6-sol"]
	svc := NewService(Config{RemoteURL: server.URL, DataDir: t.TempDir(), Overrides: map[string]*ModelPricing{"gpt-5.6-sol": want}})
	if err := svc.SyncWithRemote(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gpt-5.6-sol", "gpt-5.6", "openai/gpt-5.6-sol-2026-09-08"} {
		if got := svc.GetModelPricing(name); got != want {
			t.Fatalf("override lost for %s: %+v", name, got)
		}
	}
}

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

func TestOpenAIModelMatchingDoesNotSubstituteVariants(t *testing.T) {
	svc := NewService(Config{})
	for _, model := range []string{"gpt-unknown", "gpt-5.4-unknown", "openai/gpt-5.4-unknown"} {
		if svc.GetModelPricing(model) != nil {
			t.Fatalf("unexpected price for %s", model)
		}
	}
	for _, model := range []string{"gpt-5.4-20260908", "openai/gpt-5.4-2026-09-08"} {
		if svc.GetModelPricing(model) != svc.GetModelPricing("gpt-5.4") {
			t.Fatalf("date alias failed: %s", model)
		}
	}
}
