UPDATE video_provider_accounts
SET base_url = 'https://ark.cn-beijing.volces.com/api/v3',
    default_model = 'doubao-seedance-2-0-260128',
    updated_at = NOW()
WHERE provider = 'seedance';

CREATE UNIQUE INDEX IF NOT EXISTS uq_video_provider_seedance_group_model
    ON video_provider_accounts (group_id, provider, default_model)
    WHERE provider = 'seedance';
