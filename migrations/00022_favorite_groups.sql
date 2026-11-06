-- +goose Up
-- Favorites in user-defined groups and order (before: a client pref "fav_layout"),
-- and templates as favorites.
CREATE TABLE favorite_groups (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT     NOT NULL,
    position   INTEGER  NOT NULL DEFAULT 0,
    collapsed  BOOLEAN  NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_favorite_groups_user ON favorite_groups (user_id);

-- The cleanup triggers of 00021 reference user_favorites: drop them before the
-- table is rebuilt and create them again afterwards.
DROP TRIGGER trg_favorites_customer_deleted;
DROP TRIGGER trg_favorites_project_deleted;
DROP TRIGGER trg_favorites_checklist_deleted;

CREATE TABLE user_favorites_new (
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT     NOT NULL CHECK (kind IN ('customer', 'project', 'checklist', 'template')),
    target_id  INTEGER  NOT NULL,
    group_id   INTEGER  REFERENCES favorite_groups (id) ON DELETE SET NULL,
    position   INTEGER  NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, kind, target_id)
);
-- favorites not mentioned in a stored layout come after the mentioned ones
INSERT INTO user_favorites_new (user_id, kind, target_id, position, created_at)
SELECT user_id, kind, target_id, 1000000, created_at FROM user_favorites;
DROP TABLE user_favorites;
ALTER TABLE user_favorites_new RENAME TO user_favorites;
CREATE INDEX idx_user_favorites_target ON user_favorites (kind, target_id);
CREATE INDEX idx_user_favorites_group ON user_favorites (group_id);

-- Take over the stored layouts: [{"t":"i","k":"kind:id"} | {"t":"f","id","name","open","items":[...]}].
-- The array index is the position (gaps do not matter, only the order).
-- (Written with temp tables and correlated subqueries: sqlc cannot parse UPDATE … FROM.)
CREATE TEMP TABLE fav_layout_nodes (user_id INTEGER, pos INTEGER, node TEXT);
INSERT INTO fav_layout_nodes (user_id, pos, node)
SELECT p.user_id, j.key, j.value
FROM user_prefs p, json_each(p.value) j
WHERE p.key = 'fav_layout' AND json_valid(p.value) AND json_type(p.value) = 'array';

INSERT INTO favorite_groups (user_id, name, position, collapsed)
SELECT user_id, substr(COALESCE(json_extract(node, '$.name'), ''), 1, 60), pos,
       CASE WHEN json_extract(node, '$.open') = 0 THEN 1 ELSE 0 END
FROM fav_layout_nodes
WHERE json_extract(node, '$.t') = 'f'
ORDER BY user_id, pos;

CREATE TEMP TABLE fav_layout_items (user_id INTEGER, kind TEXT, target_id INTEGER, group_id INTEGER, pos INTEGER);
-- favorites in folders: group + index in the folder
INSERT INTO fav_layout_items (user_id, kind, target_id, group_id, pos)
SELECT n.user_id, substr(i.value, 1, instr(i.value, ':') - 1), CAST(substr(i.value, instr(i.value, ':') + 1) AS INTEGER), g.id, i.key
FROM fav_layout_nodes n
JOIN favorite_groups g ON g.user_id = n.user_id AND g.position = n.pos
JOIN json_each(json_extract(n.node, '$.items')) i
WHERE json_extract(n.node, '$.t') = 'f' AND instr(i.value, ':') > 0;
-- favorites on the top level: index among the root nodes
INSERT INTO fav_layout_items (user_id, kind, target_id, group_id, pos)
SELECT user_id, substr(json_extract(node, '$.k'), 1, instr(json_extract(node, '$.k'), ':') - 1),
       CAST(substr(json_extract(node, '$.k'), instr(json_extract(node, '$.k'), ':') + 1) AS INTEGER), NULL, pos
FROM fav_layout_nodes
WHERE json_extract(node, '$.t') = 'i' AND instr(json_extract(node, '$.k'), ':') > 0;

UPDATE user_favorites
SET group_id = (SELECT x.group_id FROM fav_layout_items x
                WHERE x.user_id = user_favorites.user_id AND x.kind = user_favorites.kind AND x.target_id = user_favorites.target_id),
    position = (SELECT x.pos FROM fav_layout_items x
                WHERE x.user_id = user_favorites.user_id AND x.kind = user_favorites.kind AND x.target_id = user_favorites.target_id)
WHERE EXISTS (SELECT 1 FROM fav_layout_items x
              WHERE x.user_id = user_favorites.user_id AND x.kind = user_favorites.kind AND x.target_id = user_favorites.target_id);

DROP TABLE fav_layout_items;
DROP TABLE fav_layout_nodes;
DELETE FROM user_prefs WHERE key = 'fav_layout';

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
-- +goose StatementBegin
CREATE TRIGGER trg_favorites_template_deleted AFTER DELETE ON checklist_templates BEGIN
    DELETE FROM user_favorites WHERE kind = 'template' AND target_id = OLD.id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER trg_favorites_template_deleted;
DROP TRIGGER trg_favorites_checklist_deleted;
DROP TRIGGER trg_favorites_project_deleted;
DROP TRIGGER trg_favorites_customer_deleted;
CREATE TABLE user_favorites_old (
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT     NOT NULL CHECK (kind IN ('customer', 'project', 'checklist')),
    target_id  INTEGER  NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, kind, target_id)
);
INSERT INTO user_favorites_old (user_id, kind, target_id, created_at)
SELECT user_id, kind, target_id, created_at FROM user_favorites WHERE kind <> 'template';
DROP TABLE user_favorites;
ALTER TABLE user_favorites_old RENAME TO user_favorites;
CREATE INDEX idx_user_favorites_target ON user_favorites (kind, target_id);
DROP TABLE favorite_groups;
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
