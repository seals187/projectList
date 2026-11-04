-- +goose Up
-- Project page: activity log, files, watchers, Easy Redmine ticket.

-- Activity per project (who did what when). Written by the server on changes;
-- user_id NULL = user deleted.
CREATE TABLE project_activity (
    id         INTEGER PRIMARY KEY,
    project_id INTEGER  NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    kind       TEXT     NOT NULL, -- item_done, checklist_created, checklist_completed, status, …
    subject    TEXT     NOT NULL DEFAULT '', -- e.g. checklist title or file name
    detail     TEXT     NOT NULL DEFAULT '', -- e.g. item title or new status
    created_at DATETIME NOT NULL
);
CREATE INDEX idx_project_activity_project ON project_activity (project_id, created_at);

-- Items checked off before the log existed.
INSERT INTO project_activity (project_id, user_id, kind, subject, detail, created_at)
SELECT c.project_id, i.done_by, 'item_done', c.title, i.title, i.done_at
FROM checklist_items i
JOIN checklists c ON c.id = i.checklist_id
WHERE i.kind = 'item' AND i.done_at IS NOT NULL;

-- Files per project. The content lives in its own table so listing files does
-- not read it; being in the database, files are part of every backup.
CREATE TABLE project_files (
    id          INTEGER PRIMARY KEY,
    project_id  INTEGER  NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name        TEXT     NOT NULL,
    mime        TEXT     NOT NULL,
    size        INTEGER  NOT NULL,
    uploaded_by INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    created_at  DATETIME NOT NULL
);
CREATE INDEX idx_project_files_project ON project_files (project_id);

CREATE TABLE project_file_data (
    file_id INTEGER PRIMARY KEY REFERENCES project_files (id) ON DELETE CASCADE,
    data    BLOB NOT NULL
);

-- People who follow a project (shown on the project page).
CREATE TABLE project_watchers (
    project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, user_id)
);

-- Easy Redmine ticket of the project (number or full URL; '' = none).
ALTER TABLE projects ADD COLUMN redmine_ticket TEXT NOT NULL DEFAULT '';

-- Global settings set by admins (e.g. the Easy Redmine base URL).
CREATE TABLE app_settings (
    key   TEXT NOT NULL PRIMARY KEY,
    value TEXT NOT NULL
);

-- +goose Down
DROP TABLE app_settings;
ALTER TABLE projects DROP COLUMN redmine_ticket;
DROP TABLE project_watchers;
DROP TABLE project_file_data;
DROP TABLE project_files;
DROP TABLE project_activity;
