-- G5: provider formal billing import & internal/external reconciliation.
-- Match engine MUST NOT write users.balance or billing_transactions.

CREATE TABLE IF NOT EXISTS provider_billing_imports (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(64) NOT NULL,
    provider_account_id VARCHAR(128) NOT NULL,
    billing_period_start TIMESTAMPTZ NOT NULL,
    billing_period_end TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(64) NOT NULL,
    original_currency VARCHAR(3) NOT NULL,
    source_type VARCHAR(32) NOT NULL,
    invoice_number VARCHAR(128),
    file_sha256 VARCHAR(64) NOT NULL,
    storage_key TEXT NOT NULL,
    original_filename VARCHAR(512) NOT NULL,
    byte_size BIGINT NOT NULL CHECK (byte_size >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'imported',
    line_count INTEGER NOT NULL DEFAULT 0 CHECK (line_count >= 0),
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT provider_billing_imports_currency_chk
        CHECK (original_currency IN ('CNY', 'USD')),
    CONSTRAINT provider_billing_imports_source_type_chk
        CHECK (source_type IN ('csv', 'xlsx')),
    CONSTRAINT provider_billing_imports_sha256_uq UNIQUE (file_sha256)
);

CREATE INDEX IF NOT EXISTS provider_billing_imports_period_idx
    ON provider_billing_imports (provider, billing_period_start, billing_period_end);

CREATE INDEX IF NOT EXISTS provider_billing_imports_account_idx
    ON provider_billing_imports (provider, provider_account_id);

CREATE TABLE IF NOT EXISTS provider_billing_lines (
    id BIGSERIAL PRIMARY KEY,
    import_id BIGINT NOT NULL REFERENCES provider_billing_imports(id) ON DELETE CASCADE,
    provider VARCHAR(64) NOT NULL,
    external_line_id VARCHAR(256) NOT NULL,
    upstream_task_id VARCHAR(512) NOT NULL DEFAULT '',
    model VARCHAR(256) NOT NULL DEFAULT '',
    sku VARCHAR(256) NOT NULL DEFAULT '',
    usage_quantity NUMERIC(30, 10) NOT NULL,
    usage_unit VARCHAR(64) NOT NULL DEFAULT '',
    net_amount NUMERIC(20, 10) NOT NULL,
    tax_amount NUMERIC(20, 10) NOT NULL,
    gross_amount NUMERIC(20, 10) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    occurred_timezone VARCHAR(64) NOT NULL,
    normalized_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT provider_billing_lines_currency_chk
        CHECK (currency IN ('CNY', 'USD')),
    CONSTRAINT provider_billing_lines_amounts_nonneg_chk
        CHECK (net_amount >= 0 AND tax_amount >= 0 AND gross_amount >= 0),
    CONSTRAINT provider_billing_lines_usage_nonneg_chk
        CHECK (usage_quantity >= 0),
    CONSTRAINT provider_billing_lines_provider_external_uq
        UNIQUE (provider, external_line_id)
);

CREATE INDEX IF NOT EXISTS provider_billing_lines_import_id_idx
    ON provider_billing_lines (import_id);

CREATE INDEX IF NOT EXISTS provider_billing_lines_upstream_task_id_idx
    ON provider_billing_lines (upstream_task_id)
    WHERE upstream_task_id <> '';

CREATE INDEX IF NOT EXISTS provider_billing_lines_occurred_at_idx
    ON provider_billing_lines (provider, occurred_at);

CREATE TABLE IF NOT EXISTS provider_reconciliation_matches (
    id BIGSERIAL PRIMARY KEY,
    import_id BIGINT NOT NULL REFERENCES provider_billing_imports(id) ON DELETE CASCADE,
    billing_line_id BIGINT REFERENCES provider_billing_lines(id) ON DELETE SET NULL,
    external_line_id VARCHAR(256) NOT NULL DEFAULT '',
    match_status VARCHAR(32) NOT NULL,
    match_mode VARCHAR(32) NOT NULL,
    internal_ref_type VARCHAR(32),
    internal_ref_id VARCHAR(128),
    provider_amount NUMERIC(20, 10),
    internal_amount NUMERIC(20, 10),
    provider_usage NUMERIC(30, 10),
    internal_usage NUMERIC(30, 10),
    currency VARCHAR(3),
    model VARCHAR(256) NOT NULL DEFAULT '',
    sku VARCHAR(256) NOT NULL DEFAULT '',
    account_day DATE,
    diff_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT provider_reconciliation_matches_status_chk
        CHECK (match_status IN (
            'matched',
            'amount_mismatch',
            'usage_mismatch',
            'internal_only',
            'provider_only',
            'adjustment'
        )),
    CONSTRAINT provider_reconciliation_matches_mode_chk
        CHECK (match_mode IN ('task_id', 'aggregate_only'))
);

CREATE INDEX IF NOT EXISTS provider_reconciliation_matches_import_id_idx
    ON provider_reconciliation_matches (import_id);

CREATE INDEX IF NOT EXISTS provider_reconciliation_matches_status_idx
    ON provider_reconciliation_matches (match_status);

CREATE INDEX IF NOT EXISTS provider_reconciliation_matches_diff_queue_idx
    ON provider_reconciliation_matches (import_id, match_status)
    WHERE match_status <> 'matched';

-- Optional adjustment ledger: requires admin confirm.
-- NEVER write users.balance / billing_transactions from match engine.
CREATE TABLE IF NOT EXISTS provider_billing_adjustments (
    id BIGSERIAL PRIMARY KEY,
    match_id BIGINT NOT NULL REFERENCES provider_reconciliation_matches(id) ON DELETE CASCADE,
    import_id BIGINT NOT NULL REFERENCES provider_billing_imports(id) ON DELETE CASCADE,
    amount NUMERIC(20, 10) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_admin_confirm',
    confirmed_by BIGINT,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT provider_billing_adjustments_currency_chk
        CHECK (currency IN ('CNY', 'USD')),
    CONSTRAINT provider_billing_adjustments_status_chk
        CHECK (status IN ('pending_admin_confirm', 'confirmed', 'rejected'))
);

CREATE INDEX IF NOT EXISTS provider_billing_adjustments_status_idx
    ON provider_billing_adjustments (status);

-- Match-key indexes for Seedance / Gemini joins.
CREATE INDEX IF NOT EXISTS video_tasks_upstream_task_id_idx
    ON video_tasks (upstream_task_id)
    WHERE upstream_task_id <> '';

CREATE INDEX IF NOT EXISTS batch_image_jobs_provider_job_name_idx
    ON batch_image_jobs (provider_job_name)
    WHERE provider_job_name IS NOT NULL AND provider_job_name <> '';
