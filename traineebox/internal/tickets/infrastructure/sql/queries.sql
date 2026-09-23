-- name: CreateTicket :exec
INSERT INTO tickets (
    id, group_id, title, body, max_attempts,
    available_from, available_until, duration_seconds, created_by, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: GetTicketByID :one
SELECT id, group_id, title, body, max_attempts,
       available_from, available_until, duration_seconds, created_by, created_at
FROM tickets
WHERE id = $1;

-- name: ListTicketsByGroup :many
SELECT id, group_id, title, body, max_attempts,
       available_from, available_until, duration_seconds, created_by, created_at
FROM tickets
WHERE group_id = $1
ORDER BY created_at;

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
    id, ticket_id, user_id, attempt_no, status, started_at, deadline_at, finished_at, score
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: CreateAttemptAnswer :exec
INSERT INTO attempt_answers (
    attempt_id, incident_type_code, applicant_last_name, applicant_first_name,
    caller_number, dictated_number, notes, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetAttemptByID :one
SELECT id, ticket_id, user_id, attempt_no, status, started_at, deadline_at, finished_at, score
FROM ticket_attempts
WHERE id = $1;

-- name: GetAttemptAnswer :one
SELECT attempt_id, incident_type_code, applicant_last_name, applicant_first_name,
       caller_number, dictated_number, notes, updated_at
FROM attempt_answers
WHERE attempt_id = $1;

-- name: ListAttemptAnswerTags :many
SELECT tag_code FROM attempt_answer_tags WHERE attempt_id = $1;

-- name: ListAttemptAnswerServices :many
SELECT service_code FROM attempt_answer_services WHERE attempt_id = $1;

-- name: FindInProgressAttempt :one
SELECT id, ticket_id, user_id, attempt_no, status, started_at, deadline_at, finished_at, score
FROM ticket_attempts
WHERE ticket_id = $1 AND user_id = $2 AND status = 'in_progress';

-- name: ListAttemptsByTicketUser :many
SELECT id, ticket_id, user_id, attempt_no, status, started_at, deadline_at, finished_at, score
FROM ticket_attempts
WHERE ticket_id = $1 AND user_id = $2
ORDER BY attempt_no;

-- name: CountFinishedAttempts :one
SELECT count(*)::int AS count
FROM ticket_attempts
WHERE ticket_id = $1 AND user_id = $2 AND status IN ('submitted', 'timed_out');

-- name: MaxAttemptNo :one
SELECT COALESCE(max(attempt_no), 0)::int AS max_no
FROM ticket_attempts
WHERE ticket_id = $1 AND user_id = $2;

-- name: UpdateAttempt :exec
UPDATE ticket_attempts
SET status = $2, finished_at = $3, score = $4
WHERE id = $1;

-- name: UpdateAttemptAnswer :exec
UPDATE attempt_answers
SET incident_type_code = $2,
    applicant_last_name = $3,
    applicant_first_name = $4,
    caller_number = $5,
    dictated_number = $6,
    notes = $7,
    updated_at = $8
WHERE attempt_id = $1;

-- name: DeleteAttemptAnswerTags :exec
DELETE FROM attempt_answer_tags WHERE attempt_id = $1;

-- name: DeleteAttemptAnswerServices :exec
DELETE FROM attempt_answer_services WHERE attempt_id = $1;

-- name: InsertAttemptAnswerTag :exec
INSERT INTO attempt_answer_tags (attempt_id, tag_code) VALUES ($1, $2);

-- name: InsertAttemptAnswerService :exec
INSERT INTO attempt_answer_services (attempt_id, service_code) VALUES ($1, $2);

-- name: GetMemberRole :one
SELECT member_role
FROM group_members
WHERE group_id = $1 AND user_id = $2;
