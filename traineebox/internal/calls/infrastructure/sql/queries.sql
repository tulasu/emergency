-- name: CreateCall :exec
INSERT INTO attempt_calls (id, attempt_id, ticket_id, user_id, scenario_id, bank_digest, channel_id, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetCallByID :one
SELECT id, attempt_id, ticket_id, user_id, scenario_id, bank_digest, channel_id, status, created_at, updated_at
FROM attempt_calls WHERE id = $1;

-- name: ListCallsByAttempt :many
SELECT id, attempt_id, ticket_id, user_id, scenario_id, bank_digest, channel_id, status, created_at, updated_at
FROM attempt_calls WHERE attempt_id = $1 ORDER BY created_at;

-- name: SetCallStatus :exec
UPDATE attempt_calls SET status = $2, updated_at = now() WHERE id = $1;

-- name: UpsertCallTurn :exec
INSERT INTO call_turns (call_id, n, utterance, reply, style)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (call_id, n) DO UPDATE SET utterance = EXCLUDED.utterance, reply = EXCLUDED.reply, style = EXCLUDED.style;
