package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	"github.com/alltomatos/dontstop/gateway/internal/auth"
)

// AdminAuthService exposes console identity and session operations.
type AdminAuthService interface {
	NeedsSetup(ctx context.Context) (bool, error)
	Setup(ctx context.Context, input admin.SetupInput) (admin.Session, error)
	Login(ctx context.Context, input admin.LoginInput) (admin.Session, error)
	Authenticate(ctx context.Context, token string) (auth.Principal, error)
	Logout(ctx context.Context, token string) error
}

func adminAuthStatus(service AdminAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		needsSetup, err := service.NeedsSetup(request.Context())
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to read setup status")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"needs_setup": needsSetup})
	}
}

func adminAuthSetup(service AdminAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.SetupInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		session, err := service.Setup(request.Context(), input)
		if errors.Is(err, admin.ErrSetupComplete) {
			writeAPIError(w, http.StatusConflict, "conflict_error", "setup already complete")
			return
		}
		if errors.Is(err, admin.ErrInvalidInput) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid setup input")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to create owner")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(session)
	}
}

func adminAuthLogin(service AdminAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(w, request.Body, 1<<20)
		var input admin.LoginInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON body")
			return
		}
		session, err := service.Login(request.Context(), input)
		if errors.Is(err, admin.ErrInvalidCredentials) {
			writeAPIError(w, http.StatusUnauthorized, "authentication_error", "invalid email or password")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to login")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(session)
	}
}

func adminAuthMe(w http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "authentication_error", "not authenticated")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true,
		"tenant_id":     principal.TenantID,
		"scopes":        principal.Scopes,
	})
}

func adminAuthLogout(service AdminAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if err := service.Logout(request.Context(), bearerToken(request)); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "api_error", "failed to logout")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func adminAuthMiddleware(verifier *auth.Verifier, service AdminAuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			token := bearerToken(request)
			if token == "" {
				writeAPIError(w, http.StatusUnauthorized, "authentication_error", "missing bearer token")
				return
			}
			if verifier != nil {
				if principal, err := verifier.Authenticate(request.Context(), request.Header.Get("Authorization")); err == nil {
					next.ServeHTTP(w, request.WithContext(auth.WithPrincipal(request.Context(), principal)))
					return
				}
			}
			if service != nil {
				if principal, err := service.Authenticate(request.Context(), token); err == nil {
					next.ServeHTTP(w, request.WithContext(auth.WithPrincipal(request.Context(), principal)))
					return
				}
			}
			writeAPIError(w, http.StatusUnauthorized, "authentication_error", "invalid credentials")
		})
	}
}

func bearerToken(request *http.Request) string {
	scheme, token, ok := strings.Cut(strings.TrimSpace(request.Header.Get("Authorization")), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}
