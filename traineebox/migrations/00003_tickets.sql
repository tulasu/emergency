-- +goose Up
CREATE TABLE incident_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL
);

CREATE TABLE incident_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_type_id UUID NOT NULL REFERENCES incident_types(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    UNIQUE (incident_type_id, code)
);

CREATE INDEX incident_tags_type_id_idx ON incident_tags(incident_type_id);

CREATE TABLE emergency_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL
);

CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    max_attempts INT NULL CHECK (max_attempts IS NULL OR max_attempts > 0),
    available_from TIMESTAMPTZ NULL,
    available_until TIMESTAMPTZ NULL,
    duration_seconds INT NULL CHECK (duration_seconds IS NULL OR duration_seconds > 0),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (available_from IS NULL OR available_until IS NULL OR available_from <= available_until)
);

CREATE INDEX tickets_group_id_idx ON tickets(group_id);

CREATE TABLE ticket_reference_answers (
    ticket_id UUID PRIMARY KEY REFERENCES tickets(id) ON DELETE CASCADE,
    incident_type_id UUID NOT NULL REFERENCES incident_types(id),
    applicant_last_name TEXT NOT NULL DEFAULT '',
    applicant_first_name TEXT NOT NULL DEFAULT '',
    caller_number TEXT NOT NULL DEFAULT '',
    dictated_number TEXT NOT NULL DEFAULT ''
);

CREATE TABLE reference_answer_tags (
    ticket_id UUID NOT NULL REFERENCES ticket_reference_answers(ticket_id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES incident_tags(id),
    PRIMARY KEY (ticket_id, tag_id)
);

CREATE TABLE reference_answer_services (
    ticket_id UUID NOT NULL REFERENCES ticket_reference_answers(ticket_id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES emergency_services(id),
    PRIMARY KEY (ticket_id, service_id)
);

CREATE TABLE ticket_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attempt_no INT NOT NULL CHECK (attempt_no > 0),
    status TEXT NOT NULL CHECK (status IN ('in_progress', 'submitted', 'timed_out')),
    started_at TIMESTAMPTZ NOT NULL,
    deadline_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    score SMALLINT NULL CHECK (score IS NULL OR (score >= 1 AND score <= 100)),
    UNIQUE (ticket_id, user_id, attempt_no)
);

CREATE UNIQUE INDEX ticket_attempts_one_in_progress_idx
    ON ticket_attempts (ticket_id, user_id)
    WHERE status = 'in_progress';

CREATE INDEX ticket_attempts_ticket_user_idx ON ticket_attempts(ticket_id, user_id);
CREATE INDEX ticket_attempts_user_id_idx ON ticket_attempts(user_id);

CREATE TABLE attempt_answers (
    attempt_id UUID PRIMARY KEY REFERENCES ticket_attempts(id) ON DELETE CASCADE,
    incident_type_id UUID NULL REFERENCES incident_types(id),
    applicant_last_name TEXT NOT NULL DEFAULT '',
    applicant_first_name TEXT NOT NULL DEFAULT '',
    caller_number TEXT NOT NULL DEFAULT '',
    dictated_number TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '' CHECK (char_length(notes) <= 1000),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attempt_answer_tags (
    attempt_id UUID NOT NULL REFERENCES attempt_answers(attempt_id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES incident_tags(id),
    PRIMARY KEY (attempt_id, tag_id)
);

CREATE TABLE attempt_answer_services (
    attempt_id UUID NOT NULL REFERENCES attempt_answers(attempt_id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES emergency_services(id),
    PRIMARY KEY (attempt_id, service_id)
);

-- +goose Down
DROP TABLE IF EXISTS attempt_answer_services;
DROP TABLE IF EXISTS attempt_answer_tags;
DROP TABLE IF EXISTS attempt_answers;
DROP TABLE IF EXISTS ticket_attempts;
DROP TABLE IF EXISTS reference_answer_services;
DROP TABLE IF EXISTS reference_answer_tags;
DROP TABLE IF EXISTS ticket_reference_answers;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS emergency_services;
DROP TABLE IF EXISTS incident_tags;
DROP TABLE IF EXISTS incident_types;
