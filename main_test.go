//go:build cgo

package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/router-for-me/cpa-quota-credit/pkg/abi"
)

func TestHostRegistrationContract(t *testing.T) {
	dir := t.TempDir()
	config := fmt.Sprintf("db_path: %q\npricing:\n  data_dir: %q\n  remote_url: http://127.0.0.1:1/pricing\n  overrides:\n    test-model:\n      input_cost_per_token: 0.000004\n", filepath.Join(dir, "usage.db"), dir)
	body, _ := json.Marshal(map[string]any{"config_yaml": []byte(config)})
	if _, err := handleMethod(abi.MethodPluginRegister, body); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cliproxyPluginShutdown() })
	if p := pricingService.GetModelPricing("test-model"); p == nil || p.InputCostPerToken != 0.000004 {
		t.Fatalf("host YAML config not applied: %+v", p)
	}
	raw, err := handleMethod(abi.MethodManagementRegister, nil)
	if err != nil {
		t.Fatal(err)
	}
	var envelope abi.Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	var routes abi.ManagementRegistrationResponse
	if err := json.Unmarshal(envelope.Result, &routes); err != nil {
		t.Fatal(err)
	}
	if len(routes.Routes) != 1 || routes.Routes[0].Path != "/plugins/cpa-quota-credit/stats" || len(routes.Resources) != 1 || routes.Resources[0].Path != "/dashboard" {
		t.Fatalf("unexpected routes: %+v", routes)
	}
}
