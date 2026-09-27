CREATE TABLE topics (
    id UUID PRIMARY KEY
);

CREATE TABLE variants (
    id UUID PRIMARY KEY
);

CREATE TABLE tickets (
    id UUID PRIMARY KEY,
    variant_id UUID NULL,
    topic_id UUID NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
	scenario JSONB NOT NULL,
	scenario_version TEXT NOT NULL,
	audio_digest TEXT NOT NULL DEFAULT '',
	audio_status TEXT NOT NULL DEFAULT 'none',
	reference JSONB NOT NULL,
	mode TEXT NOT NULL,
	briefing TEXT NOT NULL
);

CREATE TABLE ticket_reference_answers (
    ticket_id UUID PRIMARY KEY,
    incident_type_code TEXT NOT NULL,
    applicant_last_name TEXT NOT NULL,
    applicant_first_name TEXT NOT NULL,
    caller_number TEXT NOT NULL,
    dictated_number TEXT NOT NULL
);

CREATE TABLE reference_answer_tags (
    ticket_id UUID NOT NULL,
    tag_code TEXT NOT NULL,
    PRIMARY KEY (ticket_id, tag_code)
);

CREATE TABLE reference_answer_services (
    ticket_id UUID NOT NULL,
    service_code TEXT NOT NULL,
    PRIMARY KEY (ticket_id, service_code)
);

CREATE TABLE ticket_attempts (
    id UUID PRIMARY KEY,
    variant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    granted_by UUID NOT NULL,
    attempt_no INT NOT NULL,
    status TEXT NOT NULL,
    available_from TIMESTAMPTZ NULL,
    started_at TIMESTAMPTZ NULL,
    deadline_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    score SMALLINT NULL,
    report JSONB NOT NULL
);

CREATE TABLE attempt_answers (
    attempt_id UUID NOT NULL,
    ticket_id UUID NOT NULL,
    incident_type_code TEXT NULL,
    applicant_last_name TEXT NOT NULL,
    applicant_first_name TEXT NOT NULL,
    caller_number TEXT NOT NULL,
    dictated_number TEXT NOT NULL,
    notes TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (attempt_id, ticket_id)
);

CREATE TABLE attempt_answer_tags (
    attempt_id UUID NOT NULL,
    ticket_id UUID NOT NULL,
    tag_code TEXT NOT NULL,
    PRIMARY KEY (attempt_id, ticket_id, tag_code)
);

CREATE TABLE attempt_answer_services (
    attempt_id UUID NOT NULL,
    ticket_id UUID NOT NULL,
    service_code TEXT NOT NULL,
    PRIMARY KEY (attempt_id, ticket_id, service_code)
);

CREATE TABLE group_members (
    group_id UUID NOT NULL,
    user_id UUID NOT NULL,
    member_role TEXT NOT NULL,
    PRIMARY KEY (group_id, user_id)
);
