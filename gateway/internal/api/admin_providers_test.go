package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/api"
	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

type fakeAdminProviders struct {
	providers       []admin.Provider
	account         admin.Account
	configured      []admin.ConfiguredModel
	syncResult      admin.SyncModelsResult
	modelResult     admin.ModelTestResult
	modelRun        admin.ModelTestRun
	bulkResults     []admin.ModelUpdateResult
	providerDeleted bool
	err             error
}

func (f fakeAdminProviders) List(context.Context, string) ([]admin.Provider, error) {
	return f.providers, f.err
}

func (f fakeAdminProviders) Get(context.Context, string, string) (admin.Provider, error) {
	if f.err != nil {
		return admin.Provider{}, f.err
	}
	return f.providers[0], nil
}

func (f fakeAdminProviders) Create(context.Context, string, admin.CreateProviderInput) (admin.Provider, error) {
	if f.err != nil {
		return admin.Provider{}, f.err
	}
	return f.providers[0], nil
}

func (f fakeAdminProviders) AddAccount(context.Context, string, string, admin.AccountInput) (admin.Account, error) {
	return f.account, f.err
}

func (f fakeAdminProviders) DeleteAccount(context.Context, string, string, string) (bool, error) {
	return f.providerDeleted, f.err
}

func (f fakeAdminProviders) SetProxyMode(_ context.Context, _, _, _, raw string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return raw, nil
}

func (f fakeAdminProviders) RenameAccount(context.Context, string, string, string, string) error {
	return f.err
}

func (f fakeAdminProviders) AccountSummary(context.Context, string, string, string) (admin.AccountSummaryResult, error) {
	return admin.AccountSummaryResult{}, f.err
}

func (f fakeAdminProviders) TestAccount(_ context.Context, tenantID, _, accountID string) (admin.AccountTestResult, error) {
	if f.err != nil {
		return admin.AccountTestResult{}, f.err
	}
	return admin.AccountTestResult{AccountID: accountID, AccountName: "prod", OK: true, Status: 200}, nil
}

func (f fakeAdminProviders) Test(context.Context, string, string) (admin.ProviderTestResult, error) {
	return admin.ProviderTestResult{OK: true}, f.err
}

func (f fakeAdminProviders) RemoteModels(context.Context, string, string) ([]admin.RemoteModel, error) {
	return []admin.RemoteModel{{ID: "gpt-4.1"}}, f.err
}

func (f fakeAdminProviders) ImportModels(context.Context, string, string, []string) (int, error) {
	return 1, f.err
}

func (f fakeAdminProviders) ConfiguredModels(context.Context, string, string) ([]admin.ConfiguredModel, error) {
	return f.configured, f.err
}

func (f fakeAdminProviders) SyncModels(context.Context, string, string) (admin.SyncModelsResult, error) {
	return f.syncResult, f.err
}

func (f fakeAdminProviders) SetModelActive(context.Context, string, string, string, bool) error {
	return f.err
}

func (f fakeAdminProviders) SetModelsActive(context.Context, string, string, []string, bool) ([]admin.ModelUpdateResult, error) {
	return f.bulkResults, f.err
}

func (f fakeAdminProviders) TestModel(context.Context, string, string, string, string) (admin.ModelTestResult, error) {
	return f.modelResult, f.err
}

func (f fakeAdminProviders) StartModelTestRun(context.Context, string, string, []string) (admin.ModelTestRun, error) {
	return f.modelRun, f.err
}

func (f fakeAdminProviders) GetModelTestRun(context.Context, string, string, string) (admin.ModelTestRun, error) {
	return f.modelRun, f.err
}

func (f fakeAdminProviders) LatestModelTestRun(context.Context, string, string) (admin.ModelTestRun, error) {
	return f.modelRun, f.err
}

