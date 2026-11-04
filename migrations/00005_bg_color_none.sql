-- +goose Up
-- bg_color = 'none': row explicitly without background, even inside a
-- colored section ('' still means "same as section"). SQLite cannot alter a
-- CHECK, so both tables are rebuilt (same column order;
-- no other table references them).

CREATE TABLE checklist_items_new (
    id           INTEGER PRIMARY KEY,
    checklist_id INTEGER  NOT NULL REFERENCES checklists (id) ON DELETE CASCADE,
    title        TEXT     NOT NULL,
    description  TEXT     NOT NULL DEFAULT '',
    position     INTEGER  NOT NULL DEFAULT 0,
    assignee_id  INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    due_date     TEXT,
    done_at      DATETIME,
    done_by      INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    kind         TEXT     NOT NULL DEFAULT 'item' CHECK (kind IN ('item', 'h1', 'h2', 'h3', 'h4', 'text', 'bullet', 'numbered', 'quote', 'code', 'divider')),
    indent       INTEGER  NOT NULL DEFAULT 0 CHECK (indent BETWEEN 0 AND 4),
    lang         TEXT     NOT NULL DEFAULT '',
    note_color   TEXT     NOT NULL DEFAULT '' CHECK (note_color IN ('', 'green', 'orange', 'red', 'indigo')),
    bg_color     TEXT     NOT NULL DEFAULT '' CHECK (bg_color IN ('', 'none', 'gray', 'yellow', 'green', 'blue', 'red', 'purple')),
    status       TEXT     NOT NULL DEFAULT ''
);
INSERT INTO checklist_items_new
    (id, checklist_id, title, description, position, assignee_id, due_date, done_at, done_by,
     kind, indent, lang, note_color, bg_color, status)
SELECT id, checklist_id, title, description, position, assignee_id, due_date, done_at, done_by,
       kind, indent, lang, note_color, bg_color, status
FROM checklist_items;
DROP TABLE checklist_items;
ALTER TABLE checklist_items_new RENAME TO checklist_items;
CREATE INDEX idx_checklist_items_checklist_id ON checklist_items (checklist_id, position);
CREATE INDEX idx_checklist_items_assignee_id ON checklist_items (assignee_id);
CREATE INDEX idx_checklist_items_done_by ON checklist_items (done_by);

CREATE TABLE template_items_new (
    id          INTEGER PRIMARY KEY,
    template_id INTEGER NOT NULL REFERENCES checklist_templates (id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 0,
    kind        TEXT    NOT NULL DEFAULT 'item' CHECK (kind IN ('item', 'h1', 'h2', 'h3', 'h4', 'text', 'bullet', 'numbered', 'quote', 'code', 'divider')),
    indent      INTEGER NOT NULL DEFAULT 0 CHECK (indent BETWEEN 0 AND 4),
    lang        TEXT    NOT NULL DEFAULT '',
    note_color  TEXT    NOT NULL DEFAULT '' CHECK (note_color IN ('', 'green', 'orange', 'red', 'indigo')),
    bg_color    TEXT    NOT NULL DEFAULT '' CHECK (bg_color IN ('', 'none', 'gray', 'yellow', 'green', 'blue', 'red', 'purple'))
);
INSERT INTO template_items_new
    (id, template_id, title, description, position, kind, indent, lang, note_color, bg_color)
SELECT id, template_id, title, description, position, kind, indent, lang, note_color, bg_color
FROM template_items;
DROP TABLE template_items;
ALTER TABLE template_items_new RENAME TO template_items;
CREATE INDEX idx_template_items_template_id ON template_items (template_id, position);

-- +goose Down
-- The extended CHECK stays in place; 'none' is reset to "same as section".
UPDATE checklist_items SET bg_color = '' WHERE bg_color = 'none';
UPDATE template_items SET bg_color = '' WHERE bg_color = 'none';
