package pricing

// DefaultFallbackPricing provides a robust static pricing dictionary
// ensuring the plugin can operate even without external internet connectivity.
var DefaultFallbackPricing = map[string]*ModelPricing{
	// Claude models ($ per token: $3/M input = 3e-6, $15/M output = 1.5e-5)
	"claude-3-7-sonnet": {
		InputCostPerToken:           3e-06,
		OutputCostPerToken:          1.5e-05,
		CacheCreationInputTokenCost: 3.75e-06,
		CacheReadInputTokenCost:     3e-07,
		SupportsPromptCaching:       true,
		LiteLLMProvider:             "anthropic",
	},
	"claude-3-5-sonnet": {
		InputCostPerToken:           3e-06,
		OutputCostPerToken:          1.5e-05,
		CacheCreationInputTokenCost: 3.75e-06,
		CacheReadInputTokenCost:     3e-07,
		SupportsPromptCaching:       true,
		LiteLLMProvider:             "anthropic",
	},
	"claude-3-5-haiku": {
		InputCostPerToken:           8e-07,
		OutputCostPerToken:          4e-06,
		CacheCreationInputTokenCost: 1e-06,
		CacheReadInputTokenCost:     8e-08,
		SupportsPromptCaching:       true,
		LiteLLMProvider:             "anthropic",
	},
	"claude-3-opus": {
		InputCostPerToken:           1.5e-05,
		OutputCostPerToken:          7.5e-05,
		CacheCreationInputTokenCost: 1.875e-05,
		CacheReadInputTokenCost:     1.5e-06,
		SupportsPromptCaching:       true,
		LiteLLMProvider:             "anthropic",
	},
	"claude-opus-4-8": {
		InputCostPerToken:           5e-06,
		OutputCostPerToken:          2.5e-05,
		CacheCreationInputTokenCost: 6.25e-06,
		CacheReadInputTokenCost:     5e-07,
		SupportsPromptCaching:       true,
		LiteLLMProvider:             "anthropic",
	},
	"claude-opus-5": {
		InputCostPerToken:           5e-06,
		OutputCostPerToken:          2.5e-05,
		CacheCreationInputTokenCost: 6.25e-06,
		CacheReadInputTokenCost:     5e-07,
		SupportsPromptCaching:       true,
		LiteLLMProvider:             "anthropic",
	},

	// OpenAI models
	"gpt-4o": {
		InputCostPerToken:              2.5e-06,
		OutputCostPerToken:             1e-05,
		CacheReadInputTokenCost:        1.25e-06,
		LongContextInputTokenThreshold: 272000,
		LongContextInputCostMultiplier: 2.0,
		SupportsPromptCaching:          true,
		LiteLLMProvider:                "openai",
	},
	"gpt-4o-mini": {
		InputCostPerToken:       1.5e-07,
		OutputCostPerToken:      6e-07,
		CacheReadInputTokenCost: 7.5e-08,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "openai",
	},
	"o1": {
		InputCostPerToken:       1.5e-05,
		OutputCostPerToken:      6e-05,
		CacheReadInputTokenCost: 7.5e-06,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "openai",
	},
	"o1-mini": {
		InputCostPerToken:       1.1e-06,
		OutputCostPerToken:      4.4e-06,
		CacheReadInputTokenCost: 5.5e-07,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "openai",
	},
	"o3-mini": {
		InputCostPerToken:       1.1e-06,
		OutputCostPerToken:      4.4e-06,
		CacheReadInputTokenCost: 5.5e-07,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "openai",
	},
	"gpt-5.1-codex": {
		InputCostPerToken:       2.5e-06,
		OutputCostPerToken:      1.5e-05,
		CacheReadInputTokenCost: 2.5e-07,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "openai",
	},
	"gpt-5.2-codex": {
		InputCostPerToken:       2.5e-06,
		OutputCostPerToken:      1.5e-05,
		CacheReadInputTokenCost: 2.5e-07,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "openai",
	},
	"gpt-5.4": {
		InputCostPerToken:              2.5e-06,
		OutputCostPerToken:             1.5e-05,
		CacheReadInputTokenCost:        2.5e-07,
		LongContextInputTokenThreshold: 272000,
		LongContextInputCostMultiplier: 2.0,
		SupportsPromptCaching:          true,
		LiteLLMProvider:                "openai",
	},
	"gpt-5.6-sol": {
		InputCostPerToken:           5e-06,
		OutputCostPerToken:          3e-05,
		CacheCreationInputTokenCost: 6.25e-06,
		CacheReadInputTokenCost:     5e-07,
		SupportsPromptCaching:       true,
		LiteLLMProvider:             "openai",
	},

	// Google Gemini models
	"gemini-2.5-pro": {
		InputCostPerToken:              1.25e-06,
		OutputCostPerToken:             5e-06,
		CacheReadInputTokenCost:        3.125e-07,
		LongContextInputTokenThreshold: 128000,
		LongContextInputCostMultiplier: 2.0,
		SupportsPromptCaching:          true,
		LiteLLMProvider:                "vertex_ai",
	},
	"gemini-2.5-flash": {
		InputCostPerToken:       7.5e-08,
		OutputCostPerToken:      3e-07,
		CacheReadInputTokenCost: 1.875e-08,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "vertex_ai",
	},
	"gemini-3.6-flash": {
		InputCostPerToken:       7.5e-08,
		OutputCostPerToken:      3e-07,
		CacheReadInputTokenCost: 1.875e-08,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "vertex_ai",
	},

	// DeepSeek models
	"deepseek-chat": {
		InputCostPerToken:       1.4e-07,
		OutputCostPerToken:      2.8e-07,
		CacheReadInputTokenCost: 1.4e-08,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "deepseek",
	},
	"deepseek-reasoner": {
		InputCostPerToken:       5.5e-07,
		OutputCostPerToken:      2.19e-06,
		CacheReadInputTokenCost: 1.4e-07,
		SupportsPromptCaching:   true,
		LiteLLMProvider:         "deepseek",
	},
}
