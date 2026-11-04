-- name: ListFavorites :many
SELECT project_id FROM user_favorites WHERE user_id = ? ORDER BY created_at, project_id;

-- name: AddFavorite :exec
INSERT INTO user_favorites (user_id, project_id) VALUES (?, ?)
ON CONFLICT (user_id, project_id) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM user_favorites WHERE user_id = ? AND project_id = ?;

-- name: ListRecent :many
-- Newest first. rowid breaks ties between visits in the same millisecond:
-- TouchRecent re-inserts the row, so a later visit always has a larger rowid.
SELECT kind, target_id, visited_at FROM user_recent
WHERE user_id = ?
ORDER BY visited_at DESC, rowid DESC
LIMIT 20;

-- name: DeleteRecentEntry :exec
DELETE FROM user_recent WHERE user_id = ? AND kind = ? AND target_id = ?;

-- name: TouchRecent :exec
-- Always called after DeleteRecentEntry in the same transaction (new rowid).
INSERT INTO user_recent (user_id, kind, target_id, visited_at)
VALUES (?, ?, ?, strftime('%Y-%m-%d %H:%M:%f', 'now'));

-- name: DeleteRecentBefore :exec
-- Pruning: the handler passes the visited_at of the 20th newest entry.
DELETE FROM user_recent WHERE user_id = ? AND visited_at < ?;

-- name: ClearRecent :exec
DELETE FROM user_recent WHERE user_id = ?;
