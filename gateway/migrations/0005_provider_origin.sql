ALTER TABLE providers
    ADD COLUMN origin text NOT NULL DEFAULT 'allcode';

ALTER TABLE providers
    ADD CONSTRAINT providers_origin_check
    CHECK (origin IN ('9router', 'omniroute', 'allcode'));

CREATE INDEX providers_origin_idx
    ON providers (tenant_id, origin);
