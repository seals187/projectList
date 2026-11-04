-- Project page: activity, files, watchers, app settings.

-- name: ListProjectActivity :many
SELECT a.id, a.user_id, CAST(COALESCE(u.name, '') AS TEXT) AS user_name, a.kind, a.subject, a.detail, a.created_at
FROM project_activity a
LEFT JOIN users u ON u.id = a.user_id
WHERE a.project_id = ?
ORDER BY a.created_at DESC, a.id DESC
LIMIT ?;

-- name: AddProjectActivity :exec
INSERT INTO project_activity (project_id, user_id, kind, subject, detail, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ChecklistProgress :one
SELECT CAST(COUNT(*) AS INTEGER) AS total, CAST(COUNT(done_at) AS INTEGER) AS done
FROM checklist_items
WHERE checklist_id = ? AND kind = 'item';

-- name: ItemTitle :one
SELECT title FROM checklist_items WHERE id = ?;

-- name: ListProjectFiles :many
SELECT f.id, f.name, f.mime, f.size, f.uploaded_by, CAST(COALESCE(u.name, '') AS TEXT) AS uploaded_by_name, f.created_at
FROM project_files f
LEFT JOIN users u ON u.id = f.uploaded_by
WHERE f.project_id = ?
ORDER BY f.created_at DESC, f.id DESC;

-- name: CountProjectFiles :one
SELECT CAST(COUNT(*) AS INTEGER) FROM project_files WHERE project_id = ?;

-- name: CreateProjectFile :one
INSERT INTO project_files (project_id, name, mime, size, uploaded_by, created_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: SetProjectFileData :exec
INSERT INTO project_file_data (file_id, data) VALUES (?, ?);

-- name: GetProjectFile :one
SELECT * FROM project_files WHERE id = ?;

-- name: GetProjectFileData :one
SELECT data FROM project_file_data WHERE file_id = ?;

-- name: DeleteProjectFile :exec
DELETE FROM project_files WHERE id = ?;

-- name: ListProjectWatchers :many
SELECT u.id, u.name
FROM project_watchers w
JOIN users u ON u.id = w.user_id
WHERE w.project_id = ?
ORDER BY u.name COLLATE NOCASE, u.id;

-- name: ClearProjectWatchers :exec
DELETE FROM project_watchers WHERE project_id = ?;

-- name: AddProjectWatcher :exec
INSERT INTO project_watchers (project_id, user_id) VALUES (?, ?)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- name: GetAppSetting :one
SELECT value FROM app_settings WHERE key = ?;

-- name: SetAppSetting :exec
INSERT INTO app_settings (key, value) VALUES (?, ?)
ON CONFLICT (key) DO UPDATE SET value = excluded.value;

-- name: ListOverviewChecklists :many
-- Overview list: per checklist the person responsible and the number of open
-- items whose due date is before today.
SELECT c.id, c.project_id, c.assignee_id, CAST(COALESCE(a.name, '') AS TEXT) AS assignee_name,
       CAST(COUNT(CASE WHEN i.kind = 'item' AND i.done_at IS NULL AND i.due_date IS NOT NULL AND i.due_date < sqlc.arg(today) THEN 1 END) AS INTEGER) AS overdue
FROM checklists c
JOIN projects p ON p.id = c.project_id
LEFT JOIN users a ON a.id = c.assignee_id
LEFT JOIN checklist_items i ON i.checklist_id = c.id
WHERE (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
GROUP BY c.id;

-- name: ListProjectLastActivity :many
-- Overview list: time of the latest activity per project.
SELECT a.project_id, a.created_at
FROM project_activity a
JOIN projects p ON p.id = a.project_id
WHERE a.created_at = (SELECT MAX(b.created_at) FROM project_activity b WHERE b.project_id = a.project_id)
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)));
