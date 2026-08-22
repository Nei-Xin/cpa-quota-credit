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

	// 1. Stats JSON API (Checked first or when query parameter format=json)
	if strings.HasSuffix(path, "/stats") || strings.HasSuffix(path, "/api/stats") || req.Query.Get("format") == "json" {
		stats, err := h.store.GetFullStats(50)
		if err != nil {
			return abi.ManagementResponse{
				StatusCode: http.StatusInternalServerError,
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       []byte(`{"error":"failed to get stats"}`),
			}
		}
		raw, _ := json.Marshal(stats)
		return abi.ManagementResponse{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type":                []string{"application/json; charset=utf-8"},
				"Access-Control-Allow-Origin": []string{"*"},
			},
			Body: raw,
		}
	}

	// 2. Dashboard UI HTML Resource
	if path == "" || path == "/" || strings.HasSuffix(path, "/dashboard") || strings.HasSuffix(path, "/status") {
		return abi.ManagementResponse{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Content-Type": []string{"text/html; charset=utf-8"},
			},
			Body: []byte(HTMLDashboardTemplate),
		}
	}

	return abi.ManagementResponse{
		StatusCode: http.StatusNotFound,
		Headers:    http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("404 page not found"),
	}
}
