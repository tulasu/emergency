CREATE TABLE incident_types (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    title TEXT NOT NULL
);

CREATE TABLE incident_tags (
    id UUID PRIMARY KEY,
    incident_type_id UUID NOT NULL,
    code TEXT NOT NULL,
    title TEXT NOT NULL
);

CREATE TABLE emergency_services (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    title TEXT NOT NULL
);

CREATE TABLE tickets (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    max_attempts INT NULL,
    available_from TIMESTAMPTZ NULL,
    available_until TIMESTAMPTZ NULL,
    duration_seconds INT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE ticket_reference_answers (
    ticket_id UUID PRIMARY KEY,
    incident_type_id UUID NOT NULL,
    applicant_last_name TEXT NOT NULL,
    applicant_first_name TEXT NOT NULL,
    caller_number TEXT NOT NULL,
    dictated_number TEXT NOT NULL
);

CREATE TABLE reference_answer_tags (
    ticket_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    PRIMARY KEY (ticket_id, tag_id)
);

CREATE TABLE reference_answer_services (
    ticket_id UUID NOT NULL,
    service_id UUID NOT NULL,
    PRIMARY KEY (ticket_id, service_id)
);

CREATE TABLE ticket_attempts (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL,
    user_id UUID NOT NULL,
    attempt_no INT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    deadline_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    score SMALLINT NULL
);

CREATE TABLE attempt_answers (
    attempt_id UUID PRIMARY KEY,
    incident_type_id UUID NULL,
    applicant_last_name TEXT NOT NULL,
    applicant_first_name TEXT NOT NULL,
    caller_number TEXT NOT NULL,
    dictated_number TEXT NOT NULL,
    notes TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE attempt_answer_tags (
    attempt_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    PRIMARY KEY (attempt_id, tag_id)
);

CREATE TABLE attempt_answer_services (
    attempt_id UUID NOT NULL,
    service_id UUID NOT NULL,
    PRIMARY KEY (attempt_id, service_id)
);

CREATE TABLE group_members (
    group_id UUID NOT NULL,
    user_id UUID NOT NULL,
    member_role TEXT NOT NULL,
    PRIMARY KEY (group_id, user_id)
);
