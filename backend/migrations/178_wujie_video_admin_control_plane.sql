ALTER TABLE video_provider_accounts
    ADD COLUMN IF NOT EXISTS tiny_real_authorized_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS tiny_real_authorized_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS tiny_real_consumed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_video_provider_tiny_real_authorized
    ON video_provider_accounts (tiny_real_authorized_at, tiny_real_consumed_at, id);
