-- ADR-024: isolamento de IP por provedor, modo auto e poison persistente.
-- Expand & Contract: adiciona colunas/constraints sem remover as legadas.
-- Idempotente: reexecutável sem erro.

-- 1. Default de proxy no provedor (modo auto híbrido).
ALTER TABLE providers
    ADD COLUMN IF NOT EXISTS proxy_auto boolean NOT NULL DEFAULT false;

-- 2. proxy_mode por conta: acrescenta inherit/manual/auto e mantém os legados.
ALTER TABLE provider_accounts
    DROP CONSTRAINT IF EXISTS provider_accounts_proxy_mode_valid;
ALTER TABLE provider_accounts
    ADD CONSTRAINT provider_accounts_proxy_mode_valid
    CHECK (proxy_mode IN ('inherit', 'off', 'manual', 'auto', 'preferred', 'required'));

-- 3. proxy_assignments.provider_id (desnormalizado) + backfill.
ALTER TABLE proxy_assignments
    ADD COLUMN IF NOT EXISTS provider_id uuid;

UPDATE proxy_assignments pxa
SET provider_id = pa.provider_id
FROM provider_accounts pa
WHERE pa.id = pxa.provider_account_id
  AND pa.tenant_id = pxa.tenant_id
  AND pxa.provider_id IS NULL;

ALTER TABLE proxy_assignments
    ALTER COLUMN provider_id SET NOT NULL;

-- 4. Desdobra atribuições compartilhadas dentro do MESMO provedor:
--    mantém a mais antiga e libera as demais (reuso cross-provider é preservado).
WITH ranked AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY tenant_id, provider_id, proxy_endpoint_id
               ORDER BY created_at, id
           ) AS position
    FROM proxy_assignments
)
DELETE FROM proxy_assignments pxa
USING ranked
WHERE pxa.id = ranked.id
  AND ranked.position > 1;

-- 5. 1 conta por IP por provedor (reuso entre provedores diferentes é permitido).
DROP INDEX IF EXISTS proxy_assignments_provider_endpoint_key;
CREATE UNIQUE INDEX IF NOT EXISTS proxy_assignments_provider_endpoint_key
    ON proxy_assignments (tenant_id, provider_id, proxy_endpoint_id);

-- 6. Poison persistente por (provedor, endpoint).
CREATE TABLE IF NOT EXISTS proxy_poison (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id uuid NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    proxy_endpoint_id uuid NOT NULL REFERENCES proxy_endpoints(id) ON DELETE CASCADE,
    banned_until timestamptz NOT NULL,
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT proxy_poison_provider_endpoint_unique
        UNIQUE (tenant_id, provider_id, proxy_endpoint_id)
);

CREATE INDEX IF NOT EXISTS proxy_poison_tenant_provider_banned_idx
    ON proxy_poison (tenant_id, provider_id, banned_until);

-- 7. Metadados de ban/falha no endpoint.
ALTER TABLE proxy_endpoints
    ADD COLUMN IF NOT EXISTS banned_until timestamptz,
    ADD COLUMN IF NOT EXISTS ban_reason text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_failure_kind text NOT NULL DEFAULT '';

-- 8. Índice de candidatos do modo auto (somente Webshare válidos).
CREATE INDEX IF NOT EXISTS proxy_endpoints_auto_candidates_idx
    ON proxy_endpoints (tenant_id, score DESC, last_verified_at DESC)
    WHERE source = 'webshare' AND valid = true;
