package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
	"github.com/alltomatos/dontstop/gateway/internal/webshare"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var ErrSyncInProgress = errors.New("webshare sync already running")

// WebshareAPI is the subset of the Webshare client used by the admin service.
type WebshareAPI interface {
	ListDirect(ctx context.Context, apiKey, planID string) ([]webshare.Proxy, error)
	DiscoverPlanID(ctx context.Context, apiKey string) (string, error)
	Config(ctx context.Context, apiKey, planID string) (webshare.Config, error)
	Status(ctx context.Context, apiKey, planID string) (webshare.Status, error)
}

// WebshareSettings is the sanitized integration status.
type WebshareSettings struct {
	Configured       bool       `json:"configured"`
	PlanID           string     `json:"plan_id"`
	Enabled          bool       `json:"enabled"`
	KeyVersion       int        `json:"key_version"`
	LastSyncAt       *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus   string     `json:"last_sync_status"`
	LastSyncError    string     `json:"last_sync_error"`
	LastSyncTotal    int        `json:"last_sync_total"`
	LastSyncImported int        `json:"last_sync_imported"`
	ProxyCount       int        `json:"proxy_count"`
	HealthyCount     int        `json:"healthy_count"`
}

// WebshareTestResult is the result of a live connection test.
type WebshareTestResult struct {
	OK        bool   `json:"ok"`
	PlanID    string `json:"plan_id"`
	State     string `json:"state"`
	Available int    `json:"available"`
}

// WebshareSaveInput updates the integration. APIKey may be omitted to reuse
// the stored credential.
type WebshareSaveInput struct {
	APIKey  string `json:"api_key"`
	PlanID  string `json:"plan_id"`
	Enabled *bool  `json:"enabled"`
}

