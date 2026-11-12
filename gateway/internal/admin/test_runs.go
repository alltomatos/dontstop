package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	modelTestLease = 5 * time.Minute
)

// ModelTestRun is a queued batch test of provider models.
type ModelTestRun struct {
	ID          string             `json:"id"`
	ProviderID  string             `json:"provider_id"`
	Status      string             `json:"status"`
	Total       int                `json:"total"`
	Completed   int                `json:"completed"`
	OKCount     int                `json:"ok_count"`
	FailedCount int                `json:"failed_count"`
	Error       string             `json:"error,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	FinishedAt  *time.Time         `json:"finished_at,omitempty"`
	Items       []ModelTestRunItem `json:"items,omitempty"`

	tenantID string
}

// ModelTestRunItem is one model result inside a run.
type ModelTestRunItem struct {
	ModelID   string `json:"model_id"`
	Status    string `json:"status"`
	LatencyMS int    `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// StartModelTestRun queues a batch test. An active run for the same provider is
// returned instead of creating a duplicate.
func (s *ProviderService) StartModelTestRun(ctx context.Context, tenantID, providerID string, models []string) (ModelTestRun, error) {
	if !isUUID(providerID) {
		return ModelTestRun{}, ErrNotFound
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM providers WHERE tenant_id = $1 AND id = $2::uuid)
	`, tenantID, providerID).Scan(&exists); err != nil {
		return ModelTestRun{}, fmt.Errorf("check provider: %w", err)
	}
	if !exists {
		return ModelTestRun{}, ErrNotFound
	}

	ids := sanitizeModelIDs(models)
	if len(ids) == 0 {
		rows, err := s.pool.Query(ctx, `
			SELECT model_id FROM provider_models
			WHERE tenant_id = $1 AND provider_id = $2::uuid
			ORDER BY model_id
		`, tenantID, providerID)
		if err != nil {
			return ModelTestRun{}, fmt.Errorf("list provider models: %w", err)
		}
		for rows.Next() {
			var modelID string
			if err := rows.Scan(&modelID); err != nil {
				rows.Close()
				return ModelTestRun{}, fmt.Errorf("scan provider model: %w", err)
			}
			ids = append(ids, modelID)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return ModelTestRun{}, fmt.Errorf("iterate provider models: %w", err)
		}
	}
	if len(ids) == 0 {
		return ModelTestRun{}, fmt.Errorf("%w: provider has no models to test", ErrInvalidInput)
	}

	if active, ok, err := s.activeModelTestRun(ctx, tenantID, providerID); err != nil {
		return ModelTestRun{}, err
	} else if ok {
		return active, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ModelTestRun{}, fmt.Errorf("begin test run: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var runID string
	err = tx.QueryRow(ctx, `
		INSERT INTO model_test_runs (tenant_id, provider_id, total)
		VALUES ($1, $2::uuid, $3)
		RETURNING id::text
	`, tenantID, providerID, len(ids)).Scan(&runID)
	if err != nil {
		if strings.Contains(err.Error(), "model_test_runs_active_provider_idx") {
			if active, ok, activeErr := s.activeModelTestRun(ctx, tenantID, providerID); activeErr == nil && ok {
				return active, nil
			}
		}
		return ModelTestRun{}, fmt.Errorf("insert test run: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO model_test_run_items (run_id, model_id)
		SELECT $1::uuid, model_id FROM unnest($2::text[]) AS model_id
	`, runID, ids); err != nil {
		return ModelTestRun{}, fmt.Errorf("insert test run items: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ModelTestRun{}, fmt.Errorf("commit test run: %w", err)
	}
	return s.loadModelTestRun(ctx, tenantID, runID)
}

// GetModelTestRun returns one run with its items.
func (s *ProviderService) GetModelTestRun(ctx context.Context, tenantID, providerID, runID string) (ModelTestRun, error) {
	if !isUUID(providerID) || !isUUID(runID) {
		return ModelTestRun{}, ErrNotFound
	}
	return s.loadModelTestRun(ctx, tenantID, runID)
}

// LatestModelTestRun returns the most recent run for a provider.
func (s *ProviderService) LatestModelTestRun(ctx context.Context, tenantID, providerID string) (ModelTestRun, error) {
	if !isUUID(providerID) {
		return ModelTestRun{}, ErrNotFound
	}
	var runID string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text FROM model_test_runs
		WHERE tenant_id = $1 AND provider_id = $2::uuid
		ORDER BY created_at DESC
		LIMIT 1
	`, tenantID, providerID).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ModelTestRun{}, ErrNotFound
	}
	if err != nil {
		return ModelTestRun{}, fmt.Errorf("load latest test run: %w", err)
	}
	return s.loadModelTestRun(ctx, tenantID, runID)
}

// RunModelTestOnce claims and processes one queued run. It reports whether a run
// was processed so the worker can drain the queue.
func (s *ProviderService) RunModelTestOnce(ctx context.Context) (bool, error) {
	run, claimed, err := s.claimModelTestRun(ctx)
	if err != nil || !claimed {
		return false, err
	}
	if err := s.processModelTestRun(ctx, run); err != nil {
		return true, err
	}
	return true, nil
}

// RunModelTestWorker drains the model test queue while the context lives.
func (s *ProviderService) RunModelTestWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		for {
			processed, err := s.RunModelTestOnce(ctx)
			if err != nil || !processed {
				break
			}
		}
	}
}

func (s *ProviderService) activeModelTestRun(ctx context.Context, tenantID, providerID string) (ModelTestRun, bool, error) {
	var runID string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text FROM model_test_runs
		WHERE tenant_id = $1 AND provider_id = $2::uuid
		  AND status IN ('queued', 'running')
		ORDER BY created_at
		LIMIT 1
	`, tenantID, providerID).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ModelTestRun{}, false, nil
	}
	if err != nil {
		return ModelTestRun{}, false, fmt.Errorf("load active test run: %w", err)
	}
	run, err := s.loadModelTestRun(ctx, tenantID, runID)
	if err != nil {
		return ModelTestRun{}, false, err
	}
	return run, true, nil
}

