package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

type fakeAdminDLP struct{}

func (fakeAdminDLP) Rules(context.Context, string) ([]admin.DLPRule, error) {
	return []admin.DLPRule{{ID: "rule-1", Name: "CPF", Detector: "cpf", Action: "redact", Severity: "high", Enabled: true}}, nil
}

func (fakeAdminDLP) CreateRule(context.Context, string, admin.DLPInput) (admin.DLPRule, error) {
	return admin.DLPRule{ID: "rule-2", Name: "AWS", Detector: "aws", Action: "block", Severity: "critical", Enabled: true}, nil
}

func (fakeAdminDLP) SetRuleEnabled(context.Context, string, string, bool) (admin.DLPRule, error) {
	return admin.DLPRule{ID: "rule-1", Name: "CPF", Detector: "cpf", Action: "redact", Severity: "high", Enabled: false}, nil
}

func (fakeAdminDLP) Events(context.Context, string, int) ([]admin.DLPEvent, error) {
	return []admin.DLPEvent{{ID: "event-1", Detector: "cpf", Action: "redact", Severity: "high"}}, nil
}

func TestAdminDLPEndpoints(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		AdminDLP:  fakeAdminDLP{},
	})
	cases := []struct {
		method   string
		path     string
		body     string
		status   int
		expected string
	}{
		{http.MethodGet, "/admin/v1/dlp/rules", "", http.StatusOK, `"name":"CPF"`},
		{http.MethodPost, "/admin/v1/dlp/rules", `{"name":"AWS","detector":"aws","action":"block","severity":"critical"}`, http.StatusCreated, `"action":"block"`},
		{http.MethodPatch, "/admin/v1/dlp/rules/rule-1", `{"enabled":false}`, http.StatusOK, `"enabled":false`},
		{http.MethodGet, "/admin/v1/dlp/events", "", http.StatusOK, `"detector":"cpf"`},
	}
	for _, item := range cases {
		request := httptest.NewRequest(item.method, item.path, strings.NewReader(item.body))
		request.Header.Set("Authorization", "Bearer "+key)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != item.status {
			t.Fatalf("%s %s: unexpected status %d %s", item.method, item.path, response.Code, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), item.expected) {
			t.Fatalf("%s %s: missing %s in %s", item.method, item.path, item.expected, response.Body.String())
		}
	}
}
