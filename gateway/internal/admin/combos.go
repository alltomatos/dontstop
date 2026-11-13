package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var comboSlugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

// Combo is the admin representation of a virtual model.
type Combo struct {
	ID              string        `json:"id"`
	Slug            string        `json:"slug"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Strategy        string        `json:"strategy"`
	FailoverEnabled bool          `json:"failover_enabled"`
	Status          string        `json:"status"`
	Revision        int           `json:"revision"`
	RPMLimit        *int          `json:"rpm_limit,omitempty"`
	TPMLimit        *int          `json:"tpm_limit,omitempty"`
	MaxAttempts     int           `json:"max_attempts"`
	Source          string        `json:"source"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	MemberCount     int           `json:"member_count"`
	ProviderCount   int           `json:"provider_count"`
	UsableMembers   int           `json:"usable_members"`
	Members         []ComboMember `json:"members,omitempty"`
}

// ComboMember is one provider/model target inside a combo.
type ComboMember struct {
	ID           string         `json:"id"`
	ProviderID   string         `json:"provider_id"`
	ProviderName string         `json:"provider_name"`
	ModelID      string         `json:"model_id"`
	ModelName    string         `json:"model_name"`
	Priority     int            `json:"priority"`
	Position     int            `json:"position"`
	Weight       int            `json:"weight"`
	Enabled      bool           `json:"enabled"`
	Constraints  map[string]any `json:"constraints"`
	Usable       bool           `json:"usable"`
	Reason       string         `json:"reason,omitempty"`
}

// ComboMemberInput is one member accepted by the admin API.
type ComboMemberInput struct {
	ProviderID  string         `json:"provider_id"`
	ModelID     string         `json:"model_id"`
	Priority    int            `json:"priority"`
	Position    int            `json:"position"`
	Weight      int            `json:"weight"`
	Enabled     bool           `json:"enabled"`
	Constraints map[string]any `json:"constraints"`
}

