// Package admin owns the read/write admin API used by the console.
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("admin resource not found")
	ErrInvalidInput = errors.New("invalid admin input")
	ErrConflict     = errors.New("admin resource conflict")
)

// Provider is the admin representation of a configured provider.
type Provider struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"`
	Origin         string    `json:"origin"`
	CatalogSlug    string    `json:"catalog_slug,omitempty"`
	Enabled        bool      `json:"enabled"`
	ProxyAuto      bool      `json:"proxy_auto"`
	AccountsCount  int       `json:"accounts_count"`
	UsableAccounts int       `json:"usable_accounts"`
	ModelsCount    int       `json:"models_count"`
	ModelsUsable   int       `json:"models_usable"`
	Accounts       []Account `json:"accounts,omitempty"`
}

// Account is the admin representation of a provider account. Credentials are
// never returned.
type Account struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Status          string        `json:"status"`
	BaseURL         string        `json:"base_url"`
	Priority        int           `json:"priority"`
	Weight          int           `json:"weight"`
	CooldownUntil   *time.Time    `json:"cooldown_until,omitempty"`
	ProxyMode       string        `json:"proxy_mode"`
	Proxy           *AccountProxy `json:"proxy,omitempty"`
	ModelsAvailable int           `json:"models_available"`
	ModelsUsable    int           `json:"models_usable"`
	Capabilities    []string      `json:"capabilities"`
	Health          string        `json:"health"`
	Active          bool          `json:"active"`
}

// CreateProviderInput creates a provider, its first account and one model.
type CreateProviderInput struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Origin     string `json:"origin"`
	ModelID    string `json:"model_id"`
	ModelName  string `json:"model_name"`
	Credential string `json:"credential"`
	BaseURL    string `json:"base_url"`
}

