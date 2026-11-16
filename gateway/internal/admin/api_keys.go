package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKey is the admin representation of an API key. The secret is never
// returned here.
type APIKey struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Prefix          string     `json:"prefix"`
	Scopes          []string   `json:"scopes"`
	Status          string     `json:"status"`
	IPAllowlist     []string   `json:"ip_allowlist"`
	RPMLimit        *int       `json:"rpm_limit"`
	TPMLimit        *int       `json:"tpm_limit"`
	USDMonthlyLimit *float64   `json:"usd_monthly_limit"`
	USDDailyLimit   *float64   `json:"usd_daily_limit"`
	USDWeeklyLimit  *float64   `json:"usd_weekly_limit"`
	ExpiresAt       *time.Time `json:"expires_at"`
	RevokedAt       *time.Time `json:"revoked_at"`
	LastRotatedAt   *time.Time `json:"last_rotated_at"`
	CreatedAt       time.Time  `json:"created_at"`
	LastUsedAt      *time.Time `json:"last_used_at"`
}

// CreatedAPIKey is returned once, with the full secret.
type CreatedAPIKey struct {
	APIKey
	Key string `json:"key"`
}

// APIKeyInput is the full editable representation of an API key.
type APIKeyInput struct {
	Name            string     `json:"name"`
	Scopes          []string   `json:"scopes"`
	IPAllowlist     []string   `json:"ip_allowlist"`
	RPMLimit        *int       `json:"rpm_limit"`
	TPMLimit        *int       `json:"tpm_limit"`
	USDMonthlyLimit *float64   `json:"usd_monthly_limit"`
	USDDailyLimit   *float64   `json:"usd_daily_limit"`
	USDWeeklyLimit  *float64   `json:"usd_weekly_limit"`
	ExpiresAt       *time.Time `json:"expires_at"`
}

// CreateAPIKeyInput creates an API key.
type CreateAPIKeyInput = APIKeyInput

// UpdateAPIKeyInput replaces editable API key metadata.
type UpdateAPIKeyInput = APIKeyInput

var apiKeyScopeCatalog = []string{
	"chat:write",
	"models:read",
	"embeddings:write",
	"web:read",
	"usage:read",
	"combos:read",
	"combos:write",
	"manage",
}

// Scopes returns the supported scopes for API key creation.
func (s *APIKeyService) Scopes() []string {
	return Scopes()
}

// Scopes returns the supported API key scopes.
func Scopes() []string {
	result := make([]string, len(apiKeyScopeCatalog))
	copy(result, apiKeyScopeCatalog)
	return result
}

func knownScope(scope string) bool {
	for _, allowed := range apiKeyScopeCatalog {
		if scope == allowed {
			return true
		}
	}
	return false
}

// APIKeyService is the PostgreSQL-backed admin API key service.
type APIKeyService struct {
	pool *pgxpool.Pool
}

// NewAPIKeyService creates an API key service.
func NewAPIKeyService(pool *pgxpool.Pool) *APIKeyService {
	return &APIKeyService{pool: pool}
}

const apiKeyColumns = `
	id::text, name, key_prefix, scopes::text, ip_allowlist, expires_at, revoked_at,
	created_at, rotated_at, rpm_limit, tpm_limit,
	usd_monthly_limit::float8, usd_daily_limit::float8, usd_weekly_limit::float8,
	(SELECT max(r.created_at) FROM requests r WHERE r.api_key_id = k.id AND r.tenant_id = k.tenant_id)
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(row rowScanner) (APIKey, error) {
	var key APIKey
	var scopesJSON string
	if err := row.Scan(
		&key.ID, &key.Name, &key.Prefix, &scopesJSON, &key.IPAllowlist, &key.ExpiresAt,
		&key.RevokedAt, &key.CreatedAt, &key.LastRotatedAt, &key.RPMLimit, &key.TPMLimit,
		&key.USDMonthlyLimit, &key.USDDailyLimit, &key.USDWeeklyLimit, &key.LastUsedAt,
	); err != nil {
		return APIKey{}, err
	}
	if err := json.Unmarshal([]byte(scopesJSON), &key.Scopes); err != nil {
		return APIKey{}, fmt.Errorf("decode API key scopes: %w", err)
	}
	if key.IPAllowlist == nil {
		key.IPAllowlist = []string{}
	}
	key.Status = apiKeyStatus(key, time.Now())
	return key, nil
}

func apiKeyStatus(key APIKey, now time.Time) string {
	if key.RevokedAt != nil {
		return "revoked"
	}
	if key.ExpiresAt != nil && !now.Before(*key.ExpiresAt) {
		return "expired"
	}
	return "active"
}

// List returns API keys for a tenant.
func (s *APIKeyService) List(ctx context.Context, tenantID string) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+apiKeyColumns+` FROM api_keys k WHERE k.tenant_id = $1 ORDER BY k.created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query admin API keys: %w", err)
	}
	defer rows.Close()

	keys := make([]APIKey, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan admin API key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin API keys: %w", err)
	}
	return keys, nil
}

