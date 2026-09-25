-- +goose Up
-- Port legacy question-bank compiler: record which scenario each formulation
-- came from (source_ref) so the bank can measure against unseen scenarios.
ALTER TABLE slot_questions ADD COLUMN IF NOT EXISTS source_ref TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE slot_questions DROP COLUMN IF EXISTS source_ref;
