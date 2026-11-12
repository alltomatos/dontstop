package admin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ErrModelsUnsupported indicates the provider does not expose a models endpoint.
var ErrModelsUnsupported = errors.New("provider does not expose a models endpoint")

// catalogAdapterKinds maps a catalog protocol to the adapter kind it requires.
var catalogAdapterKinds = map[string]string{
	"openai":      "openai",
	"anthropic":   "anthropic",
	"agentrouter": "agentrouter",
}

// CatalogModel is one model declared by a catalog entry.
type CatalogModel struct {
	ModelID            string  `json:"model_id"`
	DisplayName        string  `json:"display_name"`
	ContextWindow      int     `json:"context_window"`
	MaxOutputTokens    int     `json:"max_output_tokens"`
	SupportsText       bool    `json:"supports_text"`
	SupportsStream     bool    `json:"supports_stream"`
	SupportsTools      bool    `json:"supports_tools"`
	SupportsVision     bool    `json:"supports_vision"`
	SupportsReasoning  bool    `json:"supports_reasoning"`
	SupportsAudio      bool    `json:"supports_audio"`
	SupportsVideo      bool    `json:"supports_video"`
	SupportsEmbeddings bool    `json:"supports_embeddings"`
	InputPrice         float64 `json:"input_price_per_million"`
	OutputPrice        float64 `json:"output_price_per_million"`
	PricingSource      string  `json:"pricing_source"`
}

// CatalogEntry is a versioned, pre-configured provider definition.
type CatalogEntry struct {
	Slug           string         `json:"slug"`
	DisplayName    string         `json:"display_name"`
	Protocol       string         `json:"protocol"`
	APIVersion     string         `json:"api_version,omitempty"`
	DefaultBaseURL string         `json:"default_base_url"`
	ModelsPath     string         `json:"models_path,omitempty"`
	AuthType       string         `json:"auth_type"`
	AuthHeader     string         `json:"auth_header"`
	AuthScheme     string         `json:"auth_scheme"`
	Headers        map[string]any `json:"headers"`
	Capabilities   map[string]any `json:"capabilities"`
	Network        map[string]any `json:"network"`
	Quirks         []string       `json:"quirks"`
	SourceRef      string         `json:"source_ref"`
	CatalogVersion string         `json:"catalog_version"`
	Models         []CatalogModel `json:"models"`
}

// CreateFromCatalogInput activates a catalog provider for a tenant.
type CreateFromCatalogInput struct {
	Slug        string `json:"slug"`
	AccountName string `json:"account_name"`
	Credential  string `json:"credential"`
	BaseURL     string `json:"base_url"`
}

// ListCatalog returns every enabled catalog entry with its models.
func (s *ProviderService) ListCatalog(ctx context.Context) ([]CatalogEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slug, display_name, protocol, coalesce(api_version, ''), default_base_url,
		       coalesce(models_path, ''), auth_type, auth_header, auth_scheme,
		       headers, capabilities, network, quirks, source_ref, catalog_version
		FROM provider_catalog
		WHERE enabled = true
		ORDER BY slug
	`)
	if err != nil {
		return nil, fmt.Errorf("query provider catalog: %w", err)
	}
	defer rows.Close()

	entries := make([]CatalogEntry, 0)
	index := make(map[string]int)
	for rows.Next() {
		var entry CatalogEntry
		if err := rows.Scan(
			&entry.Slug, &entry.DisplayName, &entry.Protocol, &entry.APIVersion,
			&entry.DefaultBaseURL, &entry.ModelsPath, &entry.AuthType, &entry.AuthHeader,
			&entry.AuthScheme, &entry.Headers, &entry.Capabilities, &entry.Network,
			&entry.Quirks, &entry.SourceRef, &entry.CatalogVersion,
		); err != nil {
			return nil, fmt.Errorf("scan provider catalog: %w", err)
		}
		index[entry.Slug] = len(entries)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider catalog: %w", err)
	}

	modelRows, err := s.pool.Query(ctx, `
		SELECT slug, model_id, display_name, coalesce(context_window, 0), coalesce(max_output_tokens, 0),
		       supports_text, supports_stream, supports_tools, supports_vision, supports_reasoning,
		       supports_audio, supports_video, supports_embeddings,
		       input_price_per_million::float8, output_price_per_million::float8, coalesce(pricing_source, '')
		FROM provider_catalog_models
		ORDER BY slug, model_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query provider catalog models: %w", err)
	}
	defer modelRows.Close()
	for modelRows.Next() {
		var slug string
		var model CatalogModel
		if err := modelRows.Scan(
			&slug, &model.ModelID, &model.DisplayName, &model.ContextWindow, &model.MaxOutputTokens,
			&model.SupportsText, &model.SupportsStream, &model.SupportsTools, &model.SupportsVision, &model.SupportsReasoning,
			&model.SupportsAudio, &model.SupportsVideo, &model.SupportsEmbeddings,
			&model.InputPrice, &model.OutputPrice, &model.PricingSource,
		); err != nil {
			return nil, fmt.Errorf("scan provider catalog model: %w", err)
		}
		if position, ok := index[slug]; ok {
			entries[position].Models = append(entries[position].Models, model)
		}
	}
	if err := modelRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider catalog models: %w", err)
	}
	return entries, nil
}

