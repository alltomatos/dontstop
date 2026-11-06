package api

import (
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/platform/health"
)

// NewRouter builds the public HTTP surface for the gateway.
func NewRouter(readiness *health.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", readiness.Liveness())
	mux.Handle("GET /readyz", readiness.Readiness())
	return mux
}
