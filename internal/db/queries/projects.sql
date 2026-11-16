-- name: ListProjectsWithProgress :many
-- Sidebar: all visible projects incl. progress in one query.
SELECT p.id, p.customer_id, p.name, p.status, p.start_date, p.due_date, p.created_at, p.include_in_customer_progress, p.owner_id, p.board_position, p.background,
       CAST(COALESCE(o.name, '') AS TEXT) AS owner_name,
       CAST(COUNT(i.id) AS INTEGER) AS total,
       CAST(COUNT(i.done_at) AS INTEGER) AS done
FROM projects p
LEFT JOIN users o ON o.id = p.owner_id
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
UPDATE projects SET name = ?, description = ?, status = ?, start_date = ?, due_date = ?, include_in_customer_progress = ?, owner_id = ?, redmine_ticket = ?
WHERE id = ?
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;

-- name: SearchProjects :many
-- scope: empty = all, 'active' = not archived, 'archived' = only archived.
-- Active projects first; archived ones carry the date of their archiving.
SELECT p.id, p.name, p.customer_id, c.name AS customer_name, p.status, a.created_at AS archived_at
FROM projects p
JOIN customers c ON c.id = p.customer_id
LEFT JOIN project_activity a ON p.status = 'archived'
     AND a.id = (SELECT MAX(x.id) FROM project_activity x WHERE x.project_id = p.id AND x.kind = 'status' AND x.detail = 'archived')
WHERE instr(lower(p.name), lower(sqlc.arg(needle))) > 0
  AND (sqlc.arg(scope) = '' OR (p.status = 'archived') = (sqlc.arg(scope) = 'archived'))
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY p.status = 'archived', p.name COLLATE NOCASE
LIMIT 10;

-- name: ListArchivedProjects :many
-- Archive page / customer archive tab: archived projects incl. progress. When
-- and by whom comes from the latest "archived" status entry of the activity
-- log (NULL for projects archived before the log existed).
SELECT p.id, p.customer_id, p.name, c.name AS customer_name,
       a.created_at AS archived_at, a.user_id AS archived_by, u.name AS archived_by_name,
       CAST((SELECT COUNT(*) FROM checklist_items i JOIN checklists cl ON cl.id = i.checklist_id
             WHERE cl.project_id = p.id AND i.kind = 'item') AS INTEGER) AS total,
       CAST((SELECT COUNT(i.done_at) FROM checklist_items i JOIN checklists cl ON cl.id = i.checklist_id
             WHERE cl.project_id = p.id AND i.kind = 'item') AS INTEGER) AS done
FROM projects p
JOIN customers c ON c.id = p.customer_id
LEFT JOIN project_activity a
       ON a.id = (SELECT MAX(x.id) FROM project_activity x WHERE x.project_id = p.id AND x.kind = 'status' AND x.detail = 'archived')
LEFT JOIN users u ON u.id = a.user_id
WHERE p.status = 'archived'
  AND (sqlc.narg(customer_id) IS NULL OR p.customer_id = sqlc.narg(customer_id))
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY a.created_at IS NULL, a.created_at DESC, p.name COLLATE NOCASE;

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

-- name: SetProjectBoardPlace :exec
-- Board drag and drop: column (status) and position within the column.
UPDATE projects SET status = ?, board_position = ? WHERE id = ?;
