-- Cross-instance single-smoke gate and provider ownership. Additive only.
ALTER TABLE video_provider_accounts
    ADD COLUMN IF NOT EXISTS group_id BIGINT REFERENCES groups(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_video_provider_accounts_group_enabled
    ON video_provider_accounts (group_id, enabled, id);

CREATE TABLE IF NOT EXISTS video_single_smoke_consumptions (
    gate_key VARCHAR(32) PRIMARY KEY CHECK (gate_key = 'global'),
    video_task_id BIGINT NOT NULL UNIQUE REFERENCES video_tasks(id) ON DELETE RESTRICT,
    consumed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
