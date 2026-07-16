-- Immutable, task-scoped delivery evidence. Additive only: existing rows remain
-- unknown (NULL) rather than being backfilled from request fields.
ALTER TABLE video_tasks
    ADD COLUMN IF NOT EXISTS upstream_model TEXT,
    ADD COLUMN IF NOT EXISTS upstream_duration_seconds INTEGER CHECK (upstream_duration_seconds > 0),
    ADD COLUMN IF NOT EXISTS upstream_resolution VARCHAR(32),
    ADD COLUMN IF NOT EXISTS billing_model TEXT,
    ADD COLUMN IF NOT EXISTS billing_duration_seconds INTEGER CHECK (billing_duration_seconds > 0),
    ADD COLUMN IF NOT EXISTS billing_resolution VARCHAR(32),
    ADD COLUMN IF NOT EXISTS balance_before_usd NUMERIC(20,8) CHECK (balance_before_usd >= 0),
    ADD COLUMN IF NOT EXISTS balance_after_usd NUMERIC(20,8) CHECK (balance_after_usd >= 0),
    ADD COLUMN IF NOT EXISTS balance_delta_usd NUMERIC(20,8),
    ADD COLUMN IF NOT EXISTS authorization_consumed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS authorization_consumed_by BIGINT REFERENCES users(id) ON DELETE SET NULL;
