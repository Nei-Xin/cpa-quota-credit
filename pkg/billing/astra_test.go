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
