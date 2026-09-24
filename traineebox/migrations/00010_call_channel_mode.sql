-- +goose Up
-- Review fixes: ARI channel id for real hangup (F), mode canon voice/text (M).
ALTER TABLE attempt_calls ADD COLUMN IF NOT EXISTS channel_id TEXT NOT NULL DEFAULT '';

-- Spec Never: no mode=both. Legacy 'both' rows (if any) fall back to voice.
UPDATE tickets SET mode = 'voice' WHERE mode = 'both';
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_mode_check;
ALTER TABLE tickets ADD CONSTRAINT tickets_mode_check CHECK (mode IN ('voice', 'text'));

-- +goose Down
ALTER TABLE tickets DROP CONSTRAINT IF EXISTS tickets_mode_check;
ALTER TABLE tickets ADD CONSTRAINT tickets_mode_check CHECK (mode IN ('voice', 'text', 'both'));
ALTER TABLE attempt_calls DROP COLUMN IF EXISTS channel_id;
