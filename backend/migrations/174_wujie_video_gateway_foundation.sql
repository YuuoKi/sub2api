-- Wujie video gateway foundation. Additive only: no provider seeds, charges,
-- external calls, or historical data rewrites are performed here.

CREATE TABLE IF NOT EXISTS video_provider_accounts (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(32) NOT NULL,
    display_name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    encrypted_api_key TEXT NOT NULL DEFAULT '',
    masked_key TEXT NOT NULL DEFAULT '',
    base_url TEXT NOT NULL DEFAULT '',
    default_model TEXT NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, display_name)
);

CREATE TABLE IF NOT EXISTS video_tasks (
    id BIGSERIAL PRIMARY KEY,
    provider_account_id BIGINT NOT NULL REFERENCES video_provider_accounts(id) ON DELETE RESTRICT,
    provider VARCHAR(32) NOT NULL,
    model TEXT NOT NULL,
    task_type VARCHAR(32) NOT NULL,
    prompt TEXT NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'submitted', 'running', 'succeeded', 'failed', 'cancelled')),
    upstream_task_id TEXT NOT NULL DEFAULT '',
    result_url TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    creation_key VARCHAR(128),
    creation_fingerprint VARCHAR(64),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    dispatch_state VARCHAR(24) NOT NULL DEFAULT 'pending',
    worker_claimed_at TIMESTAMPTZ,
    worker_claimed_until TIMESTAMPTZ,
    balance_charged_at TIMESTAMPTZ,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS video_task_events (
    id BIGSERIAL PRIMARY KEY,
    video_task_id BIGINT NOT NULL REFERENCES video_tasks(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- This is a gateway task projection, not a second balance ledger. Financial
-- charging remains owned by the existing usage/billing infrastructure.
CREATE TABLE IF NOT EXISTS video_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    video_task_id BIGINT NOT NULL REFERENCES video_tasks(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    model TEXT NOT NULL,
    status VARCHAR(24) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS video_daily_trial_reservations (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(32) NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trial_date DATE NOT NULL,
    video_task_id BIGINT REFERENCES video_tasks(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, created_by, trial_date)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_video_tasks_creation_key
    ON video_tasks (creation_key) WHERE creation_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_video_tasks_claim_due
    ON video_tasks (worker_claimed_until, updated_at, id)
    WHERE status IN ('queued', 'submitted', 'running');
CREATE INDEX IF NOT EXISTS idx_video_tasks_created_by_created_at
    ON video_tasks (created_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_video_task_events_task_created_at
    ON video_task_events (video_task_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS uq_video_usage_logs_video_task_id
    ON video_usage_logs (video_task_id);
