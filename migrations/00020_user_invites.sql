-- +goose Up
-- Settings → Users: last activity, invitations and one-time links.

-- Last authenticated request (written at most every few minutes); NULL = never.
ALTER TABLE users ADD COLUMN last_seen_at DATETIME;
-- Set while an invited user has not chosen a password yet.
ALTER TABLE users ADD COLUMN invited_at DATETIME;

-- One-time links (invitation / password reset). id = SHA-256 (hex) of the token;
-- the plaintext token only exists in the link.
CREATE TABLE user_tokens (
    id         TEXT     PRIMARY KEY,
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    TEXT     NOT NULL CHECK (purpose IN ('invite', 'reset')),
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_user_tokens_user_id ON user_tokens (user_id);

-- +goose Down
DROP TABLE user_tokens;
ALTER TABLE users DROP COLUMN invited_at;
ALTER TABLE users DROP COLUMN last_seen_at;
