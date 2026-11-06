CREATE TABLE tenants (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO tenants (id, name)
VALUES ('00000000-0000-0000-0000-000000000000', 'Default tenant')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE api_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL,
    key_prefix text NOT NULL,
    key_hash text NOT NULL,
    scopes jsonb NOT NULL DEFAULT '[]'::jsonb,
    expires_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT api_keys_key_hash_unique UNIQUE (key_hash)
);

CREATE INDEX api_keys_key_prefix_idx
    ON api_keys (key_prefix)
    WHERE revoked_at IS NULL;

CREATE TABLE providers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL,
    kind text NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT providers_tenant_name_unique UNIQUE (tenant_id, name)
);

CREATE TABLE provider_accounts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id uuid NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name text NOT NULL,
    credential_cipher bytea NOT NULL,
    priority integer NOT NULL DEFAULT 100,
    weight integer NOT NULL DEFAULT 1,
    status text NOT NULL DEFAULT 'active',
    cooldown_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT provider_accounts_priority_non_negative CHECK (priority >= 0),
    CONSTRAINT provider_accounts_weight_positive CHECK (weight > 0),
    CONSTRAINT provider_accounts_status_valid CHECK (status IN ('active', 'cooldown', 'disabled')),
    CONSTRAINT provider_accounts_tenant_name_unique UNIQUE (tenant_id, provider_id, name)
);

CREATE INDEX provider_accounts_routing_idx
    ON provider_accounts (tenant_id, provider_id, priority)
    WHERE status = 'active';

CREATE TABLE requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_key_id uuid REFERENCES api_keys(id) ON DELETE SET NULL,
    provider_id uuid REFERENCES providers(id) ON DELETE SET NULL,
    account_id uuid REFERENCES provider_accounts(id) ON DELETE SET NULL,
    model text NOT NULL,
    status text NOT NULL,
    http_status integer,
    error_code text,
    tokens_in integer NOT NULL DEFAULT 0,
    tokens_out integer NOT NULL DEFAULT 0,
    cost_usd numeric(14, 8) NOT NULL DEFAULT 0,
    latency_ms integer,
    cache_status text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT requests_tokens_non_negative CHECK (tokens_in >= 0 AND tokens_out >= 0),
    CONSTRAINT requests_cost_non_negative CHECK (cost_usd >= 0)
);

CREATE INDEX requests_tenant_created_idx
    ON requests (tenant_id, created_at DESC);

CREATE TABLE usage_daily (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    day date NOT NULL,
    provider_id uuid NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    model text NOT NULL,
    requests bigint NOT NULL DEFAULT 0,
    tokens_in bigint NOT NULL DEFAULT 0,
    tokens_out bigint NOT NULL DEFAULT 0,
    cost_usd numeric(14, 8) NOT NULL DEFAULT 0,
    errors bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, day, provider_id, model)
);

CREATE TABLE usage_outbox (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id uuid NOT NULL UNIQUE,
    payload jsonb NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX usage_outbox_unpublished_idx
    ON usage_outbox (created_at)
    WHERE published_at IS NULL;

CREATE TABLE audit_outbox (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id uuid NOT NULL UNIQUE,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    attempts integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX audit_outbox_unpublished_idx
    ON audit_outbox (created_at)
    WHERE published_at IS NULL;
