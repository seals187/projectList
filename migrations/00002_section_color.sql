-- +goose Up
-- Hintergrundfarbe fuer einen ganzen Abschnitt (nur bei Ueberschriften genutzt).
ALTER TABLE checklist_items ADD COLUMN bg_color TEXT NOT NULL DEFAULT ''
    CHECK (bg_color IN ('', 'gray', 'yellow', 'green', 'blue', 'red', 'purple'));
ALTER TABLE template_items ADD COLUMN bg_color TEXT NOT NULL DEFAULT ''
    CHECK (bg_color IN ('', 'gray', 'yellow', 'green', 'blue', 'red', 'purple'));

-- +goose Down
ALTER TABLE template_items DROP COLUMN bg_color;
ALTER TABLE checklist_items DROP COLUMN bg_color;