func TestAdminProvidersRequireManageScope(t *testing.T) {
	verifier, key := testVerifier(t, []string{"models:read"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     fakeAdminProviders{},
	})
	request := httptest.NewRequest(http.MethodGet, "/admin/v1/providers", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestAdminProvidersListAndAddAccount(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminProviders{
		providers: []admin.Provider{{ID: "provider-1", Name: "OpenAI", Kind: "openai", Origin: "native", Enabled: true, AccountsCount: 2}},
		account:   admin.Account{ID: "account-1", Name: "prod", Status: "active", Priority: 100, Weight: 1},
	}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     service,
	})

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/providers", nil)
	listRequest.Header.Set("Authorization", "Bearer "+key)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResponse.Code)
	}
	var listBody struct {
		Data []admin.Provider `json:"data"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].Origin != "native" {
		t.Fatalf("unexpected list: %+v", listBody.Data)
	}

	addRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/accounts", strings.NewReader(`{"name":"prod","credential":"secret","base_url":"https://api.openai.com"}`))
	addRequest.Header.Set("Authorization", "Bearer "+key)
	addRequest.Header.Set("Content-Type", "application/json")
	addResponse := httptest.NewRecorder()
	router.ServeHTTP(addResponse, addRequest)
	if addResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", addResponse.Code, addResponse.Body.String())
	}

	testRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/test", nil)
	testRequest.Header.Set("Authorization", "Bearer "+key)
	testResponse := httptest.NewRecorder()
	router.ServeHTTP(testResponse, testRequest)
	if testResponse.Code != http.StatusOK {
		t.Fatalf("expected test 200, got %d: %s", testResponse.Code, testResponse.Body.String())
	}

	modelsRequest := httptest.NewRequest(http.MethodGet, "/admin/v1/providers/provider-1/models", nil)
	modelsRequest.Header.Set("Authorization", "Bearer "+key)
	modelsResponse := httptest.NewRecorder()
	router.ServeHTTP(modelsResponse, modelsRequest)
	if modelsResponse.Code != http.StatusOK {
		t.Fatalf("expected models 200, got %d: %s", modelsResponse.Code, modelsResponse.Body.String())
	}

	importRequest := httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/models/import", strings.NewReader(`{"models":["gpt-4.1"]}`))
	importRequest.Header.Set("Authorization", "Bearer "+key)
	importRequest.Header.Set("Content-Type", "application/json")
	importResponse := httptest.NewRecorder()
	router.ServeHTTP(importResponse, importRequest)
	if importResponse.Code != http.StatusOK {
		t.Fatalf("expected import 200, got %d: %s", importResponse.Code, importResponse.Body.String())
	}
}

func TestAdminProviderDeleteAccount(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminProviders{providerDeleted: true}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     service,
	})
	request := httptest.NewRequest(http.MethodDelete, "/admin/v1/providers/provider-1/accounts/account-1", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Deleted         bool `json:"deleted"`
		ProviderDeleted bool `json:"provider_deleted"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Deleted || !body.ProviderDeleted {
		t.Fatalf("unexpected delete response: %+v", body)
	}
}

func TestAdminProviderDeleteAccountNotFound(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     fakeAdminProviders{err: admin.ErrNotFound},
	})
	request := httptest.NewRequest(http.MethodDelete, "/admin/v1/providers/provider-1/accounts/missing", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}
func TestAdminProviderModelRoutes(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminProviders{
		configured:  []admin.ConfiguredModel{{ModelID: "gpt-4.1", DisplayName: "GPT-4.1", Active: true}},
		syncResult:  admin.SyncModelsResult{Imported: 12, Total: 12},
		modelResult: admin.ModelTestResult{OK: true, Status: 200, LatencyMS: 42},
	}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     service,
	})

	request := httptest.NewRequest(http.MethodGet, "/admin/v1/providers/provider-1/models/configured", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected configured models 200, got %d: %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/models/sync", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected sync 200, got %d: %s", response.Code, response.Body.String())
	}
	var synced admin.SyncModelsResult
	if err := json.NewDecoder(response.Body).Decode(&synced); err != nil {
		t.Fatalf("decode sync: %v", err)
	}
	if synced.Imported != 12 || synced.Total != 12 {
		t.Fatalf("unexpected sync result: %+v", synced)
	}

	request = httptest.NewRequest(http.MethodPatch, "/admin/v1/providers/provider-1/models", strings.NewReader(`{"model_id":"gpt-4.1","active":false}`))
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected patch 200, got %d: %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPatch, "/admin/v1/providers/provider-1/models", strings.NewReader(`{"model_id":"gpt-4.1"}`))
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected patch without active to be 400, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/models/test", strings.NewReader(`{"model_id":"gpt-4.1"}`))
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected test 200, got %d: %s", response.Code, response.Body.String())
	}
	var tested admin.ModelTestResult
	if err := json.NewDecoder(response.Body).Decode(&tested); err != nil {
		t.Fatalf("decode test: %v", err)
	}
	if !tested.OK || tested.Status != 200 {
		t.Fatalf("unexpected test result: %+v", tested)
	}
}

