package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

func TestReadinessFailsWhenCheckTimesOut(t *testing.T) {
	registry := health.NewRegistry(20 * time.Millisecond)
	registry.Register("postgres", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	router := api.NewRouter(api.Dependencies{Readiness: registry})

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	started := time.Now()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("readiness took too long: %s", elapsed)
	}
}

func TestReadinessFailsWhenCheckFails(t *testing.T) {
	registry := health.NewRegistry(time.Second)
	registry.Register("redis", func(context.Context) error {
		return errors.New("connection refused")
	})
	router := api.NewRouter(api.Dependencies{Readiness: registry})

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "unavailable" {
		t.Fatalf("expected status unavailable, got %q", body.Status)
	}
	if body.Checks["redis"] != "error" {
		t.Fatalf("expected redis check error, got %q", body.Checks["redis"])
	}
}

func TestReadinessReturnsOKWithoutChecks(t *testing.T) {
	router := api.NewRouter(api.Dependencies{Readiness: health.NewRegistry(time.Second)})

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestLivenessReturnsOK(t *testing.T) {
	router := api.NewRouter(api.Dependencies{Readiness: health.NewRegistry(time.Second)})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}
