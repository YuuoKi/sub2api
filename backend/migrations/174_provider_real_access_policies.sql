-- G2: employee real-access policies + review_only flags + execution_mode stamps.

ALTER TABLE video_provider_accounts
    ADD COLUMN IF NOT EXISTS review_only BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS review_only BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE video_tasks
    ADD COLUMN IF NOT EXISTS execution_mode TEXT NOT NULL DEFAULT 'mock';

ALTER TABLE batch_image_jobs
    ADD COLUMN IF NOT EXISTS execution_mode TEXT NOT NULL DEFAULT 'mock';

CREATE TABLE IF NOT EXISTS provider_real_access_policies (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL DEFAULT 'default',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    global_kill_switch BOOLEAN NOT NULL DEFAULT FALSE,
    allow_member BOOLEAN NOT NULL DEFAULT FALSE,
    allow_group BOOLEAN NOT NULL DEFAULT FALSE,
    image_daily_cny NUMERIC(18, 6) NOT NULL DEFAULT 0,
    video_daily_cny NUMERIC(18, 6) NOT NULL DEFAULT 0,
    monthly_cny NUMERIC(18, 6) NOT NULL DEFAULT 0,
    enabled_at TIMESTAMPTZ NULL,
    disabled_at TIMESTAMPTZ NULL,
    audit_actor_id BIGINT NULL,
    audit_actor_email TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_real_access_policies_name
    ON provider_real_access_policies (name);

CREATE TABLE IF NOT EXISTS provider_real_access_reservations (
    id BIGSERIAL PRIMARY KEY,
    operation_id TEXT NOT NULL,
    user_id BIGINT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('image', 'video')),
    reserved_cny NUMERIC(18, 6) NOT NULL,
    status TEXT NOT NULL DEFAULT 'reserved' CHECK (status IN ('reserved', 'settled', 'released')),
    settled_cny NUMERIC(18, 6) NULL,
    policy_id BIGINT REFERENCES provider_real_access_policies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (operation_id)
);

CREATE INDEX IF NOT EXISTS idx_provider_real_access_reservations_user_status
    ON provider_real_access_reservations (user_id, status, created_at DESC);

INSERT INTO provider_real_access_policies (name, enabled, global_kill_switch)
VALUES ('default', FALSE, TRUE)
ON CONFLICT (name) DO NOTHING;
