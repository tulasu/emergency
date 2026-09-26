-- +goose Up

CREATE TABLE topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE topic_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body_md TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX topic_articles_topic_id_idx ON topic_articles(topic_id);

CREATE TABLE article_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES topic_articles(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    storage_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX article_attachments_article_id_idx ON article_attachments(article_id);

CREATE TABLE modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    position INT NOT NULL DEFAULT 0,
    duration_seconds INT NULL CHECK (duration_seconds IS NULL OR duration_seconds > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lessons_module_id_idx ON lessons(module_id);

CREATE TABLE variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX variants_lesson_id_idx ON variants(lesson_id);

CREATE TABLE user_modules (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    assigned_by UUID NOT NULL REFERENCES users(id),
    source_group_id UUID NULL REFERENCES groups(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, module_id)
);

DROP TABLE IF EXISTS call_turns;
DROP TABLE IF EXISTS attempt_calls;
DROP TABLE IF EXISTS attempt_answer_services;
DROP TABLE IF EXISTS attempt_answer_tags;
DROP TABLE IF EXISTS attempt_answers;
DROP TABLE IF EXISTS ticket_attempts;
DROP TABLE IF EXISTS reference_answer_services;
DROP TABLE IF EXISTS reference_answer_tags;
DROP TABLE IF EXISTS ticket_reference_answers;
DROP TABLE IF EXISTS ticket_generation_jobs;
DROP TABLE IF EXISTS tickets;

CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id),
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    scenario JSONB NOT NULL DEFAULT '{}'::jsonb,
    scenario_version TEXT NOT NULL DEFAULT '',
    reference JSONB NOT NULL DEFAULT '{}'::jsonb,
    mode TEXT NOT NULL DEFAULT 'voice' CHECK (mode IN ('voice', 'text')),
    briefing TEXT NOT NULL DEFAULT ''
);

CREATE INDEX tickets_variant_id_idx ON tickets(variant_id);
CREATE INDEX tickets_topic_id_idx ON tickets(topic_id);

CREATE TABLE ticket_reference_answers (
    ticket_id UUID PRIMARY KEY REFERENCES tickets(id) ON DELETE CASCADE,
    incident_type_code TEXT NOT NULL,
    applicant_last_name TEXT NOT NULL DEFAULT '',
    applicant_first_name TEXT NOT NULL DEFAULT '',
    caller_number TEXT NOT NULL DEFAULT '',
    dictated_number TEXT NOT NULL DEFAULT ''
);

CREATE TABLE reference_answer_tags (
    ticket_id UUID NOT NULL REFERENCES ticket_reference_answers(ticket_id) ON DELETE CASCADE,
    tag_code TEXT NOT NULL,
    PRIMARY KEY (ticket_id, tag_code)
);

CREATE TABLE reference_answer_services (
    ticket_id UUID NOT NULL REFERENCES ticket_reference_answers(ticket_id) ON DELETE CASCADE,
    service_code TEXT NOT NULL,
    PRIMARY KEY (ticket_id, service_code)
);

CREATE TABLE ticket_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted_by UUID NOT NULL REFERENCES users(id),
    attempt_no INT NOT NULL CHECK (attempt_no > 0),
    status TEXT NOT NULL CHECK (status IN ('available', 'in_progress', 'submitted', 'timed_out')),
    started_at TIMESTAMPTZ NULL,
    deadline_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    score SMALLINT NULL CHECK (score IS NULL OR (score >= 1 AND score <= 100)),
    report JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (variant_id, user_id, attempt_no)
);

CREATE UNIQUE INDEX ticket_attempts_one_open_idx
    ON ticket_attempts (variant_id, user_id)
    WHERE status IN ('available', 'in_progress');

CREATE INDEX ticket_attempts_variant_user_idx ON ticket_attempts(variant_id, user_id);
CREATE INDEX ticket_attempts_user_id_idx ON ticket_attempts(user_id);

