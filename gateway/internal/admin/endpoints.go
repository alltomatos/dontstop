package admin

// EndpointInfo describes one public API endpoint.
type EndpointInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Category string `json:"category"`
	Models   int    `json:"models"`
	Enabled  bool   `json:"enabled"`
}

// EndpointPolicy is the default policy for an endpoint.
type EndpointPolicy struct {
	EndpointID   string `json:"endpoint_id"`
	AuthRequired bool   `json:"auth_required"`
	RateLimitRPM int    `json:"rate_limit_rpm"`
	CacheEnabled bool   `json:"cache_enabled"`
	DLPEnabled   bool   `json:"dlp_enabled"`
	ProxyEnabled bool   `json:"proxy_enabled"`
	MaxBodyBytes int    `json:"max_body_bytes"`
	TimeoutMS    int    `json:"timeout_ms"`
}

var endpointCatalog = []EndpointInfo{
	{ID: "chat-completions", Name: "Chat Completions", Path: "/v1/chat/completions", Category: "main", Models: 1208, Enabled: true},
	{ID: "responses", Name: "Responses", Path: "/v1/responses", Category: "main", Models: 1208, Enabled: true},
	{ID: "messages", Name: "Messages", Path: "/v1/messages", Category: "main", Models: 1208, Enabled: true},
	{ID: "completions", Name: "Completions", Path: "/v1/completions", Category: "main", Models: 1208, Enabled: true},
	{ID: "embeddings", Name: "Embeddings", Path: "/v1/embeddings", Category: "media", Models: 1208, Enabled: true},
	{ID: "images", Name: "Images", Path: "/v1/images/generations", Category: "media", Models: 1208, Enabled: true},
	{ID: "audio", Name: "Audio", Path: "/v1/audio/transcriptions", Category: "media", Models: 1208, Enabled: true},
	{ID: "video", Name: "Video", Path: "/v1/videos/generations", Category: "media", Models: 1208, Enabled: true},
	{ID: "moderations", Name: "Moderations", Path: "/v1/moderations", Category: "utility", Models: 1208, Enabled: true},
	{ID: "batches", Name: "Batches", Path: "/v1/batches", Category: "utility", Models: 1208, Enabled: true},
	{ID: "files", Name: "Files", Path: "/v1/files", Category: "utility", Models: 1208, Enabled: true},
	{ID: "models", Name: "Models", Path: "/v1/models", Category: "utility", Models: 1208, Enabled: true},
}

// EndpointService serves the static endpoint catalog and default policies.
type EndpointService struct{}

// NewEndpointService creates an endpoint service.
func NewEndpointService() *EndpointService { return &EndpointService{} }

// List returns the endpoint catalog.
func (s *EndpointService) List() []EndpointInfo {
	result := make([]EndpointInfo, len(endpointCatalog))
	copy(result, endpointCatalog)
	return result
}

// Get returns one endpoint.
func (s *EndpointService) Get(id string) (EndpointInfo, bool) {
	for _, endpoint := range endpointCatalog {
		if endpoint.ID == id {
			return endpoint, true
		}
	}
	return EndpointInfo{}, false
}

// Policy returns the default endpoint policy.
func (s *EndpointService) Policy(id string) (EndpointPolicy, bool) {
	if _, ok := s.Get(id); !ok {
		return EndpointPolicy{}, false
	}
	return EndpointPolicy{
		EndpointID: id, AuthRequired: true, RateLimitRPM: 120, CacheEnabled: true,
		DLPEnabled: true, ProxyEnabled: false, MaxBodyBytes: 1048576, TimeoutMS: 60000,
	}, true
}
