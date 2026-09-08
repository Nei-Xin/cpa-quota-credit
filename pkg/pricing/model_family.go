package pricing

import (
	"regexp"
	"strings"
)

var (
	openAIModelDatePattern = regexp.MustCompile(`-\d{8}$`)
	openAIModelBasePattern = regexp.MustCompile(`^(gpt-\d+(?:\.\d+)?)(?:-|$)`)
)

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func lastSegment(model string) string {
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		return model[idx+1:]
	}
	return model
}

func extractBaseName(model string) string {
	parts := strings.Split(model, "-")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) == 8 && isNumeric(part) {
			continue
		}
		if strings.Contains(part, ":") {
			continue
		}
		result = append(result, part)
	}
	return strings.Join(result, "-")
}

// normalizeGeminiThinkingTierAlias aligns with sub2api Antigravity's Gemini 3.6 Flash thinking-tier aliases.
func normalizeGeminiThinkingTierAlias(model string) string {
	const baseModel = "gemini-3.6-flash"
	for _, tier := range []string{"-high", "-low", "-medium", "-tiered"} {
		if model == baseModel+tier {
			return baseModel
		}
	}
	return model
}

func normalizeModelNameForPricing(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimLeft(model, "/")
	model = strings.TrimPrefix(model, "models/")
	model = strings.TrimPrefix(model, "publishers/google/models/")

	if idx := strings.LastIndex(model, "/publishers/google/models/"); idx != -1 {
		model = model[idx+len("/publishers/google/models/"):]
	}
	if idx := strings.LastIndex(model, "/models/"); idx != -1 {
		model = model[idx+len("/models/"):]
	}

	model = strings.TrimLeft(model, "/")
	return normalizeGeminiThinkingTierAlias(model)
}

func buildModelLookupCandidates(modelLower string) []string {
	rawCandidates := []string{
		modelLower,
		strings.TrimPrefix(modelLower, "models/"),
		lastSegment(modelLower),
		lastSegment(strings.TrimPrefix(modelLower, "models/")),
	}
	normalized := normalizeModelNameForPricing(modelLower)

	candidates := rawCandidates
	if normalizeGeminiThinkingTierAlias(lastSegment(modelLower)) != lastSegment(modelLower) {
		candidates = append(candidates, normalized)
	} else {
		candidates = append([]string{normalized}, candidates...)
	}

	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	if len(out) == 0 {
		return []string{modelLower}
	}
	return out
}

type modelFamily struct {
	name    string
	match   []string
	pricing []string
}

var claudeFamilies = []modelFamily{
	{name: "opus-5", match: []string{"claude-opus-5"}, pricing: []string{"claude-opus-5", "claude-opus-4-8", "claude-3-opus"}},
	{name: "opus-4.8", match: []string{"claude-opus-4-8", "claude-opus-4.8"}, pricing: []string{"claude-opus-4-8", "claude-opus-4.8", "claude-opus-4-7", "claude-3-opus"}},
	{name: "opus-4.7", match: []string{"claude-opus-4-7", "claude-opus-4.7"}, pricing: []string{"claude-opus-4-7", "claude-opus-4.7", "claude-opus-4-6", "claude-3-opus"}},
	{name: "opus-4.6", match: []string{"claude-opus-4-6", "claude-opus-4.6"}, pricing: []string{"claude-opus-4-6", "claude-3-opus"}},
	{name: "opus-4.5", match: []string{"claude-opus-4-5", "claude-opus-4.5"}, pricing: []string{"claude-opus-4-5", "claude-3-opus"}},
	{name: "opus-4", match: []string{"claude-opus-4", "claude-3-opus"}, pricing: []string{"claude-3-opus"}},
	{name: "sonnet-4.5", match: []string{"claude-sonnet-4-5", "claude-sonnet-4.5"}, pricing: []string{"claude-sonnet-4-5", "claude-3-5-sonnet"}},
	{name: "sonnet-4", match: []string{"claude-sonnet-4", "claude-3-5-sonnet"}, pricing: []string{"claude-3-5-sonnet"}},
	{name: "sonnet-3.7", match: []string{"claude-3-7-sonnet", "claude-3.7-sonnet"}, pricing: []string{"claude-3-7-sonnet", "claude-3-5-sonnet"}},
	{name: "sonnet-3.5", match: []string{"claude-3-5-sonnet", "claude-3.5-sonnet"}, pricing: []string{"claude-3-5-sonnet"}},
	{name: "sonnet-3", match: []string{"claude-3-sonnet"}, pricing: []string{"claude-3-sonnet"}},
	{name: "haiku-3.5", match: []string{"claude-3-5-haiku", "claude-3.5-haiku"}, pricing: []string{"claude-3-5-haiku"}},
	{name: "haiku-3", match: []string{"claude-3-haiku"}, pricing: []string{"claude-3-haiku"}},
}
