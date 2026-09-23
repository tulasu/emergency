-- +goose Up

-- Catalog tags are re-seeded; clear answer links so we can reshape incident_tags.
DELETE FROM attempt_answer_tags;
DELETE FROM reference_answer_tags;
DELETE FROM incident_tags;

CREATE TABLE incident_tag_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_type_id UUID NOT NULL REFERENCES incident_types(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    selection_mode TEXT NOT NULL CHECK (selection_mode IN ('single', 'multi')),
    sort_order INT NOT NULL DEFAULT 0,
    UNIQUE (incident_type_id, code)
);

CREATE INDEX incident_tag_groups_type_id_idx ON incident_tag_groups(incident_type_id);

ALTER TABLE incident_tags
    ADD COLUMN group_id UUID NOT NULL REFERENCES incident_tag_groups(id) ON DELETE CASCADE,
    ADD COLUMN sort_order INT NOT NULL DEFAULT 0;

ALTER TABLE incident_tag_groups
    ADD COLUMN parent_tag_id UUID NULL REFERENCES incident_tags(id) ON DELETE CASCADE;

CREATE INDEX incident_tags_group_id_idx ON incident_tags(group_id);

-- +goose Down

DROP INDEX IF EXISTS incident_tags_group_id_idx;

ALTER TABLE incident_tag_groups DROP COLUMN IF EXISTS parent_tag_id;

ALTER TABLE incident_tags
    DROP COLUMN IF EXISTS sort_order,
    DROP COLUMN IF EXISTS group_id;

DROP TABLE IF EXISTS incident_tag_groups;
