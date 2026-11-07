-- name: GetItemWithCustomer :one
SELECT i.id, i.checklist_id, i.kind, i.done_at, p.customer_id
FROM checklist_items i
JOIN checklists c ON c.id = i.checklist_id
JOIN projects p ON p.id = c.project_id
WHERE i.id = ?;

-- name: GetItemFull :one
SELECT i.id, i.checklist_id, i.kind, i.title, i.description, i.note_color, i.bg_color, i.status, i.indent, i.lang, i.position,
       i.assignee_id, i.due_date, i.done_at, i.done_by,
       a.name AS assignee_name, d.name AS done_by_name
FROM checklist_items i
LEFT JOIN users a ON a.id = i.assignee_id
LEFT JOIN users d ON d.id = i.done_by
WHERE i.id = ?;

-- name: NextItemPosition :one
SELECT CAST(COALESCE(MAX(position), -1) + 1 AS INTEGER) FROM checklist_items WHERE checklist_id = ?;

-- name: ShiftItemPositions :exec
-- Make room to insert at a given position.
UPDATE checklist_items SET position = position + 1 WHERE checklist_id = ? AND position >= ?;

-- name: CreateItem :one
INSERT INTO checklist_items (checklist_id, kind, title, description, note_color, bg_color, status, indent, lang, position, assignee_id, due_date, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
RETURNING id;

-- name: UpdateItem :exec
UPDATE checklist_items
SET kind = ?, title = ?, description = ?, note_color = ?, bg_color = ?, status = ?, indent = ?, lang = ?, assignee_id = ?, due_date = ?
WHERE id = ?;

-- name: SetItemDone :exec
UPDATE checklist_items SET done_at = ?, done_by = ? WHERE id = ?;

-- name: DeleteItem :exec
DELETE FROM checklist_items WHERE id = ?;

-- name: SetItemPosition :execrows
UPDATE checklist_items SET position = ? WHERE id = ? AND checklist_id = ?;
