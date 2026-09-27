-- +goose Up

ALTER TABLE modules
    ADD COLUMN status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'archived')),
    ADD COLUMN success_threshold INT NOT NULL DEFAULT 70
        CHECK (success_threshold >= 0 AND success_threshold <= 100);

ALTER TABLE lessons
    ADD COLUMN archived_at TIMESTAMPTZ NULL;

ALTER TABLE variants
    ADD COLUMN status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'approved')),
    ADD COLUMN is_primary BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX variants_one_primary_per_lesson_idx
    ON variants (lesson_id)
    WHERE is_primary;

ALTER TABLE tickets
    ALTER COLUMN variant_id DROP NOT NULL;

CREATE INDEX tickets_library_idx ON tickets (created_at DESC) WHERE variant_id IS NULL;

ALTER TABLE ticket_attempts
    ADD COLUMN available_from TIMESTAMPTZ NULL;

-- +goose Down

ALTER TABLE ticket_attempts DROP COLUMN IF EXISTS available_from;

DROP INDEX IF EXISTS tickets_library_idx;
ALTER TABLE tickets ALTER COLUMN variant_id SET NOT NULL;

DROP INDEX IF EXISTS variants_one_primary_per_lesson_idx;
ALTER TABLE variants DROP COLUMN IF EXISTS is_primary;
ALTER TABLE variants DROP COLUMN IF EXISTS status;

ALTER TABLE lessons DROP COLUMN IF EXISTS archived_at;

ALTER TABLE modules DROP COLUMN IF EXISTS success_threshold;
ALTER TABLE modules DROP COLUMN IF EXISTS status;
