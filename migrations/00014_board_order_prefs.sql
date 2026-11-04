-- +goose Up
-- Order of the projects within a board column (shared by all users).
ALTER TABLE projects ADD COLUMN board_position INTEGER NOT NULL DEFAULT 0;

-- Personal settings that follow the user across devices (key -> JSON value),
-- e.g. the customers hidden on the board.
CREATE TABLE user_prefs (
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key     TEXT    NOT NULL,
    value   TEXT    NOT NULL,
    PRIMARY KEY (user_id, key)
);

-- +goose Down
DROP TABLE user_prefs;
ALTER TABLE projects DROP COLUMN board_position;
