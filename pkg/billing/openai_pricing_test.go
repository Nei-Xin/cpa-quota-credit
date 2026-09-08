package billing

import (
	"testing"

	"github.com/router-for-me/cpa-quota-credit/pkg/pricing"
)

func TestOpenAIFallbackRates(t *testing.T) {
	calc := NewCalculator(pricing.NewService(pricing.Config{}), MultiplierConfig{})
	for _, tt := range []struct {
		model string
		want  float64
	}{
		{"gpt-5.1-codex", 0.2375},
		{"gpt-5.2-codex", 0.3325},
		{"gpt-5.6-sol", 0.64},
		{"gpt-5.6", 0.64},
	} {
		// 100k uncached input, 100k cache reads and 10k output (including reasoning).
		got := calc.Calculate(CostInput{Model: tt.model, Provider: "openai", InputTokens: 200000, CacheReadTokens: 100000, OutputTokens: 10000, ReasoningTokens: 5000, TotalTokens: 210000})
		if got.BaseTotalCost != tt.want || got.PricingUnavailable {
			t.Fatalf("%s: %+v, want %v", tt.model, got, tt.want)
		}
	}
}

func TestOpenAILongContextBoundary(t *testing.T) {
	calc := NewCalculator(pricing.NewService(pricing.Config{}), MultiplierConfig{})
	for _, tt := range []struct {
		model string
		input int64
		want  float64
		long  bool
	}{
		{"gpt-5.4", 272000, 0.68, false},
		{"gpt-5.4", 272001, 1.247505, true},
		{"gpt-5.6-sol", 272000, 1.028, false},
		{"gpt-5.6-sol", 272001, 1.906008, true},
	} {
		got := calc.Calculate(CostInput{Model: tt.model, Provider: "openai", InputTokens: tt.input, CacheReadTokens: 100000, OutputTokens: 15000})
		if got.BaseTotalCost != tt.want || got.LongContextApplied != tt.long {
			t.Fatalf("%+v: got %+v", tt, got)
		}
	}
}

func TestMissingPricingRetainsUsageWithoutCharge(t *testing.T) {
	calc := NewCalculator(pricing.NewService(pricing.Config{}), MultiplierConfig{})
	for _, tt := range []struct{ model, tier string }{
		{"gpt-unknown", ""}, {"gpt-5.4-unknown", ""}, {"unknown-vendor-model", ""}, {"gpt-5.4", "priority"},
	} {
		got := calc.Calculate(CostInput{Model: tt.model, Provider: "openai", ServiceTier: tt.tier, InputTokens: 1000, CacheReadTokens: 100, OutputTokens: 200})
		if !got.PricingUnavailable || got.TotalTokens != 1200 || got.ActualCost != 0 || got.UserCost != 0 {
			t.Fatalf("%+v: %+v", tt, got)
		}
	}
}
