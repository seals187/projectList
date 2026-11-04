-- +goose Up
-- Seitenleiste: Favoriten (Projekte) und zuletzt besuchte Seiten je Benutzer,
-- damit sie auf allen Geräten gleich sind.
CREATE TABLE user_favorites (
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    project_id INTEGER  NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, project_id)
);
CREATE INDEX idx_user_favorites_project_id ON user_favorites (project_id);

-- Ziel ist ein Kunde, ein Projekt oder eine Checkliste (kein Fremdschlüssel
-- möglich); gelöschte Ziele blendet die Oberfläche aus, ältere Einträge werden
-- beim Schreiben auf die letzten 20 je Benutzer gekürzt.
CREATE TABLE user_recent (
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT    NOT NULL CHECK (kind IN ('customer', 'project', 'checklist')),
    target_id  INTEGER NOT NULL,
    visited_at TEXT    NOT NULL, -- UTC mit Millisekunden, sortierbar
    PRIMARY KEY (user_id, kind, target_id)
);
CREATE INDEX idx_user_recent_user_visited ON user_recent (user_id, visited_at);

-- +goose Down
DROP TABLE user_recent;
DROP TABLE user_favorites;