func TestAdminProviderModelNotFound(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     fakeAdminProviders{err: admin.ErrNotFound},
	})
	request := httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/models/test", strings.NewReader(`{"model_id":"missing"}`))
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAdminProviderModelTestRuns(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminProviders{modelRun: admin.ModelTestRun{
		ID: "run-1", ProviderID: "provider-1", Status: "queued", Total: 2,
		Items: []admin.ModelTestRunItem{{ModelID: "a", Status: "pending"}, {ModelID: "b", Status: "pending"}},
	}}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     service,
	})

	start := httptest.NewRequest(http.MethodPost, "/admin/v1/providers/provider-1/models/test-all", strings.NewReader(`{"models":["a","b"]}`))
	start.Header.Set("Authorization", "Bearer "+key)
	start.Header.Set("Content-Type", "application/json")
	startResponse := httptest.NewRecorder()
	router.ServeHTTP(startResponse, start)
	if startResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", startResponse.Code, startResponse.Body.String())
	}
	var run admin.ModelTestRun
	if err := json.NewDecoder(startResponse.Body).Decode(&run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.ID != "run-1" || run.Total != 2 {
		t.Fatalf("unexpected run: %+v", run)
	}

	latest := httptest.NewRequest(http.MethodGet, "/admin/v1/providers/provider-1/models/test-runs/latest", nil)
	latest.Header.Set("Authorization", "Bearer "+key)
	latestResponse := httptest.NewRecorder()
	router.ServeHTTP(latestResponse, latest)
	if latestResponse.Code != http.StatusOK {
		t.Fatalf("expected latest 200, got %d", latestResponse.Code)
	}

	single := httptest.NewRequest(http.MethodGet, "/admin/v1/providers/provider-1/models/test-runs/run-1", nil)
	single.Header.Set("Authorization", "Bearer "+key)
	singleResponse := httptest.NewRecorder()
	router.ServeHTTP(singleResponse, single)
	if singleResponse.Code != http.StatusOK {
		t.Fatalf("expected run 200, got %d", singleResponse.Code)
	}
}

func TestAdminProviderModelTestRunNotFound(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     fakeAdminProviders{err: admin.ErrNotFound},
	})
	request := httptest.NewRequest(http.MethodGet, "/admin/v1/providers/provider-1/models/test-runs/latest", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestAdminProviderBulkModelState(t *testing.T) {
	verifier, key := testVerifier(t, []string{"manage"})
	service := fakeAdminProviders{bulkResults: []admin.ModelUpdateResult{
		{ModelID: "a", OK: true},
		{ModelID: "missing", Error: "model not found"},
	}}
	router := api.NewRouter(api.Dependencies{
		Readiness: health.NewRegistry(0),
		Verifier:  verifier,
		Admin:     service,
	})
	request := httptest.NewRequest(http.MethodPatch, "/admin/v1/providers/provider-1/models/bulk", strings.NewReader(`{"models":["a","missing"],"active":false}`))
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Updated int                       `json:"updated"`
		Results []admin.ModelUpdateResult `json:"results"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Updated != 1 || len(body.Results) != 2 || body.Results[1].Error == "" {
		t.Fatalf("unexpected bulk response: %+v", body)
	}
}
