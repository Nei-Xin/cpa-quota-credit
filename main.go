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

static int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
	if (stored_host == NULL || stored_host->call == NULL) {
		return 1;
	}
	return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}

static void free_host_buffer(void* ptr, size_t len) {
	if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) {
		stored_host->free_buffer(ptr, len);
	}
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
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
				Version:          "1.0.7",
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
		if isStatsRequest(mgmtReq) {
			if activeAuthIDs, ok := listActiveAuthIDs(); ok {
				resp := dashboardHandler.HandleRequestWithAuthFilter(mgmtReq, activeAuthIDs)
				raw, _ := json.Marshal(resp)
				return okEnvelope(raw), nil
			}
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

type hostAuthFileEntry struct {
	ID        string `json:"id,omitempty"`
	AuthIndex string `json:"auth_index,omitempty"`
	Name      string `json:"name,omitempty"`
}

type hostAuthListResponse struct {
	Files []hostAuthFileEntry `json:"files"`
}

func isStatsRequest(req abi.ManagementRequest) bool {
	path := strings.TrimSpace(req.Path)
	return strings.HasSuffix(path, "/stats") || strings.HasSuffix(path, "/api/stats") || req.Query.Get("format") == "json"
}

func listActiveAuthIDs() (map[string]struct{}, bool) {
	raw, errCall := callHostCallback(abi.MethodHostAuthList, []byte(`{}`))
	if errCall != nil {
		return nil, false
	}
	var response hostAuthListResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, false
	}
	ids := make(map[string]struct{}, len(response.Files)*3)
	for _, file := range response.Files {
		addAuthIDVariants(ids, file.ID)
		addAuthIDVariants(ids, file.AuthIndex)
		addAuthIDVariants(ids, file.Name)
	}
	return ids, true
}

func addAuthIDVariants(ids map[string]struct{}, raw string) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return
	}
	ids[value] = struct{}{}
	base := filepath.Base(strings.ReplaceAll(value, "\\", "/"))
	ids[base] = struct{}{}
	ids[strings.TrimSuffix(base, ".json")] = struct{}{}
}

func callHostCallback(method string, payload []byte) (json.RawMessage, error) {
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	var requestPtr *C.uint8_t
	if len(payload) > 0 {
		cPayload := C.CBytes(payload)
		if cPayload == nil {
			return nil, fmt.Errorf("allocate host callback payload")
		}
		defer C.free(cPayload)
		requestPtr = (*C.uint8_t)(cPayload)
	}
	var response C.cliproxy_buffer
	callCode := C.call_host_api(cMethod, requestPtr, C.size_t(len(payload)), &response)
	var rawResponse []byte
	if response.ptr != nil && response.len > 0 {
		rawResponse = C.GoBytes(response.ptr, C.int(response.len))
	}
	if response.ptr != nil {
		C.free_host_buffer(response.ptr, response.len)
	}
	if len(rawResponse) == 0 {
		return nil, fmt.Errorf("host callback returned no response, code=%d", int(callCode))
	}
	var envelope abi.Envelope
	if err := json.Unmarshal(rawResponse, &envelope); err != nil {
		return nil, fmt.Errorf("decode host callback response: %w", err)
	}
	if !envelope.OK {
		if envelope.Error != nil {
			return nil, fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return nil, fmt.Errorf("host callback failed")
	}
	if callCode != 0 {
		return nil, fmt.Errorf("host callback returned code=%d", int(callCode))
	}
	return envelope.Result, nil
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
