package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProxyLimit is returned when a proxy already serves the maximum accounts.
var ErrProxyLimit = errors.New("proxy account limit reached")

const maxAccountsPerProxy = 3

// ProxyEndpointAccount is a sanitized account assigned to a proxy.
type ProxyEndpointAccount struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProviderID   string `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	ProxyMode    string `json:"proxy_mode"`
}

// ProxyEndpoint is the admin representation of a proxy in the pool.
// Credentials are never returned.
type ProxyEndpoint struct {
	ID               string                 `json:"id"`
	Label            string                 `json:"label"`
	Address          string                 `json:"address"`
	Port             int                    `json:"port"`
	CountryCode      string                 `json:"country_code"`
	CityName         string                 `json:"city_name"`
	Source           string                 `json:"source"`
	Valid            bool                   `json:"valid"`
	Health           string                 `json:"health"`
	QuarantinedUntil *time.Time             `json:"quarantined_until,omitempty"`
	LastVerifiedAt   *time.Time             `json:"last_verified_at,omitempty"`
	AssignedCount    int                    `json:"assigned_count"`
	Accounts         []ProxyEndpointAccount `json:"accounts"`
}

// AccountProxy is the sanitized proxy attached to a provider account.
type AccountProxy struct {
	ID               string     `json:"id"`
	Label            string     `json:"label"`
	Address          string     `json:"address"`
	Port             int        `json:"port"`
	CountryCode      string     `json:"country_code"`
	CityName         string     `json:"city_name"`
	Health           string     `json:"health"`
	QuarantinedUntil *time.Time `json:"quarantined_until,omitempty"`
}

// CreateProxyEndpointInput registers a proxy manually.
type CreateProxyEndpointInput struct {
	Label       string `json:"label"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	CountryCode string `json:"country_code"`
	CityName    string `json:"city_name"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

// ProxyService is the PostgreSQL-backed proxy pool service.
type ProxyService struct {
	pool *pgxpool.Pool
	box  *secret.Box
}

// NewProxyService creates a proxy service.
func NewProxyService(pool *pgxpool.Pool, box *secret.Box) *ProxyService {
	return &ProxyService{pool: pool, box: box}
}

// List returns the tenant proxy pool without credentials.
func (s *ProxyService) List(ctx context.Context, tenantID string) ([]ProxyEndpoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT pe.id::text, pe.label, pe.address, pe.port, pe.country_code, pe.city_name,
		       pe.source, pe.valid, pe.quarantined_until, pe.last_verified_at,
		       (SELECT count(*) FROM proxy_assignments pxa
		        WHERE pxa.tenant_id = pe.tenant_id AND pxa.proxy_endpoint_id = pe.id)
		FROM proxy_endpoints pe
		WHERE pe.tenant_id = $1
		ORDER BY pe.created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query proxy endpoints: %w", err)
	}
	defer rows.Close()

	endpoints := make([]ProxyEndpoint, 0)
	for rows.Next() {
		var endpoint ProxyEndpoint
		if err := rows.Scan(
			&endpoint.ID, &endpoint.Label, &endpoint.Address, &endpoint.Port,
			&endpoint.CountryCode, &endpoint.CityName, &endpoint.Source, &endpoint.Valid,
			&endpoint.QuarantinedUntil, &endpoint.LastVerifiedAt, &endpoint.AssignedCount,
		); err != nil {
			return nil, fmt.Errorf("scan proxy endpoint: %w", err)
		}
		endpoint.Health = proxyHealth(endpoint.Valid, endpoint.QuarantinedUntil)
		endpoint.Accounts = make([]ProxyEndpointAccount, 0)
		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy endpoints: %w", err)
	}
	rows.Close()

	index := make(map[string]int, len(endpoints))
	for position, endpoint := range endpoints {
		index[endpoint.ID] = position
	}
	assigned, err := s.pool.Query(ctx, `
		SELECT pxa.proxy_endpoint_id::text, pa.id::text, pa.name, p.id::text, p.name, pa.proxy_mode
		FROM proxy_assignments pxa
		JOIN provider_accounts pa
		  ON pa.id = pxa.provider_account_id AND pa.tenant_id = pxa.tenant_id
		JOIN providers p
		  ON p.id = pa.provider_id AND p.tenant_id = pa.tenant_id
		WHERE pxa.tenant_id = $1
		ORDER BY p.name, pa.name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query proxy assignments: %w", err)
	}
	defer assigned.Close()
	for assigned.Next() {
		var endpointID string
		var account ProxyEndpointAccount
		if err := assigned.Scan(&endpointID, &account.ID, &account.Name, &account.ProviderID, &account.ProviderName, &account.ProxyMode); err != nil {
			return nil, fmt.Errorf("scan proxy assignment: %w", err)
		}
		if position, ok := index[endpointID]; ok {
			endpoints[position].Accounts = append(endpoints[position].Accounts, account)
		}
	}
	if err := assigned.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy assignments: %w", err)
	}
	return endpoints, nil
}

