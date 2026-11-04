-- Sichtbarkeit: user_id NULL = Admin (alle Kunden), sonst nur zugeordnete Kunden.

-- name: ListCustomers :many
SELECT * FROM customers c
WHERE (sqlc.narg(user_id) IS NULL
       OR c.id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY c.name COLLATE NOCASE, c.id;

-- name: GetCustomer :one
SELECT * FROM customers WHERE id = ?;

-- name: CreateCustomer :one
INSERT INTO customers (name, notes, location, contact_name, contact_email) VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers SET name = ?, notes = ?, location = ?, contact_name = ?, contact_email = ? WHERE id = ? RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = ?;

-- name: SearchCustomers :many
SELECT c.id, c.name FROM customers c
WHERE instr(lower(c.name), lower(sqlc.arg(needle))) > 0
  AND (sqlc.narg(user_id) IS NULL
       OR c.id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)))
ORDER BY c.name COLLATE NOCASE
LIMIT 10;

-- name: ListCustomerActivity :many
-- Customer overview: latest activity per visible customer, from project creation
-- and checked items; falls back to the customer creation time.
SELECT c.id AS customer_id,
       CAST(MAX(c.created_at,
                COALESCE((SELECT MAX(p.created_at) FROM projects p WHERE p.customer_id = c.id), c.created_at),
                COALESCE((SELECT MAX(i.done_at) FROM checklist_items i
                          JOIN checklists cl ON cl.id = i.checklist_id
                          JOIN projects p ON p.id = cl.project_id
                          WHERE p.customer_id = c.id), c.created_at)) AS TEXT) AS last_activity
FROM customers c
WHERE (sqlc.narg(user_id) IS NULL
       OR c.id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)));
