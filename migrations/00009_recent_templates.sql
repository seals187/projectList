-- +goose Up
-- "Recently visited" for templates too. Rebuilding is safe here:
-- user_recent is not referenced by any other table.
CREATE TABLE user_recent_new (
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT    NOT NULL CHECK (kind IN ('customer', 'project', 'checklist', 'template')),
    target_id  INTEGER NOT NULL,
    visited_at TEXT    NOT NULL,
    PRIMARY KEY (user_id, kind, target_id)
);
INSERT INTO user_recent_new (user_id, kind, target_id, visited_at)
SELECT user_id, kind, target_id, visited_at FROM user_recent;
DROP TABLE user_recent;
ALTER TABLE user_recent_new RENAME TO user_recent;
CREATE INDEX idx_user_recent_user_visited ON user_recent (user_id, visited_at);

-- +goose Down
DELETE FROM user_recent WHERE kind = 'template';
CREATE TABLE user_recent_old (
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT    NOT NULL CHECK (kind IN ('customer', 'project', 'checklist')),
    target_id  INTEGER NOT NULL,
    visited_at TEXT    NOT NULL,
    PRIMARY KEY (user_id, kind, target_id)
);
INSERT INTO user_recent_old SELECT user_id, kind, target_id, visited_at FROM user_recent;
DROP TABLE user_recent;
ALTER TABLE user_recent_old RENAME TO user_recent;
CREATE INDEX idx_user_recent_user_visited ON user_recent (user_id, visited_at);
