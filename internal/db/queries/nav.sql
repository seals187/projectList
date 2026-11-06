-- name: ListFavorites :many
-- Only existing targets (the triggers of migration 00021 also clean up).
SELECT f.kind, f.target_id FROM user_favorites f
WHERE f.user_id = ?
  AND ((f.kind = 'customer' AND EXISTS (SELECT 1 FROM customers c WHERE c.id = f.target_id))
    OR (f.kind = 'project' AND EXISTS (SELECT 1 FROM projects p WHERE p.id = f.target_id))
    OR (f.kind = 'checklist' AND EXISTS (SELECT 1 FROM checklists cl WHERE cl.id = f.target_id)))
ORDER BY f.created_at, f.kind, f.target_id;

-- name: AddFavorite :exec
INSERT INTO user_favorites (user_id, kind, target_id) VALUES (?, ?, ?)
ON CONFLICT (user_id, kind, target_id) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM user_favorites WHERE user_id = ? AND kind = ? AND target_id = ?;

-- name: GetUserPref :one
SELECT value FROM user_prefs WHERE user_id = ? AND key = ?;

-- name: SetUserPref :exec
INSERT INTO user_prefs (user_id, key, value) VALUES (?, ?, ?)
ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value;
