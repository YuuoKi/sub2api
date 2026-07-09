ALTER TABLE video_tasks
    ADD COLUMN IF NOT EXISTS upstream_video_id TEXT,
    ADD COLUMN IF NOT EXISTS audio_id TEXT;
