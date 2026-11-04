-- +goose Up
-- Whether a project counts towards its customer's overall progress
-- (e.g. to exclude completed or internal projects).
ALTER TABLE projects ADD COLUMN include_in_customer_progress BOOLEAN NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE projects DROP COLUMN include_in_customer_progress;
