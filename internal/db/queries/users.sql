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
