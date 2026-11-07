-- name: ListBackgrounds :many
SELECT id, mime, size, uploaded_by, created_at FROM backgrounds ORDER BY created_at DESC, id DESC;

-- name: CreateBackground :one
INSERT INTO backgrounds (mime, size, uploaded_by, created_at, data) VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: GetBackground :one
SELECT id, mime, size, uploaded_by, created_at FROM backgrounds WHERE id = ?;

-- name: GetBackgroundData :one
SELECT data FROM backgrounds WHERE id = ?;

-- name: DeleteBackground :exec
DELETE FROM backgrounds WHERE id = ?;

-- name: ClearBackgroundRefs :exec
-- A deleted image is removed from customers and projects that used it.
UPDATE customers SET background = '' WHERE background = sqlc.arg(ref);

-- name: ClearProjectBackgroundRefs :exec
UPDATE projects SET background = '' WHERE background = sqlc.arg(ref);

-- name: SetCustomerBackground :exec
UPDATE customers SET background = ? WHERE id = ?;

-- name: SetProjectBackground :exec
UPDATE projects SET background = ? WHERE id = ?;

-- name: SetCustomerColor :exec
-- Shared customer color (overview cards; with color_everywhere in all views).
UPDATE customers SET color = ?, color_everywhere = ? WHERE id = ?;
