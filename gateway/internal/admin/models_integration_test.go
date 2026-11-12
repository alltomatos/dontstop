package admin_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/platform/database"
	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/migrations"
)

func TestProviderModelsSyncToggleAndTest(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch request.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"remote-alpha","name":"Remote Alpha"},{"id":"remote-beta"}]}`))
		case "/v1/chat/completions":
			var payload struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(request.Body).Decode(&payload)
			if payload.Model == "remote-beta" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"message":"model disabled upstream"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"chat-1","model":"remote-alpha","choices":[{"message":{"content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	box, err := secret.NewBox(make([]byte, 32))
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	service := admin.NewProviderService(pool, box)
	const tenantID = "00000000-0000-0000-0000-000000000000"

	provider, err := service.Create(ctx, tenantID, admin.CreateProviderInput{
		Name: "Mock Provider", Kind: "openai", Origin: "custom",
		ModelID: "seed-model", Credential: "test-key", BaseURL: upstream.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	models, err := service.ConfiguredModels(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("configured models: %v", err)
	}
	ids := make(map[string]admin.ConfiguredModel, len(models))
	for _, model := range models {
		ids[model.ModelID] = model
	}
	if _, ok := ids["seed-model"]; !ok {
		t.Fatal("seed model was not persisted")
	}
	if alpha, ok := ids["remote-alpha"]; !ok || !alpha.Active || alpha.DisplayName != "Remote Alpha" {
		t.Fatalf("remote model was not imported: %+v", alpha)
	}
	if _, ok := ids["remote-beta"]; !ok {
		t.Fatal("remote-beta was not imported")
	}

	if err := service.SetModelActive(ctx, tenantID, provider.ID, "remote-alpha", false); err != nil {
		t.Fatalf("disable model: %v", err)
	}
	if err := service.SetModelActive(ctx, tenantID, provider.ID, "missing-model", false); err != admin.ErrNotFound {
		t.Fatalf("expected ErrNotFound for missing model, got %v", err)
	}

	result, err := service.TestModel(ctx, tenantID, provider.ID, "remote-alpha", "")
	if err != nil {
		t.Fatalf("test model: %v", err)
	}
	if !result.OK || result.Status != 200 {
		t.Fatalf("expected successful model test, got %+v", result)
	}
	brokenResult, err := service.TestModel(ctx, tenantID, provider.ID, "remote-beta", "")
	if err != nil {
		t.Fatalf("test broken model: %v", err)
	}
	if brokenResult.OK || brokenResult.Error == "" {
		t.Fatalf("expected failed model test with message, got %+v", brokenResult)
	}

	synced, err := service.SyncModels(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("sync models: %v", err)
	}
	if synced.Total != 2 || synced.Error != "" {
		t.Fatalf("unexpected sync result: %+v", synced)
	}

	models, err = service.ConfiguredModels(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("configured models after sync: %v", err)
	}
	for _, model := range models {
		if model.ModelID == "remote-alpha" && model.Active {
			t.Fatal("sync must not reactivate a disabled model")
		}
	}
}

func TestBulkModelStateAndMetadataEnrichment(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[
			{"id":"rich-model","name":"Rich","context_length":128000,
			 "pricing":{"prompt":"0.000001","completion":"0.000002"},
			 "architecture":{"input_modalities":["text","image","audio"]},
			 "supported_parameters":["tools","reasoning"],
			 "top_provider":{"max_completion_tokens":4096},
			 "endpoints":["/chat/completions"]},
			{"id":"free-model","name":"Free",
			 "pricing":{"prompt":"0","completion":"0"},
			 "architecture":{"input_modalities":["text"]},
			 "endpoints":["/chat/completions"]}
		]}`))
	}))
	defer upstream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	box, err := secret.NewBox(make([]byte, 32))
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	service := admin.NewProviderService(pool, box)
	const tenantID = "00000000-0000-0000-0000-000000000000"

	provider, err := service.Create(ctx, tenantID, admin.CreateProviderInput{
		Name: "Metadata Provider", Kind: "openai", Origin: "custom",
		ModelID: "seed", Credential: "key", BaseURL: upstream.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	models, err := service.ConfiguredModels(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("configured models: %v", err)
	}
	byID := make(map[string]admin.ConfiguredModel, len(models))
	for _, model := range models {
		byID[model.ModelID] = model
	}
	rich := byID["rich-model"]
	caps, _ := rich.Capabilities.(map[string]any)
	if rich.InputPrice != 1 || rich.OutputPrice != 2 {
		t.Fatalf("unexpected rich prices: %+v", rich)
	}
	if caps["vision"] != true || caps["audio"] != true || caps["tools"] != true || caps["reasoning"] != true || caps["stream"] != true || caps["pricing_known"] != true {
		t.Fatalf("unexpected rich capabilities: %+v", caps)
	}
	free := byID["free-model"]
	freeCaps, _ := free.Capabilities.(map[string]any)
	if free.InputPrice != 0 || free.OutputPrice != 0 || freeCaps["pricing_known"] != true {
		t.Fatalf("unexpected free metadata: %+v %+v", free, freeCaps)
	}

	results, err := service.SetModelsActive(ctx, tenantID, provider.ID, []string{"rich-model", "missing-model"}, false)
	if err != nil {
		t.Fatalf("bulk update: %v", err)
	}
	if len(results) != 2 || !results[0].OK || results[1].OK || results[1].Error == "" {
		t.Fatalf("unexpected bulk results: %+v", results)
	}
	models, err = service.ConfiguredModels(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("configured models after bulk: %v", err)
	}
	for _, model := range models {
		if model.ModelID == "rich-model" && model.Active {
			t.Fatal("bulk update must deactivate the model")
		}
	}
}
