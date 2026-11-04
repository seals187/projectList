-- name: ListChecklistsByProject :many
-- Checklists of a project incl. progress (items only, no headings)
-- and latest due date, computed by aggregation.
SELECT c.id, c.project_id, c.title, c.template_id, c.position, c.assignee_id,
       CAST(COALESCE(a.name, '') AS TEXT) AS assignee_name,
       CAST(COUNT(i.id) AS INTEGER) AS total,
       CAST(COUNT(i.done_at) AS INTEGER) AS done,
       CAST(COALESCE(MAX(i.due_date), '') AS TEXT) AS due_date
FROM checklists c
LEFT JOIN checklist_items i ON i.checklist_id = c.id AND i.kind = 'item'
LEFT JOIN users a ON a.id = c.assignee_id
WHERE c.project_id = ?
GROUP BY c.id
ORDER BY c.position, c.id;

-- name: ListChecklistsWithProgress :many
-- Sidebar: all visible checklists incl. progress in one query.
SELECT c.id, c.project_id, c.title,
       CAST(COUNT(i.id) AS INTEGER) AS total,
       CAST(COUNT(i.done_at) AS INTEGER) AS done,
       CAST(COALESCE(MIN(CASE WHEN i.done_at IS NULL THEN i.due_date END), '') AS TEXT) AS next_due
FROM checklists c
JOIN projects p ON p.id = c.project_id
LEFT JOIN checklist_items i ON i.checklist_id = c.id AND i.kind = 'item'
WHERE (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
GROUP BY c.id
ORDER BY c.position, c.id;

-- name: ListItemsByChecklist :many
SELECT i.id, i.checklist_id, i.kind, i.title, i.description, i.note_color, i.bg_color, i.status, i.indent, i.lang, i.position,
       i.assignee_id, i.due_date, i.done_at, i.done_by,
       a.name AS assignee_name, d.name AS done_by_name
FROM checklist_items i
LEFT JOIN users a ON a.id = i.assignee_id
LEFT JOIN users d ON d.id = i.done_by
WHERE i.checklist_id = ?
ORDER BY i.position, i.id;

-- name: GetChecklistWithCustomer :one
SELECT c.id, c.project_id, c.title, c.template_id, c.position, c.assignee_id,
       CAST(COALESCE(a.name, '') AS TEXT) AS assignee_name, p.customer_id
FROM checklists c
JOIN projects p ON p.id = c.project_id
LEFT JOIN users a ON a.id = c.assignee_id
WHERE c.id = ?;

-- name: NextChecklistPosition :one
SELECT CAST(COALESCE(MAX(position), -1) + 1 AS INTEGER) FROM checklists WHERE project_id = ?;

-- name: CreateChecklist :one
INSERT INTO checklists (project_id, title, template_id, position)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateChecklistTitle :one
UPDATE checklists SET title = ? WHERE id = ? RETURNING *;

-- name: DeleteChecklist :exec
DELETE FROM checklists WHERE id = ?;

-- name: SetChecklistPosition :execrows
UPDATE checklists SET position = ? WHERE id = ? AND project_id = ?;

-- name: SearchChecklists :many
SELECT c.id, c.title, c.project_id, p.name AS project_name, p.customer_id
FROM checklists c
JOIN projects p ON p.id = c.project_id
WHERE instr(lower(c.title), lower(sqlc.arg(needle))) > 0
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY c.title COLLATE NOCASE
LIMIT 10;

-- name: SetChecklistAssignee :exec
UPDATE checklists SET assignee_id = ? WHERE id = ?;

-- name: ListMyChecklists :many
-- Checklists assigned to a user (not archived), incl. progress.
SELECT c.id, c.title, c.project_id, p.name AS project_name, cu.id AS customer_id, cu.name AS customer_name,
       CAST(COUNT(i.id) AS INTEGER) AS total,
       CAST(COUNT(i.done_at) AS INTEGER) AS done,
       CAST(COALESCE(MIN(CASE WHEN i.done_at IS NULL THEN i.due_date END), '') AS TEXT) AS due_date
FROM checklists c
JOIN projects p ON p.id = c.project_id
JOIN customers cu ON cu.id = p.customer_id
LEFT JOIN checklist_items i ON i.checklist_id = c.id AND i.kind = 'item'
WHERE c.assignee_id = sqlc.arg(assignee_id)
  AND p.status != 'archived'
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
GROUP BY c.id
ORDER BY cu.name COLLATE NOCASE, p.name COLLATE NOCASE, c.position;
