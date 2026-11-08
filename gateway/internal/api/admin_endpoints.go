package api

import (
	"net/http"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
)

// AdminEndpointService exposes the endpoint catalog and policies.
type AdminEndpointService interface {
	List() []admin.EndpointInfo
	Policy(id string) (admin.EndpointPolicy, bool)
}

func adminEndpointsList(service AdminEndpointService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if _, ok := requireManage(w, request); !ok {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": service.List()})
	}
}

func adminEndpointPolicy(service AdminEndpointService) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if _, ok := requireManage(w, request); !ok {
			return
		}
		policy, found := service.Policy(request.PathValue("id"))
		if !found {
			writeAPIError(w, http.StatusNotFound, "not_found", "endpoint not found")
			return
		}
		writeJSON(w, http.StatusOK, policy)
	}
}
