-- Immutable task-scoped pricing provenance. Existing rows intentionally remain
-- NULL/unknown: current configuration is not evidence of historical pricing.
ALTER TABLE video_tasks
    ADD COLUMN IF NOT EXISTS pricing_source TEXT,
    ADD COLUMN IF NOT EXISTS pricing_version TEXT,
    ADD COLUMN IF NOT EXISTS pricing_cny_per_million_completion_tokens NUMERIC(20,8)
        CHECK (pricing_cny_per_million_completion_tokens > 0),
    ADD COLUMN IF NOT EXISTS pricing_usd_cny_exchange_rate NUMERIC(20,8)
        CHECK (pricing_usd_cny_exchange_rate > 0),
    ADD COLUMN IF NOT EXISTS pricing_maximum_cny NUMERIC(20,8)
        CHECK (pricing_maximum_cny > 0);
