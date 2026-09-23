package billing

import (
	"math"
	"testing"

	"github.com/router-for-me/cpa-quota-credit/pkg/pricing"
)

func TestAstraPricing(t *testing.T) {
	calc := NewCalculator(pricing.NewService(pricing.Config{}), MultiplierConfig{})
	for _, tt := range []struct {
		name, tier string
		input      int64
		want       float64
	}{
		{"standard boundary", "", 272000, 2.67},
		{"long context", "", 272001, 5.32002},
		{"priority boundary", "priority", 272000, 5.34},
		{"long priority", "priority", 272001, 10.64004},
		{"long fast", "fast", 272001, 10.64004},
		{"long flex", "flex", 272001, 2.66001},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Input includes 10k cached tokens; output includes reasoning.
			got := calc.Calculate(CostInput{
				Model: "gpt-6-astra", Provider: "codex", ServiceTier: tt.tier,
				InputTokens: tt.input, CacheReadTokens: 10000, OutputTokens: 800,
			})
			if math.Abs(got.BaseTotalCost-tt.want) > 1e-8 {
				t.Fatalf("cost = %v, want %v", got.BaseTotalCost, tt.want)
			}
		})
	}
}

func TestGPT6CacheWriteAndServiceTierPricing(t *testing.T) {
	calc := NewCalculator(pricing.NewService(pricing.Config{}), MultiplierConfig{})
	for _, tt := range []struct {
		model, tier string
		input       int64
		want        float64
	}{
		{"gpt-6-sol", "", 272000, 0.541},
		{"gpt-6-sol", "priority", 272000, 1.082},
		{"gpt-6-sol", "fast", 272001, 2.154008},
		{"gpt-6-sol", "flex", 272001, 0.538502},
		{"gpt-6-luna", "", 272000, 0.02705},
		{"gpt-6-luna", "priority", 272000, 0.0541},
		{"gpt-6-luna", "fast", 272001, 0.1077004},
		{"gpt-6-luna", "flex", 272001, 0.0269251},
	} {
		t.Run(tt.model+"/"+tt.tier, func(t *testing.T) {
			got := calc.Calculate(CostInput{
				Model: tt.model, Provider: "openai", ServiceTier: tt.tier,
				InputTokens: tt.input, CacheReadTokens: 10000,
				CacheCreationTokens: 10000, OutputTokens: 1000,
			})
			if got.PricingUnavailable || math.Abs(got.BaseTotalCost-tt.want) > 1e-8 || got.LongContextApplied != (tt.input > 272000) {
				t.Fatalf("cost = %+v, want %v", got, tt.want)
			}
		})
	}
}
