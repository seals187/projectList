-- name: ListFavorites :many
SELECT project_id FROM user_favorites WHERE user_id = ? ORDER BY created_at, project_id;

-- name: AddFavorite :exec
INSERT INTO user_favorites (user_id, project_id) VALUES (?, ?)
ON CONFLICT (user_id, project_id) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM user_favorites WHERE user_id = ? AND project_id = ?;

-- name: ListRecent :many
SELECT kind, target_id, visited_at FROM user_recent
WHERE user_id = ?
ORDER BY visited_at DESC
LIMIT 20;

-- name: TouchRecent :exec
INSERT INTO user_recent (user_id, kind, target_id, visited_at)
VALUES (?, ?, ?, strftime('%Y-%m-%d %H:%M:%f', 'now'))
ON CONFLICT (user_id, kind, target_id) DO UPDATE SET visited_at = excluded.visited_at;

-- name: DeleteRecentBefore :exec
-- Pruning: the handler passes the visited_at of the 20th newest entry.
DELETE FROM user_recent WHERE user_id = ? AND visited_at < ?;

-- name: ClearRecent :exec
DELETE FROM user_recent WHERE user_id = ?;
