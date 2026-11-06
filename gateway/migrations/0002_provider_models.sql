CREATE TABLE provider_models (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id uuid NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    model_id text NOT NULL,
    display_name text NOT NULL DEFAULT '',
    capabilities jsonb NOT NULL DEFAULT '{}'::jsonb,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT provider_models_tenant_provider_model_unique UNIQUE (tenant_id, provider_id, model_id)
);

CREATE INDEX provider_models_active_idx
    ON provider_models (tenant_id, provider_id, model_id)
    WHERE active = true;
