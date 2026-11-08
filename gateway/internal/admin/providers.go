// Package admin owns the read/write admin API used by the console.
package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
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
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Kind          string    `json:"kind"`
	Origin        string    `json:"origin"`
	Enabled       bool      `json:"enabled"`
	AccountsCount int       `json:"accounts_count"`
	ModelsCount   int       `json:"models_count"`
	Accounts      []Account `json:"accounts,omitempty"`
}

// Account is the admin representation of a provider account. Credentials are
// never returned.
type Account struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	BaseURL       string     `json:"base_url"`
	Priority      int        `json:"priority"`
	Weight        int        `json:"weight"`
	CooldownUntil *time.Time `json:"cooldown_until,omitempty"`
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
	pool *pgxpool.Pool
	box  *secret.Box
}

// NewProviderService creates a provider service.
func NewProviderService(pool *pgxpool.Pool, box *secret.Box) *ProviderService {
	return &ProviderService{pool: pool, box: box}
}

// List returns configured providers for a tenant.
func (s *ProviderService) List(ctx context.Context, tenantID string) ([]Provider, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id::text, p.name, p.kind, p.origin, p.enabled,
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
		if err := rows.Scan(&provider.ID, &provider.Name, &provider.Kind, &provider.Origin, &provider.Enabled, &provider.AccountsCount, &provider.ModelsCount); err != nil {
			return nil, fmt.Errorf("scan admin provider: %w", err)
		}
		providers = append(providers, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin providers: %w", err)
	}
	return providers, nil
}

// Get returns one provider and its accounts.
func (s *ProviderService) Get(ctx context.Context, tenantID, providerID string) (Provider, error) {
	var provider Provider
	err := s.pool.QueryRow(ctx, `
		SELECT p.id::text, p.name, p.kind, p.origin, p.enabled,
		       (SELECT count(*) FROM provider_accounts pa WHERE pa.provider_id = p.id AND pa.tenant_id = p.tenant_id),
		       (SELECT count(*) FROM provider_models pm WHERE pm.provider_id = p.id AND pm.tenant_id = p.tenant_id AND pm.active = true)
		FROM providers p
		WHERE p.tenant_id = $1 AND p.id = $2::uuid
	`, tenantID, providerID).Scan(&provider.ID, &provider.Name, &provider.Kind, &provider.Origin, &provider.Enabled, &provider.AccountsCount, &provider.ModelsCount)
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
	return provider, nil
}

func (s *ProviderService) accounts(ctx context.Context, tenantID, providerID string) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, status, base_url, priority, weight, cooldown_until
		FROM provider_accounts
		WHERE tenant_id = $1 AND provider_id = $2::uuid
		ORDER BY priority, name
	`, tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("query provider accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]Account, 0)
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.Name, &account.Status, &account.BaseURL, &account.Priority, &account.Weight, &account.CooldownUntil); err != nil {
			return nil, fmt.Errorf("scan provider account: %w", err)
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider accounts: %w", err)
	}
	return accounts, nil
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
		SELECT $1, p.id, $3, $4, $5, $6, $7
		FROM providers p
		WHERE p.id = $2::uuid AND p.tenant_id = $1
		RETURNING id::text, name, status, base_url, priority, weight, cooldown_until
	`, tenantID, providerID, input.Name, ciphertext, input.BaseURL, input.Priority, input.Weight).Scan(
		&account.ID, &account.Name, &account.Status, &account.BaseURL, &account.Priority, &account.Weight, &account.CooldownUntil,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("insert provider account: %w", err)
	}
	return account, nil
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
	return s.Get(ctx, tenantID, providerID)
}
