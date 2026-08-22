package pricing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ModelPricing holds the per-token pricing details for a model
type ModelPricing struct {
	InputCostPerToken                   float64 `json:"input_cost_per_token"`
	InputCostPerTokenPriority           float64 `json:"input_cost_per_token_priority,omitempty"`
	OutputCostPerToken                  float64 `json:"output_cost_per_token"`
	OutputCostPerTokenPriority          float64 `json:"output_cost_per_token_priority,omitempty"`
	CacheCreationInputTokenCost         float64 `json:"cache_creation_input_token_cost,omitempty"`
	CacheCreationInputTokenCostPriority float64 `json:"cache_creation_input_token_cost_priority,omitempty"`
	CacheCreationInputTokenCostAbove1hr float64 `json:"cache_creation_input_token_cost_above_1hr,omitempty"`
	CacheReadInputTokenCost             float64 `json:"cache_read_input_token_cost,omitempty"`
	CacheReadInputTokenCostPriority     float64 `json:"cache_read_input_token_cost_priority,omitempty"`
	LongContextInputTokenThreshold      int     `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier      float64 `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier     float64 `json:"long_context_output_cost_multiplier,omitempty"`
	SupportsServiceTier                 bool    `json:"supports_service_tier,omitempty"`
	LiteLLMProvider                     string  `json:"litellm_provider,omitempty"`
	Mode                                string  `json:"mode,omitempty"`
	SupportsPromptCaching               bool    `json:"supports_prompt_caching,omitempty"`
}

type rawEntry struct {
	InputCostPerToken                   *float64 `json:"input_cost_per_token"`
	InputCostPerTokenPriority           *float64 `json:"input_cost_per_token_priority"`
	OutputCostPerToken                  *float64 `json:"output_cost_per_token"`
	OutputCostPerTokenPriority          *float64 `json:"output_cost_per_token_priority"`
	CacheCreationInputTokenCost         *float64 `json:"cache_creation_input_token_cost"`
	CacheCreationInputTokenCostPriority *float64 `json:"cache_creation_input_token_cost_priority"`
	CacheCreationInputTokenCostAbove1hr *float64 `json:"cache_creation_input_token_cost_above_1hr"`
	CacheReadInputTokenCost             *float64 `json:"cache_read_input_token_cost"`
	CacheReadInputTokenCostPriority     *float64 `json:"cache_read_input_token_cost_priority"`
	LongContextInputTokenThreshold      *int     `json:"long_context_input_token_threshold"`
	LongContextInputCostMultiplier      *float64 `json:"long_context_input_cost_multiplier"`
	LongContextOutputCostMultiplier     *float64 `json:"long_context_output_cost_multiplier"`
	SupportsServiceTier                 bool     `json:"supports_service_tier"`
	LiteLLMProvider                     string   `json:"litellm_provider"`
	Mode                                string   `json:"mode"`
	SupportsPromptCaching               bool     `json:"supports_prompt_caching"`
}

type Config struct {
	RemoteURL           string `json:"remote_url"`
	HashURL             string `json:"hash_url"`
	DataDir             string `json:"data_dir"`
	FallbackFile        string `json:"fallback_file"`
	UpdateIntervalHours int    `json:"update_interval_hours"`
}

type Service struct {
	cfg         Config
	httpClient  *http.Client
	mu          sync.RWMutex
	pricingData map[string]*ModelPricing
	lastUpdated time.Time
	localHash   string
	stopCh      chan struct{}
}

func NewService(cfg Config) *Service {
	if cfg.RemoteURL == "" {
		cfg.RemoteURL = "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.json"
	}
	if cfg.HashURL == "" {
		cfg.HashURL = "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.sha256"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}
	if cfg.UpdateIntervalHours <= 0 {
		cfg.UpdateIntervalHours = 24
	}

	s := &Service{
		cfg:         cfg,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		pricingData: make(map[string]*ModelPricing),
		stopCh:      make(chan struct{}),
	}
	// Preload fallback prices
	for k, v := range DefaultFallbackPricing {
		s.pricingData[k] = v
	}
	return s
}

func (s *Service) Initialize() error {
	_ = os.MkdirAll(s.cfg.DataDir, 0755)

	pricingFile := filepath.Join(s.cfg.DataDir, "model_pricing.json")
	if data, err := os.ReadFile(pricingFile); err == nil {
		if parsed, err := s.parseData(data); err == nil && len(parsed) > 0 {
			s.mu.Lock()
			for k, v := range parsed {
				s.pricingData[k] = v
			}
			s.lastUpdated = time.Now()
			s.mu.Unlock()
		}
	}

	// Try remote sync in background
	go func() {
		_ = s.SyncWithRemote()
		s.startScheduler()
	}()

	return nil
}

func (s *Service) Stop() {
	close(s.stopCh)
}

