-- +goose Up
-- Assignee for a whole checklist (optional).
ALTER TABLE checklists ADD COLUMN assignee_id INTEGER REFERENCES users (id) ON DELETE SET NULL;
CREATE INDEX idx_checklists_assignee_id ON checklists (assignee_id);

-- +goose Down
DROP INDEX idx_checklists_assignee_id;
ALTER TABLE checklists DROP COLUMN assignee_id;