// WebshareSyncResult reports one sync run.
type WebshareSyncResult struct {
	Imported int    `json:"imported"`
	Total    int    `json:"total"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

// WebshareService manages the encrypted Webshare credential and sync.
type WebshareService struct {
	pool           *pgxpool.Pool
	box            *secret.Box
	client         WebshareAPI
	redis          redis.UniversalClient
	now            func() time.Time
	refreshMu      sync.Mutex
	lastRefresh    map[string]time.Time
	refreshCounter metric.Int64Counter
}

// refreshDebounce bounds failure-driven re-syncs per tenant (ADR-024).
const refreshDebounce = 30 * time.Second

// NewWebshareService creates a Webshare admin service.
func NewWebshareService(pool *pgxpool.Pool, box *secret.Box, client WebshareAPI, redisClient redis.UniversalClient) *WebshareService {
	meter := otel.Meter("dontstop/gateway/webshare")
	refresh, _ := meter.Int64Counter("proxy.refresh")
	return &WebshareService{
		pool: pool, box: box, client: client, redis: redisClient,
		now: time.Now, lastRefresh: map[string]time.Time{}, refreshCounter: refresh,
	}
}

// RefreshPool re-syncs the pool after a proxy failure. It is debounced per
// tenant (in-process and, when available, via Redis) and is a no-op when the
// integration is not configured.
func (s *WebshareService) RefreshPool(ctx context.Context, tenantID string) error {
	now := s.now()
	s.refreshMu.Lock()
	if last, ok := s.lastRefresh[tenantID]; ok && now.Sub(last) < refreshDebounce {
		s.refreshMu.Unlock()
		return nil
	}
	s.lastRefresh[tenantID] = now
	s.refreshMu.Unlock()

	if s.redis != nil {
		key := fmt.Sprintf("t:%s:proxy:refresh", tenantID)
		acquired, err := s.redis.SetNX(ctx, key, "1", refreshDebounce).Result()
		if err == nil && !acquired {
			return nil
		}
	}
	_, err := s.Sync(ctx, tenantID, "failure")
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrSyncInProgress) {
		return nil
	}
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	s.refreshCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("dontstop.tenant_id", tenantID),
		attribute.String("proxy.refresh_status", status),
	))
	return err
}

// Status returns the sanitized integration state.
func (s *WebshareService) Status(ctx context.Context, tenantID string) (WebshareSettings, error) {
	var settings WebshareSettings
	var lastSync *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT plan_id, enabled, key_version, last_sync_at, last_sync_status, last_sync_error,
		       last_sync_total, last_sync_imported
		FROM webshare_settings WHERE tenant_id = $1
	`, tenantID).Scan(
		&settings.PlanID, &settings.Enabled, &settings.KeyVersion, &lastSync,
		&settings.LastSyncStatus, &settings.LastSyncError, &settings.LastSyncTotal, &settings.LastSyncImported,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return WebshareSettings{}, fmt.Errorf("query webshare settings: %w", err)
	default:
		settings.Configured = true
		settings.LastSyncAt = lastSync
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE valid AND (quarantined_until IS NULL OR quarantined_until <= now()))
		FROM proxy_endpoints WHERE tenant_id = $1 AND source = 'webshare'
	`, tenantID).Scan(&settings.ProxyCount, &settings.HealthyCount); err != nil {
		return WebshareSettings{}, fmt.Errorf("count webshare proxies: %w", err)
	}
	return settings, nil
}

// Test validates a key/plan pair without persisting it. An empty API key uses
// the stored credential.
func (s *WebshareService) Test(ctx context.Context, tenantID, actor, apiKey, planID string) (WebshareTestResult, error) {
	key, resolvedPlan, err := s.credentials(ctx, tenantID, apiKey, planID)
	if err != nil {
		return WebshareTestResult{}, err
	}
	if resolvedPlan == "" {
		discovered, err := s.client.DiscoverPlanID(ctx, key)
		if err != nil {
			return WebshareTestResult{}, err
		}
		resolvedPlan = discovered
	}
	config, err := s.client.Config(ctx, key, resolvedPlan)
	if err != nil {
		return WebshareTestResult{}, err
	}
	status, err := s.client.Status(ctx, key, resolvedPlan)
	if err != nil {
		return WebshareTestResult{}, err
	}
	proxies, err := s.client.ListDirect(ctx, key, resolvedPlan)
	if err != nil {
		return WebshareTestResult{}, err
	}
	result := WebshareTestResult{
		OK:        true,
		PlanID:    firstNonEmpty(resolvedPlan, config.PlanID),
		State:     status.State,
		Available: len(proxies),
	}
	s.audit(ctx, tenantID, actor, "webshare.test", map[string]any{
		"plan_id": result.PlanID, "state": result.State, "available": result.Available,
	})
	return result, nil
}

// Save validates the credential, encrypts it and persists the settings.
func (s *WebshareService) Save(ctx context.Context, tenantID, actor string, input WebshareSaveInput) (WebshareSettings, error) {
	input.APIKey = strings.TrimSpace(input.APIKey)
	input.PlanID = strings.TrimSpace(input.PlanID)
	var existingCipher []byte
	var existingVersion int
	var existingPlan string
	err := s.pool.QueryRow(ctx, `
		SELECT api_key_cipher, key_version, plan_id FROM webshare_settings WHERE tenant_id = $1
	`, tenantID).Scan(&existingCipher, &existingVersion, &existingPlan)
	hasExisting := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return WebshareSettings{}, fmt.Errorf("query webshare settings: %w", err)
	}
	key := input.APIKey
	if key == "" {
		if !hasExisting {
			return WebshareSettings{}, fmt.Errorf("%w: api_key is required", ErrInvalidInput)
		}
		plain, err := s.box.Decrypt(existingCipher)
		if err != nil {
			return WebshareSettings{}, fmt.Errorf("decrypt webshare key: %w", err)
		}
		key = string(plain)
	}
	if input.PlanID == "" {
		input.PlanID = existingPlan
	}
	test, err := s.Test(ctx, tenantID, actor, key, input.PlanID)
	if err != nil {
		return WebshareSettings{}, err
	}
	if input.PlanID == "" {
		input.PlanID = test.PlanID
	}
	ciphertext, err := s.box.Encrypt([]byte(key))
	if err != nil {
		return WebshareSettings{}, fmt.Errorf("encrypt webshare key: %w", err)
	}
	version := existingVersion
	if !hasExisting {
		version = 1
	} else if input.APIKey != "" {
		version = existingVersion + 1
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO webshare_settings (tenant_id, api_key_cipher, key_version, plan_id, enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id) DO UPDATE
		SET api_key_cipher = excluded.api_key_cipher,
		    key_version = excluded.key_version,
		    plan_id = excluded.plan_id,
		    enabled = excluded.enabled,
		    updated_at = now()
	`, tenantID, ciphertext, version, input.PlanID, enabled); err != nil {
		return WebshareSettings{}, fmt.Errorf("save webshare settings: %w", err)
	}
	s.audit(ctx, tenantID, actor, "webshare.configure", map[string]any{
		"plan_id": input.PlanID, "key_version": version, "enabled": enabled,
	})
	return s.Status(ctx, tenantID)
}

