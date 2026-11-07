-- +goose Up
-- Card colors in the overview (shared by all users). Values: "" (none) or a
-- color name from the web app's palette (web/src/lib/cardColors.ts).
-- A customer has one color per view (cards, list) plus a customer color that,
-- with color_everywhere = 1, wins in all views (also on its projects in the board).
ALTER TABLE customers ADD COLUMN color TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN color_everywhere BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE customers ADD COLUMN color_cards TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN color_list TEXT NOT NULL DEFAULT '';
-- color of the project card in the board
ALTER TABLE projects ADD COLUMN color TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projects DROP COLUMN color;
ALTER TABLE customers DROP COLUMN color_list;
ALTER TABLE customers DROP COLUMN color_cards;
ALTER TABLE customers DROP COLUMN color_everywhere;
ALTER TABLE customers DROP COLUMN color;
