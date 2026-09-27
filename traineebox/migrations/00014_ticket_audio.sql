-- +goose Up
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS audio_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS audio_status TEXT NOT NULL DEFAULT 'none';

-- +goose Down
ALTER TABLE tickets DROP COLUMN IF EXISTS audio_status;
ALTER TABLE tickets DROP COLUMN IF EXISTS audio_digest;
