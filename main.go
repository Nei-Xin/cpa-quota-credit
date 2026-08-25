package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);

static const cliproxy_host_api* stored_host;

static void store_host_api(const cliproxy_host_api* host) {
	stored_host = host;
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/router-for-me/cpa-quota-credit/pkg/abi"
	"github.com/router-for-me/cpa-quota-credit/pkg/billing"
	"github.com/router-for-me/cpa-quota-credit/pkg/dashboard"
	"github.com/router-for-me/cpa-quota-credit/pkg/pricing"
	"github.com/router-for-me/cpa-quota-credit/pkg/storage"
)

var (
	mu               sync.Mutex
	pricingService   *pricing.Service
	billingCalc      *billing.Calculator
	quotaStore       *storage.Store
	dashboardHandler *dashboard.Handler
)

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	C.store_host_api(host)
	plugin.abi_version = C.uint32_t(abi.ABIVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}

	methodStr := C.GoString(method)
	var reqBody []byte
	if request != nil && requestLen > 0 {
		reqBody = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}

	raw, err := handleMethod(methodStr, reqBody)
	if err != nil {
		writeResponse(response, errorEnvelope("plugin_error", err.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, len C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
	_ = len
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {
	mu.Lock()
	defer mu.Unlock()
	if quotaStore != nil {
		_ = quotaStore.Close()
	}
	if pricingService != nil {
		pricingService.Stop()
	}
}

type pluginConfig struct {
	DBPath      string                   `json:"db_path"`
	Pricing     pricing.Config           `json:"pricing"`
	Multipliers billing.MultiplierConfig `json:"multipliers"`
}

func initPluginComponents(cfgMap map[string]interface{}) error {
	mu.Lock()
	defer mu.Unlock()

	var cfg pluginConfig
	if cfgMap != nil {
		raw, _ := json.Marshal(cfgMap)
		_ = json.Unmarshal(raw, &cfg)
	}

	if cfg.DBPath == "" {
		cfg.DBPath = "./data/quota_credit.db"
	}

	if pricingService == nil {
		pricingService = pricing.NewService(cfg.Pricing)
		_ = pricingService.Initialize()
	}

	billingCalc = billing.NewCalculator(pricingService, cfg.Multipliers)

	if quotaStore == nil {
		store, err := storage.NewStore(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("init quota store: %w", err)
		}
		quotaStore = store
		dashboardHandler = dashboard.NewHandler(quotaStore)
	}

	return nil
}

func handleMethod(method string, reqBody []byte) ([]byte, error) {
	switch method {
	case abi.MethodPluginRegister, abi.MethodPluginReconfigure:
		var regReq struct {
			Config map[string]interface{} `json:"config"`
		}
		if len(reqBody) > 0 {
			_ = json.Unmarshal(reqBody, &regReq)
		}
		_ = initPluginComponents(regReq.Config)

		regResp := abi.PluginRegistration{
			SchemaVersion: abi.SchemaVersion,
			Metadata: abi.PluginMetadata{
				Name:             "cpa-quota-credit",
				Version:          "1.0.5",
				Author:           "router-for-me",
				GitHubRepository: "https://github.com/router-for-me/cpa-quota-credit",
				Logo:             "https://raw.githubusercontent.com/router-for-me/CLIProxyAPI/main/assets/logo/antigravity.svg",
				ConfigFields: []abi.ConfigField{
					{Name: "db_path", Type: "string", Description: "Path to the embedded quota database file"},
					{Name: "multipliers", Type: "object", Description: "Cost multipliers for A $ (Account) and U $ (User)"},
				},
			},
			Capabilities: abi.PluginCapabilities{
				UsagePlugin:   true,
				ManagementAPI: true,
			},
		}
		raw, _ := json.Marshal(regResp)
		return okEnvelope(raw), nil

	case abi.MethodUsageHandle:
		var record abi.UsageRecord
		if err := json.Unmarshal(reqBody, &record); err != nil {
			return nil, err
		}

		if billingCalc != nil && quotaStore != nil {
			model := record.Model
			if model == "" {
				model = record.Alias
			}

			cost := billingCalc.Calculate(billing.CostInput{
				Model:               model,
				Provider:            record.Provider,
				ExecutorType:        record.ExecutorType,
				InputTokens:         record.Detail.InputTokens,
				OutputTokens:        record.Detail.OutputTokens,
				ReasoningTokens:     record.Detail.ReasoningTokens,
				CacheReadTokens:     record.Detail.CacheReadTokens,
				CacheCreationTokens: record.Detail.CacheCreationTokens,
				TotalTokens:         record.Detail.TotalTokens,
				ServiceTier:         record.ServiceTier,
				APIKey:              record.APIKey,
				AuthID:              record.AuthID,
			})

			reqTimestamp := record.RequestedAt
			if reqTimestamp.IsZero() {
				reqTimestamp = time.Now()
			}

			_ = quotaStore.UpdateCodexQuota(record.AuthID, record.ResponseHeaders, time.Now())
			_ = quotaStore.RecordUsage(storage.Record{
				APIKey:    record.APIKey,
				AuthID:    record.AuthID,
				Provider:  record.Provider,
				Model:     model,
				Cost:      cost,
				LatencyMs: record.Latency.Milliseconds(),
				Timestamp: reqTimestamp,
				Failed:    record.Failed,
			})
		}
		return okEnvelope([]byte("{}")), nil

	case abi.MethodManagementRegister:
		routes := abi.ManagementRegistrationResponse{
			Resources: []abi.ResourceRoute{
				{
					Path:        "/dashboard",
					Menu:        "额度与计费",
					Description: "CLIProxyAPI 额度与计费控制看板",
				},
				{
					Path:        "/stats",
					Menu:        "",
					Description: "Public JSON stats for dashboard resource",
				},
			},
			Routes: []abi.ManagementRoute{
				{
					Method:      "GET",
					Path:        "/stats",
					Description: "获取全量额度、消耗与 API Key 统计 JSON",
				},
			},
		}
		raw, _ := json.Marshal(routes)
		return okEnvelope(raw), nil

	case abi.MethodManagementHandle:
		if dashboardHandler == nil {
			return nil, fmt.Errorf("dashboard handler not initialized")
		}
		var mgmtReq abi.ManagementRequest
		if err := json.Unmarshal(reqBody, &mgmtReq); err != nil {
			return nil, err
		}
		resp := dashboardHandler.HandleRequest(mgmtReq)
		raw, _ := json.Marshal(resp)
		return okEnvelope(raw), nil

	case abi.MethodPluginShutdown:
		cliproxyPluginShutdown()
		return okEnvelope([]byte("{}")), nil

	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func okEnvelope(result []byte) []byte {
	raw, _ := json.Marshal(abi.Envelope{OK: true, Result: result})
	return raw
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(abi.Envelope{OK: false, Error: &abi.EnvelopeError{Code: code, Message: message}})
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
