package dashboard

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/router-for-me/cpa-quota-credit/pkg/abi"
	"github.com/router-for-me/cpa-quota-credit/pkg/storage"
)

type Handler struct {
	store *storage.Store
}

// IsStatsRequest only accepts the full management path forwarded by CPA after
// its management authentication middleware has succeeded.
func IsStatsRequest(req abi.ManagementRequest) bool {
	return req.Method == http.MethodGet && req.Path == "/v0/management/plugins/cpa-quota-credit/stats"
}

func NewHandler(store *storage.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HandleRequest(req abi.ManagementRequest) abi.ManagementResponse {
	return h.handleRequest(req, nil)
}

// HandleRequestWithAuthFilter hides historical auth rows that are no longer
// present in the host's current auth list. A non-nil empty map hides all rows.
func (h *Handler) HandleRequestWithAuthFilter(req abi.ManagementRequest, activeAuthIDs map[string]struct{}) abi.ManagementResponse {
	return h.handleRequest(req, activeAuthIDs)
}

func (h *Handler) handleRequest(req abi.ManagementRequest, activeAuthIDs map[string]struct{}) abi.ManagementResponse {
	// Allow the static dashboard to be embedded by the same-origin management UI.
	commonHeaders := http.Header{
		"Content-Security-Policy": []string{"frame-ancestors 'self';"},
		"X-Frame-Options":         []string{"SAMEORIGIN"},
	}

	if IsStatsRequest(req) {
		commonHeaders.Set("Cache-Control", "no-store")
		stats, err := h.store.GetFullStats(50)
		if err != nil {
			commonHeaders.Set("Content-Type", "application/json; charset=utf-8")
			return abi.ManagementResponse{
				StatusCode: http.StatusInternalServerError,
				Headers:    commonHeaders,
				Body:       []byte(`{"error":"failed to get stats"}`),
			}
		}
		if activeAuthIDs != nil {
			visible := stats.Auths[:0]
			for _, auth := range stats.Auths {
				if authIDIsActive(auth.AuthID, activeAuthIDs) {
					visible = append(visible, auth)
				}
			}
			stats.Auths = visible
		}
		raw, _ := json.Marshal(stats)
		commonHeaders.Set("Content-Type", "application/json; charset=utf-8")
		return abi.ManagementResponse{
			StatusCode: http.StatusOK,
			Headers:    commonHeaders,
			Body:       raw,
		}
	}

	// 2. Dashboard UI HTML Resource (Supports iframe embedding inside CPAMC)
	if req.Method == http.MethodGet && req.Path == "/v0/resource/plugins/cpa-quota-credit/dashboard" {
		commonHeaders.Set("Content-Type", "text/html; charset=utf-8")
		return abi.ManagementResponse{
			StatusCode: http.StatusOK,
			Headers:    commonHeaders,
			Body:       []byte(HTMLDashboardTemplate),
		}
	}

	return abi.ManagementResponse{
		StatusCode: http.StatusNotFound,
		Headers:    http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("404 page not found"),
	}
}

func authIDIsActive(authID string, activeAuthIDs map[string]struct{}) bool {
	value := strings.ToLower(strings.TrimSpace(authID))
	if value == "" {
		return false
	}
	if _, ok := activeAuthIDs[value]; ok {
		return true
	}
	if _, ok := activeAuthIDs[value+".json"]; ok {
		return true
	}
	base := filepath.Base(strings.ReplaceAll(value, "\\", "/"))
	if _, ok := activeAuthIDs[base]; ok {
		return true
	}
	_, ok := activeAuthIDs[strings.TrimSuffix(base, ".json")]
	return ok
}
