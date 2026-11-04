-- +goose Up
-- Person responsible for a project (optional). Deleting the user keeps the
-- project and just removes the assignment.
ALTER TABLE projects ADD COLUMN owner_id INTEGER REFERENCES users (id) ON DELETE SET NULL;
CREATE INDEX idx_projects_owner_id ON projects (owner_id);

-- +goose Down
DROP INDEX idx_projects_owner_id;
ALTER TABLE projects DROP COLUMN owner_id;
