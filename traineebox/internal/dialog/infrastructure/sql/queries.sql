-- name: ListSlotIDs :many
SELECT id FROM dialog_slots;

-- name: BankVersion :one
SELECT version, digest FROM bank_state ORDER BY updated_at DESC LIMIT 1;