// AccountInput creates a provider account.
type AccountInput struct {
	Name       string `json:"name"`
	Credential string `json:"credential"`
	BaseURL    string `json:"base_url"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
}

// ProviderService is the PostgreSQL-backed admin provider service.
type ProviderService struct {
	pool                *pgxpool.Pool
	box                 *secret.Box
	agentRouterRelayURL string
}

// NewProviderService creates a provider service.
func NewProviderService(pool *pgxpool.Pool, box *secret.Box) *ProviderService {
	return &ProviderService{pool: pool, box: box}
}

// SetAgentRouterRelayURL routes new AgentRouter accounts to the internal relay
// sidecar (ADR-025). An explicit account base_url still wins.
func (s *ProviderService) SetAgentRouterRelayURL(relayURL string) {
	s.agentRouterRelayURL = strings.TrimSpace(relayURL)
}

// List returns configured providers for a tenant.
func (s *ProviderService) List(ctx context.Context, tenantID string) ([]Provider, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id::text, p.name, p.kind, p.origin, coalesce(p.catalog_slug, ''), p.enabled, p.proxy_auto,
		       (SELECT count(*) FROM provider_accounts pa WHERE pa.provider_id = p.id AND pa.tenant_id = p.tenant_id),
		       (SELECT count(*) FROM provider_models pm WHERE pm.provider_id = p.id AND pm.tenant_id = p.tenant_id AND pm.active = true)
		FROM providers p
		WHERE p.tenant_id = $1
		ORDER BY p.name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query admin providers: %w", err)
	}
	defer rows.Close()

	providers := make([]Provider, 0)
	for rows.Next() {
		var provider Provider
		if err := rows.Scan(&provider.ID, &provider.Name, &provider.Kind, &provider.Origin, &provider.CatalogSlug, &provider.Enabled, &provider.ProxyAuto, &provider.AccountsCount, &provider.ModelsCount); err != nil {
			return nil, fmt.Errorf("scan admin provider: %w", err)
		}
		providers = append(providers, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin providers: %w", err)
	}
	return providers, nil
}

// isUUID reports whether the value is a canonical UUID, so slugs reach the
// not-found path instead of a PostgreSQL cast error.
func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			isHex := (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}

// Get returns one provider and its accounts.
func (s *ProviderService) Get(ctx context.Context, tenantID, providerID string) (Provider, error) {
	if !isUUID(providerID) {
		return Provider{}, ErrNotFound
	}
	var provider Provider
	err := s.pool.QueryRow(ctx, `
		SELECT p.id::text, p.name, p.kind, p.origin, coalesce(p.catalog_slug, ''), p.enabled, p.proxy_auto,
		       (SELECT count(*) FROM provider_accounts pa WHERE pa.provider_id = p.id AND pa.tenant_id = p.tenant_id),
		       (SELECT count(*) FROM provider_models pm WHERE pm.provider_id = p.id AND pm.tenant_id = p.tenant_id AND pm.active = true)
		FROM providers p
		WHERE p.tenant_id = $1 AND p.id = $2::uuid
	`, tenantID, providerID).Scan(&provider.ID, &provider.Name, &provider.Kind, &provider.Origin, &provider.CatalogSlug, &provider.Enabled, &provider.ProxyAuto, &provider.AccountsCount, &provider.ModelsCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Provider{}, ErrNotFound
	}
	if err != nil {
		return Provider{}, fmt.Errorf("query admin provider: %w", err)
	}
	accounts, err := s.accounts(ctx, tenantID, providerID)
	if err != nil {
		return Provider{}, err
	}
	provider.Accounts = accounts
	for _, account := range accounts {
		if account.Active {
			provider.UsableAccounts++
		}
	}
	if provider.UsableAccounts > 0 {
		provider.ModelsUsable = provider.ModelsCount
	}
	return provider, nil
}

func (s *ProviderService) accounts(ctx context.Context, tenantID, providerID string) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT pa.id::text, pa.name, pa.status, pa.base_url, pa.priority, pa.weight, pa.cooldown_until,
		       pa.proxy_mode,
		       pe.id::text, pe.label, pe.address, pe.port, pe.country_code, pe.city_name,
		       pe.valid, pe.quarantined_until, pp.banned_until
		FROM provider_accounts pa
		LEFT JOIN proxy_assignments pxa
		  ON pxa.tenant_id = pa.tenant_id AND pxa.provider_account_id = pa.id
		LEFT JOIN proxy_endpoints pe
		  ON pe.id = pxa.proxy_endpoint_id AND pe.tenant_id = pa.tenant_id
		LEFT JOIN proxy_poison pp
		  ON pp.tenant_id = pa.tenant_id
		 AND pp.provider_id = pa.provider_id
		 AND pp.proxy_endpoint_id = pe.id
		 AND pp.banned_until > now()
		WHERE pa.tenant_id = $1 AND pa.provider_id = $2::uuid
		ORDER BY pa.priority, pa.name
	`, tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("query provider accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]Account, 0)
	for rows.Next() {
		var account Account
		var endpointID, label, address, country, city *string
		var port *int
		var valid *bool
		var quarantinedUntil *time.Time
		var bannedUntil *time.Time
		if err := rows.Scan(
			&account.ID, &account.Name, &account.Status, &account.BaseURL, &account.Priority,
			&account.Weight, &account.CooldownUntil, &account.ProxyMode,
			&endpointID, &label, &address, &port, &country, &city, &valid, &quarantinedUntil, &bannedUntil,
		); err != nil {
			return nil, fmt.Errorf("scan provider account: %w", err)
		}
		if endpointID != nil {
			health := proxyHealth(derefBool(valid), quarantinedUntil)
			if bannedUntil != nil && bannedUntil.After(time.Now()) {
				health = "quarantined"
			}
			account.Proxy = &AccountProxy{
				ID:               *endpointID,
				Label:            deref(label),
				Address:          deref(address),
				Port:             derefInt(port),
				CountryCode:      deref(country),
				CityName:         deref(city),
				Health:           health,
				QuarantinedUntil: quarantinedUntil,
			}
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider accounts: %w", err)
	}
	modelCount, capabilities, err := s.providerModelSummary(ctx, tenantID, providerID)
	if err != nil {
		return nil, err
	}
	for index := range accounts {
		accounts[index].ModelsAvailable = modelCount
		accounts[index].Capabilities = capabilities
		accounts[index].Health = accountHealth(accounts[index])
		accounts[index].Active = accounts[index].Health == "active"
		if accounts[index].Active {
			accounts[index].ModelsUsable = modelCount
		}
	}
	return accounts, nil
}

func accountHealth(account Account) string {
	if account.Status == "disabled" {
		return "disabled"
	}
	if account.Status == "cooldown" || (account.CooldownUntil != nil && account.CooldownUntil.After(time.Now())) {
		return "cooldown"
	}
	if account.Proxy != nil {
		switch account.Proxy.Health {
		case "quarantined":
			return "quarantined"
		case "unhealthy":
			return "unhealthy"
		}
	}
	return "active"
}

func (s *ProviderService) providerModelSummary(ctx context.Context, tenantID, providerID string) (int, []string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT capabilities FROM provider_models
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND active = true
	`, tenantID, providerID)
	if err != nil {
		return 0, nil, fmt.Errorf("query provider model capabilities: %w", err)
	}
	defer rows.Close()
	count := 0
	seen := map[string]bool{}
	for rows.Next() {
		var capabilities map[string]any
		if err := rows.Scan(&capabilities); err != nil {
			return 0, nil, fmt.Errorf("scan provider model capabilities: %w", err)
		}
		count++
		for _, key := range []string{"text_input", "image_input", "audio_input", "video_input", "tools", "reasoning", "embeddings", "stream"} {
			if capabilities[key] == true {
				seen[key] = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("iterate provider model capabilities: %w", err)
	}
	order := []string{"text_input", "image_input", "audio_input", "video_input", "tools", "reasoning", "embeddings", "stream"}
	result := make([]string, 0, len(seen))
	for _, key := range order {
		if seen[key] {
			result = append(result, key)
		}
	}
	return count, result, nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func derefBool(value *bool) bool {
	return value != nil && *value
}

// SetProxyMode validates and persists the account proxy mode.
func (s *ProviderService) SetProxyMode(ctx context.Context, tenantID, providerID, accountID, raw string) (string, error) {
	mode, ok := provider.ParseProxyMode(raw)
	if !ok {
		return "", fmt.Errorf("%w: proxy_mode must be off, preferred or required", ErrInvalidInput)
	}
	if !isUUID(providerID) || !isUUID(accountID) {
		return "", ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE provider_accounts
		SET proxy_mode = $4
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND id = $3::uuid
	`, tenantID, providerID, accountID, string(mode))
	if err != nil {
		return "", fmt.Errorf("update proxy mode: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	return string(mode), nil
}

// SetProxyAuto enables or disables the provider proxy default (ADR-024).
func (s *ProviderService) SetProxyAuto(ctx context.Context, tenantID, providerID string, enabled bool) (bool, error) {
	if !isUUID(providerID) {
		return false, ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE providers SET proxy_auto = $3
		WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, providerID, enabled)
	if err != nil {
		return false, fmt.Errorf("update proxy_auto: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, ErrNotFound
	}
	return enabled, nil
}

// RenameAccount changes only the account name; the provider name is immutable.
func (s *ProviderService) RenameAccount(ctx context.Context, tenantID, providerID, accountID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return fmt.Errorf("%w: account name is required and must be at most 120 characters", ErrInvalidInput)
	}
	if !isUUID(providerID) || !isUUID(accountID) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE provider_accounts
		SET name = $4
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND id = $3::uuid
	`, tenantID, providerID, accountID, name)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: account name already exists", ErrConflict)
		}
		return fmt.Errorf("rename provider account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AccountSummaryResult is one account summary plus the provider aggregate.
type AccountSummaryResult struct {
	Provider Provider `json:"provider"`
	Account  Account  `json:"account"`
}

// AccountSummary returns the selected account summary after tenant/provider validation.
func (s *ProviderService) AccountSummary(ctx context.Context, tenantID, providerID, accountID string) (AccountSummaryResult, error) {
	provider, err := s.Get(ctx, tenantID, providerID)
	if err != nil {
		return AccountSummaryResult{}, err
	}
	for _, account := range provider.Accounts {
		if account.ID == accountID {
			return AccountSummaryResult{Provider: provider, Account: account}, nil
		}
	}
	return AccountSummaryResult{}, ErrNotFound
}

// AddAccount encrypts and inserts a provider account.
func (s *ProviderService) AddAccount(ctx context.Context, tenantID, providerID string, input AccountInput) (Account, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Credential = strings.TrimSpace(input.Credential)
	if input.Name == "" || input.Credential == "" {
		return Account{}, fmt.Errorf("%w: name and credential are required", ErrInvalidInput)
	}
	if input.Priority < 0 {
		return Account{}, fmt.Errorf("%w: priority must be non-negative", ErrInvalidInput)
	}
	if input.Priority == 0 {
		input.Priority = 100
	}
	if input.Weight <= 0 {
		input.Weight = 1
	}
	if s.box == nil {
		return Account{}, fmt.Errorf("credential box is not configured")
	}
	ciphertext, err := s.box.Encrypt([]byte(input.Credential))
	if err != nil {
		return Account{}, fmt.Errorf("encrypt credential: %w", err)
	}

	var account Account
	err = s.pool.QueryRow(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, credential_cipher, base_url, priority, weight)
		SELECT $1, p.id, $3, $4,
		       COALESCE(
		           NULLIF($5, ''),
		           CASE WHEN p.kind = 'agentrouter' THEN NULLIF($8, '') END,
		           (SELECT pc.default_base_url FROM provider_catalog pc WHERE pc.slug = p.catalog_slug),
		           (SELECT peer.base_url FROM provider_accounts peer
		            WHERE peer.provider_id = p.id AND peer.tenant_id = p.tenant_id AND peer.base_url <> ''
		            ORDER BY peer.priority, peer.created_at LIMIT 1),
		           ''
		       ),
		       $6, $7
		FROM providers p
		WHERE p.id = $2::uuid AND p.tenant_id = $1
		RETURNING id::text, name, status, base_url, priority, weight, cooldown_until, proxy_mode
	`, tenantID, providerID, input.Name, ciphertext, input.BaseURL, input.Priority, input.Weight, s.agentRouterRelayURL).Scan(
		&account.ID, &account.Name, &account.Status, &account.BaseURL, &account.Priority, &account.Weight, &account.CooldownUntil, &account.ProxyMode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("insert provider account: %w", err)
	}
	return account, nil
}

// DeleteAccount removes one account. When it was the provider's last account the
// provider is removed as well, so it leaves the configured providers list.
func (s *ProviderService) DeleteAccount(ctx context.Context, tenantID, providerID, accountID string) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin account delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		DELETE FROM provider_accounts
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND id = $3::uuid
	`, tenantID, providerID, accountID)
	if err != nil {
		return false, fmt.Errorf("delete provider account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, ErrNotFound
	}

	var remaining int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM provider_accounts
		WHERE tenant_id = $1 AND provider_id = $2::uuid
	`, tenantID, providerID).Scan(&remaining); err != nil {
		return false, fmt.Errorf("count provider accounts: %w", err)
	}
	providerDeleted := false
	if remaining == 0 {
		if _, err := tx.Exec(ctx, `
			DELETE FROM providers WHERE tenant_id = $1 AND id = $2::uuid
		`, tenantID, providerID); err != nil {
			return false, fmt.Errorf("delete empty provider: %w", err)
		}
		providerDeleted = true
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit account delete: %w", err)
	}
	return providerDeleted, nil
}

