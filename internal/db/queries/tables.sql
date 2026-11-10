-- Tables (spreadsheets) of a project; the workbook is stored as JSON in data.

-- name: ListTablesByProject :many
-- Tables of a project without their data (number of sheets and last change only).
SELECT t.id, t.project_id, t.title, t.template_id, t.position, t.version, t.updated_at,
       CAST(CASE WHEN json_valid(t.data) THEN json_array_length(t.data) ELSE 0 END AS INTEGER) AS sheets,
       CAST(COALESCE(u.name, '') AS TEXT) AS updated_by_name
FROM project_tables t
LEFT JOIN users u ON u.id = t.updated_by
WHERE t.project_id = ?
ORDER BY t.position, t.id;

-- name: ListTablesForSidebar :many
-- Sidebar: id and title of all visible tables (breadcrumb, project tab).
SELECT t.id, t.project_id, t.title
FROM project_tables t
JOIN projects p ON p.id = t.project_id
WHERE (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY t.position, t.id;

-- name: GetTableWithCustomer :one
SELECT t.id, t.project_id, t.title, t.template_id, t.position, t.data, t.version, t.created_at, t.updated_at, t.updated_by,
       CAST(COALESCE(u.name, '') AS TEXT) AS updated_by_name, p.customer_id
FROM project_tables t
JOIN projects p ON p.id = t.project_id
LEFT JOIN users u ON u.id = t.updated_by
WHERE t.id = ?;

-- name: ListTableDataByProject :many
-- complete tables of a project (duplicating a project)
SELECT * FROM project_tables WHERE project_id = ? ORDER BY position, id;

-- name: NextTablePosition :one
SELECT CAST(COALESCE(MAX(position), -1) + 1 AS INTEGER) FROM project_tables WHERE project_id = ?;

-- name: CreateTable :one
INSERT INTO project_tables (project_id, title, template_id, position, data, created_at, updated_at, updated_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateTableTitle :one
UPDATE project_tables SET title = ? WHERE id = ? RETURNING *;

-- name: SaveTableData :one
-- only when nobody saved in between (version); no row = conflict
UPDATE project_tables
SET data = sqlc.arg(data), version = version + 1, updated_at = sqlc.arg(updated_at), updated_by = sqlc.arg(updated_by)
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING version, updated_at;

-- name: DeleteTable :exec
DELETE FROM project_tables WHERE id = ?;

-- name: SetTablePosition :execrows
UPDATE project_tables SET position = ? WHERE id = ? AND project_id = ?;

-- name: CreateTableTemplate :one
INSERT INTO checklist_templates (name, description, kind, sheet_data) VALUES (?, ?, 'table', ?) RETURNING *;

-- name: UpdateTemplateSheet :execrows
UPDATE checklist_templates SET sheet_data = ? WHERE id = ? AND kind = 'table';

-- name: AddTableTemplateUse :exec
INSERT INTO template_uses (template_id, project_id, table_id, rule_id, used_at) VALUES (?, ?, ?, ?, ?);
