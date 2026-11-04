-- name: ListProjectsWithProgress :many
-- Sidebar: all visible projects incl. progress in one query.
SELECT p.id, p.customer_id, p.name, p.status, p.due_date, p.include_in_customer_progress,
       CAST(COUNT(i.id) AS INTEGER) AS total,
       CAST(COUNT(i.done_at) AS INTEGER) AS done
FROM projects p
LEFT JOIN checklists c ON c.project_id = p.id
LEFT JOIN checklist_items i ON i.checklist_id = c.id AND i.kind = 'item'
WHERE (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
GROUP BY p.id
ORDER BY p.name COLLATE NOCASE, p.id;

-- name: ListProjectsByCustomer :many
SELECT p.id, p.customer_id, p.name, p.description, p.status, p.due_date, p.include_in_customer_progress, p.created_at,
       CAST(COUNT(i.id) AS INTEGER) AS total,
       CAST(COUNT(i.done_at) AS INTEGER) AS done
FROM projects p
LEFT JOIN checklists c ON c.project_id = p.id
LEFT JOIN checklist_items i ON i.checklist_id = c.id AND i.kind = 'item'
WHERE p.customer_id = ?
GROUP BY p.id
ORDER BY p.name COLLATE NOCASE, p.id;

-- name: GetProject :one
SELECT * FROM projects WHERE id = ?;

-- name: CreateProject :one
INSERT INTO projects (customer_id, name, description, status, due_date)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects SET name = ?, description = ?, status = ?, due_date = ?, include_in_customer_progress = ?
WHERE id = ?
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;

-- name: SearchProjects :many
SELECT p.id, p.name, p.customer_id, c.name AS customer_name
FROM projects p
JOIN customers c ON c.id = p.customer_id
WHERE instr(lower(p.name), lower(sqlc.arg(needle))) > 0
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY p.name COLLATE NOCASE
LIMIT 10;

-- name: ListProjectTeams :many
-- Overview: everyone assigned to a checklist or an item of a project,
-- without duplicates per project.
SELECT DISTINCT c.project_id, u.id AS user_id, u.name AS user_name
FROM checklists c
JOIN projects p ON p.id = c.project_id
LEFT JOIN checklist_items i ON i.checklist_id = c.id AND i.kind = 'item'
JOIN users u ON u.id = c.assignee_id OR u.id = i.assignee_id
WHERE (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY c.project_id, u.name COLLATE NOCASE;

-- name: ListProjectNextSteps :many
-- Overview: first open item per project, in checklist and item order.
SELECT p.id AS project_id,
       CAST(COALESCE((SELECT i.title
                      FROM checklist_items i
                      JOIN checklists c ON c.id = i.checklist_id
                      WHERE c.project_id = p.id AND i.kind = 'item' AND i.done_at IS NULL
                      ORDER BY c.position, c.id, i.position, i.id
                      LIMIT 1), '') AS TEXT) AS next_step
FROM projects p
WHERE (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)));
