CREATE TABLE users (
    id UUID PRIMARY KEY
);

CREATE TABLE topics (
    id UUID PRIMARY KEY
);

CREATE TABLE variants (
    id UUID PRIMARY KEY
);

CREATE TABLE tickets (
    id UUID PRIMARY KEY
);

CREATE TABLE ticket_generation_jobs (
    id UUID PRIMARY KEY,
    variant_id UUID NOT NULL REFERENCES variants(id),
    topic_id UUID NOT NULL REFERENCES topics(id),
    created_by UUID NOT NULL REFERENCES users(id),
    prompt TEXT NOT NULL,
    status TEXT NOT NULL,
    version INT NOT NULL,
    scenario_text TEXT NOT NULL,
    draft_title TEXT NOT NULL,
    draft_reference JSONB NOT NULL,
    error_message TEXT NOT NULL,
    attempts INT NOT NULL,
    published_ticket_id UUID NULL REFERENCES tickets(id),
    claimed_by TEXT NOT NULL,
    claimed_at TIMESTAMPTZ NULL,
    lease_until TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
