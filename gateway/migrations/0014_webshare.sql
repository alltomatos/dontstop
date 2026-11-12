ALTER TABLE proxy_endpoints
    ADD COLUMN IF NOT EXISTS external_id text,
    ADD COLUMN IF NOT EXISTS plan_id text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_seen_at timestamptz,
    ADD COLUMN IF NOT EXISTS score integer NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS last_failure_at timestamptz;

ALTER TABLE proxy_endpoints
    DROP CONSTRAINT IF EXISTS proxy_endpoints_tenant_address_unique;

CREATE UNIQUE INDEX IF NOT EXISTS proxy_endpoints_webshare_external_idx
    ON proxy_endpoints (tenant_id, external_id)
    WHERE source = 'webshare' AND external_id IS NOT NULL;

ALTER TABLE proxy_assignments
    ADD COLUMN IF NOT EXISTS sticky_until timestamptz;

CREATE TABLE webshare_settings (
    tenant_id uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_cipher bytea NOT NULL,
    key_version integer NOT NULL DEFAULT 1,
    plan_id text NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT true,
    last_sync_at timestamptz,
    last_sync_status text NOT NULL DEFAULT '',
    last_sync_error text NOT NULL DEFAULT '',
    last_sync_total integer NOT NULL DEFAULT 0,
    last_sync_imported integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT webshare_settings_key_version_positive CHECK (key_version > 0)
);
