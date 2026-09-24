-- sqlc schema mirror (post-00008)
CREATE TABLE dialog_slots (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    kind TEXT NOT NULL,
    family TEXT NOT NULL
);

CREATE TABLE slot_questions (
    slot_id TEXT NOT NULL,
    question TEXT NOT NULL,
    source TEXT NOT NULL
);

CREATE TABLE bank_state (
    version TEXT PRIMARY KEY,
    digest TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
