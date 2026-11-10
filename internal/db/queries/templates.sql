-- name: ListTemplates :many
SELECT t.id, t.name, t.description, t.category_id, t.kind,
       CAST((SELECT COUNT(*) FROM template_items ti WHERE ti.template_id = t.id AND ti.kind = 'item') AS INTEGER) AS item_count,
       CAST((SELECT COUNT(*) FROM template_uses u WHERE u.template_id = t.id) AS INTEGER) AS used_count,
       CAST((SELECT COALESCE(MAX(u.used_at), '') FROM template_uses u WHERE u.template_id = t.id) AS TEXT) AS last_used
FROM checklist_templates t
ORDER BY t.name COLLATE NOCASE, t.id;

-- name: GetTemplate :one
SELECT * FROM checklist_templates WHERE id = ?;

-- name: CreateTemplate :one
INSERT INTO checklist_templates (name, description) VALUES (?, ?) RETURNING *;

-- name: UpdateTemplate :one
UPDATE checklist_templates SET name = ?, description = ?, category_id = ? WHERE id = ? RETURNING *;

-- name: DeleteTemplate :exec
DELETE FROM checklist_templates WHERE id = ?;

-- name: ListTemplateItems :many
SELECT * FROM template_items WHERE template_id = ? ORDER BY position, id;

-- name: GetTemplateItem :one
SELECT * FROM template_items WHERE id = ?;

-- name: NextTemplateItemPosition :one
SELECT CAST(COALESCE(MAX(position), -1) + 1 AS INTEGER) FROM template_items WHERE template_id = ?;

-- name: ShiftTemplateItemPositions :exec
UPDATE template_items SET position = position + 1 WHERE template_id = ? AND position >= ?;

-- name: CreateTemplateItem :one
INSERT INTO template_items (template_id, kind, title, description, note_color, bg_color, indent, lang, position)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateTemplateItem :one
UPDATE template_items SET kind = ?, title = ?, description = ?, note_color = ?, bg_color = ?, indent = ?, lang = ? WHERE id = ? RETURNING *;

-- name: DeleteTemplateItem :exec
DELETE FROM template_items WHERE id = ?;

-- name: SetTemplateItemPosition :execrows
UPDATE template_items SET position = ? WHERE id = ? AND template_id = ?;
