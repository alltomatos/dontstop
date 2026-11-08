CREATE TABLE dlp_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL,
    detector text NOT NULL,
    action text NOT NULL,
    severity text NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT dlp_rules_action_valid CHECK (action IN ('allow', 'redact', 'block', 'audit')),
    CONSTRAINT dlp_rules_severity_valid CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT dlp_rules_tenant_name_unique UNIQUE (tenant_id, name)
);

CREATE TABLE dlp_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    request_id text,
    detector text NOT NULL,
    action text NOT NULL,
    severity text NOT NULL,
    value_hash text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX dlp_events_tenant_created_idx ON dlp_events (tenant_id, created_at DESC);
