-- +goose Up
-- Customer overview: location and contact person (name, email).
ALTER TABLE customers ADD COLUMN location TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN contact_name TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN contact_email TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE customers DROP COLUMN contact_email;
ALTER TABLE customers DROP COLUMN contact_name;
ALTER TABLE customers DROP COLUMN location;
