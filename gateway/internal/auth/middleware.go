package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Middleware authenticates Bearer API keys and attaches the principal.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		principal, err := v.Authenticate(request.Context(), request.Header.Get("Authorization"))
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidAuthorization), errors.Is(err, ErrInvalidKey),
				errors.Is(err, ErrKeyRevoked), errors.Is(err, ErrKeyExpired):
				writeError(w, http.StatusUnauthorized, "authentication_error", "invalid API key")
			default:
				writeError(w, http.StatusInternalServerError, "api_error", "authentication failed")
			}
			return
		}
		next.ServeHTTP(w, request.WithContext(WithPrincipal(request.Context(), principal)))
	})
}

func writeError(w http.ResponseWriter, status int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"type":    errorType,
			"message": message,
		},
	})
}