// Create registers a manual proxy endpoint. Credentials are encrypted and
// never returned.
func (s *ProxyService) Create(ctx context.Context, tenantID string, input CreateProxyEndpointInput) (ProxyEndpoint, error) {
	input.Label = strings.TrimSpace(input.Label)
	input.Address = strings.TrimSpace(input.Address)
	input.CountryCode = strings.ToUpper(strings.TrimSpace(input.CountryCode))
	input.CityName = strings.TrimSpace(input.CityName)
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" && input.Password != "" {
		return ProxyEndpoint{}, fmt.Errorf("%w: username is required with password", ErrInvalidInput)
	}
	if input.Username == "" {
		input.Password = ""
	}
	if len(input.Label) > 120 || len(input.CityName) > 120 || len(input.Username) > 256 || len(input.Password) > 512 {
		return ProxyEndpoint{}, fmt.Errorf("%w: proxy fields are too long", ErrInvalidInput)
	}
	if len(input.CountryCode) > 3 {
		return ProxyEndpoint{}, fmt.Errorf("%w: country code is invalid", ErrInvalidInput)
	}
	if _, err := provider.BuildProxyURL(input.Address, input.Port, input.Username, input.Password); err != nil {
		return ProxyEndpoint{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	var usernameCipher, passwordCipher []byte
	if input.Username != "" {
		if s.box == nil {
			return ProxyEndpoint{}, fmt.Errorf("credential box is not configured")
		}
		var err error
		usernameCipher, err = s.box.Encrypt([]byte(input.Username))
		if err != nil {
			return ProxyEndpoint{}, fmt.Errorf("encrypt proxy username: %w", err)
		}
		passwordCipher, err = s.box.Encrypt([]byte(input.Password))
		if err != nil {
			return ProxyEndpoint{}, fmt.Errorf("encrypt proxy password: %w", err)
		}
	}

	var endpoint ProxyEndpoint
	err := s.pool.QueryRow(ctx, `
		INSERT INTO proxy_endpoints
		    (tenant_id, label, address, port, country_code, city_name, username_cipher, password_cipher, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'manual')
		RETURNING id::text, label, address, port, country_code, city_name, source, valid,
		          quarantined_until, last_verified_at
	`, tenantID, input.Label, input.Address, input.Port, input.CountryCode, input.CityName, usernameCipher, passwordCipher).Scan(
		&endpoint.ID, &endpoint.Label, &endpoint.Address, &endpoint.Port, &endpoint.CountryCode,
		&endpoint.CityName, &endpoint.Source, &endpoint.Valid, &endpoint.QuarantinedUntil, &endpoint.LastVerifiedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ProxyEndpoint{}, fmt.Errorf("%w: proxy endpoint already exists", ErrConflict)
		}
		return ProxyEndpoint{}, fmt.Errorf("insert proxy endpoint: %w", err)
	}
	endpoint.Health = proxyHealth(endpoint.Valid, endpoint.QuarantinedUntil)
	return endpoint, nil
}

// Assign attaches a proxy to an account, replacing any previous assignment.
func (s *ProxyService) Assign(ctx context.Context, tenantID, providerID, accountID, endpointID string) error {
	if !isUUID(providerID) || !isUUID(accountID) || !isUUID(endpointID) {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin proxy assignment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists int
	if err := tx.QueryRow(ctx, `
		SELECT 1 FROM provider_accounts
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND id = $3::uuid
	`, tenantID, providerID, accountID).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("query provider account: %w", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT 1 FROM proxy_endpoints WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, endpointID).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("query proxy endpoint: %w", err)
	}

	var assigned int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM proxy_assignments
		WHERE tenant_id = $1 AND proxy_endpoint_id = $2::uuid AND provider_account_id <> $3::uuid
	`, tenantID, endpointID, accountID).Scan(&assigned); err != nil {
		return fmt.Errorf("count proxy assignments: %w", err)
	}
	if assigned >= maxAccountsPerProxy {
		return ErrProxyLimit
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO proxy_assignments (tenant_id, provider_account_id, proxy_endpoint_id)
		VALUES ($1, $2::uuid, $3::uuid)
		ON CONFLICT (tenant_id, provider_account_id)
		DO UPDATE SET proxy_endpoint_id = excluded.proxy_endpoint_id, updated_at = now()
	`, tenantID, accountID, endpointID); err != nil {
		return fmt.Errorf("assign proxy endpoint: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit proxy assignment: %w", err)
	}
	return nil
}

// Unassign removes the account proxy assignment.
func (s *ProxyService) Unassign(ctx context.Context, tenantID, providerID, accountID string) error {
	if !isUUID(providerID) || !isUUID(accountID) {
		return ErrNotFound
	}
	var exists int
	if err := s.pool.QueryRow(ctx, `
		SELECT 1 FROM provider_accounts
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND id = $3::uuid
	`, tenantID, providerID, accountID).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("query provider account: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM proxy_assignments
		WHERE tenant_id = $1 AND provider_account_id = $2::uuid
	`, tenantID, accountID); err != nil {
		return fmt.Errorf("delete proxy assignment: %w", err)
	}
	return nil
}

func proxyHealth(valid bool, quarantinedUntil *time.Time) string {
	if quarantinedUntil != nil && quarantinedUntil.After(time.Now()) {
		return "quarantined"
	}
	if !valid {
		return "unhealthy"
	}
	return "healthy"
}

func isUniqueViolation(err error) bool {
	type sqlState interface{ SQLState() string }
	var state sqlState
	if errors.As(err, &state) {
		return state.SQLState() == "23505"
	}
	return false
}