// CreateFromCatalog activates a catalog provider, its first account and its models atomically.
func (s *ProviderService) CreateFromCatalog(ctx context.Context, tenantID string, input CreateFromCatalogInput) (Provider, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.AccountName = strings.TrimSpace(input.AccountName)
	input.Credential = strings.TrimSpace(input.Credential)
	input.BaseURL = strings.TrimSpace(input.BaseURL)
	if input.Slug == "" || input.Credential == "" {
		return Provider{}, fmt.Errorf("%w: slug and credential are required", ErrInvalidInput)
	}
	if s.box == nil {
		return Provider{}, fmt.Errorf("credential box is not configured")
	}

	var entry CatalogEntry
	err := s.pool.QueryRow(ctx, `
		SELECT slug, display_name, protocol, default_base_url
		FROM provider_catalog
		WHERE slug = $1 AND enabled = true
	`, input.Slug).Scan(&entry.Slug, &entry.DisplayName, &entry.Protocol, &entry.DefaultBaseURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("load provider catalog entry: %w", err)
	}

	kind, ok := catalogAdapterKind(entry.Protocol)
	if !ok {
		return Provider{}, fmt.Errorf("%w: unsupported catalog protocol %q", ErrInvalidInput, entry.Protocol)
	}
	baseURL := input.BaseURL
	if baseURL == "" {
		baseURL = entry.DefaultBaseURL
	}
	if err := validateProviderBaseURL(baseURL); err != nil {
		return Provider{}, err
	}
	if input.AccountName == "" {
		input.AccountName = entry.DisplayName
	}
	ciphertext, err := s.box.Encrypt([]byte(input.Credential))
	if err != nil {
		return Provider{}, fmt.Errorf("encrypt credential: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Provider{}, fmt.Errorf("begin catalog provider create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var providerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled, catalog_slug)
		VALUES ($1, $2, $3, 'native', true, $4)
		RETURNING id::text
	`, tenantID, entry.DisplayName, kind, entry.Slug).Scan(&providerID)
	if err != nil {
		if strings.Contains(err.Error(), "providers_tenant_name_unique") {
			return Provider{}, ErrConflict
		}
		return Provider{}, fmt.Errorf("insert catalog provider: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, credential_cipher, base_url, priority, weight)
		VALUES ($1, $2::uuid, $3, $4, $5, 100, 1)
	`, tenantID, providerID, input.AccountName, ciphertext, baseURL); err != nil {
		return Provider{}, fmt.Errorf("insert catalog provider account: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, capabilities, active,
		                             input_price_per_million, output_price_per_million)
		SELECT $1, $2::uuid, model_id, display_name,
		       jsonb_build_object(
		           'context_window', context_window,
		           'max_output_tokens', max_output_tokens,
		           'text_input', supports_text,
		           'text_output', supports_text,
		           'image_input', supports_vision,
		           'audio_input', supports_audio,
		           'video_input', supports_video,
		           'image_output', false,
		           'audio_output', false,
		           'video_output', false,
		           'vision', supports_vision,
		           'audio', supports_audio,
		           'stream', supports_stream,
		           'tools', supports_tools,
		           'reasoning', supports_reasoning,
		           'embeddings', supports_embeddings,
		           'metadata_source', 'catalog:provider_catalog_models',
		           'pricing_source', pricing_source
		       ),
		       true, input_price_per_million, output_price_per_million
		FROM provider_catalog_models
		WHERE slug = $3
	`, tenantID, providerID, entry.Slug); err != nil {
		return Provider{}, fmt.Errorf("insert catalog provider models: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Provider{}, fmt.Errorf("commit catalog provider create: %w", err)
	}
	// Import advertised models right away; a failing upstream must not block creation.
	_, _ = s.SyncModels(ctx, tenantID, providerID)
	return s.Get(ctx, tenantID, providerID)
}

// catalogAdapterKind resolves the adapter kind for a catalog protocol.
func catalogAdapterKind(protocol string) (string, bool) {
	kind, ok := catalogAdapterKinds[strings.ToLower(strings.TrimSpace(protocol))]
	return kind, ok
}

// validateProviderBaseURL accepts only absolute HTTP(S) provider endpoints.
func validateProviderBaseURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("%w: base_url is required", ErrInvalidInput)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("%w: base_url is not a valid URL", ErrInvalidInput)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: base_url must use http or https", ErrInvalidInput)
	}
	return nil
}
