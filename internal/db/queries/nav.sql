-- name: ListFavorites :many
SELECT project_id FROM user_favorites WHERE user_id = ? ORDER BY created_at, project_id;

-- name: AddFavorite :exec
INSERT INTO user_favorites (user_id, project_id) VALUES (?, ?)
ON CONFLICT (user_id, project_id) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM user_favorites WHERE user_id = ? AND project_id = ?;

-- name: GetUserPref :one
SELECT value FROM user_prefs WHERE user_id = ? AND key = ?;

-- name: SetUserPref :exec
INSERT INTO user_prefs (user_id, key, value) VALUES (?, ?, ?)
ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value;
