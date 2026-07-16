-- Runtime contract for the employee-scoped Seedance gateway. Additive only.
ALTER TABLE video_tasks
    ADD COLUMN IF NOT EXISTS api_key_id BIGINT REFERENCES api_keys(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS group_id BIGINT REFERENCES groups(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS duration_seconds INTEGER NOT NULL DEFAULT 4 CHECK (duration_seconds = 4),
    ADD COLUMN IF NOT EXISTS resolution VARCHAR(16) NOT NULL DEFAULT '720p' CHECK (resolution = '720p'),
    ADD COLUMN IF NOT EXISTS last_frame_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS usage_total_tokens BIGINT,
    ADD COLUMN IF NOT EXISTS cost_amount NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS currency VARCHAR(8) NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS real_dispatch_count INTEGER NOT NULL DEFAULT 0 CHECK (real_dispatch_count BETWEEN 0 AND 1),
    ADD COLUMN IF NOT EXISTS provider_error_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS provider_error_message TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_video_tasks_employee_scope
    ON video_tasks (created_by, api_key_id, group_id, created_at DESC);
