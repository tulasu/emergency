-- name: CreateUser :exec
INSERT INTO users (id, login, password_hash, role, blocked_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetUserByID :one
SELECT id, login, password_hash, role, blocked_at, created_at
FROM users
WHERE id = $1;

-- name: GetUserByLogin :one
SELECT id, login, password_hash, role, blocked_at, created_at
FROM users
WHERE login = $1;

-- name: SetUserBlocked :exec
UPDATE users
SET blocked_at = $2
WHERE id = $1;

-- name: SetUserRole :exec
UPDATE users
SET role = $2
WHERE id = $1;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, expires_at, created_at
FROM sessions
WHERE token_hash = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < $1;
