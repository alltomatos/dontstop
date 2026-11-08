ALTER TABLE providers
    DROP CONSTRAINT IF EXISTS providers_origin_check;

UPDATE providers
SET origin = 'native'
WHERE origin NOT IN ('native', 'custom');

ALTER TABLE providers
    ALTER COLUMN origin SET DEFAULT 'native';

ALTER TABLE providers
    ADD CONSTRAINT providers_origin_check
    CHECK (origin IN ('native', 'custom'));
