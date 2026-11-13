CREATE TABLE IF NOT EXISTS combos (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    slug text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    strategy text NOT NULL DEFAULT 'priority_round_robin',
    failover_enabled boolean NOT NULL DEFAULT true,
    status text NOT NULL DEFAULT 'draft',
    revision integer NOT NULL DEFAULT 1,
    rpm_limit integer,
    tpm_limit integer,
    max_attempts integer NOT NULL DEFAULT 5,
    cursor bigint NOT NULL DEFAULT 0,
    source text NOT NULL DEFAULT 'manual',
    created_by uuid REFERENCES admin_users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT combos_tenant_slug_unique UNIQUE (tenant_id, slug),
    CONSTRAINT combos_slug_check CHECK (slug ~ '^[a-z][a-z0-9-]{1,62}$'),
    CONSTRAINT combos_strategy_check CHECK (strategy IN ('priority', 'round_robin', 'priority_round_robin')),
    CONSTRAINT combos_status_check CHECK (status IN ('draft', 'active', 'inactive')),
    CONSTRAINT combos_source_check CHECK (source IN ('manual', 'ai')),
    CONSTRAINT combos_limits_check CHECK (
        (rpm_limit IS NULL OR rpm_limit > 0) AND
        (tpm_limit IS NULL OR tpm_limit > 0) AND
        max_attempts BETWEEN 1 AND 8
    )
);

CREATE INDEX IF NOT EXISTS combos_tenant_status_idx
    ON combos (tenant_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS combo_members (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    combo_id uuid NOT NULL REFERENCES combos(id) ON DELETE CASCADE,
    provider_id uuid NOT NULL REFERENCES providers(id) ON DELETE RESTRICT,
    model_id text NOT NULL,
    priority smallint NOT NULL DEFAULT 0,
    position integer NOT NULL DEFAULT 0,
    weight integer NOT NULL DEFAULT 1,
    enabled boolean NOT NULL DEFAULT true,
    constraints jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT combo_members_combo_provider_model_unique UNIQUE (combo_id, provider_id, model_id),
    CONSTRAINT combo_members_combo_position_unique UNIQUE (combo_id, priority, position),
    CONSTRAINT combo_members_priority_check CHECK (priority BETWEEN 0 AND 100),
    CONSTRAINT combo_members_weight_check CHECK (weight BETWEEN 1 AND 100),
    CONSTRAINT combo_members_tenant_provider_model_fk
        FOREIGN KEY (tenant_id, provider_id, model_id)
        REFERENCES provider_models (tenant_id, provider_id, model_id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS combo_members_combo_order_idx
    ON combo_members (combo_id, priority, position);
