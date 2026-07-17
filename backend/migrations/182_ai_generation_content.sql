-- Fail-open, redacted generation-content side table for the internal console.
-- All attribution references are nullable so source records can be deleted
-- without deleting the audit/capture row.
CREATE TABLE IF NOT EXISTS ai_generation_content (
    id                   BIGSERIAL PRIMARY KEY,
    request_id           VARCHAR(128) NOT NULL DEFAULT '',
    api_key_id           BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    user_id              BIGINT REFERENCES users(id) ON DELETE SET NULL,
    group_id             BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    account_id           BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    task_id              BIGINT REFERENCES video_tasks(id) ON DELETE SET NULL,
    model                VARCHAR(255) NOT NULL DEFAULT '',
    request_payload_hash VARCHAR(128) NOT NULL DEFAULT '',
    prompt_redacted      TEXT NOT NULL DEFAULT '',
    response_redacted    TEXT NOT NULL DEFAULT '',
    prompt_bytes         INT NOT NULL DEFAULT 0,
    response_bytes       INT NOT NULL DEFAULT 0,
    response_truncated   BOOLEAN NOT NULL DEFAULT FALSE,
    redaction_version    INT NOT NULL DEFAULT 1,
    adoption_status      VARCHAR(32) NOT NULL DEFAULT '',
    quality_score        DECIMAL(8,6),
    adoption_notes       TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_ai_generation_content_apikey_request
    ON ai_generation_content(api_key_id, request_id)
    WHERE request_id <> '';
CREATE INDEX IF NOT EXISTS idx_ai_generation_content_created_at
    ON ai_generation_content(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_generation_content_user_created_at
    ON ai_generation_content(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_generation_content_group_created_at
    ON ai_generation_content(group_id, created_at DESC);
