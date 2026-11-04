
-- name: CountMyOpenTasks :one
SELECT COUNT(*)
FROM checklist_items i
JOIN checklists c ON c.id = i.checklist_id
JOIN projects p ON p.id = c.project_id
WHERE i.assignee_id = sqlc.arg(assignee_id)
  AND i.done_at IS NULL
  AND i.kind = 'item'
  AND p.status != 'archived'
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)));

-- name: ListMyTasks :many
SELECT i.id, i.title, i.description, i.note_color, i.due_date, i.done_at,
       c.id AS checklist_id, c.title AS checklist_title,
       p.id AS project_id, p.name AS project_name,
       cu.id AS customer_id, cu.name AS customer_name
FROM checklist_items i
JOIN checklists c ON c.id = i.checklist_id
JOIN projects p ON p.id = c.project_id
JOIN customers cu ON cu.id = p.customer_id
WHERE i.assignee_id = sqlc.arg(assignee_id)
  AND i.done_at IS NULL
  AND i.kind = 'item'
  AND p.status != 'archived'
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY i.due_date IS NULL, i.due_date, cu.name COLLATE NOCASE, p.name COLLATE NOCASE, c.position, i.position
LIMIT 500;

-- name: ListMyDoneTasks :many
-- Done items assigned to the user within the last 30 days, newest first.
SELECT i.id, i.title, i.description, i.note_color, i.due_date, i.done_at,
       c.id AS checklist_id, c.title AS checklist_title,
       p.id AS project_id, p.name AS project_name,
       cu.id AS customer_id, cu.name AS customer_name
FROM checklist_items i
JOIN checklists c ON c.id = i.checklist_id
JOIN projects p ON p.id = c.project_id
JOIN customers cu ON cu.id = p.customer_id
WHERE i.assignee_id = sqlc.arg(assignee_id)
  AND i.done_at IS NOT NULL
  AND i.done_at >= datetime('now', '-30 days')
  AND i.kind = 'item'
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY i.done_at DESC
LIMIT 200;