CREATE TABLE attempt_answers (
    attempt_id UUID NOT NULL REFERENCES ticket_attempts(id) ON DELETE CASCADE,
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    incident_type_code TEXT NULL,
    applicant_last_name TEXT NOT NULL DEFAULT '',
    applicant_first_name TEXT NOT NULL DEFAULT '',
    caller_number TEXT NOT NULL DEFAULT '',
    dictated_number TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '' CHECK (char_length(notes) <= 1000),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (attempt_id, ticket_id)
);

CREATE TABLE attempt_answer_tags (
    attempt_id UUID NOT NULL,
    ticket_id UUID NOT NULL,
    tag_code TEXT NOT NULL,
    PRIMARY KEY (attempt_id, ticket_id, tag_code),
    FOREIGN KEY (attempt_id, ticket_id) REFERENCES attempt_answers(attempt_id, ticket_id) ON DELETE CASCADE
);

CREATE TABLE attempt_answer_services (
    attempt_id UUID NOT NULL,
    ticket_id UUID NOT NULL,
    service_code TEXT NOT NULL,
    PRIMARY KEY (attempt_id, ticket_id, service_code),
    FOREIGN KEY (attempt_id, ticket_id) REFERENCES attempt_answers(attempt_id, ticket_id) ON DELETE CASCADE
);

CREATE TABLE ticket_generation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id),
    created_by UUID NOT NULL REFERENCES users(id),
    prompt TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN (
            'queued', 'enriching', 'filling_pii', 'picking_type', 'tagging_type',
            'tagging_common', 'building_services', 'building_dialog', 'checking_dialog',
            'building_ref', 'ready', 'failed', 'cancelled', 'published'
        )),
    version INT NOT NULL DEFAULT 1,
    scenario_text TEXT NOT NULL DEFAULT '',
    draft_title TEXT NOT NULL DEFAULT '',
    draft_reference JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    attempts INT NOT NULL DEFAULT 0,
    published_ticket_id UUID NULL REFERENCES tickets(id) ON DELETE SET NULL,
    claimed_by TEXT NOT NULL DEFAULT '',
    claimed_at TIMESTAMPTZ NULL,
    lease_until TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ticket_generation_jobs_variant_id_idx ON ticket_generation_jobs(variant_id);
CREATE INDEX ticket_generation_jobs_status_idx ON ticket_generation_jobs(status);
CREATE INDEX ticket_generation_jobs_claim_idx
    ON ticket_generation_jobs(status, lease_until)
    WHERE status IN (
        'queued', 'enriching', 'filling_pii', 'picking_type', 'tagging_type',
        'tagging_common', 'building_services', 'building_dialog', 'checking_dialog', 'building_ref'
    );