// Get returns one API key.
func (s *APIKeyService) Get(ctx context.Context, tenantID, keyID string) (APIKey, error) {
	key, err := scanAPIKey(s.pool.QueryRow(ctx, `SELECT `+apiKeyColumns+` FROM api_keys k WHERE k.tenant_id = $1 AND k.id = $2::uuid`, tenantID, keyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return APIKey{}, ErrNotFound
	}
	if err != nil {
		return APIKey{}, fmt.Errorf("load API key: %w", err)
	}
	return key, nil
}

// Create generates, hashes and persists a new API key.
func (s *APIKeyService) Create(ctx context.Context, tenantID, actor string, input CreateAPIKeyInput) (CreatedAPIKey, error) {
	normalized, err := normalizeAPIKeyInput(input)
	if err != nil {
		return CreatedAPIKey{}, err
	}
	secret, err := generateAPIKey()
	if err != nil {
		return CreatedAPIKey{}, err
	}
	hash, err := auth.HashKey(secret)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("hash API key: %w", err)
	}
	scopesJSON, err := json.Marshal(normalized.Scopes)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("encode API key scopes: %w", err)
	}
	var keyID string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO api_keys (
			tenant_id, name, key_prefix, key_hash, scopes, expires_at, ip_allowlist,
			rpm_limit, tpm_limit, usd_monthly_limit, usd_daily_limit, usd_weekly_limit
		)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id::text
	`, tenantID, normalized.Name, secret[:auth.PrefixLength], hash, string(scopesJSON),
		normalized.ExpiresAt, normalized.IPAllowlist, normalized.RPMLimit, normalized.TPMLimit,
		normalized.USDMonthlyLimit, normalized.USDDailyLimit, normalized.USDWeeklyLimit).Scan(&keyID)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("insert API key: %w", err)
	}
	created, err := s.Get(ctx, tenantID, keyID)
	if err != nil {
		return CreatedAPIKey{}, err
	}
	s.audit(ctx, tenantID, actor, "api_key.create", auditPayload(created))
	return CreatedAPIKey{APIKey: created, Key: secret}, nil
}

// Update replaces editable metadata without changing the secret.
func (s *APIKeyService) Update(ctx context.Context, tenantID, actor, keyID string, input UpdateAPIKeyInput) (APIKey, error) {
	normalized, err := normalizeAPIKeyInput(input)
	if err != nil {
		return APIKey{}, err
	}
	scopesJSON, err := json.Marshal(normalized.Scopes)
	if err != nil {
		return APIKey{}, fmt.Errorf("encode API key scopes: %w", err)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE api_keys
		SET name = $3, scopes = $4::jsonb, expires_at = $5, ip_allowlist = $6,
		    rpm_limit = $7, tpm_limit = $8, usd_monthly_limit = $9,
		    usd_daily_limit = $10, usd_weekly_limit = $11
		WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, keyID, normalized.Name, string(scopesJSON), normalized.ExpiresAt,
		normalized.IPAllowlist, normalized.RPMLimit, normalized.TPMLimit,
		normalized.USDMonthlyLimit, normalized.USDDailyLimit, normalized.USDWeeklyLimit)
	if err != nil {
		return APIKey{}, fmt.Errorf("update API key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return APIKey{}, ErrNotFound
	}
	updated, err := s.Get(ctx, tenantID, keyID)
	if err != nil {
		return APIKey{}, err
	}
	s.audit(ctx, tenantID, actor, "api_key.update", auditPayload(updated))
	return updated, nil
}

// Rotate replaces the secret while preserving metadata.
func (s *APIKeyService) Rotate(ctx context.Context, tenantID, actor, keyID string) (CreatedAPIKey, error) {
	secret, err := generateAPIKey()
	if err != nil {
		return CreatedAPIKey{}, err
	}
	hash, err := auth.HashKey(secret)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("hash API key: %w", err)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE api_keys
		SET key_prefix = $3, key_hash = $4, rotated_at = now()
		WHERE tenant_id = $1 AND id = $2::uuid AND revoked_at IS NULL
	`, tenantID, keyID, secret[:auth.PrefixLength], hash)
	if err != nil {
		return CreatedAPIKey{}, fmt.Errorf("rotate API key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.Get(ctx, tenantID, keyID); errors.Is(err, ErrNotFound) {
			return CreatedAPIKey{}, ErrNotFound
		}
		return CreatedAPIKey{}, ErrConflict
	}
	rotated, err := s.Get(ctx, tenantID, keyID)
	if err != nil {
		return CreatedAPIKey{}, err
	}
	s.audit(ctx, tenantID, actor, "api_key.rotate", auditPayload(rotated))
	return CreatedAPIKey{APIKey: rotated, Key: secret}, nil
}

