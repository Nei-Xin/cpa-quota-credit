package billing

import (
	"strings"

	"github.com/router-for-me/cpa-quota-credit/pkg/pricing"
)

type MultiplierConfig struct {
	DefaultUserMultiplier    float64            `json:"default_user_multiplier"`
	DefaultAccountMultiplier float64            `json:"default_account_multiplier"`
	KeyMultipliers           map[string]float64 `json:"key_multipliers,omitempty"`
	AccountMultipliers       map[string]float64 `json:"account_multipliers,omitempty"` // Per-AuthID multiplier (e.g. "claude-oauth-1": 0.8)
	ModelMultipliers         map[string]float64 `json:"model_multipliers,omitempty"`
}

type CostInput struct {
	Model               string
	InputTokens         int64
	OutputTokens        int64
	ReasoningTokens     int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	ServiceTier         string
	APIKey              string
	AuthID              string
	UserMultiplier      float64
	AccountMultiplier   float64
}

type CostBreakdown struct {
	Model             string  `json:"model"`
	InputTokens       int64   `json:"input_tokens"`
	OutputTokens      int64   `json:"output_tokens"`
	ReasoningTokens   int64   `json:"reasoning_tokens"`
	CacheReadTokens   int64   `json:"cache_read_tokens"`
	CacheCreateTokens int64   `json:"cache_create_tokens"`
	TotalTokens       int64   `json:"total_tokens"`

	InputCost         float64 `json:"input_cost"`
	OutputCost        float64 `json:"output_cost"`
	CacheReadCost     float64 `json:"cache_read_cost"`
	CacheCreationCost float64 `json:"cache_creation_cost"`
	BaseTotalCost     float64 `json:"base_total_cost"`

	UserMultiplier    float64 `json:"user_multiplier"`
	AccountMultiplier float64 `json:"account_multiplier"`

	// A: Actual / Admin / Upstream cost
	ActualCost float64 `json:"actual_cost"`
	// U: User cost / Billed quota
	UserCost float64 `json:"user_cost"`

	LongContextApplied bool `json:"long_context_applied"`
	ServiceTierApplied bool `json:"service_tier_applied"`
}

type Calculator struct {
	pricingService *pricing.Service
	config         MultiplierConfig
}

func NewCalculator(pricingService *pricing.Service, config MultiplierConfig) *Calculator {
	if config.DefaultUserMultiplier <= 0 {
		config.DefaultUserMultiplier = 1.0
	}
	if config.DefaultAccountMultiplier <= 0 {
		config.DefaultAccountMultiplier = 1.0
	}
	return &Calculator{
		pricingService: pricingService,
		config:         config,
	}
}

