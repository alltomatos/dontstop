ALTER TABLE provider_accounts
    ADD COLUMN IF NOT EXISTS proxy_mode text NOT NULL DEFAULT 'off';

ALTER TABLE provider_accounts
    ADD CONSTRAINT provider_accounts_proxy_mode_valid
    CHECK (proxy_mode IN ('off', 'preferred', 'required'));

CREATE TABLE proxy_endpoints (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    label text NOT NULL DEFAULT '',
    address text NOT NULL,
    port integer NOT NULL,
    country_code text NOT NULL DEFAULT '',
    city_name text NOT NULL DEFAULT '',
    username_cipher bytea,
    password_cipher bytea,
    source text NOT NULL DEFAULT 'manual',
    valid boolean NOT NULL DEFAULT true,
    quarantined_until timestamptz,
    last_verified_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT proxy_endpoints_port_valid CHECK (port BETWEEN 1 AND 65535),
    CONSTRAINT proxy_endpoints_source_valid CHECK (source IN ('manual', 'webshare')),
    CONSTRAINT proxy_endpoints_tenant_address_unique UNIQUE (tenant_id, address, port)
);

CREATE INDEX proxy_endpoints_tenant_idx
    ON proxy_endpoints (tenant_id, created_at DESC);

CREATE TABLE proxy_assignments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_account_id uuid NOT NULL REFERENCES provider_accounts(id) ON DELETE CASCADE,
    proxy_endpoint_id uuid NOT NULL REFERENCES proxy_endpoints(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT proxy_assignments_account_unique UNIQUE (tenant_id, provider_account_id)
);

CREATE INDEX proxy_assignments_endpoint_idx
    ON proxy_assignments (tenant_id, proxy_endpoint_id);
