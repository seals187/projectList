-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountAdmins :one
SELECT COUNT(*) FROM users WHERE role = 'admin';

-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, role)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY name COLLATE NOCASE, id;

-- name: UpdateUser :one
UPDATE users SET email = ?, name = ?, role = ?
WHERE id = ?
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: ListAllMemberships :many
SELECT user_id, customer_id FROM memberships;

-- name: ListMembershipsForUser :many
SELECT customer_id FROM memberships WHERE user_id = ?;

-- name: DeleteMembershipsForUser :exec
DELETE FROM memberships WHERE user_id = ?;

-- name: AddMembership :exec
INSERT OR IGNORE INTO memberships (user_id, customer_id) VALUES (?, ?);

-- name: HasMembership :one
SELECT EXISTS (
    SELECT 1 FROM memberships WHERE user_id = ? AND customer_id = ?
);

-- name: ListUsersWithAccessToCustomer :many
SELECT u.id, u.name, u.email
FROM users u
WHERE u.role = 'admin'
   OR u.id IN (SELECT m.user_id FROM memberships m WHERE m.customer_id = ?)
ORDER BY u.name COLLATE NOCASE;

-- name: TouchUserSeen :exec
UPDATE users SET last_seen_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND (last_seen_at IS NULL OR last_seen_at < sqlc.arg(before));

-- name: SetUserInvited :exec
UPDATE users SET invited_at = ? WHERE id = ?;

-- name: CreateUserToken :exec
INSERT INTO user_tokens (id, user_id, purpose, expires_at) VALUES (?, ?, ?, ?);

-- name: DeleteUserTokens :exec
DELETE FROM user_tokens WHERE user_id = ?;

-- name: GetUserToken :one
SELECT t.id, t.user_id, t.purpose, t.expires_at, u.name, u.email
FROM user_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.id = ? AND t.expires_at > ?;

-- name: DeleteExpiredUserTokens :execrows
DELETE FROM user_tokens WHERE expires_at <= ?;
