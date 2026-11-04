-- +goose Up
-- Freitext-Status eines Checklisten-Eintrags (z. B. "erfuellt", "erreicht").
ALTER TABLE checklist_items ADD COLUMN status TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE checklist_items DROP COLUMN status;