// ComboInput creates or replaces a combo definition.
type ComboInput struct {
	Slug            string             `json:"slug"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Strategy        string             `json:"strategy"`
	FailoverEnabled bool               `json:"failover_enabled"`
	RPMLimit        *int               `json:"rpm_limit"`
	TPMLimit        *int               `json:"tpm_limit"`
	MaxAttempts     int                `json:"max_attempts"`
	Source          string             `json:"source"`
	Members         []ComboMemberInput `json:"members"`
}

// ComboCatalogModel is a selectable model in the combo editor.
type ComboCatalogModel struct {
	ProviderID   string         `json:"provider_id"`
	ProviderName string         `json:"provider_name"`
	ModelID      string         `json:"model_id"`
	DisplayName  string         `json:"display_name"`
	Capabilities map[string]any `json:"capabilities"`
	Usable       bool           `json:"usable"`
	Reason       string         `json:"reason,omitempty"`
}

// ComboService manages tenant-scoped combos.
type ComboService struct {
	pool *pgxpool.Pool
}

// NewComboService creates a combo service.
func NewComboService(pool *pgxpool.Pool) *ComboService {
	return &ComboService{pool: pool}
}

// List returns combos for one tenant.
func (s *ComboService) List(ctx context.Context, tenantID string) ([]Combo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text, c.slug, c.name, c.description, c.strategy, c.failover_enabled,
		       c.status, c.revision, c.rpm_limit, c.tpm_limit, c.max_attempts, c.source,
		       c.created_at, c.updated_at,
		       (SELECT count(*) FROM combo_members cm WHERE cm.combo_id = c.id),
		       (SELECT count(DISTINCT cm.provider_id) FROM combo_members cm WHERE cm.combo_id = c.id),
		       (SELECT count(*) FROM combo_members cm
		         JOIN providers p ON p.id = cm.provider_id AND p.tenant_id = cm.tenant_id
		         JOIN provider_models pm ON pm.tenant_id = cm.tenant_id AND pm.provider_id = cm.provider_id AND pm.model_id = cm.model_id
		         WHERE cm.combo_id = c.id AND cm.enabled = true AND p.enabled = true AND pm.active = true
		           AND EXISTS (SELECT 1 FROM provider_accounts pa
		                       WHERE pa.tenant_id = cm.tenant_id AND pa.provider_id = cm.provider_id
		                         AND pa.status = 'active' AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now())))
		FROM combos c
		WHERE c.tenant_id = $1
		ORDER BY c.updated_at DESC, c.name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query combos: %w", err)
	}
	defer rows.Close()
	combos := make([]Combo, 0)
	for rows.Next() {
		var combo Combo
		if err := rows.Scan(
			&combo.ID, &combo.Slug, &combo.Name, &combo.Description, &combo.Strategy,
			&combo.FailoverEnabled, &combo.Status, &combo.Revision, &combo.RPMLimit,
			&combo.TPMLimit, &combo.MaxAttempts, &combo.Source, &combo.CreatedAt,
			&combo.UpdatedAt, &combo.MemberCount, &combo.ProviderCount, &combo.UsableMembers,
		); err != nil {
			return nil, fmt.Errorf("scan combo: %w", err)
		}
		combos = append(combos, combo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate combos: %w", err)
	}
	return combos, nil
}

// Get returns one combo with members.
func (s *ComboService) Get(ctx context.Context, tenantID, comboID string) (Combo, error) {
	var combo Combo
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, slug, name, description, strategy, failover_enabled, status,
		       revision, rpm_limit, tpm_limit, max_attempts, source, created_at, updated_at
		FROM combos WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, comboID).Scan(
		&combo.ID, &combo.Slug, &combo.Name, &combo.Description, &combo.Strategy,
		&combo.FailoverEnabled, &combo.Status, &combo.Revision, &combo.RPMLimit,
		&combo.TPMLimit, &combo.MaxAttempts, &combo.Source, &combo.CreatedAt, &combo.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Combo{}, ErrNotFound
	}
	if err != nil {
		return Combo{}, fmt.Errorf("load combo: %w", err)
	}
	members, err := s.members(ctx, tenantID, comboID)
	if err != nil {
		return Combo{}, err
	}
	combo.Members = members
	combo.MemberCount = len(members)
	combo.ProviderCount = providerCount(members)
	for _, member := range members {
		if member.Usable {
			combo.UsableMembers++
		}
	}
	return combo, nil
}

// Create inserts a combo in draft status.
func (s *ComboService) Create(ctx context.Context, tenantID string, input ComboInput) (Combo, error) {
	if err := validateComboInput(input); err != nil {
		return Combo{}, err
	}
	members, err := s.validateMembers(ctx, tenantID, input.Members)
	if err != nil {
		return Combo{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Combo{}, fmt.Errorf("begin combo: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var comboID string
	source := input.Source
	if source != "ai" {
		source = "manual"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO combos (tenant_id, slug, name, description, strategy, failover_enabled,
		                    status, rpm_limit, tpm_limit, max_attempts, source)
		VALUES ($1, $2, $3, $4, $5, $6, 'draft', $7, $8, $9, $10)
		RETURNING id::text
	`, tenantID, input.Slug, strings.TrimSpace(input.Name), strings.TrimSpace(input.Description),
		input.Strategy, input.FailoverEnabled, input.RPMLimit, input.TPMLimit,
		normalizeAttempts(input.MaxAttempts), source).Scan(&comboID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Combo{}, ErrConflict
		}
		return Combo{}, fmt.Errorf("insert combo: %w", err)
	}
	if err := insertComboMembers(ctx, tx, tenantID, comboID, members); err != nil {
		return Combo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Combo{}, fmt.Errorf("commit combo: %w", err)
	}
	return s.Get(ctx, tenantID, comboID)
}

