CREATE TABLE users (
    id UUID PRIMARY KEY
);

CREATE TABLE groups (
    id UUID PRIMARY KEY
);

CREATE TABLE topics (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE topic_articles (
    id UUID PRIMARY KEY,
    topic_id UUID NOT NULL,
    title TEXT NOT NULL,
    body_md TEXT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE article_attachments (
    id UUID PRIMARY KEY,
    article_id UUID NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE modules (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL,
    success_threshold INT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE lessons (
    id UUID PRIMARY KEY,
    module_id UUID NOT NULL,
    title TEXT NOT NULL,
    position INT NOT NULL,
    duration_seconds INT NULL,
    archived_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE variants (
    id UUID PRIMARY KEY,
    lesson_id UUID NOT NULL,
    title TEXT NOT NULL,
    position INT NOT NULL,
    status TEXT NOT NULL,
    is_primary BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE user_modules (
    user_id UUID NOT NULL,
    module_id UUID NOT NULL,
    assigned_by UUID NOT NULL,
    source_group_id UUID NULL,
    assigned_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, module_id)
);

CREATE TABLE tickets (
    id UUID PRIMARY KEY,
    variant_id UUID NULL,
    topic_id UUID NOT NULL
);

CREATE TABLE ticket_attempts (
    id UUID PRIMARY KEY,
    variant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status TEXT NOT NULL
);
