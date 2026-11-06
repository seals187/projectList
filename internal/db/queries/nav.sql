-- name: ListFavorites :many
-- Only existing targets (the cleanup triggers also remove deleted ones).
-- Order: position, then the time added (group order: ListFavoriteGroups).
SELECT f.kind, f.target_id, f.group_id, f.position FROM user_favorites f
WHERE f.user_id = ?
  AND ((f.kind = 'customer' AND EXISTS (SELECT 1 FROM customers c WHERE c.id = f.target_id))
    OR (f.kind = 'project' AND EXISTS (SELECT 1 FROM projects p WHERE p.id = f.target_id))
    OR (f.kind = 'checklist' AND EXISTS (SELECT 1 FROM checklists cl WHERE cl.id = f.target_id))
    OR (f.kind = 'template' AND EXISTS (SELECT 1 FROM checklist_templates t WHERE t.id = f.target_id)))
ORDER BY f.position, f.created_at, f.kind, f.target_id;

-- name: AddFavorite :exec
-- New favorites go to the end of the ungrouped list.
INSERT INTO user_favorites (user_id, kind, target_id, position)
VALUES (sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(target_id),
        (SELECT COALESCE(MAX(u2.position), -1) + 1 FROM user_favorites u2 WHERE u2.user_id = sqlc.arg(user_id) AND u2.group_id IS NULL))
ON CONFLICT (user_id, kind, target_id) DO NOTHING;

-- name: IsFavorite :one
SELECT EXISTS (SELECT 1 FROM user_favorites WHERE user_id = ? AND kind = ? AND target_id = ?);

-- name: SetFavoritePlacement :exec
UPDATE user_favorites SET group_id = ?, position = ?
WHERE user_id = ? AND kind = ? AND target_id = ?;

-- name: NextUngroupedPosition :one
SELECT CAST(COALESCE(MAX(position), -1) + 1 AS INTEGER) FROM user_favorites WHERE user_id = ? AND group_id IS NULL;

-- name: UngroupFavorites :exec
-- Favorites of a group move to the ungrouped list, starting at @offset (relative order kept).
UPDATE user_favorites SET position = position + sqlc.arg(offset), group_id = NULL
WHERE user_id = sqlc.arg(user_id) AND group_id = sqlc.arg(group_id);

-- name: ListFavoriteGroups :many
SELECT id, name, position, collapsed FROM favorite_groups WHERE user_id = ? ORDER BY position, id;

-- name: GetFavoriteGroup :one
SELECT id, name, position, collapsed FROM favorite_groups WHERE id = ? AND user_id = ?;

-- name: CreateFavoriteGroup :one
INSERT INTO favorite_groups (user_id, name, position)
VALUES (sqlc.arg(user_id), sqlc.arg(name),
        (SELECT COALESCE(MAX(g2.position), -1) + 1 FROM favorite_groups g2 WHERE g2.user_id = sqlc.arg(user_id)))
RETURNING id, name, position, collapsed;

-- name: UpdateFavoriteGroup :one
UPDATE favorite_groups SET name = ?, collapsed = ? WHERE id = ? AND user_id = ?
RETURNING id, name, position, collapsed;

-- name: SetFavoriteGroupPosition :exec
UPDATE favorite_groups SET position = ? WHERE id = ? AND user_id = ?;

-- name: DeleteFavoriteGroup :exec
DELETE FROM favorite_groups WHERE id = ? AND user_id = ?;

-- name: RemoveFavorite :exec
DELETE FROM user_favorites WHERE user_id = ? AND kind = ? AND target_id = ?;

-- name: GetUserPref :one
SELECT value FROM user_prefs WHERE user_id = ? AND key = ?;

-- name: SetUserPref :exec
INSERT INTO user_prefs (user_id, key, value) VALUES (?, ?, ?)
ON CONFLICT (user_id, key) DO UPDATE SET value = excluded.value;
