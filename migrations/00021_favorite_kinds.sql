-- +goose Up
-- Favorites can be customers, projects or checklists (before: projects only).
-- No foreign key on the target (three tables): triggers remove favorites of
-- deleted targets, and the sidebar query only returns existing targets.
CREATE TABLE user_favorites_new (
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT     NOT NULL CHECK (kind IN ('customer', 'project', 'checklist')),
    target_id  INTEGER  NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, kind, target_id)
);
INSERT INTO user_favorites_new (user_id, kind, target_id, created_at)
SELECT user_id, 'project', project_id, created_at FROM user_favorites;
DROP TABLE user_favorites;
ALTER TABLE user_favorites_new RENAME TO user_favorites;
CREATE INDEX idx_user_favorites_target ON user_favorites (kind, target_id);

-- +goose StatementBegin
CREATE TRIGGER trg_favorites_customer_deleted AFTER DELETE ON customers BEGIN
    DELETE FROM user_favorites WHERE kind = 'customer' AND target_id = OLD.id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_favorites_project_deleted AFTER DELETE ON projects BEGIN
    DELETE FROM user_favorites WHERE kind = 'project' AND target_id = OLD.id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_favorites_checklist_deleted AFTER DELETE ON checklists BEGIN
    DELETE FROM user_favorites WHERE kind = 'checklist' AND target_id = OLD.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER trg_favorites_checklist_deleted;
DROP TRIGGER trg_favorites_project_deleted;
DROP TRIGGER trg_favorites_customer_deleted;
CREATE TABLE user_favorites_old (
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    project_id INTEGER  NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, project_id)
);
INSERT INTO user_favorites_old (user_id, project_id, created_at)
SELECT user_id, target_id, created_at FROM user_favorites
WHERE kind = 'project' AND target_id IN (SELECT id FROM projects);
DROP TABLE user_favorites;
ALTER TABLE user_favorites_old RENAME TO user_favorites;
CREATE INDEX idx_user_favorites_project_id ON user_favorites (project_id);
