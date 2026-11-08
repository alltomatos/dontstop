package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminDLPService exposes DLP rules and events.
type AdminDLPService interface {
	Rules(ctx context.Context, tenantID string) ([]admin.DLPRule, error)
	CreateRule(ctx context.Context, tenantID string, input admin.DLPInput) (admin.DLPRule, error)
	SetRuleEnabled(ctx context.Context, tenantID, ruleID string, enabled bool) (admin.DLPRule, error)
	Events(ctx context.Context, tenantID string, limit int) ([]admin.DLPEvent, error)
}

func adminDLPRules(service AdminDLPService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		rules, err := service.Rules(request.Context(), principal.TenantID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load DLP rules")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": rules})
	}
}

func adminDLPCreateRule(service AdminDLPService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.DLPInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		rule, err := service.CreateRule(request.Context(), principal.TenantID, input)
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid DLP rule")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to create DLP rule")
			return
		}
		writeJSON(w, http.StatusCreated, rule)
	}
}

func adminDLPSetRule(service AdminDLPService) http.HandlerFunc {
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
		rule, err := service.SetRuleEnabled(request.Context(), principal.TenantID, request.PathValue("id"), *input.Enabled)
		if errors.Is(err, admin.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "not_found", "DLP rule not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to update DLP rule")
			return
		}
		writeJSON(w, http.StatusOK, rule)
	}
}

func adminDLPEvents(service AdminDLPService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		principal, ok := requireManage(w, request)
		if !ok {
			return
		}
		limit, err := parseLimit(request.URL.Query().Get("limit"), 50)
		if err != nil {
			limit = 50
		}
		events, err := service.Events(request.Context(), principal.TenantID, limit)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to load DLP events")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": events})
	}
}

func parseLimit(value string, fallback int) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback, err
	}
	return parsed, nil
}