func (s *Service) startScheduler() {
	ticker := time.NewTicker(time.Duration(s.cfg.UpdateIntervalHours) * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = s.SyncWithRemote()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) SyncWithRemote() error {
	if s.cfg.RemoteURL == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.cfg.RemoteURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote pricing returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	parsed, err := s.parseData(body)
	if err != nil {
		return err
	}

	// Save to disk cache
	pricingFile := filepath.Join(s.cfg.DataDir, "model_pricing.json")
	_ = os.WriteFile(pricingFile, body, 0644)

	dataHash := sha256.Sum256(body)
	hashStr := hex.EncodeToString(dataHash[:])

	s.mu.Lock()
	for k, v := range parsed {
		s.pricingData[k] = v
	}
	s.lastUpdated = time.Now()
	s.localHash = hashStr
	s.mu.Unlock()

	return nil
}

func (s *Service) parseData(body []byte) (map[string]*ModelPricing, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]*ModelPricing, len(raw))
	for name, item := range raw {
		if name == "sample_spec" {
			continue
		}
		var entry rawEntry
		if err := json.Unmarshal(item, &entry); err != nil {
			continue
		}
		if entry.InputCostPerToken == nil && entry.OutputCostPerToken == nil {
			continue
		}

		p := &ModelPricing{
			LiteLLMProvider:       entry.LiteLLMProvider,
			Mode:                  entry.Mode,
			SupportsPromptCaching: entry.SupportsPromptCaching,
			SupportsServiceTier:   entry.SupportsServiceTier,
		}
		if entry.InputCostPerToken != nil {
			p.InputCostPerToken = *entry.InputCostPerToken
		}
		if entry.InputCostPerTokenPriority != nil {
			p.InputCostPerTokenPriority = *entry.InputCostPerTokenPriority
		}
		if entry.OutputCostPerToken != nil {
			p.OutputCostPerToken = *entry.OutputCostPerToken
		}
		if entry.OutputCostPerTokenPriority != nil {
			p.OutputCostPerTokenPriority = *entry.OutputCostPerTokenPriority
		}
		if entry.CacheCreationInputTokenCost != nil {
			p.CacheCreationInputTokenCost = *entry.CacheCreationInputTokenCost
		}
		if entry.CacheCreationInputTokenCostPriority != nil {
			p.CacheCreationInputTokenCostPriority = *entry.CacheCreationInputTokenCostPriority
		}
		if entry.CacheCreationInputTokenCostAbove1hr != nil {
			p.CacheCreationInputTokenCostAbove1hr = *entry.CacheCreationInputTokenCostAbove1hr
		}
		if entry.CacheReadInputTokenCost != nil {
			p.CacheReadInputTokenCost = *entry.CacheReadInputTokenCost
		}
		if entry.CacheReadInputTokenCostPriority != nil {
			p.CacheReadInputTokenCostPriority = *entry.CacheReadInputTokenCostPriority
		}
		if entry.LongContextInputTokenThreshold != nil {
			p.LongContextInputTokenThreshold = *entry.LongContextInputTokenThreshold
		}
		if entry.LongContextInputCostMultiplier != nil {
			p.LongContextInputCostMultiplier = *entry.LongContextInputCostMultiplier
		}
		if entry.LongContextOutputCostMultiplier != nil {
			p.LongContextOutputCostMultiplier = *entry.LongContextOutputCostMultiplier
		}

		result[name] = p
	}
	return result, nil
}

// GetModelPricing queries model pricing with candidates, base name stripping, Claude family and OpenAI fallback matching.
func (s *Service) GetModelPricing(modelName string) *ModelPricing {
	if modelName == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	modelLower := strings.ToLower(strings.TrimSpace(modelName))
	candidates := buildModelLookupCandidates(modelLower)

	// 1. Exact match across candidates
	for _, cand := range candidates {
		if p, ok := s.pricingData[cand]; ok {
			return p
		}
		normalized := strings.ReplaceAll(cand, "-4-5-", "-4.5-")
		if p, ok := s.pricingData[normalized]; ok {
			return p
		}
	}

	// 2. Base name match (without date suffix)
	baseName := extractBaseName(candidates[0])
	for key, p := range s.pricingData {
		if extractBaseName(strings.ToLower(key)) == baseName {
			return p
		}
	}

	// 3. Claude family ordered slice match
	for _, fam := range claudeFamilies {
		matched := false
		for _, pat := range fam.match {
			if strings.Contains(modelLower, pat) || strings.Contains(modelLower, strings.ReplaceAll(pat, "-", "")) {
				matched = true
				break
			}
		}
		if matched {
			for _, pKey := range fam.pricing {
				for key, p := range s.pricingData {
					if strings.Contains(strings.ToLower(key), pKey) {
						return p
					}
				}
			}
		}
	}

	// 4. OpenAI fallback matching
	if strings.HasPrefix(modelLower, "gpt-") || strings.HasPrefix(modelLower, "o1") || strings.HasPrefix(modelLower, "o3") {
		// Date stripped
		withoutDate := openAIModelDatePattern.ReplaceAllString(modelLower, "")
		if p, ok := s.pricingData[withoutDate]; ok {
			return p
		}
		// Base version
		if matches := openAIModelBasePattern.FindStringSubmatch(modelLower); len(matches) > 1 {
			if p, ok := s.pricingData[matches[1]]; ok {
				return p
			}
		}
		if p, ok := s.pricingData["gpt-4o"]; ok {
			return p
		}
	}

	return nil
}
