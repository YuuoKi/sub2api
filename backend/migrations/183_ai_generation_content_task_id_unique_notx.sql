-- The migration runner executes *_notx.sql files outside a transaction.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_ai_generation_content_task_id
    ON ai_generation_content(task_id)
    WHERE task_id IS NOT NULL;
