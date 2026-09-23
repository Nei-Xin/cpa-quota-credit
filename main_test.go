package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/router-for-me/cpa-quota-credit/pkg/abi"
	"github.com/router-for-me/cpa-quota-credit/pkg/billing"
)

func registerTestPlugin(t *testing.T) (string, string, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	dbPath := filepath.Join(t.TempDir(), "usage.db")
	dataDir := t.TempDir()
	t.Cleanup(cliproxyPluginShutdown)
	configureTestPlugin(t, abi.MethodPluginRegister, dbPath, dataDir, server.URL, "")
	return dbPath, dataDir, server.URL
}

func configureTestPlugin(t *testing.T, method, dbPath, dataDir, remoteURL, overrides string) {
	t.Helper()
	configYAML := fmt.Sprintf("db_path: %q\npricing:\n  data_dir: %q\n  remote_url: %q\n%s", dbPath, dataDir, remoteURL, overrides)
	request, err := json.Marshal(struct {
		ConfigYAML []byte `json:"config_yaml"`
	}{ConfigYAML: []byte(configYAML)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handleMethod(method, request); err != nil {
		t.Fatal(err)
	}
}

func TestUsageUsesResponseServiceTier(t *testing.T) {
	registerTestPlugin(t)
	for _, tt := range []struct {
		name, requestTier, responseTier string
		want                            float64
	}{
		{"response priority", "default", "priority", 0.024},
		{"response default", "priority", "default", 0.012},
		{"legacy request fallback", "priority", "", 0.024},
	} {
		t.Run(tt.name, func(t *testing.T) {
			record, err := json.Marshal(abi.UsageRecord{
				Model: "gpt-6-sol", Provider: "openai",
				ServiceTier: tt.requestTier, ResponseServiceTier: tt.responseTier,
				Detail: abi.UsageDetail{InputTokens: 1000, OutputTokens: 1000},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := handleMethod(abi.MethodUsageHandle, record); err != nil {
				t.Fatal(err)
			}
			stats, err := quotaStore.GetFullStats(1)
			if err != nil {
				t.Fatal(err)
			}
			if got := stats.RecentLogs[0].Cost.BaseTotalCost; got != tt.want {
				t.Fatalf("cost = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPricingReconfigureReplacesOverrides(t *testing.T) {
	dbPath, dataDir, remoteURL := registerTestPlugin(t)
	model := "gpt-6-sol"
	if got := pricingService.GetModelPricing(model).InputCostPerToken; got != 2e-6 {
		t.Fatalf("initial rate = %v", got)
	}
	configureTestPlugin(t, abi.MethodPluginReconfigure, dbPath, dataDir, remoteURL,
		"  overrides:\n    gpt-6-sol:\n      input_cost_per_token: 0.000004\n      output_cost_per_token: 0.00001\n")
	if got := pricingService.GetModelPricing(model).InputCostPerToken; got != 4e-6 {
		t.Fatalf("reconfigured rate = %v", got)
	}
	if got := billingCalc.Calculate(billing.CostInput{Model: model, Provider: "openai", InputTokens: 1000}).BaseTotalCost; got != 0.004 {
		t.Fatalf("reconfigured cost = %v", got)
	}
	if err := initPluginComponents(map[string]interface{}{
		"pricing": map[string]interface{}{
			"overrides": map[string]interface{}{
				model: map[string]interface{}{"input_cost_per_token": "invalid"},
			},
		},
	}); err == nil {
		t.Fatal("expected invalid reconfiguration to fail")
	}
	if got := pricingService.GetModelPricing(model).InputCostPerToken; got != 4e-6 {
		t.Fatalf("invalid reconfiguration changed rate to %v", got)
	}
	configureTestPlugin(t, abi.MethodPluginReconfigure, dbPath, dataDir, remoteURL, "")
	if got := pricingService.GetModelPricing(model).InputCostPerToken; got != 2e-6 {
		t.Fatalf("removed override rate = %v", got)
	}
}

func TestUsageWriteFailureIsReturned(t *testing.T) {
	registerTestPlugin(t)
	if err := quotaStore.Close(); err != nil {
		t.Fatal(err)
	}
	record, err := json.Marshal(abi.UsageRecord{
		Model: "gpt-6-sol", Provider: "openai",
		Detail: abi.UsageDetail{InputTokens: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handleMethod(abi.MethodUsageHandle, record); err == nil || !strings.Contains(err.Error(), "record usage") {
		t.Fatalf("expected usage write failure, got %v", err)
	}
}
