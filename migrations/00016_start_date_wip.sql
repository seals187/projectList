-- +goose Up
-- Start of a project (timeline view); NULL = creation date is used.
ALTER TABLE projects ADD COLUMN start_date TEXT; -- YYYY-MM-DD or NULL

-- WIP limit per status column on the board (0 = no limit).
ALTER TABLE status_labels ADD COLUMN wip_limit INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE status_labels DROP COLUMN wip_limit;
ALTER TABLE projects DROP COLUMN start_date;
