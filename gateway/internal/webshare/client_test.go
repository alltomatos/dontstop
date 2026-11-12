package webshare_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/webshare"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func TestListDirectFollowsPaginationAndContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token test-key" {
			t.Errorf("unexpected authorization header")
		}
		if r.URL.Query().Get("mode") != "direct" || r.URL.Query().Get("plan_id") != "plan-1" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = w.Write(fixture(t, "list_page_1.json"))
		case "2":
			_, _ = w.Write(fixture(t, "list_page_2.json"))
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	client := webshare.NewClient(server.Client())
	client.V2Base = server.URL
	proxies, err := client.ListDirect(context.Background(), "test-key", "plan-1")
	if err != nil {
		t.Fatalf("list direct: %v", err)
	}
	if len(proxies) != 3 || proxies[0].ID != "p-1" || proxies[2].Address != "203.0.113.12" || proxies[2].Valid {
		t.Fatalf("unexpected proxies: %+v", proxies)
	}
}

func TestV3ConfigStatusAndStatsContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token test-key" {
			t.Errorf("unexpected authorization header")
		}
		switch r.URL.Path {
		case "/proxy/config":
			_, _ = w.Write(fixture(t, "config.json"))
		case "/proxy/list/status":
			_, _ = w.Write(fixture(t, "status.json"))
		case "/proxy/list/stats":
			_, _ = w.Write(fixture(t, "stats.json"))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := webshare.NewClient(server.Client())
	client.V3Base = server.URL
	config, err := client.Config(context.Background(), "test-key", "plan-1")
	if err != nil || config.ProxyListDownloadToken != "download-token" || config.PlanID != "plan-1" {
		t.Fatalf("config: %+v err=%v", config, err)
	}
	status, err := client.Status(context.Background(), "test-key", "plan-1")
	if err != nil || status.State != "completed" || status.Countries["BR"] != 100 {
		t.Fatalf("status: %+v err=%v", status, err)
	}
	stats, err := client.Stats(context.Background(), "test-key", "plan-1")
	if err != nil || stats.AvailableCountries["BR"] != 100 {
		t.Fatalf("stats: %+v err=%v", stats, err)
	}
}

func TestRefreshAndDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/proxy/list/refresh/"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/proxy/list/download/"):
			if r.Header.Get("Authorization") != "" {
				t.Errorf("download must not send authorization")
			}
			_, _ = w.Write([]byte("203.0.113.10:8080:user:pass\n"))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := webshare.NewClient(server.Client())
	client.V2Base = server.URL
	if err := client.Refresh(context.Background(), "test-key", "plan-1"); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	body, err := client.Download(context.Background(), "token", "", "", "direct", "", "plan-1")
	if err != nil || !strings.Contains(body, "203.0.113.10:8080") {
		t.Fatalf("download: %q err=%v", body, err)
	}
}

func TestRateLimitBacksOffAndRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write(fixture(t, "list_page_2.json"))
	}))
	defer server.Close()

	waited := time.Duration(0)
	client := webshare.NewClient(server.Client())
	client.V2Base = server.URL
	client.Sleep = func(_ context.Context, duration time.Duration) error {
		waited = duration
		return nil
	}
	if _, err := client.ListDirect(context.Background(), "test-key", ""); err != nil {
		t.Fatalf("list after retry: %v", err)
	}
	if attempts != 2 || waited != 60*time.Second {
		t.Fatalf("expected one 60s backoff and two attempts, got attempts=%d wait=%s", attempts, waited)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, webshare.ErrUnauthorized},
		{http.StatusForbidden, webshare.ErrForbidden},
		{http.StatusInternalServerError, webshare.ErrUnavailable},
	}
	for _, test := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(test.status)
		}))
		client := webshare.NewClient(server.Client())
		client.V2Base = server.URL
		_, err := client.ListDirect(context.Background(), "secret-key", "")
		if !errors.Is(err, test.want) {
			t.Fatalf("status %d: expected %v, got %v", test.status, test.want, err)
		}
		if strings.Contains(err.Error(), "secret-key") {
			t.Fatalf("error leaked the api key: %v", err)
		}
		server.Close()
	}
}
