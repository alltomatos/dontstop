ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS ip_allowlist text[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS rpm_limit integer,
    ADD COLUMN IF NOT EXISTS tpm_limit integer,
    ADD COLUMN IF NOT EXISTS usd_monthly_limit numeric(12,4),
    ADD COLUMN IF NOT EXISTS usd_daily_limit numeric(12,4),
    ADD COLUMN IF NOT EXISTS usd_weekly_limit numeric(12,4),
    ADD COLUMN IF NOT EXISTS rotated_at timestamptz;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_limits_nonnegative CHECK (
        (rpm_limit IS NULL OR rpm_limit >= 0) AND
        (tpm_limit IS NULL OR tpm_limit >= 0) AND
        (usd_monthly_limit IS NULL OR usd_monthly_limit >= 0) AND
        (usd_daily_limit IS NULL OR usd_daily_limit >= 0) AND
        (usd_weekly_limit IS NULL OR usd_weekly_limit >= 0)
    );
