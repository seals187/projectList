-- +goose Up
-- Custom color (#rrggbb) per project status; empty = default color.
ALTER TABLE status_labels ADD COLUMN color TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE status_labels DROP COLUMN color;
