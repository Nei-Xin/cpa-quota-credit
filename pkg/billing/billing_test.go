package billing

import (
	"testing"

	"github.com/router-for-me/cpa-quota-credit/pkg/pricing"
)

func TestCalculator_Calculate(t *testing.T) {
	priceSvc := pricing.NewService(pricing.Config{})
	calc := NewCalculator(priceSvc, MultiplierConfig{
		DefaultUserMultiplier:    1.2,
		DefaultAccountMultiplier: 1.0,
		KeyMultipliers: map[string]float64{
			"sk-vip-custom": 0.9,
		},
	})

	// 1. Standard Claude 3.5 Sonnet calculation
	// 1000 input tokens ($3/M) = $0.003
	// 500 output tokens ($15/M) = $0.0075
	// Base = 0.0105
	// A = 0.0105 * 1.0 = 0.0105
	// U = 0.0105 * 1.2 = 0.0126
	input := CostInput{
		Model:        "claude-3-5-sonnet",
		InputTokens:  1000,
		OutputTokens: 500,
	}
	res := calc.Calculate(input)
	if res.InputCost != 0.003 {
		t.Errorf("InputCost = %v, want 0.003", res.InputCost)
	}
	if res.OutputCost != 0.0075 {
		t.Errorf("OutputCost = %v, want 0.0075", res.OutputCost)
	}
	if res.BaseTotalCost != 0.0105 {
		t.Errorf("BaseTotalCost = %v, want 0.0105", res.BaseTotalCost)
	}
	if res.ActualCost != 0.0105 {
		t.Errorf("ActualCost (A) = %v, want 0.0105", res.ActualCost)
	}
	if res.UserCost != 0.0126 {
		t.Errorf("UserCost (U) = %v, want 0.0126", res.UserCost)
	}

	// 2. Custom Key Multiplier
	inputVIP := CostInput{
		Model:        "claude-3-5-sonnet",
		InputTokens:  1000,
		OutputTokens: 500,
		APIKey:       "sk-vip-custom",
	}
	resVIP := calc.Calculate(inputVIP)
	if resVIP.UserCost != 0.00945 { // 0.0105 * 0.9 = 0.00945
		t.Errorf("VIP UserCost = %v, want 0.00945", resVIP.UserCost)
	}

	// 3. Prompt Caching
	inputCache := CostInput{
		Model:               "claude-3-5-sonnet",
		InputTokens:         100,
		OutputTokens:        100,
		CacheReadTokens:     5000, // 5000 * 3e-7 = 0.0015
		CacheCreationTokens: 2000, // 2000 * 3.75e-6 = 0.0075
	}
	resCache := calc.Calculate(inputCache)
	if resCache.CacheReadCost != 0.0015 {
		t.Errorf("CacheReadCost = %v, want 0.0015", resCache.CacheReadCost)
	}
	if resCache.CacheCreationCost != 0.0075 {
		t.Errorf("CacheCreationCost = %v, want 0.0075", resCache.CacheCreationCost)
	}
}

func TestQuantizeAmount(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.000078125, 0.00007813},
		{0.123456781, 0.12345678},
		{0.123456789, 0.12345679},
		{0.0, 0.0},
	}
	for _, tc := range cases {
		got := QuantizeAmount(tc.in)
		if got != tc.want {
			t.Errorf("QuantizeAmount(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
