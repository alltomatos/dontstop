package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Check reports whether a dependency is ready.
type Check func(context.Context) error

// Registry holds dependency checks used by readiness probes.
type Registry struct {
	timeout time.Duration

	mu     sync.RWMutex
	checks map[string]Check
}

type statusResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// NewRegistry creates a readiness registry with a bounded check timeout.
func NewRegistry(timeout time.Duration) *Registry {
	if timeout <= 0 {
		timeout = time.Second
	}
	return &Registry{
		timeout: timeout,
		checks:  make(map[string]Check),
	}
}

// Register adds or replaces a named readiness check.
func (r *Registry) Register(name string, check Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[name] = check
}

// Liveness reports that the process is running and can serve HTTP.
func (r *Registry) Liveness() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeStatus(w, http.StatusOK, statusResponse{Status: "ok"})
	})
}

// Readiness reports whether all registered dependencies are available.
func (r *Registry) Readiness() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		checks := r.snapshot()
		if len(checks) == 0 {
			writeStatus(w, http.StatusOK, statusResponse{Status: "ok"})
			return
		}

		ctx, cancel := context.WithTimeout(request.Context(), r.timeout)
		defer cancel()

		results := make(map[string]string, len(checks))
		var mu sync.Mutex
		var wg sync.WaitGroup
		for name, check := range checks {
			wg.Add(1)
			go func(name string, check Check) {
				defer wg.Done()
				status := "ok"
				if err := check(ctx); err != nil {
					status = "error"
				}
				mu.Lock()
				results[name] = status
				mu.Unlock()
			}(name, check)
		}
		wg.Wait()

		status := http.StatusOK
		overall := "ok"
		for _, result := range results {
			if result != "ok" {
				status = http.StatusServiceUnavailable
				overall = "unavailable"
				break
			}
		}
		writeStatus(w, status, statusResponse{Status: overall, Checks: results})
	})
}

func (r *Registry) snapshot() map[string]Check {
	r.mu.RLock()
	defer r.mu.RUnlock()
	checks := make(map[string]Check, len(r.checks))
	for name, check := range r.checks {
		checks[name] = check
	}
	return checks
}

func writeStatus(w http.ResponseWriter, status int, payload statusResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
