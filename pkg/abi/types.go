package abi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

const (
	ABIVersion    uint32 = 1
	SchemaVersion uint32 = 1

	MethodPluginRegister    = "plugin.register"
	MethodPluginReconfigure = "plugin.reconfigure"
	MethodPluginShutdown    = "plugin.shutdown"

	MethodUsageHandle        = "usage.handle"
	MethodManagementRegister = "management.register"
	MethodManagementHandle   = "management.handle"
	MethodHostAuthList       = "host.auth.list"
)

type Envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *EnvelopeError  `json:"error,omitempty"`
}

type EnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PluginRegistration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      PluginMetadata         `json:"metadata"`
	Capabilities  PluginCapabilities     `json:"capabilities"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

type PluginMetadata struct {
	Name             string        `json:"Name"`
	Version          string        `json:"Version"`
	Author           string        `json:"Author"`
	GitHubRepository string        `json:"GitHubRepository"`
	Logo             string        `json:"Logo"`
	ConfigFields     []ConfigField `json:"ConfigFields"`
}

type ConfigField struct {
	Name        string   `json:"Name"`
	Type        string   `json:"Type"`
	EnumValues  []string `json:"EnumValues,omitempty"`
	Description string   `json:"Description"`
}

type PluginCapabilities struct {
	UsagePlugin   bool `json:"usage_plugin"`
	ManagementAPI bool `json:"management_api"`
}

// UsageRecord describes request usage passed to usage.handle
type UsageRecord struct {
	Provider        string          `json:"Provider"`
	ExecutorType    string          `json:"ExecutorType"`
	Model           string          `json:"Model"`
	Alias           string          `json:"Alias"`
	APIKey          string          `json:"APIKey"`
	AuthID          string          `json:"AuthID"`
	AuthIndex       string          `json:"AuthIndex"`
	AuthType        string          `json:"AuthType"`
	Source          string          `json:"Source"`
	ReasoningEffort string          `json:"ReasoningEffort"`
	ServiceTier     string          `json:"ServiceTier"`
	Generate        bool            `json:"Generate"`
	RequestedAt     time.Time       `json:"RequestedAt"`
	Latency         time.Duration   `json:"Latency"`
	TTFT            time.Duration   `json:"TTFT"`
	Failed          bool            `json:"Failed"`
	Failure         UsageFailure    `json:"Failure"`
	Detail          UsageDetail     `json:"Detail"`
	ResponseHeaders http.Header     `json:"ResponseHeaders"`
}

type UsageFailure struct {
	StatusCode int    `json:"StatusCode"`
	Body       string `json:"Body"`
}

type UsageDetail struct {
	InputTokens         int64 `json:"InputTokens"`
	OutputTokens        int64 `json:"OutputTokens"`
	ReasoningTokens     int64 `json:"ReasoningTokens"`
	CachedTokens        int64 `json:"CachedTokens"`
	CacheReadTokens     int64 `json:"CacheReadTokens"`
	CacheCreationTokens int64 `json:"CacheCreationTokens"`
	TotalTokens         int64 `json:"TotalTokens"`
}

type ManagementRegistrationRequest struct {
	Plugin           PluginMetadata `json:"Plugin"`
	BasePath         string         `json:"BasePath"`
	ResourceBasePath string         `json:"ResourceBasePath"`
}

type ManagementRegistrationResponse struct {
	Routes    []ManagementRoute `json:"routes,omitempty"`
	Resources []ResourceRoute   `json:"resources,omitempty"`
}

type ManagementRoute struct {
	Method      string `json:"Method"`
	Path        string `json:"Path"`
	Menu        string `json:"Menu,omitempty"`
	Description string `json:"Description,omitempty"`
}

type ResourceRoute struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type ManagementRequest struct {
	Method  string      `json:"Method"`
	Path    string      `json:"Path"`
	Headers http.Header `json:"Headers"`
	Query   url.Values  `json:"Query"`
	Body    []byte      `json:"Body"`
}

type ManagementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}
