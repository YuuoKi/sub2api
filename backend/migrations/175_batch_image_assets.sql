-- G4: persist Gemini batch image assets for local preview/download/reuse.

CREATE TABLE IF NOT EXISTS batch_image_assets (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) NOT NULL REFERENCES batch_image_jobs(batch_id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL REFERENCES batch_image_items(id) ON DELETE CASCADE,
    image_index INTEGER NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    byte_size BIGINT NOT NULL CHECK (byte_size >= 0),
    sha256 VARCHAR(64) NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_provider VARCHAR(32) NOT NULL,
    source_ref TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT batch_image_assets_image_index_nonneg CHECK (image_index >= 0),
    CONSTRAINT batch_image_assets_batch_item_index_uq UNIQUE (batch_id, item_id, image_index)
);

CREATE INDEX IF NOT EXISTS batch_image_assets_batch_id_idx
    ON batch_image_assets (batch_id);

CREATE INDEX IF NOT EXISTS batch_image_assets_item_id_idx
    ON batch_image_assets (item_id);

CREATE INDEX IF NOT EXISTS batch_image_assets_storage_key_idx
    ON batch_image_assets (storage_key);

ALTER TABLE batch_image_jobs
    ADD COLUMN IF NOT EXISTS response_mime_type VARCHAR(128),
    ADD COLUMN IF NOT EXISTS aspect_ratio VARCHAR(32),
    ADD COLUMN IF NOT EXISTS image_size VARCHAR(16);
