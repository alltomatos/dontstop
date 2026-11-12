package admin_test

import (
	"context"
	"encoding/json"
	"errors"
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

func TestModelTestRunQueueWorker(t *testing.T) {
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
				_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"chat-1","model":"remote-alpha","choices":[{"message":{"content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
		Name: "Queue Provider", Kind: "openai", Origin: "custom",
		ModelID: "seed-model", Credential: "test-key", BaseURL: upstream.URL + "/v1",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	run, err := service.StartModelTestRun(ctx, tenantID, provider.ID, []string{"remote-alpha", "remote-beta", "remote-alpha"})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Total != 2 || run.Status != "queued" {
		t.Fatalf("unexpected run: %+v", run)
	}
	duplicate, err := service.StartModelTestRun(ctx, tenantID, provider.ID, nil)
	if err != nil {
		t.Fatalf("start duplicate run: %v", err)
	}
	if duplicate.ID != run.ID {
		t.Fatalf("expected the active run to be reused, got %s and %s", run.ID, duplicate.ID)
	}

	processed, err := service.RunModelTestOnce(ctx)
	if err != nil {
		t.Fatalf("run worker once: %v", err)
	}
	if !processed {
		t.Fatal("expected the worker to process the queued run")
	}

	final, err := service.GetModelTestRun(ctx, tenantID, provider.ID, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if final.Status != "completed" || final.Completed != 2 || final.OKCount != 1 || final.FailedCount != 1 {
		t.Fatalf("unexpected final run: %+v", final)
	}
	for _, item := range final.Items {
		switch item.ModelID {
		case "remote-alpha":
			if item.Status != "ok" || item.LatencyMS < 0 {
				t.Fatalf("unexpected alpha item: %+v", item)
			}
		case "remote-beta":
			if item.Status != "failed" || item.Error == "" {
				t.Fatalf("unexpected beta item: %+v", item)
			}
		}
	}

	latest, err := service.LatestModelTestRun(ctx, tenantID, provider.ID)
	if err != nil {
		t.Fatalf("latest run: %v", err)
	}
	if latest.ID != run.ID || latest.Status != "completed" {
		t.Fatalf("unexpected latest run: %+v", latest)
	}

	all, err := service.StartModelTestRun(ctx, tenantID, provider.ID, nil)
	if err != nil {
		t.Fatalf("start run for all models: %v", err)
	}
	if all.ID == run.ID || all.Total < 2 {
		t.Fatalf("expected a new run covering every model, got %+v", all)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM provider_models WHERE provider_id = $1::uuid`, provider.ID); err != nil {
		t.Fatalf("delete models: %v", err)
	}
	if _, err := service.StartModelTestRun(ctx, tenantID, provider.ID, nil); !errors.Is(err, admin.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for a provider without models, got %v", err)
	}
}
