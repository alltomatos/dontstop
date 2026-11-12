CREATE TABLE model_test_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id uuid NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    total integer NOT NULL DEFAULT 0,
    completed integer NOT NULL DEFAULT 0,
    ok_count integer NOT NULL DEFAULT 0,
    failed_count integer NOT NULL DEFAULT 0,
    error text NOT NULL DEFAULT '',
    lease_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    finished_at timestamptz
);

CREATE INDEX model_test_runs_queue_idx
    ON model_test_runs (status, created_at)
    WHERE status IN ('queued', 'running');

CREATE UNIQUE INDEX model_test_runs_active_provider_idx
    ON model_test_runs (tenant_id, provider_id)
    WHERE status IN ('queued', 'running');

CREATE TABLE model_test_run_items (
    run_id uuid NOT NULL REFERENCES model_test_runs (id) ON DELETE CASCADE,
    model_id text NOT NULL,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'ok', 'failed')),
    latency_ms integer NOT NULL DEFAULT 0,
    error text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, model_id)
);

CREATE INDEX model_test_run_items_pending_idx
    ON model_test_run_items (run_id)
    WHERE status = 'pending';
