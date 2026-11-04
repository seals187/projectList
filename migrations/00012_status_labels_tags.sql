-- +goose Up
-- Custom display names for the project statuses (global, set by admins).
-- No row = default translation of the UI language.
CREATE TABLE status_labels (
    status TEXT NOT NULL PRIMARY KEY,
    label  TEXT NOT NULL
);

-- Project tags: global catalog with a free color (#rrggbb) ...
CREATE TABLE tags (
    id         INTEGER PRIMARY KEY,
    name       TEXT     NOT NULL UNIQUE COLLATE NOCASE,
    color      TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ... assigned to any number of projects.
CREATE TABLE project_tags (
    project_id INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    tag_id     INTEGER NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, tag_id)
);
CREATE INDEX idx_project_tags_tag_id ON project_tags (tag_id);

-- +goose Down
DROP TABLE project_tags;
DROP TABLE tags;
DROP TABLE status_labels;
