-- Template library: categories, usage log, custom variables.

-- name: ListTemplateCategories :many
SELECT * FROM template_categories ORDER BY position, name COLLATE NOCASE, id;

-- name: GetTemplateCategory :one
SELECT * FROM template_categories WHERE id = ?;

-- name: CreateTemplateCategory :one
INSERT INTO template_categories (name, color, position)
VALUES (?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM template_categories))
RETURNING *;

-- name: UpdateTemplateCategory :one
UPDATE template_categories SET name = ?, color = ? WHERE id = ? RETURNING *;

-- name: DeleteTemplateCategory :exec
DELETE FROM template_categories WHERE id = ?;

-- name: AddTemplateUse :exec
INSERT INTO template_uses (template_id, project_id, checklist_id, rule_id, used_at) VALUES (?, ?, ?, ?, ?);

-- name: ListTemplateVariables :many
SELECT * FROM template_variables ORDER BY name;

-- name: GetTemplateVariable :one
SELECT * FROM template_variables WHERE id = ?;

-- name: CreateTemplateVariable :one
INSERT INTO template_variables (name, description, default_value, source) VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateTemplateVariable :one
UPDATE template_variables SET name = ?, description = ?, default_value = ?, source = ? WHERE id = ? RETURNING *;

-- name: DeleteTemplateVariable :exec
DELETE FROM template_variables WHERE id = ?;

-- name: ListAllTemplateTexts :many
-- name and items of all templates (to find the variables they use)
SELECT t.id AS template_id, t.name AS text FROM checklist_templates t
UNION ALL
SELECT ti.template_id, ti.title || ' ' || ti.description FROM template_items ti;

-- name: VariableContext :one
-- values for the system variables of a project
SELECT p.name AS project_name, p.redmine_ticket, p.start_date, p.due_date,
       c.id AS customer_id, c.name AS customer_name, c.location, c.contact_name, c.contact_email
FROM projects p JOIN customers c ON c.id = p.customer_id
WHERE p.id = ?;

-- name: LatestProjectID :one
SELECT CAST(COALESCE(MAX(id), 0) AS INTEGER) FROM projects;
