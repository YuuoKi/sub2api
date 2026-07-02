\pset tuples_only on
\pset format unaligned
\o /tmp/g3_sql_recent_count.txt
SELECT COUNT(*) FROM ai_generation_content WHERE created_at >= NOW() - INTERVAL '15 minutes';
\o /tmp/g3_sql_suspicious_rows.txt
SELECT COUNT(*) AS suspicious_rows
FROM ai_generation_content
WHERE created_at >= NOW() - INTERVAL '15 minutes'
  AND (
    prompt_redacted ~ 'sk-[A-Za-z0-9_-]{20,}'
    OR response_redacted ~ 'sk-[A-Za-z0-9_-]{20,}'
    OR prompt_redacted ~ '13800138000'
    OR response_redacted ~ '13800138000'
  );
\o /tmp/g3_sql_recent_rows.txt
SELECT id, task_id, model, left(prompt_redacted, 120) AS prompt_preview,
       left(response_redacted, 120) AS response_preview,
       prompt_bytes, response_bytes, response_truncated
FROM ai_generation_content
WHERE created_at >= NOW() - INTERVAL '15 minutes'
ORDER BY created_at DESC;
\o
