-- +goose Up
DELETE FROM attempt_answer_services;
DELETE FROM attempt_answer_tags;
DELETE FROM attempt_answers;
DELETE FROM reference_answer_services;
DELETE FROM reference_answer_tags;
DELETE FROM ticket_reference_answers;

ALTER TABLE ticket_reference_answers
    DROP CONSTRAINT IF EXISTS ticket_reference_answers_incident_type_id_fkey;
ALTER TABLE attempt_answers
    DROP CONSTRAINT IF EXISTS attempt_answers_incident_type_id_fkey;
ALTER TABLE reference_answer_tags
    DROP CONSTRAINT IF EXISTS reference_answer_tags_tag_id_fkey;
ALTER TABLE reference_answer_services
    DROP CONSTRAINT IF EXISTS reference_answer_services_service_id_fkey;
ALTER TABLE attempt_answer_tags
    DROP CONSTRAINT IF EXISTS attempt_answer_tags_tag_id_fkey;
ALTER TABLE attempt_answer_services
    DROP CONSTRAINT IF EXISTS attempt_answer_services_service_id_fkey;

ALTER TABLE ticket_reference_answers RENAME COLUMN incident_type_id TO incident_type_code;
ALTER TABLE ticket_reference_answers
    ALTER COLUMN incident_type_code TYPE TEXT USING '';

ALTER TABLE attempt_answers RENAME COLUMN incident_type_id TO incident_type_code;
ALTER TABLE attempt_answers
    ALTER COLUMN incident_type_code TYPE TEXT USING NULL;
ALTER TABLE attempt_answers
    ALTER COLUMN incident_type_code DROP NOT NULL;

ALTER TABLE reference_answer_tags RENAME COLUMN tag_id TO tag_code;
ALTER TABLE reference_answer_tags
    ALTER COLUMN tag_code TYPE TEXT USING '';

ALTER TABLE reference_answer_services RENAME COLUMN service_id TO service_code;
ALTER TABLE reference_answer_services
    ALTER COLUMN service_code TYPE TEXT USING '';

ALTER TABLE attempt_answer_tags RENAME COLUMN tag_id TO tag_code;
ALTER TABLE attempt_answer_tags
    ALTER COLUMN tag_code TYPE TEXT USING '';

ALTER TABLE attempt_answer_services RENAME COLUMN service_id TO service_code;
ALTER TABLE attempt_answer_services
    ALTER COLUMN service_code TYPE TEXT USING '';

DROP TABLE IF EXISTS incident_tag_groups CASCADE;
DROP TABLE IF EXISTS incident_tags CASCADE;
DROP TABLE IF EXISTS incident_types CASCADE;
DROP TABLE IF EXISTS emergency_services CASCADE;

-- +goose Down
CREATE TABLE incident_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL
);

CREATE TABLE incident_tag_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_type_id UUID NOT NULL REFERENCES incident_types(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    selection_mode TEXT NOT NULL CHECK (selection_mode IN ('single', 'multi')),
    parent_tag_id UUID NULL,
    sort_order INT NOT NULL DEFAULT 0,
    UNIQUE (incident_type_id, code)
);

CREATE TABLE incident_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_type_id UUID NOT NULL REFERENCES incident_types(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES incident_tag_groups(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    UNIQUE (incident_type_id, code)
);

ALTER TABLE incident_tag_groups
    ADD CONSTRAINT incident_tag_groups_parent_tag_id_fkey
    FOREIGN KEY (parent_tag_id) REFERENCES incident_tags(id) ON DELETE SET NULL;

CREATE TABLE emergency_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL
);

DELETE FROM attempt_answer_services;
DELETE FROM attempt_answer_tags;
DELETE FROM attempt_answers;
DELETE FROM reference_answer_services;
DELETE FROM reference_answer_tags;
DELETE FROM ticket_reference_answers;

ALTER TABLE ticket_reference_answers RENAME COLUMN incident_type_code TO incident_type_id;
ALTER TABLE ticket_reference_answers
    ALTER COLUMN incident_type_id TYPE UUID USING gen_random_uuid();

ALTER TABLE attempt_answers RENAME COLUMN incident_type_code TO incident_type_id;
ALTER TABLE attempt_answers
    ALTER COLUMN incident_type_id TYPE UUID USING NULL;

ALTER TABLE reference_answer_tags RENAME COLUMN tag_code TO tag_id;
ALTER TABLE reference_answer_tags
    ALTER COLUMN tag_id TYPE UUID USING gen_random_uuid();

ALTER TABLE reference_answer_services RENAME COLUMN service_code TO service_id;
ALTER TABLE reference_answer_services
    ALTER COLUMN service_id TYPE UUID USING gen_random_uuid();

ALTER TABLE attempt_answer_tags RENAME COLUMN tag_code TO tag_id;
ALTER TABLE attempt_answer_tags
    ALTER COLUMN tag_id TYPE UUID USING gen_random_uuid();

ALTER TABLE attempt_answer_services RENAME COLUMN service_code TO service_id;
ALTER TABLE attempt_answer_services
    ALTER COLUMN service_id TYPE UUID USING gen_random_uuid();
