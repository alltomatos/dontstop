package routing

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrComboNotFound  = errors.New("combo not found")
	ErrComboExhausted = errors.New("combo has no eligible member")
)

// ComboTarget is one concrete member selected by a combo.
type ComboTarget struct {
	ProviderID string
	ModelID    string
	Priority   int
	Position   int
	Weight     int
}

// ComboRepository resolves active combos for the data plane.
type ComboRepository struct {
	pool *pgxpool.Pool
}

// NewComboRepository creates a combo repository.
func NewComboRepository(pool *pgxpool.Pool) *ComboRepository {
	return &ComboRepository{pool: pool}
}

// ResolveTargets returns eligible members ordered for this request. The cursor
// is advanced atomically in PostgreSQL so multiple replicas share rotation.
func (r *ComboRepository) ResolveTargets(ctx context.Context, tenantID, slug string) ([]ComboTarget, int, error) {
	var comboID, strategy string
	var revision int
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, revision, strategy FROM combos
		WHERE tenant_id = $1 AND slug = $2 AND status = 'active'
	`, tenantID, slug).Scan(&comboID, &revision, &strategy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, ErrComboNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("load combo: %w", err)
	}
	var cursor int64
	if strategy == "round_robin" || strategy == "priority_round_robin" {
		if err := r.pool.QueryRow(ctx, `
			UPDATE combos SET cursor = cursor + 1
			WHERE tenant_id = $1 AND id = $2::uuid
			RETURNING cursor
		`, tenantID, comboID).Scan(&cursor); err != nil {
			return nil, 0, fmt.Errorf("advance combo cursor: %w", err)
		}
	}
	rows, err := r.pool.Query(ctx, `
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
		return nil, 0, fmt.Errorf("query combo members: %w", err)
	}
	defer rows.Close()
	targets := make([]ComboTarget, 0)
	for rows.Next() {
		var target ComboTarget
		if err := rows.Scan(&target.ProviderID, &target.ModelID, &target.Priority, &target.Position, &target.Weight); err != nil {
			return nil, 0, fmt.Errorf("scan combo member: %w", err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate combo members: %w", err)
	}
	if len(targets) == 0 {
		return nil, revision, ErrComboExhausted
	}
	return rotateTargets(targets, strategy, cursor), revision, nil
}

func rotateTargets(targets []ComboTarget, strategy string, cursor int64) []ComboTarget {
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
