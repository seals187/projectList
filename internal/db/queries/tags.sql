-- name: ListStatusLabels :many
SELECT status, label, color, wip_limit FROM status_labels ORDER BY status;

-- name: UpsertStatusLabel :exec
INSERT INTO status_labels (status, label, color, wip_limit) VALUES (?, ?, ?, ?)
ON CONFLICT (status) DO UPDATE SET label = excluded.label, color = excluded.color, wip_limit = excluded.wip_limit;

-- name: DeleteStatusLabel :exec
DELETE FROM status_labels WHERE status = ?;

-- name: ListTags :many
SELECT id, name, color FROM tags ORDER BY name COLLATE NOCASE, id;

-- name: GetTag :one
SELECT id, name, color FROM tags WHERE id = ?;

-- name: CreateTag :one
INSERT INTO tags (name, color) VALUES (?, ?)
RETURNING id, name, color;

-- name: UpdateTag :one
UPDATE tags SET name = ?, color = ? WHERE id = ?
RETURNING id, name, color;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: ListProjectTags :many
-- All assignments of the visible projects (sidebar, overview).
SELECT pt.project_id, pt.tag_id
FROM project_tags pt
JOIN projects p ON p.id = pt.project_id
JOIN tags t ON t.id = pt.tag_id
WHERE (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY pt.project_id, t.name COLLATE NOCASE;

-- name: ListTagIDsByProject :many
SELECT pt.tag_id FROM project_tags pt
JOIN tags t ON t.id = pt.tag_id
WHERE pt.project_id = ?
ORDER BY t.name COLLATE NOCASE;

-- name: ClearProjectTags :exec
DELETE FROM project_tags WHERE project_id = ?;

-- name: AddProjectTag :exec
INSERT INTO project_tags (project_id, tag_id) VALUES (?, ?)
ON CONFLICT (project_id, tag_id) DO NOTHING;
