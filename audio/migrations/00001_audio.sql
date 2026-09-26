-- +goose Up
CREATE TABLE blobs (
    hash BYTEA PRIMARY KEY,
    voice TEXT NOT NULL,
    rate INT NOT NULL,
    bytes INT NOT NULL,
    dur_s REAL NOT NULL,
    refcount INT NOT NULL DEFAULT 0 CHECK (refcount >= 0),
    unreferenced_since TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX blobs_orphan_idx ON blobs (unreferenced_since) WHERE refcount = 0;

CREATE TABLE refs (
    ticket_id UUID NOT NULL,
    hash BYTEA NOT NULL REFERENCES blobs(hash) ON DELETE RESTRICT,
    PRIMARY KEY (ticket_id, hash)
);

CREATE TABLE manifests (
    ticket_id UUID PRIMARY KEY,
    scenario_digest TEXT NOT NULL,
    voice TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'ready', 'partial', 'error')),
    fragments JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE synth_queue (
    hash BYTEA PRIMARY KEY,
    ticket_id UUID NOT NULL,
    frag_id TEXT NOT NULL,
    voice TEXT NOT NULL,
    text TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    claim_token UUID,
    claim_until TIMESTAMPTZ,
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

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

DROP TRIGGER IF EXISTS refs_adjust_blob_refcount_trigger ON refs;
CREATE TRIGGER refs_adjust_blob_refcount_trigger
AFTER INSERT OR DELETE ON refs
FOR EACH ROW EXECUTE FUNCTION refs_adjust_blob_refcount();

-- +goose Down
DROP TRIGGER IF EXISTS refs_adjust_blob_refcount_trigger ON refs;
DROP FUNCTION IF EXISTS refs_adjust_blob_refcount();
DROP TABLE IF EXISTS synth_queue;
DROP TABLE IF EXISTS manifests;
DROP TABLE IF EXISTS refs;
DROP TABLE IF EXISTS blobs;
