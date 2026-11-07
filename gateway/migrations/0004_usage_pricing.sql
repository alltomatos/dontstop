ALTER TABLE provider_models
    ADD COLUMN IF NOT EXISTS input_price_per_million numeric(14, 8) NOT NULL DEFAULT 0;

ALTER TABLE provider_models
    ADD COLUMN IF NOT EXISTS output_price_per_million numeric(14, 8) NOT NULL DEFAULT 0;

ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS request_id text;

ALTER TABLE requests
    ADD CONSTRAINT requests_request_id_unique UNIQUE (request_id);
