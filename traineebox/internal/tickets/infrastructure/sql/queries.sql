-- name: CreateTicket :exec
INSERT INTO tickets (
    id, variant_id, topic_id, title, body, created_by, created_at,
    scenario, scenario_version, mode, briefing
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: GetTicketByID :one
SELECT id, variant_id, topic_id, title, body, created_by, created_at,
       scenario, scenario_version, audio_digest, audio_status, reference, mode, briefing
FROM tickets
WHERE id = $1;

-- name: ListTicketsByVariant :many
SELECT id, variant_id, topic_id, title, body, created_by, created_at,
       scenario, scenario_version, audio_digest, audio_status, reference, mode, briefing
FROM tickets
WHERE variant_id = $1
ORDER BY created_at;

-- name: UpdateTicket :exec
UPDATE tickets SET title = $2, body = $3, topic_id = $4 WHERE id = $1;

-- name: DeleteTicket :exec
DELETE FROM tickets WHERE id = $1;

-- name: UpdateDialogSnapshot :execrows
UPDATE tickets
SET scenario = $2, scenario_version = $3, audio_digest = $4, audio_status = $5
WHERE id = $1;

-- name: UpdateTicketAudioStatus :execrows
UPDATE tickets SET audio_status = $3 WHERE id = $1 AND audio_digest = $2 AND (audio_status = 'pending' OR $3 = 'ready');

-- name: UpsertReferenceAnswer :exec
INSERT INTO ticket_reference_answers (
    ticket_id, incident_type_code, applicant_last_name, applicant_first_name,
    caller_number, dictated_number
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (ticket_id) DO UPDATE SET
    incident_type_code = EXCLUDED.incident_type_code,
    applicant_last_name = EXCLUDED.applicant_last_name,
    applicant_first_name = EXCLUDED.applicant_first_name,
    caller_number = EXCLUDED.caller_number,
    dictated_number = EXCLUDED.dictated_number;

-- name: DeleteReferenceAnswerTags :exec
DELETE FROM reference_answer_tags WHERE ticket_id = $1;

-- name: DeleteReferenceAnswerServices :exec
DELETE FROM reference_answer_services WHERE ticket_id = $1;

-- name: InsertReferenceAnswerTag :exec
INSERT INTO reference_answer_tags (ticket_id, tag_code) VALUES ($1, $2);

-- name: InsertReferenceAnswerService :exec
INSERT INTO reference_answer_services (ticket_id, service_code) VALUES ($1, $2);

-- name: GetReferenceAnswer :one
SELECT ticket_id, incident_type_code, applicant_last_name, applicant_first_name,
       caller_number, dictated_number
FROM ticket_reference_answers
WHERE ticket_id = $1;

-- name: ListReferenceAnswerTags :many
SELECT tag_code FROM reference_answer_tags WHERE ticket_id = $1;

-- name: ListReferenceAnswerServices :many
SELECT service_code FROM reference_answer_services WHERE ticket_id = $1;

-- name: CreateAttempt :exec
INSERT INTO ticket_attempts (
    id, variant_id, user_id, granted_by, attempt_no, status,
    started_at, deadline_at, finished_at, score, report
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetAttemptByID :one
SELECT id, variant_id, user_id, granted_by, attempt_no, status,
       started_at, deadline_at, finished_at, score, report
FROM ticket_attempts
WHERE id = $1;

-- name: FindOpenAttempt :one
SELECT id, variant_id, user_id, granted_by, attempt_no, status,
       started_at, deadline_at, finished_at, score, report
FROM ticket_attempts
WHERE variant_id = $1 AND user_id = $2 AND status IN ('available', 'in_progress');

-- name: ListAttemptsByVariantUser :many
SELECT id, variant_id, user_id, granted_by, attempt_no, status,
       started_at, deadline_at, finished_at, score, report
FROM ticket_attempts
WHERE variant_id = $1 AND user_id = $2
ORDER BY attempt_no;

-- name: ListAttemptsByUser :many
SELECT id, variant_id, user_id, granted_by, attempt_no, status,
       started_at, deadline_at, finished_at, score, report
FROM ticket_attempts
WHERE user_id = $1
ORDER BY attempt_no;

-- name: HasAnyAttempt :one
SELECT EXISTS(
    SELECT 1 FROM ticket_attempts WHERE variant_id = $1 AND user_id = $2
) AS exists;

-- name: MaxAttemptNo :one
SELECT COALESCE(max(attempt_no), 0)::int AS max_no
FROM ticket_attempts
WHERE variant_id = $1 AND user_id = $2;

-- name: UpdateAttempt :exec
UPDATE ticket_attempts
SET status = $2, started_at = $3, deadline_at = $4, finished_at = $5, score = $6, report = $7
WHERE id = $1;

-- name: UpsertAttemptAnswer :exec
INSERT INTO attempt_answers (
    attempt_id, ticket_id, incident_type_code, applicant_last_name, applicant_first_name,
    caller_number, dictated_number, notes, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (attempt_id, ticket_id) DO UPDATE SET
    incident_type_code = EXCLUDED.incident_type_code,
    applicant_last_name = EXCLUDED.applicant_last_name,
    applicant_first_name = EXCLUDED.applicant_first_name,
    caller_number = EXCLUDED.caller_number,
    dictated_number = EXCLUDED.dictated_number,
    notes = EXCLUDED.notes,
    updated_at = EXCLUDED.updated_at;

-- name: ListAttemptAnswers :many
SELECT attempt_id, ticket_id, incident_type_code, applicant_last_name, applicant_first_name,
       caller_number, dictated_number, notes, updated_at
FROM attempt_answers
WHERE attempt_id = $1;

-- name: GetAttemptAnswer :one
SELECT attempt_id, ticket_id, incident_type_code, applicant_last_name, applicant_first_name,
       caller_number, dictated_number, notes, updated_at
FROM attempt_answers
WHERE attempt_id = $1 AND ticket_id = $2;

-- name: ListAttemptAnswerTags :many
SELECT tag_code FROM attempt_answer_tags WHERE attempt_id = $1 AND ticket_id = $2;

-- name: ListAttemptAnswerServices :many
SELECT service_code FROM attempt_answer_services WHERE attempt_id = $1 AND ticket_id = $2;

-- name: DeleteAttemptAnswerTags :exec
DELETE FROM attempt_answer_tags WHERE attempt_id = $1 AND ticket_id = $2;

-- name: DeleteAttemptAnswerServices :exec
DELETE FROM attempt_answer_services WHERE attempt_id = $1 AND ticket_id = $2;

-- name: InsertAttemptAnswerTag :exec
INSERT INTO attempt_answer_tags (attempt_id, ticket_id, tag_code) VALUES ($1, $2, $3);

-- name: InsertAttemptAnswerService :exec
INSERT INTO attempt_answer_services (attempt_id, ticket_id, service_code) VALUES ($1, $2, $3);

-- name: GetMemberRole :one
SELECT member_role
FROM group_members
WHERE group_id = $1 AND user_id = $2;