func (s *ProviderService) loadModelTestRun(ctx context.Context, tenantID, runID string) (ModelTestRun, error) {
	var run ModelTestRun
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, provider_id::text, status, total, completed, ok_count, failed_count,
		       error, created_at, started_at, finished_at
		FROM model_test_runs
		WHERE tenant_id = $1 AND id = $2::uuid
	`, tenantID, runID).Scan(
		&run.ID, &run.tenantID, &run.ProviderID, &run.Status, &run.Total, &run.Completed, &run.OKCount,
		&run.FailedCount, &run.Error, &run.CreatedAt, &run.StartedAt, &run.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ModelTestRun{}, ErrNotFound
	}
	if err != nil {
		return ModelTestRun{}, fmt.Errorf("load test run: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT model_id, status, latency_ms, error
		FROM model_test_run_items
		WHERE run_id = $1::uuid
		ORDER BY model_id
	`, runID)
	if err != nil {
		return ModelTestRun{}, fmt.Errorf("load test run items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item ModelTestRunItem
		if err := rows.Scan(&item.ModelID, &item.Status, &item.LatencyMS, &item.Error); err != nil {
			return ModelTestRun{}, fmt.Errorf("scan test run item: %w", err)
		}
		run.Items = append(run.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ModelTestRun{}, fmt.Errorf("iterate test run items: %w", err)
	}
	return run, nil
}

func (s *ProviderService) claimModelTestRun(ctx context.Context) (ModelTestRun, bool, error) {
	var runID, tenantID string
	err := s.pool.QueryRow(ctx, `
		UPDATE model_test_runs
		SET status = 'running',
		    started_at = COALESCE(started_at, now()),
		    lease_until = now() + $1::interval
		WHERE id = (
			SELECT id FROM model_test_runs
			WHERE status = 'queued'
			   OR (status = 'running' AND (lease_until IS NULL OR lease_until < now()))
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id::text, tenant_id::text
	`, fmt.Sprintf("%d seconds", int(modelTestLease.Seconds()))).Scan(&runID, &tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ModelTestRun{}, false, nil
	}
	if err != nil {
		return ModelTestRun{}, false, fmt.Errorf("claim test run: %w", err)
	}
	run, err := s.loadModelTestRun(ctx, tenantID, runID)
	if err != nil {
		return ModelTestRun{}, false, err
	}
	return run, true, nil
}

func (s *ProviderService) renewModelTestLease(ctx context.Context, runID string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE model_test_runs
		SET lease_until = now() + $2::interval
		WHERE id = $1::uuid AND status = 'running'
	`, runID, fmt.Sprintf("%d seconds", int(modelTestLease.Seconds())))
	if err != nil {
		return false, fmt.Errorf("renew test run lease: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *ProviderService) processModelTestRun(ctx context.Context, run ModelTestRun) error {
	for _, item := range run.Items {
		if item.Status != "pending" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		owned, err := s.renewModelTestLease(ctx, run.ID)
		if err != nil {
			return err
		}
		if !owned {
			// Another worker owns this run now; stop without overwriting results.
			return nil
		}
		if _, err := s.pool.Exec(ctx, `
			UPDATE model_test_run_items SET status = 'running', updated_at = now()
			WHERE run_id = $1::uuid AND model_id = $2
		`, run.ID, item.ModelID); err != nil {
			return fmt.Errorf("mark test item running: %w", err)
		}
		result, testErr := s.TestModel(ctx, run.tenantID, run.ProviderID, item.ModelID, "")
		status := "ok"
		message := ""
		if testErr != nil || !result.OK {
			status = "failed"
			message = result.Error
			if message == "" && testErr != nil {
				message = providerTestMessage(testErr)
			}
			if message == "" {
				message = "model test failed"
			}
		}
		if _, err := s.pool.Exec(ctx, `
			UPDATE model_test_run_items
			SET status = $3, latency_ms = $4, error = $5, updated_at = now()
			WHERE run_id = $1::uuid AND model_id = $2
		`, run.ID, item.ModelID, status, result.LatencyMS, message); err != nil {
			return fmt.Errorf("update test item: %w", err)
		}
		if err := s.refreshModelTestRunCounters(ctx, run.ID); err != nil {
			return err
		}
	}

	var pending int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM model_test_run_items
		WHERE run_id = $1::uuid AND status IN ('pending', 'running')
	`, run.ID).Scan(&pending); err != nil {
		return fmt.Errorf("count pending test items: %w", err)
	}
	if pending == 0 {
		if _, err := s.pool.Exec(ctx, `
			UPDATE model_test_runs
			SET status = 'completed', finished_at = now(), lease_until = NULL
			WHERE id = $1::uuid
		`, run.ID); err != nil {
			return fmt.Errorf("complete test run: %w", err)
		}
	}
	return nil
}

func (s *ProviderService) refreshModelTestRunCounters(ctx context.Context, runID string) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE model_test_runs r
		SET completed = counters.done,
		    ok_count = counters.ok,
		    failed_count = counters.failed
		FROM (
			SELECT count(*) FILTER (WHERE status IN ('ok', 'failed')) AS done,
			       count(*) FILTER (WHERE status = 'ok') AS ok,
			       count(*) FILTER (WHERE status = 'failed') AS failed
			FROM model_test_run_items
			WHERE run_id = $1::uuid
		) counters
		WHERE r.id = $1::uuid
	`, runID); err != nil {
		return fmt.Errorf("refresh test run counters: %w", err)
	}
	return nil
}

func sanitizeModelIDs(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	ids := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		ids = append(ids, model)
	}
	return ids
}
