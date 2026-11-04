-- +goose Up
-- Sidebar: favorites (projects) and recently visited pages per user,
-- so they are the same on all devices.
CREATE TABLE user_favorites (
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    project_id INTEGER  NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, project_id)
);
CREATE INDEX idx_user_favorites_project_id ON user_favorites (project_id);

-- The target is a customer, a project or a checklist (no foreign key
-- possible); the UI hides deleted targets, older entries are trimmed
-- to the last 20 per user on write.
CREATE TABLE user_recent (
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT    NOT NULL CHECK (kind IN ('customer', 'project', 'checklist')),
    target_id  INTEGER NOT NULL,
    visited_at TEXT    NOT NULL, -- UTC with milliseconds, sortable
    PRIMARY KEY (user_id, kind, target_id)
);
CREATE INDEX idx_user_recent_user_visited ON user_recent (user_id, visited_at);

-- +goose Down
DROP TABLE user_recent;
DROP TABLE user_favorites;
