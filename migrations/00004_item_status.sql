-- +goose Up
-- Free-text status of a checklist entry (e.g. "erfuellt", "erreicht").
ALTER TABLE checklist_items ADD COLUMN status TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE checklist_items DROP COLUMN status;