// Revoke marks an API key as revoked.
func (s *APIKeyService) Revoke(ctx context.Context, tenantID, actor, keyID string) (APIKey, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE api_keys SET revoked_at = now()
		WHERE tenant_id = $1 AND id = $2::uuid AND revoked_at IS NULL
	`, tenantID, keyID)
	if err != nil {
		return APIKey{}, fmt.Errorf("revoke API key: %w", err)
	}
	revoked, err := s.Get(ctx, tenantID, keyID)
	if err != nil {
		return APIKey{}, err
	}
	if tag.RowsAffected() > 0 {
		s.audit(ctx, tenantID, actor, "api_key.revoke", auditPayload(revoked))
	}
	return revoked, nil
}

func normalizeAPIKeyInput(input APIKeyInput) (APIKeyInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 80 {
		return APIKeyInput{}, fmt.Errorf("%w: name is required and must be at most 80 characters", ErrInvalidInput)
	}
	scopes := make([]string, 0, len(input.Scopes))
	seenScopes := map[string]bool{}
	for _, scope := range input.Scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" || seenScopes[trimmed] {
			continue
		}
		if !knownScope(trimmed) {
			return APIKeyInput{}, fmt.Errorf("%w: unknown scope %q", ErrInvalidInput, trimmed)
		}
		seenScopes[trimmed] = true
		scopes = append(scopes, trimmed)
	}
	if len(scopes) == 0 {
		return APIKeyInput{}, fmt.Errorf("%w: at least one scope is required", ErrInvalidInput)
	}
	input.Scopes = scopes

	allowlist, err := normalizeIPAllowlist(input.IPAllowlist)
	if err != nil {
		return APIKeyInput{}, err
	}
	input.IPAllowlist = allowlist

	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		return APIKeyInput{}, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidInput)
	}
	if input.RPMLimit != nil && *input.RPMLimit < 0 {
		return APIKeyInput{}, fmt.Errorf("%w: rpm_limit must be non-negative", ErrInvalidInput)
	}
	if input.TPMLimit != nil && *input.TPMLimit < 0 {
		return APIKeyInput{}, fmt.Errorf("%w: tpm_limit must be non-negative", ErrInvalidInput)
	}
	if input.USDMonthlyLimit != nil && *input.USDMonthlyLimit < 0 {
		return APIKeyInput{}, fmt.Errorf("%w: usd_monthly_limit must be non-negative", ErrInvalidInput)
	}
	if input.USDDailyLimit != nil && *input.USDDailyLimit < 0 {
		return APIKeyInput{}, fmt.Errorf("%w: usd_daily_limit must be non-negative", ErrInvalidInput)
	}
	if input.USDWeeklyLimit != nil && *input.USDWeeklyLimit < 0 {
		return APIKeyInput{}, fmt.Errorf("%w: usd_weekly_limit must be non-negative", ErrInvalidInput)
	}
	return input, nil
}

func normalizeIPAllowlist(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		if strings.Contains(trimmed, "/") {
			if _, _, err := net.ParseCIDR(trimmed); err != nil {
				return nil, fmt.Errorf("%w: invalid CIDR %q", ErrInvalidInput, trimmed)
			}
		} else if net.ParseIP(trimmed) == nil {
			return nil, fmt.Errorf("%w: invalid IP %q", ErrInvalidInput, trimmed)
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	if len(result) > 64 {
		return nil, fmt.Errorf("%w: at most 64 IP/CIDR entries are allowed", ErrInvalidInput)
	}
	return result, nil
}

func auditPayload(key APIKey) map[string]any {
	return map[string]any{
		"api_key_id":      key.ID,
		"name":            key.Name,
		"scopes":          key.Scopes,
		"ip_allowlist":    key.IPAllowlist,
		"rpm_limit":       key.RPMLimit,
		"tpm_limit":       key.TPMLimit,
		"usd_monthly":     key.USDMonthlyLimit,
		"usd_daily":       key.USDDailyLimit,
		"usd_weekly":      key.USDWeeklyLimit,
		"expires_at":      key.ExpiresAt,
		"last_rotated_at": key.LastRotatedAt,
	}
}

func (s *APIKeyService) audit(ctx context.Context, tenantID, actor, event string, payload map[string]any) {
	values := make(map[string]any, len(payload)+3)
	for key, value := range payload {
		values[key] = value
	}
	values["tenant_id"] = tenantID
	values["actor"] = actor
	values["at"] = time.Now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(values)
	if err != nil {
		return
	}
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO audit_outbox (event_id, event_type, payload)
		VALUES (gen_random_uuid(), $1, $2::jsonb)
	`, event, encoded)
}

func generateAPIKey() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate API key: %w", err)
	}
	return "sk-ds-" + hex.EncodeToString(buffer), nil
}
