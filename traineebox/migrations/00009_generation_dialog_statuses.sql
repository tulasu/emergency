-- +goose Up
-- ticketgen is a pure function; traineebox owns the job pipeline.
-- New mid-pipeline states for dialog snapshot build + lint check.
ALTER TABLE ticket_generation_jobs DROP CONSTRAINT IF EXISTS ticket_generation_jobs_status_check;

ALTER TABLE ticket_generation_jobs
    ADD CONSTRAINT ticket_generation_jobs_status_check CHECK (status IN (
        'queued',
        'enriching',
        'filling_pii',
        'picking_type',
        'tagging_type',
        'tagging_common',
        'building_services',
        'building_dialog',
        'checking_dialog',
        'building_ref',
        'ready',
        'failed',
        'cancelled',
        'published'
    ));

DROP INDEX IF EXISTS ticket_generation_jobs_claim_idx;
CREATE INDEX ticket_generation_jobs_claim_idx
    ON ticket_generation_jobs(status, lease_until)
    WHERE status IN (
        'queued',
        'enriching',
        'filling_pii',
        'picking_type',
        'tagging_type',
        'tagging_common',
        'building_services',
        'building_dialog',
        'checking_dialog',
        'building_ref'
    );

-- +goose Down
ALTER TABLE ticket_generation_jobs DROP CONSTRAINT IF EXISTS ticket_generation_jobs_status_check;

ALTER TABLE ticket_generation_jobs
    ADD CONSTRAINT ticket_generation_jobs_status_check CHECK (status IN (
        'queued',
        'enriching',
        'filling_pii',
        'picking_type',
        'tagging_type',
        'tagging_common',
        'building_services',
        'building_ref',
        'ready',
        'failed',
        'cancelled',
        'published'
    ));

DROP INDEX IF EXISTS ticket_generation_jobs_claim_idx;
CREATE INDEX ticket_generation_jobs_claim_idx
    ON ticket_generation_jobs(status, lease_until)
    WHERE status IN (
        'queued',
        'enriching',
        'filling_pii',
        'picking_type',
        'tagging_type',
        'tagging_common',
        'building_services',
        'building_ref'
    );
