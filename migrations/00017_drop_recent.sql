-- +goose Up
-- "Recently visited" was removed from the sidebar (replaced by saved views).
DROP TABLE user_recent;

-- +goose Down
CREATE TABLE user_recent (
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT     NOT NULL CHECK (kind IN ('customer', 'project', 'checklist', 'template')),
    target_id  INTEGER  NOT NULL,
    visited_at TEXT     NOT NULL,
    PRIMARY KEY (user_id, kind, target_id)
);
CREATE INDEX idx_user_recent_user_visited ON user_recent (user_id, visited_at);
