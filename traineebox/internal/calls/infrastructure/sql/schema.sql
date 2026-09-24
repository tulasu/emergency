-- sqlc schema mirror (post-00008)
CREATE TABLE attempt_calls (
    id UUID PRIMARY KEY,
    attempt_id UUID NOT NULL,
    ticket_id UUID NOT NULL,
    user_id UUID NOT NULL,
    scenario_id TEXT NOT NULL,
    bank_digest TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE call_turns (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    call_id UUID NOT NULL,
    n INT NOT NULL,
    utterance TEXT NOT NULL,
    reply TEXT NOT NULL,
    style TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