CREATE TABLE attempt_calls (
    id UUID PRIMARY KEY,
    attempt_id UUID NOT NULL REFERENCES ticket_attempts(id) ON DELETE CASCADE,
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scenario_id TEXT NOT NULL DEFAULT '',
    bank_digest TEXT NOT NULL DEFAULT '',
    channel_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'originating'
        CHECK (status IN ('originating', 'ringing', 'answered', 'completed', 'no_answer', 'failed', 'timed_out')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX attempt_calls_attempt_idx ON attempt_calls(attempt_id);
CREATE INDEX attempt_calls_ticket_user_idx ON attempt_calls(ticket_id, user_id);

CREATE TABLE call_turns (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    call_id UUID NOT NULL REFERENCES attempt_calls(id) ON DELETE CASCADE,
    n INT NOT NULL CHECK (n > 0),
    utterance TEXT NOT NULL DEFAULT '',
    reply TEXT NOT NULL DEFAULT '',
    style TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (call_id, n)
);

CREATE INDEX call_turns_call_idx ON call_turns(call_id);

-- +goose Down

DROP TABLE IF EXISTS call_turns;
DROP TABLE IF EXISTS attempt_calls;
DROP TABLE IF EXISTS ticket_generation_jobs;
DROP TABLE IF EXISTS attempt_answer_services;
DROP TABLE IF EXISTS attempt_answer_tags;
DROP TABLE IF EXISTS attempt_answers;
DROP TABLE IF EXISTS ticket_attempts;
DROP TABLE IF EXISTS reference_answer_services;
DROP TABLE IF EXISTS reference_answer_tags;
DROP TABLE IF EXISTS ticket_reference_answers;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS user_modules;
DROP TABLE IF EXISTS variants;
DROP TABLE IF EXISTS lessons;
DROP TABLE IF EXISTS modules;
DROP TABLE IF EXISTS article_attachments;
DROP TABLE IF EXISTS topic_articles;
DROP TABLE IF EXISTS topics;

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
    scenario JSONB NOT NULL DEFAULT '{}'::jsonb,
    scenario_version TEXT NOT NULL DEFAULT '',
    reference JSONB NOT NULL DEFAULT '{}'::jsonb,
    mode TEXT NOT NULL DEFAULT 'voice' CHECK (mode IN ('voice', 'text')),
    briefing TEXT NOT NULL DEFAULT '',
    CHECK (available_from IS NULL OR available_until IS NULL OR available_from <= available_until)
);

CREATE TABLE ticket_reference_answers (
    ticket_id UUID PRIMARY KEY REFERENCES tickets(id) ON DELETE CASCADE,
    incident_type_code TEXT NOT NULL,
    applicant_last_name TEXT NOT NULL DEFAULT '',
    applicant_first_name TEXT NOT NULL DEFAULT '',
    caller_number TEXT NOT NULL DEFAULT '',
    dictated_number TEXT NOT NULL DEFAULT ''
);

CREATE TABLE reference_answer_tags (
    ticket_id UUID NOT NULL REFERENCES ticket_reference_answers(ticket_id) ON DELETE CASCADE,
    tag_code TEXT NOT NULL,
    PRIMARY KEY (ticket_id, tag_code)
);

CREATE TABLE reference_answer_services (
    ticket_id UUID NOT NULL REFERENCES ticket_reference_answers(ticket_id) ON DELETE CASCADE,
    service_code TEXT NOT NULL,
    PRIMARY KEY (ticket_id, service_code)
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

CREATE TABLE attempt_answers (
    attempt_id UUID PRIMARY KEY REFERENCES ticket_attempts(id) ON DELETE CASCADE,
    incident_type_code TEXT NULL,
    applicant_last_name TEXT NOT NULL DEFAULT '',
    applicant_first_name TEXT NOT NULL DEFAULT '',
    caller_number TEXT NOT NULL DEFAULT '',
    dictated_number TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '' CHECK (char_length(notes) <= 1000),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attempt_answer_tags (
    attempt_id UUID NOT NULL REFERENCES attempt_answers(attempt_id) ON DELETE CASCADE,
    tag_code TEXT NOT NULL,
    PRIMARY KEY (attempt_id, tag_code)
);

CREATE TABLE attempt_answer_services (
    attempt_id UUID NOT NULL REFERENCES attempt_answers(attempt_id) ON DELETE CASCADE,
    service_code TEXT NOT NULL,
    PRIMARY KEY (attempt_id, service_code)
);

CREATE TABLE ticket_generation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),
    prompt TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    version INT NOT NULL DEFAULT 1,
    scenario_text TEXT NOT NULL DEFAULT '',
    draft_title TEXT NOT NULL DEFAULT '',
    draft_reference JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    attempts INT NOT NULL DEFAULT 0,
    published_ticket_id UUID NULL REFERENCES tickets(id) ON DELETE SET NULL,
    claimed_by TEXT NOT NULL DEFAULT '',
    claimed_at TIMESTAMPTZ NULL,
    lease_until TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attempt_calls (
    id UUID PRIMARY KEY,
    attempt_id UUID NOT NULL REFERENCES ticket_attempts(id) ON DELETE CASCADE,
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scenario_id TEXT NOT NULL DEFAULT '',
    bank_digest TEXT NOT NULL DEFAULT '',
    channel_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'originating',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE call_turns (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    call_id UUID NOT NULL REFERENCES attempt_calls(id) ON DELETE CASCADE,
    n INT NOT NULL CHECK (n > 0),
    utterance TEXT NOT NULL DEFAULT '',
    reply TEXT NOT NULL DEFAULT '',
    style TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (call_id, n)
);
