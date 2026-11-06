ALTER TABLE provider_accounts
    ADD COLUMN IF NOT EXISTS base_url text NOT NULL DEFAULT '';

ALTER TABLE provider_accounts
    ADD COLUMN IF NOT EXISTS credential_key_version integer NOT NULL DEFAULT 1;

ALTER TABLE provider_accounts
    ADD CONSTRAINT provider_accounts_key_version_positive CHECK (credential_key_version > 0);