// Sync imports the direct proxy list and applies an idempotent diff.
func (s *WebshareService) Sync(ctx context.Context, tenantID, actor string) (result WebshareSyncResult, err error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return WebshareSyncResult{}, fmt.Errorf("acquire sync connection: %w", err)
	}
	defer conn.Release()
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1, 0))`, "webshare:"+tenantID).Scan(&locked); err != nil {
		return WebshareSyncResult{}, fmt.Errorf("acquire webshare sync lock: %w", err)
	}
	if !locked {
		return WebshareSyncResult{}, ErrSyncInProgress
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtextextended($1, 0))`, "webshare:"+tenantID)
		if err != nil && !errors.Is(err, ErrSyncInProgress) {
			message := sanitizeWebshareError(err)
			s.recordSync(context.Background(), tenantID, 0, 0, "failed", message)
			s.audit(context.Background(), tenantID, actor, "webshare.sync_failed", map[string]any{"error": message})
		}
	}()

	var ciphertext []byte
	var planID string
	var enabled bool
	if err := conn.QueryRow(ctx, `
		SELECT api_key_cipher, plan_id, enabled FROM webshare_settings WHERE tenant_id = $1
	`, tenantID).Scan(&ciphertext, &planID, &enabled); errors.Is(err, pgx.ErrNoRows) {
		return WebshareSyncResult{}, fmt.Errorf("%w: webshare is not configured", ErrInvalidInput)
	} else if err != nil {
		return WebshareSyncResult{}, fmt.Errorf("query webshare settings: %w", err)
	}
	if !enabled {
		return WebshareSyncResult{Status: "disabled"}, nil
	}
	plain, err := s.box.Decrypt(ciphertext)
	if err != nil {
		return WebshareSyncResult{}, fmt.Errorf("decrypt webshare key: %w", err)
	}
	var started time.Time
	if err := conn.QueryRow(ctx, `SELECT now()`).Scan(&started); err != nil {
		return WebshareSyncResult{}, fmt.Errorf("read sync start time: %w", err)
	}
	proxies, err := s.client.ListDirect(ctx, string(plain), planID)
	if err != nil {
		return WebshareSyncResult{Status: "failed", Error: sanitizeWebshareError(err)}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WebshareSyncResult{}, fmt.Errorf("begin webshare sync: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, item := range proxies {
		if item.ID == "" || item.Address == "" || item.Port < 1 || item.Port > 65535 {
			continue
		}
		lastVerified := parseTime(item.LastVerifiedAt)
		usernameCipher, err := encryptString(s.box, item.Username)
		if err != nil {
			return WebshareSyncResult{}, fmt.Errorf("encrypt proxy username: %w", err)
		}
		passwordCipher, err := encryptString(s.box, item.Password)
		if err != nil {
			return WebshareSyncResult{}, fmt.Errorf("encrypt proxy password: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO proxy_endpoints
			    (tenant_id, label, address, port, country_code, city_name, username_cipher, password_cipher,
			     source, valid, last_verified_at, external_id, plan_id, last_seen_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'webshare', $9, $10, $11, $12, now())
			ON CONFLICT (tenant_id, external_id) WHERE source = 'webshare' AND external_id IS NOT NULL
			DO UPDATE SET address = excluded.address,
			              port = excluded.port,
			              country_code = excluded.country_code,
			              city_name = excluded.city_name,
			              username_cipher = excluded.username_cipher,
			              password_cipher = excluded.password_cipher,
			              valid = excluded.valid,
			              last_verified_at = excluded.last_verified_at,
			              plan_id = excluded.plan_id,
			              last_seen_at = now(),
			              label = excluded.label
		`, tenantID, webshareLabel(item), item.Address, item.Port, item.CountryCode, item.CityName,
			usernameCipher, passwordCipher, item.Valid,
			lastVerified, item.ID, planID); err != nil {
			return WebshareSyncResult{}, fmt.Errorf("upsert webshare proxy: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE proxy_endpoints
		SET valid = false
		WHERE tenant_id = $1 AND source = 'webshare'
		  AND (last_seen_at IS NULL OR last_seen_at < $2)
	`, tenantID, started); err != nil {
		return WebshareSyncResult{}, fmt.Errorf("mark stale webshare proxies: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return WebshareSyncResult{}, fmt.Errorf("commit webshare sync: %w", err)
	}

	result = WebshareSyncResult{Imported: len(proxies), Total: len(proxies), Status: "succeeded"}
	s.recordSync(ctx, tenantID, result.Total, result.Imported, "succeeded", "")
	s.audit(ctx, tenantID, actor, "webshare.sync", map[string]any{
		"imported": result.Imported, "total": result.Total,
	})
	return result, nil
}

// SyncAll runs the sync for every enabled tenant, used by the worker.
func (s *WebshareService) SyncAll(ctx context.Context, actor string) {
	rows, err := s.pool.Query(ctx, `
		SELECT tenant_id::text FROM webshare_settings WHERE enabled = true
	`)
	if err != nil {
		return
	}
	tenants := make([]string, 0)
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err == nil {
			tenants = append(tenants, tenantID)
		}
	}
	rows.Close()
	for _, tenantID := range tenants {
		if ctx.Err() != nil {
			return
		}
		_, _ = s.Sync(ctx, tenantID, actor)
	}
}

// Run periodically syncs every enabled tenant until the context ends.
func (s *WebshareService) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	s.SyncAll(ctx, "worker")
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.SyncAll(ctx, "worker")
			timer.Reset(interval)
		}
	}
}

