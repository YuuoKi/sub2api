-- Idempotent internal mock video provider for zero-cost simulation.
-- No real credentials, base URLs, or paid-provider seeds.

INSERT INTO video_provider_accounts (
    provider,
    display_name,
    enabled,
    encrypted_api_key,
    masked_key,
    base_url,
    default_model,
    metadata_json
)
VALUES (
    'mock',
    'Internal Mock Video',
    TRUE,
    '',
    '',
    '',
    'mock-video-v1',
    '{"pricing_source":"internal_simulation","pricing_version":"simulation-v1"}'::jsonb
)
ON CONFLICT (provider, display_name) DO NOTHING;
