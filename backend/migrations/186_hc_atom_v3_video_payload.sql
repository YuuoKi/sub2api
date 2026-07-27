-- Persist the exact employee V3 video request for asynchronous dispatch.
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS request_payload JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE video_tasks DROP CONSTRAINT IF EXISTS video_tasks_duration_seconds_check;
ALTER TABLE video_tasks DROP CONSTRAINT IF EXISTS video_tasks_resolution_check;
ALTER TABLE video_tasks ADD CONSTRAINT video_tasks_duration_seconds_check CHECK (
    (provider = 'hc_atom_seedance_v3' AND duration_seconds > 0)
    OR (provider <> 'hc_atom_seedance_v3' AND duration_seconds = 4)
);
ALTER TABLE video_tasks ADD CONSTRAINT video_tasks_resolution_check CHECK (
    provider = 'hc_atom_seedance_v3' OR resolution = '720p'
);
