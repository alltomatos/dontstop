package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DLPRule is one data-loss-prevention rule.
type DLPRule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Detector  string    `json:"detector"`
	Action    string    `json:"action"`
	Severity  string    `json:"severity"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// DLPInput creates a DLP rule.
type DLPInput struct {
	Name     string `json:"name"`
	Detector string `json:"detector"`
	Action   string `json:"action"`
	Severity string `json:"severity"`
	Enabled  *bool  `json:"enabled"`
}

// DLPEvent is a payload-free DLP event.
type DLPEvent struct {
	ID        string    `json:"id"`
	RequestID string    `json:"request_id,omitempty"`
	Detector  string    `json:"detector"`
	Action    string    `json:"action"`
	Severity  string    `json:"severity"`
	ValueHash string    `json:"value_hash,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// DLPService persists and reads DLP rules/events.
type DLPService struct {
	pool *pgxpool.Pool
}

// NewDLPService creates a DLP service.
func NewDLPService(pool *pgxpool.Pool) *DLPService {
	return &DLPService{pool: pool}
}

// Rules lists DLP rules.
func (s *DLPService) Rules(ctx context.Context, tenantID string) ([]DLPRule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, detector, action, severity, enabled, created_at
		FROM dlp_rules
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query DLP rules: %w", err)
	}
	defer rows.Close()
	rules := make([]DLPRule, 0)
	for rows.Next() {
		var rule DLPRule
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Detector, &rule.Action, &rule.Severity, &rule.Enabled, &rule.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan DLP rule: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate DLP rules: %w", err)
	}
	return rules, nil
}

// CreateRule inserts a validated DLP rule.
func (s *DLPService) CreateRule(ctx context.Context, tenantID string, input DLPInput) (DLPRule, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Detector = strings.TrimSpace(input.Detector)
	input.Action = strings.TrimSpace(input.Action)
	input.Severity = strings.TrimSpace(input.Severity)
	if input.Name == "" || input.Detector == "" {
		return DLPRule{}, fmt.Errorf("%w: name and detector are required", ErrInvalidInput)
	}
	if !validDLPAction(input.Action) || !validDLPSeverity(input.Severity) {
		return DLPRule{}, fmt.Errorf("%w: invalid action or severity", ErrInvalidInput)
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	var rule DLPRule
	err := s.pool.QueryRow(ctx, `
		INSERT INTO dlp_rules (tenant_id, name, detector, action, severity, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, name, detector, action, severity, enabled, created_at
	`, tenantID, input.Name, input.Detector, input.Action, input.Severity, enabled).Scan(
		&rule.ID, &rule.Name, &rule.Detector, &rule.Action, &rule.Severity, &rule.Enabled, &rule.CreatedAt,
	)
	if err != nil {
		return DLPRule{}, fmt.Errorf("insert DLP rule: %w", err)
	}
	return rule, nil
}

// SetRuleEnabled toggles a rule.
func (s *DLPService) SetRuleEnabled(ctx context.Context, tenantID, ruleID string, enabled bool) (DLPRule, error) {
	var rule DLPRule
	err := s.pool.QueryRow(ctx, `
		UPDATE dlp_rules SET enabled = $3
		WHERE tenant_id = $1 AND id = $2::uuid
		RETURNING id::text, name, detector, action, severity, enabled, created_at
	`, tenantID, ruleID, enabled).Scan(&rule.ID, &rule.Name, &rule.Detector, &rule.Action, &rule.Severity, &rule.Enabled, &rule.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DLPRule{}, ErrNotFound
	}
	if err != nil {
		return DLPRule{}, fmt.Errorf("update DLP rule: %w", err)
	}
	return rule, nil
}

// Events lists recent DLP events without raw values.
func (s *DLPService) Events(ctx context.Context, tenantID string, limit int) ([]DLPEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, COALESCE(request_id, ''), detector, action, severity, COALESCE(value_hash, ''), created_at
		FROM dlp_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("query DLP events: %w", err)
	}
	defer rows.Close()
	events := make([]DLPEvent, 0)
	for rows.Next() {
		var event DLPEvent
		if err := rows.Scan(&event.ID, &event.RequestID, &event.Detector, &event.Action, &event.Severity, &event.ValueHash, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan DLP event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate DLP events: %w", err)
	}
	return events, nil
}

func validDLPAction(value string) bool {
	return value == "allow" || value == "redact" || value == "block" || value == "audit"
}

func validDLPSeverity(value string) bool {
	return value == "low" || value == "medium" || value == "high" || value == "critical"
}
