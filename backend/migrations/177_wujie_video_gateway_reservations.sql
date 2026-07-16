-- Transactional video budget reservations and settlement evidence. Additive only.
ALTER TABLE video_tasks
    ADD COLUMN IF NOT EXISTS reserved_cost_usd NUMERIC(20,8) NOT NULL DEFAULT 0 CHECK (reserved_cost_usd >= 0),
    ADD COLUMN IF NOT EXISTS reservation_state VARCHAR(16) NOT NULL DEFAULT 'none'
        CHECK (reservation_state IN ('none', 'reserved', 'released', 'captured')),
    ADD COLUMN IF NOT EXISTS reserved_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reservation_window_5h_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reservation_window_1d_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reservation_window_7d_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS provider_actual_cost_usd NUMERIC(20,8) NOT NULL DEFAULT 0
        CHECK (provider_actual_cost_usd >= 0);

ALTER TABLE video_usage_logs
    ADD COLUMN IF NOT EXISTS completion_tokens BIGINT,
    ADD COLUMN IF NOT EXISTS charged_cost_usd NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provider_actual_cost_usd NUMERIC(20,8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS currency VARCHAR(8) NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS result_url TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_video_tasks_reservation_state
    ON video_tasks (reservation_state, updated_at, id)
    WHERE reservation_state = 'reserved';
