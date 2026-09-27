-- +goose Up
ALTER TABLE blobs ALTER COLUMN unreferenced_since SET DEFAULT now();

-- Correct refcounts produced by the original trigger, and make every
-- unreferenced blob immediately eligible for TTL-based collection.
WITH counts AS (
    SELECT b.hash, count(r.hash)::INT AS refcount
    FROM blobs b
    LEFT JOIN refs r ON r.hash = b.hash
    GROUP BY b.hash
)
UPDATE blobs b
SET refcount = c.refcount,
    unreferenced_since = CASE
        WHEN c.refcount = 0 THEN COALESCE(b.unreferenced_since, now())
        ELSE NULL
    END
FROM counts c
WHERE b.hash = c.hash;

-- The original trigger updated a newly-referenced row twice. Replace it with
-- one update so every refs row contributes exactly one reference.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION refs_adjust_blob_refcount() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE blobs
           SET refcount = refcount + 1,
               unreferenced_since = NULL
         WHERE hash = NEW.hash;
        RETURN NEW;
    END IF;
    UPDATE blobs
       SET refcount = refcount - 1,
           unreferenced_since = CASE WHEN refcount = 1 THEN now() ELSE unreferenced_since END
     WHERE hash = OLD.hash;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

ALTER TABLE synth_queue ADD COLUMN IF NOT EXISTS voice TEXT;
ALTER TABLE synth_queue ADD COLUMN IF NOT EXISTS claim_token UUID;
ALTER TABLE synth_queue ADD COLUMN IF NOT EXISTS claim_until TIMESTAMPTZ;
ALTER TABLE synth_queue ADD COLUMN IF NOT EXISTS error TEXT NOT NULL DEFAULT '';

-- Existing queue work can use its owner manifest's voice. Work no longer
-- referenced by any manifest has no consumer, so it is safe to discard.
UPDATE synth_queue q
SET voice = (
    SELECT m.voice
    FROM manifests m
    WHERE m.ticket_id = q.ticket_id
       OR EXISTS (
           SELECT 1 FROM jsonb_each_text(m.fragments) e
           WHERE e.value = encode(q.hash, 'hex')
       )
    ORDER BY (m.ticket_id = q.ticket_id) DESC
    LIMIT 1
)
WHERE q.voice IS NULL;
DELETE FROM synth_queue WHERE voice IS NULL;
ALTER TABLE synth_queue ALTER COLUMN voice SET NOT NULL;
CREATE INDEX IF NOT EXISTS synth_queue_claim_idx
    ON synth_queue (claim_until, created_at) WHERE attempts < 3;

-- +goose Down
DROP INDEX IF EXISTS synth_queue_claim_idx;
ALTER TABLE synth_queue DROP COLUMN IF EXISTS error;
ALTER TABLE synth_queue DROP COLUMN IF EXISTS claim_until;
ALTER TABLE synth_queue DROP COLUMN IF EXISTS claim_token;
ALTER TABLE synth_queue DROP COLUMN IF EXISTS voice;
