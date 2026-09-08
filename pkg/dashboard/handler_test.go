package dashboard

import (
	"bytes"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/router-for-me/cpa-quota-credit/pkg/abi"
	"github.com/router-for-me/cpa-quota-credit/pkg/billing"
	"github.com/router-for-me/cpa-quota-credit/pkg/storage"
)

func TestPublicRoutesNeverReadStats(t *testing.T) {
	// A nil store also ensures public requests never reach database access.
	h := NewHandler(nil)
	for _, path := range []string{
		"/v0/resource/plugins/cpa-quota-credit/dashboard",
		"/v0/resource/plugins/cpa-quota-credit/stats",
		"/v0/resource/plugins/cpa-quota-credit/api/stats",
		"/stats", "/api/stats", "/unknown",
	} {
		for _, query := range []url.Values{nil, {"format": {"json"}}} {
			resp := h.HandleRequest(abi.ManagementRequest{Method: http.MethodGet, Path: path, Query: query})
			if path == "/v0/resource/plugins/cpa-quota-credit/dashboard" {
				if resp.StatusCode != http.StatusOK || string(resp.Body) != HTMLDashboardTemplate {
					t.Fatalf("%s must return only static HTML", path)
				}
			} else if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("%s returned %d, want 404", path, resp.StatusCode)
			}
			if resp.Headers.Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("unexpected CORS override")
			}
		}
	}
}

func TestManagementStats(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "stats.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.RecordUsage(storage.Record{
		APIKey: "sk-test-private", AuthID: "private-auth", Model: "test-model",
		Cost: &billing.CostBreakdown{TotalTokens: 12}, Timestamp: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(store)
	req := abi.ManagementRequest{Method: http.MethodGet, Path: "/v0/management/plugins/cpa-quota-credit/stats"}
	resp := h.HandleRequest(req)
	if resp.StatusCode != http.StatusOK || !bytes.Contains(resp.Body, []byte("sk-test-private")) {
		t.Fatalf("management stats did not return recorded data: %s", resp.Body)
	}
	if resp.Headers.Get("Cache-Control") != "no-store" {
		t.Fatal("stats must not be cached")
	}
	filtered := h.HandleRequestWithAuthFilter(req, map[string]struct{}{})
	if !bytes.Contains(filtered.Body, []byte(`"auths":[]`)) {
		t.Fatalf("auth filter not applied: %s", filtered.Body)
	}
	for _, method := range []string{http.MethodPost, http.MethodOptions, ""} {
		req.Method = method
		if resp := h.HandleRequest(req); resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s must not serve stats", method)
		}
	}
}

func TestAuthIDIsActive(t *testing.T) {
	active := map[string]struct{}{
		"codex-current.json": {},
		"codex-index-1":      {},
	}

	tests := []struct {
		name   string
		authID string
		want   bool
	}{
		{name: "exact name", authID: "codex-current.json", want: true},
		{name: "name without extension", authID: "codex-current", want: true},
		{name: "path name", authID: `auths\\codex-current.json`, want: true},
		{name: "auth index", authID: "codex-index-1", want: true},
		{name: "deleted account", authID: "codex-deleted.json", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authIDIsActive(tt.authID, active); got != tt.want {
				t.Fatalf("authIDIsActive(%q) = %v, want %v", tt.authID, got, tt.want)
			}
		})
	}
}
