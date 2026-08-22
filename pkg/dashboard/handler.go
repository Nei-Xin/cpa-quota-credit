package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/router-for-me/cpa-quota-credit/pkg/abi"
	"github.com/router-for-me/cpa-quota-credit/pkg/storage"
)

type Handler struct {
	store *storage.Store
}

func NewHandler(store *storage.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HandleRequest(req abi.ManagementRequest) abi.ManagementResponse {
	path := strings.TrimSpace(req.Path)

	// Common security and iframe compatibility headers
	commonHeaders := http.Header{
		"Access-Control-Allow-Origin": []string{"*"},
		"Access-Control-Allow-Methods": []string{"GET, POST, OPTIONS"},
		"Access-Control-Allow-Headers": []string{"*"},
		"Content-Security-Policy":     []string{"frame-ancestors 'self' *;"},
		"X-Frame-Options":             []string{"ALLOWALL"},
	}

	// 1. Stats JSON API (Checked first or when query parameter format=json)
	if strings.HasSuffix(path, "/stats") || strings.HasSuffix(path, "/api/stats") || req.Query.Get("format") == "json" {
		stats, err := h.store.GetFullStats(50)
		if err != nil {
			commonHeaders.Set("Content-Type", "application/json; charset=utf-8")
			return abi.ManagementResponse{
				StatusCode: http.StatusInternalServerError,
				Headers:    commonHeaders,
				Body:       []byte(`{"error":"failed to get stats"}`),
			}
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
	if path == "" || path == "/" || strings.HasSuffix(path, "/dashboard") || strings.HasSuffix(path, "/status") {
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
