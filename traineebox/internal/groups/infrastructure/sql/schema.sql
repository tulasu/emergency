CREATE TABLE groups (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE group_members (
    group_id UUID NOT NULL,
    user_id UUID NOT NULL,
    member_role TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    role TEXT NOT NULL,
    blocked_at TIMESTAMPTZ NULL
);
