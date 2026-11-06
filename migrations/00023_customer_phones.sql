-- +goose Up
-- Phone numbers of a customer: JSON list [{"label": "Zentrale", "number": "+41 …"}].
ALTER TABLE customers ADD COLUMN phones TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE customers DROP COLUMN phones;
