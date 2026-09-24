-- name: CreateGenerationJob :exec
INSERT INTO ticket_generation_jobs (
    id, group_id, created_by, prompt, status, version,
    scenario_text, draft_title, draft_reference, error_message, attempts,
    published_ticket_id, claimed_by, claimed_at, lease_until, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16, $17
);

-- name: GetGenerationJobByID :one
SELECT
    id, group_id, created_by, prompt, status, version,
    scenario_text, draft_title, draft_reference, error_message, attempts,
    published_ticket_id, claimed_by, claimed_at, lease_until, created_at, updated_at
FROM ticket_generation_jobs
WHERE id = $1;

-- name: ListGenerationJobsByGroup :many
SELECT
    id, group_id, created_by, prompt, status, version,
    scenario_text, draft_title, draft_reference, error_message, attempts,
    published_ticket_id, claimed_by, claimed_at, lease_until, created_at, updated_at
FROM ticket_generation_jobs
WHERE group_id = $1
ORDER BY created_at DESC;

-- name: UpdateGenerationJobCAS :execrows
UPDATE ticket_generation_jobs SET
    status = $1,
    version = version + 1,
    scenario_text = $2,
    draft_title = $3,
    draft_reference = $4,
    error_message = $5,
    attempts = $6,
    published_ticket_id = $7,
    claimed_by = $8,
    claimed_at = $9,
    lease_until = $10,
    updated_at = $11
WHERE id = $12
  AND status = $13
  AND version = $14;

-- name: DeleteGenerationJob :exec
DELETE FROM ticket_generation_jobs WHERE id = $1;
