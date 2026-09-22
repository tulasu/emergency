-- name: CreateGroup :exec
INSERT INTO groups (id, name, created_at)
VALUES ($1, $2, $3);

-- name: CreateGroupMember :exec
INSERT INTO group_members (group_id, user_id, member_role, joined_at)
VALUES ($1, $2, $3, $4);

-- name: GetGroupByID :one
SELECT id, name, created_at
FROM groups
WHERE id = $1;

-- name: ListGroupMembers :many
SELECT group_id, user_id, member_role, joined_at
FROM group_members
WHERE group_id = $1
ORDER BY joined_at;

-- name: ListAllGroups :many
SELECT id, name, created_at
FROM groups
ORDER BY created_at;

-- name: ListGroupIDsByMember :many
SELECT group_id
FROM group_members
WHERE user_id = $1;

-- name: UpdateGroupName :exec
UPDATE groups
SET name = $2
WHERE id = $1;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = $1;

-- name: DeleteGroupMember :exec
DELETE FROM group_members
WHERE group_id = $1 AND user_id = $2;

-- name: GetAccountByID :one
SELECT id, role, blocked_at
FROM users
WHERE id = $1;
