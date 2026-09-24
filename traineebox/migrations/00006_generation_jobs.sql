-- +goose Up
CREATE TABLE ticket_generation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),
    prompt TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN (
            'queued', 'enriching', 'filling_pii', 'building_ref',
            'ready', 'failed', 'cancelled', 'published'
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

CREATE INDEX ticket_generation_jobs_group_id_idx ON ticket_generation_jobs(group_id);
CREATE INDEX ticket_generation_jobs_status_idx ON ticket_generation_jobs(status);
CREATE INDEX ticket_generation_jobs_claim_idx
    ON ticket_generation_jobs(status, lease_until)
    WHERE status IN ('queued', 'enriching', 'filling_pii', 'building_ref');

-- +goose Down
DROP TABLE IF EXISTS ticket_generation_jobs;