func (s *WebshareService) credentials(ctx context.Context, tenantID, apiKey, planID string) (string, string, error) {
	key := strings.TrimSpace(apiKey)
	plan := strings.TrimSpace(planID)
	if key != "" {
		return key, plan, nil
	}
	var ciphertext []byte
	var storedPlan string
	err := s.pool.QueryRow(ctx, `
		SELECT api_key_cipher, plan_id FROM webshare_settings WHERE tenant_id = $1
	`, tenantID).Scan(&ciphertext, &storedPlan)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("%w: api_key is required", ErrInvalidInput)
	}
	if err != nil {
		return "", "", fmt.Errorf("query webshare settings: %w", err)
	}
	plain, err := s.box.Decrypt(ciphertext)
	if err != nil {
		return "", "", fmt.Errorf("decrypt webshare key: %w", err)
	}
	if plan == "" {
		plan = storedPlan
	}
	return string(plain), plan, nil
}

func (s *WebshareService) recordSync(ctx context.Context, tenantID string, total, imported int, status, message string) {
	_, _ = s.pool.Exec(ctx, `
		UPDATE webshare_settings
		SET last_sync_at = now(), last_sync_status = $2, last_sync_error = $3,
		    last_sync_total = $4, last_sync_imported = $5, updated_at = now()
		WHERE tenant_id = $1
	`, tenantID, status, message, total, imported)
}

func (s *WebshareService) audit(ctx context.Context, tenantID, actor, event string, payload map[string]any) {
	payload["tenant_id"] = tenantID
	payload["actor"] = actor
	payload["at"] = s.now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO audit_outbox (event_id, event_type, payload)
		VALUES (gen_random_uuid(), $1, $2::jsonb)
	`, event, encoded)
}

func encryptString(box *secret.Box, value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	if box == nil {
		return nil, fmt.Errorf("credential box is not configured")
	}
	return box.Encrypt([]byte(value))
}

func webshareLabel(item webshare.Proxy) string {
	parts := make([]string, 0, 2)
	if item.CityName != "" {
		parts = append(parts, item.CityName)
	}
	if item.CountryCode != "" {
		parts = append(parts, item.CountryCode)
	}
	if len(parts) == 0 {
		return "Webshare"
	}
	return strings.Join(parts, ", ")
}

func parseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func sanitizeWebshareError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, webshare.ErrUnauthorized):
		return "invalid api key"
	case errors.Is(err, webshare.ErrForbidden):
		return "plan access forbidden"
	case errors.Is(err, webshare.ErrRateLimited):
		return "rate limited; retry later"
	case errors.Is(err, webshare.ErrUnavailable):
		return "webshare unavailable"
	case errors.Is(err, webshare.ErrInvalidAnswer):
		return "invalid webshare response"
	default:
		return "webshare request failed"
	}
}
