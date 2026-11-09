package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
)

type stubAdminAuth struct {
	loginErr error
}

func (s stubAdminAuth) NeedsSetup(context.Context) (bool, error) { return false, nil }

func (s stubAdminAuth) Setup(context.Context, admin.SetupInput) (admin.Session, error) {
	return admin.Session{}, nil
}

func (s stubAdminAuth) Login(context.Context, admin.LoginInput) (admin.Session, error) {
	if s.loginErr != nil {
		return admin.Session{}, s.loginErr
	}
	return admin.Session{Token: "session-token"}, nil
}

func (s stubAdminAuth) Authenticate(context.Context, string) (auth.Principal, error) {
	return auth.Principal{}, nil
}

func (s stubAdminAuth) Logout(context.Context, string) error { return nil }

func postLogin(handler http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/admin/v1/auth/login", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestLoginLimiterBlocksAfterRepeatedFailures(t *testing.T) {
	clock := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	limiter := newLoginLimiter()
	limiter.now = func() time.Time { return clock }
	handler := adminAuthLogin(stubAdminAuth{loginErr: admin.ErrInvalidCredentials}, limiter)
	body := `{"email":"owner@example.com","password":"wrong"}`

	for attempt := 0; attempt < loginMaxAttempts; attempt++ {
		if code := postLogin(handler, body).Code; code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401, got %d", attempt+1, code)
		}
	}
	if code := postLogin(handler, body).Code; code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after %d failures, got %d", loginMaxAttempts, code)
	}

	clock = clock.Add(loginWindow + time.Second)
	if code := postLogin(handler, body).Code; code != http.StatusUnauthorized {
		t.Fatalf("expected window to reset, got %d", code)
	}
}

func TestLoginLimiterResetsOnSuccess(t *testing.T) {
	limiter := newLoginLimiter()
	handler := adminAuthLogin(stubAdminAuth{}, limiter)
	email := strings.ToLower("owner@example.com")

	for attempt := 0; attempt < loginMaxAttempts-1; attempt++ {
		limiter.Fail(email)
	}
	if !limiter.Allow(email) {
		t.Fatal("expected limiter to allow before threshold")
	}

	if code := postLogin(handler, `{"email":"owner@example.com","password":"right"}`).Code; code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !limiter.Allow(email) {
		t.Fatal("expected successful login to reset the email counter")
	}
}

func TestLoginLimiterIsPerEmail(t *testing.T) {
	limiter := newLoginLimiter()
	for attempt := 0; attempt < loginMaxAttempts; attempt++ {
		limiter.Fail("owner@example.com")
	}
	if limiter.Allow("owner@example.com") {
		t.Fatal("email should be locked out")
	}
	if !limiter.Allow("other@example.com") {
		t.Fatal("other emails should not inherit the lockout")
	}
}
