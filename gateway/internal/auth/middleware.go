package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Middleware authenticates Bearer API keys and attaches the principal.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		ctx, span := otel.Tracer("dontstop/gateway/auth").Start(request.Context(), "auth.authenticate")
		defer span.End()
		principal, err := v.Authenticate(ctx, request.Header.Get("Authorization"))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "authentication failed")
			span.SetAttributes(attribute.Bool("auth.authenticated", false))
			switch {
			case errors.Is(err, ErrInvalidAuthorization), errors.Is(err, ErrInvalidKey),
				errors.Is(err, ErrKeyRevoked), errors.Is(err, ErrKeyExpired):
				writeError(w, http.StatusUnauthorized, "authentication_error", "invalid API key")
			default:
				writeError(w, http.StatusInternalServerError, "api_error", "authentication failed")
			}
			return
		}
		span.SetAttributes(attribute.Bool("auth.authenticated", true))
		next.ServeHTTP(w, request.WithContext(WithPrincipal(ctx, principal)))
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
