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
	Provider            string
	ExecutorType        string
	InputTokens         int64
	OutputTokens        int64
	ReasoningTokens     int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalTokens         int64
	ServiceTier         string
	APIKey              string
	AuthID              string
	UserMultiplier      float64
	AccountMultiplier   float64
}

type CostBreakdown struct {
	Model             string `json:"model"`
	InputTokens       int64  `json:"input_tokens"`
	OutputTokens      int64  `json:"output_tokens"`
	ReasoningTokens   int64  `json:"reasoning_tokens"`
	CacheReadTokens   int64  `json:"cache_read_tokens"`
	CacheCreateTokens int64  `json:"cache_create_tokens"`
	TotalTokens       int64  `json:"total_tokens"`

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

type tokenSemantics int

const (
	tokenSemanticsSubset tokenSemantics = iota
	tokenSemanticsIndependentCache
	tokenSemanticsSeparateReasoning
	tokenSemanticsFullyIndependent
)

type normalizedTokens struct {
	input         int64
	output        int64
	reasoning     int64
	cacheRead     int64
	cacheCreation int64
}

func normalizeUsageTokens(input CostInput) normalizedTokens {
	semantics := usageTokenSemantics(input)
	tokens := normalizedTokens{
		input:         nonNegative(input.InputTokens),
		output:        nonNegative(input.OutputTokens),
		reasoning:     nonNegative(input.ReasoningTokens),
		cacheRead:     nonNegative(input.CacheReadTokens),
		cacheCreation: nonNegative(input.CacheCreationTokens),
	}

	if semantics == tokenSemanticsSubset || semantics == tokenSemanticsSeparateReasoning {
		tokens.input -= tokens.cacheRead + tokens.cacheCreation
		if tokens.input < 0 {
			tokens.input = 0
		}
	}
	if semantics == tokenSemanticsSeparateReasoning || semantics == tokenSemanticsFullyIndependent {
		tokens.output += tokens.reasoning
	}
	return tokens
}

func usageTokenSemantics(input CostInput) tokenSemantics {
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	executor := strings.ToLower(strings.TrimSpace(input.ExecutorType))
	value := strings.TrimSpace(provider + " " + executor)
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(input.Model))
	}

	if executor == "openaicompatexecutor" || provider == "openai-compatibility" || strings.HasPrefix(provider, "openai-compatible-") {
		return tokenSemanticsSubset
	}
	if strings.Contains(value, "claude") || strings.Contains(value, "anthropic") {
		return tokenSemanticsIndependentCache
	}
	for _, marker := range []string{"gemini", "aistudio", "antigravity", "vertex", "interaction"} {
		if strings.Contains(value, marker) {
			return tokenSemanticsSeparateReasoning
		}
	}
	for _, marker := range []string{"openai", "codex", "xai", "grok", "kimi", "qwen", "deepseek", "openrouter"} {
		if strings.Contains(value, marker) {
			baseTotal := nonNegative(input.InputTokens) + nonNegative(input.OutputTokens)
			reasoningTokens := nonNegative(input.ReasoningTokens)
			if reasoningTokens > 0 && input.TotalTokens == baseTotal+reasoningTokens {
				return tokenSemanticsSeparateReasoning
			}
			return tokenSemanticsSubset
		}
	}

	cacheTokens := nonNegative(input.CacheReadTokens) + nonNegative(input.CacheCreationTokens)
	reasoningTokens := nonNegative(input.ReasoningTokens)
	baseTotal := nonNegative(input.InputTokens) + nonNegative(input.OutputTokens)
	switch input.TotalTokens {
	case baseTotal + cacheTokens + reasoningTokens:
		return tokenSemanticsFullyIndependent
	case baseTotal + cacheTokens:
		return tokenSemanticsIndependentCache
	case baseTotal + reasoningTokens:
		return tokenSemanticsSeparateReasoning
	default:
		return tokenSemanticsSubset
	}
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
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
	tokens := normalizeUsageTokens(input)

	// Fallback rates if unknown model: $2.5/M input, $10/M output
	var (
		inputPrice       = 2.5e-06
		outputPrice      = 1.0e-05
		cacheReadPrice   = 2.5e-07
		cacheCreatePrice = 3.125e-06
		longThreshold    = 0
		longInMult       = 1.0
		longOutMult      = 1.0
		tierMultiplier   = 1.0
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

		// Apply explicit tier prices when available; otherwise use Sub2API's tier multiplier.
		tier := strings.ToLower(strings.TrimSpace(input.ServiceTier))
		if tier == "priority" || tier == "fast" {
			hasPriorityPrices := modelPricing.InputCostPerTokenPriority > 0 ||
				modelPricing.OutputCostPerTokenPriority > 0 ||
				modelPricing.CacheReadInputTokenCostPriority > 0 ||
				modelPricing.CacheCreationInputTokenCostPriority > 0
			if hasPriorityPrices && modelPricing.InputCostPerTokenPriority > 0 {
				inputPrice = modelPricing.InputCostPerTokenPriority
			}
			if hasPriorityPrices && modelPricing.OutputCostPerTokenPriority > 0 {
				outputPrice = modelPricing.OutputCostPerTokenPriority
			}
			if hasPriorityPrices && modelPricing.CacheReadInputTokenCostPriority > 0 {
				cacheReadPrice = modelPricing.CacheReadInputTokenCostPriority
			}
			if hasPriorityPrices && modelPricing.CacheCreationInputTokenCostPriority > 0 {
				cacheCreatePrice = modelPricing.CacheCreationInputTokenCostPriority
			}
			if !hasPriorityPrices {
				tierMultiplier = 2.0
			}
		} else if tier == "flex" {
			tierMultiplier = 0.5
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
	totalInputTokens := tokens.input + tokens.cacheRead + tokens.cacheCreation
	if longThreshold > 0 && totalInputTokens > int64(longThreshold) {
		longContextApplied = true
		inputPrice *= longInMult
		outputPrice *= longOutMult
		cacheReadPrice *= longInMult
		cacheCreatePrice *= longInMult
	}

	inCost := float64(tokens.input) * inputPrice * tierMultiplier
	outCost := float64(tokens.output) * outputPrice * tierMultiplier
	cReadCost := float64(tokens.cacheRead) * cacheReadPrice * tierMultiplier
	cCreateCost := float64(tokens.cacheCreation) * cacheCreatePrice * tierMultiplier

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
		InputTokens:        tokens.input,
		OutputTokens:       tokens.output,
		ReasoningTokens:    tokens.reasoning,
		CacheReadTokens:    tokens.cacheRead,
		CacheCreateTokens:  tokens.cacheCreation,
		TotalTokens:        tokens.input + tokens.output + tokens.cacheRead + tokens.cacheCreation,
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
