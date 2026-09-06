package pricing

// LongContext272K mirrors the explicit long-context rates in the price feed.
type LongContext272K struct {
	Input               float64 `json:"input_cost_per_token_above_272k_tokens"`
	Output              float64 `json:"output_cost_per_token_above_272k_tokens"`
	CacheRead           float64 `json:"cache_read_input_token_cost_above_272k_tokens"`
	CacheCreate         float64 `json:"cache_creation_input_token_cost_above_272k_tokens"`
	InputPriority       float64 `json:"input_cost_per_token_above_272k_tokens_priority"`
	OutputPriority      float64 `json:"output_cost_per_token_above_272k_tokens_priority"`
	CacheReadPriority   float64 `json:"cache_read_input_token_cost_above_272k_tokens_priority"`
	CacheCreatePriority float64 `json:"cache_creation_input_token_cost_above_272k_tokens_priority"`
	InputFlex           float64 `json:"input_cost_per_token_above_272k_tokens_flex"`
	OutputFlex          float64 `json:"output_cost_per_token_above_272k_tokens_flex"`
	CacheReadFlex       float64 `json:"cache_read_input_token_cost_above_272k_tokens_flex"`
	CacheCreateFlex     float64 `json:"cache_creation_input_token_cost_above_272k_tokens_flex"`
}

// Rates returns explicit per-token rates; zero fields retain the caller's rate.
func (p LongContext272K) Rates(tier string) [4]float64 {
	rates := [4]float64{p.Input, p.Output, p.CacheRead, p.CacheCreate}
	var overrides [4]float64
	switch tier {
	case "priority", "fast":
		overrides = [4]float64{p.InputPriority, p.OutputPriority, p.CacheReadPriority, p.CacheCreatePriority}
		for i := range rates {
			rates[i] *= 2
		}
	case "flex":
		overrides = [4]float64{p.InputFlex, p.OutputFlex, p.CacheReadFlex, p.CacheCreateFlex}
		for i := range rates {
			rates[i] *= 0.5
		}
	}
	for i, rate := range overrides {
		if rate > 0 {
			rates[i] = rate
		}
	}
	return rates
}