func (c *Calculator) Calculate(input CostInput) *CostBreakdown {
	modelPricing := c.pricingService.GetModelPricing(input.Model)

	// Fallback rates if unknown model: $2.5/M input, $10/M output
	var (
		inputPrice       = 2.5e-06
		outputPrice      = 1.0e-05
		cacheReadPrice   = 2.5e-07
		cacheCreatePrice = 3.125e-06
		longThreshold    = 0
		longInMult       = 1.0
		longOutMult      = 1.0
	)

	if modelPricing != nil {
		if modelPricing.InputCostPerToken > 0 {
			inputPrice = modelPricing.InputCostPerToken
		}
		if modelPricing.OutputCostPerToken > 0 {
			outputPrice = modelPricing.OutputCostPerToken
		}
		if modelPricing.CacheReadInputTokenCost > 0 {
			cacheReadPrice = modelPricing.CacheReadInputTokenCost
		} else if modelPricing.SupportsPromptCaching {
			cacheReadPrice = inputPrice * 0.1
		}
		if modelPricing.CacheCreationInputTokenCost > 0 {
			cacheCreatePrice = modelPricing.CacheCreationInputTokenCost
		} else if modelPricing.SupportsPromptCaching {
			cacheCreatePrice = inputPrice * 1.25
		}

		// Check service tier overrides (e.g. priority tier)
		tier := strings.ToLower(strings.TrimSpace(input.ServiceTier))
		if tier == "priority" || tier == "fast" {
			if modelPricing.InputCostPerTokenPriority > 0 {
				inputPrice = modelPricing.InputCostPerTokenPriority
			} else {
				inputPrice *= 2.0
			}
			if modelPricing.OutputCostPerTokenPriority > 0 {
				outputPrice = modelPricing.OutputCostPerTokenPriority
			} else {
				outputPrice *= 2.0
			}
			if modelPricing.CacheReadInputTokenCostPriority > 0 {
				cacheReadPrice = modelPricing.CacheReadInputTokenCostPriority
			}
			if modelPricing.CacheCreationInputTokenCostPriority > 0 {
				cacheCreatePrice = modelPricing.CacheCreationInputTokenCostPriority
			}
		}

		longThreshold = modelPricing.LongContextInputTokenThreshold
		if modelPricing.LongContextInputCostMultiplier > 1.0 {
			longInMult = modelPricing.LongContextInputCostMultiplier
		}
		if modelPricing.LongContextOutputCostMultiplier > 1.0 {
			longOutMult = modelPricing.LongContextOutputCostMultiplier
		}
	}

	longContextApplied := false
	if longThreshold > 0 && input.InputTokens > int64(longThreshold) {
		longContextApplied = true
		inputPrice *= longInMult
		outputPrice *= longOutMult
	}

	inCost := float64(input.InputTokens) * inputPrice
	outCost := float64(input.OutputTokens+input.ReasoningTokens) * outputPrice
	cReadCost := float64(input.CacheReadTokens) * cacheReadPrice
	cCreateCost := float64(input.CacheCreationTokens) * cacheCreatePrice

	baseTotal := inCost + outCost + cReadCost + cCreateCost

	// Multipliers resolution for User (U $)
	userMult := input.UserMultiplier
	if userMult <= 0 {
		if c.config.KeyMultipliers != nil && input.APIKey != "" {
			if m, ok := c.config.KeyMultipliers[input.APIKey]; ok && m > 0 {
				userMult = m
			}
		}
	}
	if userMult <= 0 {
		userMult = c.config.DefaultUserMultiplier
	}

	// Multipliers resolution for Account / Upstream Auth (A $)
	accountMult := input.AccountMultiplier
	if accountMult <= 0 {
		if c.config.AccountMultipliers != nil && input.AuthID != "" {
			if m, ok := c.config.AccountMultipliers[input.AuthID]; ok && m > 0 {
				accountMult = m
			}
		}
	}
	if accountMult <= 0 {
		accountMult = c.config.DefaultAccountMultiplier
	}

	actualCost := QuantizeAmount(baseTotal * accountMult)
	userCost := QuantizeAmount(baseTotal * userMult)

	return &CostBreakdown{
		Model:              input.Model,
		InputTokens:        input.InputTokens,
		OutputTokens:       input.OutputTokens,
		ReasoningTokens:    input.ReasoningTokens,
		CacheReadTokens:    input.CacheReadTokens,
		CacheCreateTokens:  input.CacheCreationTokens,
		TotalTokens:        input.InputTokens + input.OutputTokens + input.ReasoningTokens + input.CacheReadTokens + input.CacheCreationTokens,
		InputCost:          QuantizeAmount(inCost),
		OutputCost:         QuantizeAmount(outCost),
		CacheReadCost:      QuantizeAmount(cReadCost),
		CacheCreationCost:  QuantizeAmount(cCreateCost),
		BaseTotalCost:      QuantizeAmount(baseTotal),
		UserMultiplier:     userMult,
		AccountMultiplier:  accountMult,
		ActualCost:         actualCost,
		UserCost:           userCost,
		LongContextApplied: longContextApplied,
		ServiceTierApplied: strings.ToLower(strings.TrimSpace(input.ServiceTier)) != "",
	}
}
