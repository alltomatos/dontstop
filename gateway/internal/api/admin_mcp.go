package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminMCPService exposes MCP settings administration.
type AdminMCPService interface {
	Status(ctx context.Context, tenantID string) (admin.MCPStatus, error)
	UpdateSettings(ctx context.Context, tenantID string, enabled *bool, transport *string) (admin.MCPStatus, error)
	UpdateTool(ctx context.Context, tenantID, toolID string, enabled bool) (admin.MCPStatus, error)
}

func adminMCPStatus(service AdminMCPService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		status, err := service.Status(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load MCP status")
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

func adminMCPUpdateSettings(service AdminMCPService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			Enabled   *bool   `json:"enabled"`
			Transport *string `json:"transport"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		status, err := service.UpdateSettings(request.Context(), principal.TenantID, input.Enabled, input.Transport)
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid MCP settings")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to update MCP settings")
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

func adminMCPUpdateTool(service AdminMCPService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.Enabled == nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "enabled is required")
			return
		}
		status, err := service.UpdateTool(request.Context(), principal.TenantID, request.PathValue("id"), *input.Enabled)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "MCP tool not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to update MCP tool")
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
