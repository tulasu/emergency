-- name: CreateTopic :exec
INSERT INTO topics (id, title, created_by, created_at)
VALUES ($1, $2, $3, $4);

-- name: GetTopicByID :one
SELECT id, title, created_by, created_at
FROM topics
WHERE id = $1;

-- name: ListTopics :many
SELECT id, title, created_by, created_at
FROM topics
ORDER BY created_at;

-- name: UpdateTopic :exec
UPDATE topics SET title = $2 WHERE id = $1;

-- name: DeleteTopic :exec
DELETE FROM topics WHERE id = $1;

-- name: CreateArticle :exec
INSERT INTO topic_articles (id, topic_id, title, body_md, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetArticleByID :one
SELECT id, topic_id, title, body_md, created_by, created_at, updated_at
FROM topic_articles
WHERE id = $1;

-- name: ListArticlesByTopic :many
SELECT id, topic_id, title, body_md, created_by, created_at, updated_at
FROM topic_articles
WHERE topic_id = $1
ORDER BY created_at;

-- name: UpdateArticle :exec
UPDATE topic_articles SET title = $2, body_md = $3, updated_at = $4 WHERE id = $1;

-- name: DeleteArticle :exec
DELETE FROM topic_articles WHERE id = $1;

-- name: CreateAttachment :exec
INSERT INTO article_attachments (id, article_id, filename, content_type, size_bytes, storage_key, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetAttachmentByID :one
SELECT id, article_id, filename, content_type, size_bytes, storage_key, created_at
FROM article_attachments
WHERE id = $1;

-- name: ListAttachmentsByArticle :many
SELECT id, article_id, filename, content_type, size_bytes, storage_key, created_at
FROM article_attachments
WHERE article_id = $1
ORDER BY created_at;

-- name: DeleteAttachment :exec
DELETE FROM article_attachments WHERE id = $1;

-- name: CreateModule :exec
INSERT INTO modules (id, title, description, status, success_threshold, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetModuleByID :one
SELECT id, title, description, status, success_threshold, created_by, created_at
FROM modules
WHERE id = $1;

-- name: ListModules :many
SELECT id, title, description, status, success_threshold, created_by, created_at
FROM modules
ORDER BY created_at DESC;

-- name: ListModulesFiltered :many
SELECT id, title, description, status, success_threshold, created_by, created_at
FROM modules
WHERE (
    sqlc.narg(q)::text IS NULL OR sqlc.narg(q)::text = ''
    OR title ILIKE '%' || sqlc.narg(q)::text || '%'
    OR description ILIKE '%' || sqlc.narg(q)::text || '%'
)
AND (
    sqlc.narg(scope)::text IS NULL OR sqlc.narg(scope)::text = '' OR sqlc.narg(scope)::text = 'all'
    OR (sqlc.narg(scope)::text = 'active' AND status = 'active')
    OR (sqlc.narg(scope)::text = 'draft' AND status = 'draft')
    OR (sqlc.narg(scope)::text = 'archived' AND status = 'archived')
    OR (sqlc.narg(scope)::text = 'mine' AND created_by = sqlc.narg(actor_id)::uuid)
    OR (sqlc.narg(scope)::text = 'team' AND created_by <> sqlc.narg(actor_id)::uuid AND status <> 'archived')
)
ORDER BY created_at DESC;

-- name: CountLessonsByModule :one
SELECT COUNT(*)::int AS count FROM lessons WHERE module_id = $1 AND archived_at IS NULL;

-- name: UpdateModule :exec
UPDATE modules SET title = $2, description = $3, status = $4, success_threshold = $5 WHERE id = $1;

-- name: DeleteModule :exec
DELETE FROM modules WHERE id = $1;

-- name: CreateLesson :exec
INSERT INTO lessons (id, module_id, title, position, duration_seconds, archived_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetLessonByID :one
SELECT id, module_id, title, position, duration_seconds, archived_at, created_at
FROM lessons
WHERE id = $1;

-- name: ListLessonsByModule :many
SELECT id, module_id, title, position, duration_seconds, archived_at, created_at
FROM lessons
WHERE module_id = $1 AND archived_at IS NULL
ORDER BY position, created_at;

-- name: ListLessonsPool :many
SELECT id, module_id, title, position, duration_seconds, archived_at, created_at
FROM lessons
WHERE archived_at IS NULL
AND (sqlc.narg(q)::text IS NULL OR sqlc.narg(q)::text = '' OR title ILIKE '%' || sqlc.narg(q)::text || '%')
ORDER BY created_at DESC;

-- name: CountVariantsByLesson :one
SELECT COUNT(*)::int AS count FROM variants WHERE lesson_id = $1;

-- name: CountTicketsByLesson :one
SELECT COUNT(*)::int AS count
FROM tickets t
JOIN variants v ON v.id = t.variant_id
WHERE v.lesson_id = $1;

-- name: UpdateLesson :exec
UPDATE lessons SET title = $2, position = $3, duration_seconds = $4, archived_at = $5 WHERE id = $1;

-- name: DeleteLesson :exec
DELETE FROM lessons WHERE id = $1;

-- name: CreateVariant :exec
INSERT INTO variants (id, lesson_id, title, position, status, is_primary, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetVariantByID :one
SELECT id, lesson_id, title, position, status, is_primary, created_at
FROM variants
WHERE id = $1;

-- name: ListVariantsByLesson :many
SELECT id, lesson_id, title, position, status, is_primary, created_at
FROM variants
WHERE lesson_id = $1
ORDER BY position, created_at;

-- name: ClearPrimaryVariants :exec
UPDATE variants SET is_primary = false WHERE lesson_id = $1;

-- name: UpdateVariant :exec
UPDATE variants SET title = $2, position = $3, status = $4, is_primary = $5 WHERE id = $1;

-- name: DeleteVariant :exec
DELETE FROM variants WHERE id = $1;

-- name: GetLessonByVariant :one
SELECT l.id, l.module_id, l.title, l.position, l.duration_seconds, l.archived_at, l.created_at
FROM lessons l
JOIN variants v ON v.lesson_id = l.id
WHERE v.id = $1;

-- name: UpsertUserModule :exec
INSERT INTO user_modules (user_id, module_id, assigned_by, source_group_id, assigned_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, module_id) DO UPDATE SET
    assigned_by = EXCLUDED.assigned_by,
    source_group_id = EXCLUDED.source_group_id,
    assigned_at = EXCLUDED.assigned_at;

-- name: ListUserModules :many
SELECT user_id, module_id, assigned_by, source_group_id, assigned_at
FROM user_modules
WHERE user_id = $1
ORDER BY assigned_at;

-- name: GetUserModule :one
SELECT user_id, module_id, assigned_by, source_group_id, assigned_at
FROM user_modules
WHERE user_id = $1 AND module_id = $2;

-- name: HasUserModule :one
SELECT EXISTS(
    SELECT 1 FROM user_modules WHERE user_id = $1 AND module_id = $2
) AS exists;

-- name: ListUserIDsByModule :many
SELECT user_id FROM user_modules WHERE module_id = $1;

-- name: ListUserModulesByModule :many
SELECT user_id, module_id, assigned_by, source_group_id, assigned_at
FROM user_modules
WHERE module_id = $1;

-- name: ListTopicIDsForUser :many
SELECT DISTINCT t.topic_id
FROM tickets t
JOIN ticket_attempts a ON a.variant_id = t.variant_id
WHERE a.user_id = $1 AND t.variant_id IS NOT NULL;
