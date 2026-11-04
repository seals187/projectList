-- +goose Up
-- Extended project status (planning, active, review, blocked, done, archived)
-- and due date for projects.
--
-- No table rebuild: DROP TABLE projects would delete all checklists because of
-- ON DELETE CASCADE. Instead, add a new column without CHECK, copy the values,
-- drop the old column (incl. CHECK) and rename. The allowed values are
-- checked by the server (httpapi.projectStatuses).
ALTER TABLE projects ADD COLUMN status_new TEXT NOT NULL DEFAULT 'active';
UPDATE projects SET status_new = status;
ALTER TABLE projects DROP COLUMN status;
ALTER TABLE projects RENAME COLUMN status_new TO status;
ALTER TABLE projects ADD COLUMN due_date TEXT; -- YYYY-MM-DD or NULL

-- +goose Down
ALTER TABLE projects DROP COLUMN due_date;
ALTER TABLE projects ADD COLUMN status_old TEXT NOT NULL DEFAULT 'active' CHECK (status_old IN ('active', 'done', 'archived'));
UPDATE projects SET status_old = CASE status WHEN 'done' THEN 'done' WHEN 'archived' THEN 'archived' ELSE 'active' END;
ALTER TABLE projects DROP COLUMN status;
ALTER TABLE projects RENAME COLUMN status_old TO status;
