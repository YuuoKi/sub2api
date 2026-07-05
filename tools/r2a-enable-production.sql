UPDATE video_provider_accounts
SET enabled = true,
    metadata_json = metadata_json || '{"production_authorized": true, "single_smoke_authorized": true}'::jsonb
WHERE id = 2;
