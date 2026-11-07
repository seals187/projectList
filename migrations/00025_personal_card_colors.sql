-- +goose Up
-- The card colors per overview view are personal now (user pref "card_colors");
-- only the customer color (color, color_everywhere) stays shared.
ALTER TABLE customers DROP COLUMN color_cards;
ALTER TABLE customers DROP COLUMN color_list;
ALTER TABLE projects DROP COLUMN color;

-- +goose Down
ALTER TABLE customers ADD COLUMN color_cards TEXT NOT NULL DEFAULT '';
ALTER TABLE customers ADD COLUMN color_list TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN color TEXT NOT NULL DEFAULT '';