// Create inserts a provider, its first account and one model atomically.
func (s *ProviderService) Create(ctx context.Context, tenantID string, input CreateProviderInput) (Provider, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Origin = strings.ToLower(strings.TrimSpace(input.Origin))
	input.ModelID = strings.TrimSpace(input.ModelID)
	input.ModelName = strings.TrimSpace(input.ModelName)
	input.Credential = strings.TrimSpace(input.Credential)
	if input.Name == "" || input.ModelID == "" || input.Credential == "" {
		return Provider{}, fmt.Errorf("%w: name, model_id and credential are required", ErrInvalidInput)
	}
	switch input.Kind {
	case "openai", "anthropic", "agentrouter":
	default:
		return Provider{}, fmt.Errorf("%w: unsupported kind", ErrInvalidInput)
	}
	if input.Origin == "" {
		input.Origin = "native"
	}
	if input.Origin != "native" && input.Origin != "custom" {
		return Provider{}, fmt.Errorf("%w: origin must be native or custom", ErrInvalidInput)
	}
	if input.ModelName == "" {
		input.ModelName = input.ModelID
	}
	if input.Kind == "agentrouter" && input.BaseURL == "" && s.agentRouterRelayURL != "" {
		input.BaseURL = s.agentRouterRelayURL
	}
	if s.box == nil {
		return Provider{}, fmt.Errorf("credential box is not configured")
	}
	ciphertext, err := s.box.Encrypt([]byte(input.Credential))
	if err != nil {
		return Provider{}, fmt.Errorf("encrypt credential: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Provider{}, fmt.Errorf("begin provider create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var providerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO providers (tenant_id, name, kind, origin, enabled)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id::text
	`, tenantID, input.Name, input.Kind, input.Origin).Scan(&providerID)
	if err != nil {
		if strings.Contains(err.Error(), "providers_tenant_name_unique") {
			return Provider{}, ErrConflict
		}
		return Provider{}, fmt.Errorf("insert provider: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO provider_accounts (tenant_id, provider_id, name, credential_cipher, base_url, priority, weight)
		VALUES ($1, $2::uuid, $3, $4, $5, 100, 1)
	`, tenantID, providerID, input.Name+"-default", ciphertext, input.BaseURL); err != nil {
		return Provider{}, fmt.Errorf("insert provider account: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name)
		VALUES ($1, $2::uuid, $3, $4)
	`, tenantID, providerID, input.ModelID, input.ModelName); err != nil {
		return Provider{}, fmt.Errorf("insert provider model: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Provider{}, fmt.Errorf("commit provider create: %w", err)
	}
	// Import advertised models right away; a failing upstream must not block creation.
	_, _ = s.SyncModels(ctx, tenantID, providerID)
	return s.Get(ctx, tenantID, providerID)
}

// RemoteModel is a model reported by a provider account. Optional metadata is
// filled only when the provider's /models payload exposes it.
type RemoteModel struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	OwnedBy     string `json:"owned_by,omitempty"`
	Created     int64  `json:"created,omitempty"`

	Type          string `json:"type,omitempty"`
	ContextLength int    `json:"context_length,omitempty"`
	Pricing       struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing,omitempty"`
	Architecture struct {
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture,omitempty"`
	Modalities struct {
		Input  []string `json:"input"`
		Output []string `json:"output"`
	} `json:"modalities,omitempty"`
	Capabilities struct {
		Tools struct {
			Supported bool `json:"supported"`
		} `json:"tools"`
		Reasoning struct {
			Supported bool `json:"supported"`
		} `json:"reasoning"`
	} `json:"capabilities,omitempty"`
	Limit struct {
		Context int `json:"context"`
		Output  int `json:"output"`
	} `json:"limit,omitempty"`
	Cost struct {
		Text struct {
			Input struct {
				Price float64 `json:"price"`
			} `json:"input"`
			Output struct {
				Price float64 `json:"price"`
			} `json:"output"`
		} `json:"text"`
	} `json:"cost,omitempty"`
	SupportedParameters []string `json:"supported_parameters,omitempty"`
	TopProvider         struct {
		MaxCompletionTokens int `json:"max_completion_tokens"`
	} `json:"top_provider,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`
}

// ProviderTestResult is a sanitized connectivity check.
type ProviderTestResult struct {
	OK         bool   `json:"ok"`
	Status     int    `json:"status"`
	LatencyMS  int64  `json:"latency_ms"`
	ModelCount int    `json:"model_count"`
	Error      string `json:"error,omitempty"`
}

// AccountTestResult reports one account without affecting provider state.
type AccountTestResult struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	OK          bool   `json:"ok"`
	Status      int    `json:"status"`
	LatencyMS   int64  `json:"latency_ms"`
	ModelCount  int    `json:"model_count"`
	Error       string `json:"error,omitempty"`
}

// Test checks the provider account against its models endpoint.
func (s *ProviderService) Test(ctx context.Context, tenantID, providerID string) (ProviderTestResult, error) {
	started := time.Now()
	models, status, err := s.remoteModels(ctx, tenantID, providerID)
	result := ProviderTestResult{OK: err == nil, Status: status, LatencyMS: time.Since(started).Milliseconds(), ModelCount: len(models)}
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	return result, nil
}

// RemoteModels returns models currently advertised by the provider account.
func (s *ProviderService) RemoteModels(ctx context.Context, tenantID, providerID string) ([]RemoteModel, error) {
	models, _, err := s.remoteModels(ctx, tenantID, providerID)
	return models, err
}

// ImportModels persists selected remote models for the provider.
func (s *ProviderService) ImportModels(ctx context.Context, tenantID, providerID string, ids []string) (int, error) {
	models, _, err := s.remoteModels(ctx, tenantID, providerID)
	if err != nil {
		return 0, err
	}
	allowed := make(map[string]RemoteModel, len(models))
	for _, model := range models {
		allowed[model.ID] = model
	}
	imported := 0
	for _, id := range ids {
		id = strings.TrimSpace(id)
		model, ok := allowed[id]
		if !ok {
			continue
		}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = id
		}
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, active)
			SELECT $1, p.id, $3, $4, true
			FROM providers p
			WHERE p.id = $2::uuid AND p.tenant_id = $1
			ON CONFLICT (tenant_id, provider_id, model_id) DO UPDATE SET active = true, display_name = EXCLUDED.display_name
		`, tenantID, providerID, id, name); err != nil {
			return imported, fmt.Errorf("import provider model: %w", err)
		}
		imported++
	}
	return imported, nil
}

type accountTarget struct {
	ID               string
	Name             string
	Kind             string
	BaseURL          string
	CatalogSlug      string
	ModelsPath       string
	CredentialCipher []byte
}

func (s *ProviderService) loadAccountTarget(ctx context.Context, tenantID, providerID, accountID string) (accountTarget, error) {
	if !isUUID(providerID) || (accountID != "" && !isUUID(accountID)) {
		return accountTarget{}, ErrNotFound
	}
	var target accountTarget
	err := s.pool.QueryRow(ctx, `
		SELECT pa.id::text, pa.name, p.kind, pa.base_url, coalesce(p.catalog_slug, ''),
		       coalesce(pc.models_path, ''), pa.credential_cipher
		FROM providers p
		JOIN provider_accounts pa ON pa.provider_id = p.id AND pa.tenant_id = p.tenant_id
		LEFT JOIN provider_catalog pc ON pc.slug = p.catalog_slug
		WHERE p.tenant_id = $1 AND p.id = $2::uuid AND pa.status = 'active'
		  AND ($3::text = '' OR pa.id::text = $3)
		ORDER BY pa.priority, pa.created_at
		LIMIT 1
	`, tenantID, providerID, accountID).Scan(
		&target.ID, &target.Name, &target.Kind, &target.BaseURL, &target.CatalogSlug,
		&target.ModelsPath, &target.CredentialCipher,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return accountTarget{}, ErrNotFound
	}
	if err != nil {
		return accountTarget{}, fmt.Errorf("load provider account: %w", err)
	}
	return target, nil
}

func (s *ProviderService) accountTargets(ctx context.Context, tenantID, providerID string) ([]accountTarget, error) {
	if !isUUID(providerID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		SELECT pa.id::text, pa.name, p.kind, pa.base_url, coalesce(p.catalog_slug, ''),
		       coalesce(pc.models_path, ''), pa.credential_cipher
		FROM providers p
		JOIN provider_accounts pa ON pa.provider_id = p.id AND pa.tenant_id = p.tenant_id
		LEFT JOIN provider_catalog pc ON pc.slug = p.catalog_slug
		WHERE p.tenant_id = $1 AND p.id = $2::uuid AND pa.status = 'active'
		ORDER BY pa.priority, pa.created_at
	`, tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("query provider accounts: %w", err)
	}
	defer rows.Close()
	targets := make([]accountTarget, 0)
	for rows.Next() {
		var target accountTarget
		if err := rows.Scan(
			&target.ID, &target.Name, &target.Kind, &target.BaseURL, &target.CatalogSlug,
			&target.ModelsPath, &target.CredentialCipher,
		); err != nil {
			return nil, fmt.Errorf("scan provider account: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider accounts: %w", err)
	}
	if len(targets) == 0 {
		return nil, ErrNotFound
	}
	return targets, nil
}

// TestAccount checks one account's own models endpoint. It never mutates the
// provider or other accounts.
func (s *ProviderService) TestAccount(ctx context.Context, tenantID, providerID, accountID string) (AccountTestResult, error) {
	target, err := s.loadAccountTarget(ctx, tenantID, providerID, accountID)
	if err != nil {
		return AccountTestResult{}, err
	}
	started := time.Now()
	models, status, testErr := s.fetchRemoteModels(ctx, target)
	result := AccountTestResult{
		AccountID: target.ID, AccountName: target.Name, OK: testErr == nil,
		Status: status, LatencyMS: time.Since(started).Milliseconds(), ModelCount: len(models),
	}
	if testErr != nil {
		result.Error = accountTestMessage(status, testErr)
	}
	return result, nil
}

func accountTestMessage(status int, err error) string {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return "account credential rejected by provider"
	case status == http.StatusTooManyRequests:
		return "account rate limit reached"
	case status >= 500:
		return "provider unavailable for this account"
	case errors.Is(err, ErrModelsUnsupported):
		return "provider does not expose a models endpoint"
	default:
		return "account models endpoint failed"
	}
}

func (s *ProviderService) remoteModelsForAccount(ctx context.Context, tenantID, providerID, accountID string) ([]RemoteModel, int, error) {
	target, err := s.loadAccountTarget(ctx, tenantID, providerID, accountID)
	if err != nil {
		return nil, 0, err
	}
	return s.fetchRemoteModels(ctx, target)
}

func (s *ProviderService) remoteModels(ctx context.Context, tenantID, providerID string) ([]RemoteModel, int, error) {
	return s.remoteModelsForAccount(ctx, tenantID, providerID, "")
}

func (s *ProviderService) fetchRemoteModels(ctx context.Context, target accountTarget) ([]RemoteModel, int, error) {
	if s.box == nil {
		return nil, 0, fmt.Errorf("credential box is not configured")
	}
	credential, err := s.box.Decrypt(target.CredentialCipher)
	if err != nil {
		return nil, 0, fmt.Errorf("decrypt credential: %w", err)
	}
	url := ""
	if target.CatalogSlug != "" {
		if target.ModelsPath == "" {
			return nil, 0, ErrModelsUnsupported
		}
		url = joinModelsURL(target.BaseURL, target.ModelsPath)
	} else {
		url = modelsURL(target.Kind, target.BaseURL)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create models request: %w", err)
	}
	switch strings.ToLower(target.Kind) {
	case "anthropic", "agentrouter":
		request.Header.Set("x-api-key", string(credential))
		request.Header.Set("anthropic-version", "2023-06-01")
	default:
		request.Header.Set("Authorization", "Bearer "+string(credential))
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("provider models request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, response.StatusCode, fmt.Errorf("provider models returned %d", response.StatusCode)
	}
	var payload struct {
		Data []RemoteModel `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, response.StatusCode, fmt.Errorf("decode provider models: %w", err)
	}
	models := make([]RemoteModel, 0, len(payload.Data))
	for _, item := range payload.Data {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		if item.Name == "" {
			item.Name = item.DisplayName
		}
		models = append(models, item)
	}
	return models, response.StatusCode, nil
}

func modelsURL(kind, baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	switch strings.ToLower(kind) {
	case "anthropic", "agentrouter":
		if strings.HasSuffix(base, "/v1") {
			return base + "/models"
		}
		return base + "/v1/models"
	default:
		return base + "/models"
	}
}

// joinModelsURL appends a catalog-declared models path to a provider base URL.
func joinModelsURL(baseURL, modelsPath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path := strings.TrimSpace(modelsPath)
	if path == "" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}