// Update replaces combo metadata and members when revision matches.
func (s *ComboService) Update(ctx context.Context, tenantID, comboID string, revision int, input ComboInput) (Combo, error) {
	if err := validateComboInput(input); err != nil {
		return Combo{}, err
	}
	members, err := s.validateMembers(ctx, tenantID, input.Members)
	if err != nil {
		return Combo{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Combo{}, fmt.Errorf("begin combo update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE combos
		SET name = $4, description = $5, strategy = $6, failover_enabled = $7,
		    rpm_limit = $8, tpm_limit = $9, max_attempts = $10,
		    revision = revision + 1, updated_at = now()
		WHERE tenant_id = $1 AND id = $2::uuid AND revision = $3
	`, tenantID, comboID, revision, strings.TrimSpace(input.Name), strings.TrimSpace(input.Description),
		input.Strategy, input.FailoverEnabled, input.RPMLimit, input.TPMLimit,
		normalizeAttempts(input.MaxAttempts))
	if err != nil {
		return Combo{}, fmt.Errorf("update combo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Combo{}, ErrConflict
	}
	if _, err := tx.Exec(ctx, `DELETE FROM combo_members WHERE tenant_id = $1 AND combo_id = $2::uuid`, tenantID, comboID); err != nil {
		return Combo{}, fmt.Errorf("replace combo members: %w", err)
	}
	if err := insertComboMembers(ctx, tx, tenantID, comboID, members); err != nil {
		return Combo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Combo{}, fmt.Errorf("commit combo update: %w", err)
	}
	return s.Get(ctx, tenantID, comboID)
}

// SetStatus activates or deactivates a combo.
func (s *ComboService) SetStatus(ctx context.Context, tenantID, comboID, status string) (Combo, error) {
	if status != "active" && status != "inactive" && status != "draft" {
		return Combo{}, fmt.Errorf("%w: invalid combo status", ErrInvalidInput)
	}
	if status == "active" {
		var usable int
		err := s.pool.QueryRow(ctx, `
			SELECT count(*) FROM combo_members cm
			JOIN providers p ON p.id = cm.provider_id AND p.tenant_id = cm.tenant_id
			JOIN provider_models pm ON pm.tenant_id = cm.tenant_id AND pm.provider_id = cm.provider_id AND pm.model_id = cm.model_id
			WHERE cm.tenant_id = $1 AND cm.combo_id = $2::uuid
			  AND cm.enabled = true AND p.enabled = true AND pm.active = true
			  AND EXISTS (SELECT 1 FROM provider_accounts pa
			              WHERE pa.tenant_id = cm.tenant_id AND pa.provider_id = cm.provider_id
			                AND pa.status = 'active' AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now()))
		`, tenantID, comboID).Scan(&usable)
		if err != nil {
			return Combo{}, fmt.Errorf("check combo members: %w", err)
		}
		if usable == 0 {
			return Combo{}, fmt.Errorf("%w: combo has no usable member", ErrConflict)
		}
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE combos SET status = $3, revision = revision + 1, updated_at = now()
		WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, comboID, status)
	if err != nil {
		return Combo{}, fmt.Errorf("update combo status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Combo{}, ErrNotFound
	}
	return s.Get(ctx, tenantID, comboID)
}

// Delete removes a combo and its members.
func (s *ComboService) Delete(ctx context.Context, tenantID, comboID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM combos WHERE tenant_id = $1 AND id = $2::uuid`, tenantID, comboID)
	if err != nil {
		return fmt.Errorf("delete combo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Catalog returns selectable models for the combo editor.
func (s *ComboService) Catalog(ctx context.Context, tenantID string) ([]ComboCatalogModel, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id::text, p.name, pm.model_id, pm.display_name, pm.capabilities,
		       EXISTS (SELECT 1 FROM provider_accounts pa
		               WHERE pa.tenant_id = pm.tenant_id AND pa.provider_id = pm.provider_id
		                 AND pa.status = 'active' AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now()))
		FROM provider_models pm
		JOIN providers p ON p.id = pm.provider_id AND p.tenant_id = pm.tenant_id
		WHERE pm.tenant_id = $1 AND p.enabled = true AND pm.active = true
		ORDER BY p.name, pm.model_id
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query combo catalog: %w", err)
	}
	defer rows.Close()
	models := make([]ComboCatalogModel, 0)
	for rows.Next() {
		var model ComboCatalogModel
		if err := rows.Scan(&model.ProviderID, &model.ProviderName, &model.ModelID, &model.DisplayName, &model.Capabilities, &model.Usable); err != nil {
			return nil, fmt.Errorf("scan combo catalog: %w", err)
		}
		if !model.Usable {
			model.Reason = "no active account"
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate combo catalog: %w", err)
	}
	return models, nil
}

// ComboTarget is one resolved concrete target for the data plane.
type ComboTarget struct {
	ProviderID string
	ModelID    string
	Priority   int
	Position   int
	Weight     int
}

// ResolveTargets returns enabled combo members in deterministic order.
func (s *ComboService) ResolveTargets(ctx context.Context, tenantID, slug string) ([]ComboTarget, int, error) {
	var comboID, strategy string
	var revision int
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, revision, strategy FROM combos
		WHERE tenant_id = $1 AND slug = $2 AND status = 'active'
	`, tenantID, slug).Scan(&comboID, &revision, &strategy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("load combo target: %w", err)
	}
	var cursor int64
	if strategy == "round_robin" || strategy == "priority_round_robin" {
		if err := s.pool.QueryRow(ctx, `
			UPDATE combos SET cursor = cursor + 1
			WHERE tenant_id = $1 AND id = $2::uuid
			RETURNING cursor
		`, tenantID, comboID).Scan(&cursor); err != nil {
			return nil, 0, fmt.Errorf("advance combo cursor: %w", err)
		}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT cm.provider_id::text, cm.model_id, cm.priority, cm.position, cm.weight
		FROM combo_members cm
		JOIN providers p ON p.id = cm.provider_id AND p.tenant_id = cm.tenant_id
		JOIN provider_models pm ON pm.tenant_id = cm.tenant_id AND pm.provider_id = cm.provider_id AND pm.model_id = cm.model_id
		WHERE cm.tenant_id = $1 AND cm.combo_id = $2::uuid
		  AND cm.enabled = true AND p.enabled = true AND pm.active = true
		  AND EXISTS (SELECT 1 FROM provider_accounts pa
		              WHERE pa.tenant_id = cm.tenant_id AND pa.provider_id = cm.provider_id
		                AND pa.status = 'active' AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now()))
		ORDER BY cm.priority, cm.position
	`, tenantID, comboID)
	if err != nil {
		return nil, 0, fmt.Errorf("query combo targets: %w", err)
	}
	defer rows.Close()
	targets := make([]ComboTarget, 0)
	for rows.Next() {
		var target ComboTarget
		if err := rows.Scan(&target.ProviderID, &target.ModelID, &target.Priority, &target.Position, &target.Weight); err != nil {
			return nil, 0, fmt.Errorf("scan combo target: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate combo targets: %w", err)
	}
	if len(targets) == 0 {
		return nil, revision, ErrConflict
	}
	targets = rotateComboTargets(targets, strategy, cursor)
	return targets, revision, nil
}

func rotateComboTargets(targets []ComboTarget, strategy string, cursor int64) []ComboTarget {
	if len(targets) < 2 {
		return targets
	}
	if strategy == "round_robin" {
		offset := int(cursor % int64(len(targets)))
		return append(append([]ComboTarget{}, targets[offset:]...), targets[:offset]...)
	}
	if strategy != "priority_round_robin" {
		return targets
	}
	result := make([]ComboTarget, 0, len(targets))
	for start := 0; start < len(targets); {
		end := start + 1
		for end < len(targets) && targets[end].Priority == targets[start].Priority {
			end++
		}
		tier := targets[start:end]
		offset := int(cursor % int64(len(tier)))
		result = append(result, tier[offset:]...)
		result = append(result, tier[:offset]...)
		start = end
	}
	return result
}

func (s *ComboService) members(ctx context.Context, tenantID, comboID string) ([]ComboMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cm.id::text, cm.provider_id::text, p.name, cm.model_id,
		       coalesce(nullif(pm.display_name, ''), pm.model_id), cm.priority, cm.position,
		       cm.weight, cm.enabled, cm.constraints,
		       EXISTS (SELECT 1 FROM provider_accounts pa
		               WHERE pa.tenant_id = cm.tenant_id AND pa.provider_id = cm.provider_id
		                 AND pa.status = 'active' AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now()))
		       AND p.enabled AND pm.active
		FROM combo_members cm
		JOIN providers p ON p.id = cm.provider_id AND p.tenant_id = cm.tenant_id
		JOIN provider_models pm ON pm.tenant_id = cm.tenant_id AND pm.provider_id = cm.provider_id AND pm.model_id = cm.model_id
		WHERE cm.tenant_id = $1 AND cm.combo_id = $2::uuid
		ORDER BY cm.priority, cm.position
	`, tenantID, comboID)
	if err != nil {
		return nil, fmt.Errorf("query combo members: %w", err)
	}
	defer rows.Close()
	members := make([]ComboMember, 0)
	for rows.Next() {
		var member ComboMember
		if err := rows.Scan(
			&member.ID, &member.ProviderID, &member.ProviderName, &member.ModelID,
			&member.ModelName, &member.Priority, &member.Position, &member.Weight,
			&member.Enabled, &member.Constraints, &member.Usable,
		); err != nil {
			return nil, fmt.Errorf("scan combo member: %w", err)
		}
		if !member.Usable {
			member.Reason = "member is not currently usable"
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate combo members: %w", err)
	}
	return members, nil
}

func insertComboMembers(ctx context.Context, tx pgx.Tx, tenantID, comboID string, members []ComboMemberInput) error {
	for _, member := range members {
		if _, err := tx.Exec(ctx, `
			INSERT INTO combo_members (tenant_id, combo_id, provider_id, model_id, priority, position, weight, enabled, constraints)
			VALUES ($1, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, coalesce($9, '{}'::jsonb))
		`, tenantID, comboID, member.ProviderID, member.ModelID, member.Priority, member.Position, member.Weight, member.Enabled, member.Constraints); err != nil {
			return fmt.Errorf("insert combo member: %w", err)
		}
	}
	return nil
}

func validateComboInput(input ComboInput) error {
	if !comboSlugPattern.MatchString(strings.TrimSpace(input.Slug)) {
		return fmt.Errorf("%w: invalid combo slug", ErrInvalidInput)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 80 {
		return fmt.Errorf("%w: invalid combo name", ErrInvalidInput)
	}
	if input.Strategy != "priority" && input.Strategy != "round_robin" && input.Strategy != "priority_round_robin" {
		return fmt.Errorf("%w: invalid combo strategy", ErrInvalidInput)
	}
	if len(input.Members) == 0 || len(input.Members) > 32 {
		return fmt.Errorf("%w: combo requires 1-32 members", ErrInvalidInput)
	}
	seen := map[string]bool{}
	positions := map[string]bool{}
	for _, member := range input.Members {
		if !isUUID(member.ProviderID) || strings.TrimSpace(member.ModelID) == "" {
			return fmt.Errorf("%w: invalid combo member", ErrInvalidInput)
		}
		key := member.ProviderID + "/" + member.ModelID
		if seen[key] {
			return fmt.Errorf("%w: duplicate combo member", ErrConflict)
		}
		seen[key] = true
		if member.Priority < 0 || member.Priority > 100 || member.Weight < 1 || member.Weight > 100 {
			return fmt.Errorf("%w: invalid combo member order or weight", ErrInvalidInput)
		}
		positionKey := fmt.Sprintf("%d/%d", member.Priority, member.Position)
		if positions[positionKey] {
			return fmt.Errorf("%w: duplicate combo position", ErrConflict)
		}
		positions[positionKey] = true
	}
	return nil
}

func (s *ComboService) validateMembers(ctx context.Context, tenantID string, members []ComboMemberInput) ([]ComboMemberInput, error) {
	for index := range members {
		members[index].ProviderID = strings.TrimSpace(members[index].ProviderID)
		members[index].ModelID = strings.TrimSpace(members[index].ModelID)
		var exists bool
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM provider_models pm
				JOIN providers p ON p.id = pm.provider_id AND p.tenant_id = pm.tenant_id
				WHERE pm.tenant_id = $1 AND pm.provider_id = $2::uuid AND pm.model_id = $3
				  AND pm.active = true AND p.enabled = true
			)
		`, tenantID, members[index].ProviderID, members[index].ModelID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("validate combo member: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("%w: combo member provider/model is not active", ErrInvalidInput)
		}
	}
	return members, nil
}

func normalizeAttempts(value int) int {
	if value <= 0 {
		return 5
	}
	if value > 8 {
		return 8
	}
	return value
}

func providerCount(members []ComboMember) int {
	seen := map[string]bool{}
	for _, member := range members {
		seen[member.ProviderID] = true
	}
	return len(seen)
}

// MarshalConstraints keeps the API contract explicit even when empty.
func comboConstraints(value map[string]any) json.RawMessage {
	if value == nil {
		return json.RawMessage(`{}`)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}
