-- +goose Up
CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    email         TEXT     NOT NULL UNIQUE COLLATE NOCASE,
    name          TEXT     NOT NULL,
    password_hash TEXT     NOT NULL,
    role          TEXT     NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- id = SHA-256 of the session token (the plaintext token exists only in the cookie)
CREATE TABLE sessions (
    id         TEXT     PRIMARY KEY,
    user_id    INTEGER  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE customers (
    id         INTEGER PRIMARY KEY,
    name       TEXT     NOT NULL,
    notes      TEXT     NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE memberships (
    user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    customer_id INTEGER NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, customer_id)
);
CREATE INDEX idx_memberships_customer_id ON memberships (customer_id);

CREATE TABLE projects (
    id          INTEGER PRIMARY KEY,
    customer_id INTEGER  NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    name        TEXT     NOT NULL,
    description TEXT     NOT NULL DEFAULT '',
    status      TEXT     NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'done', 'archived')),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_projects_customer_id ON projects (customer_id);

CREATE TABLE checklist_templates (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE template_items (
    id          INTEGER PRIMARY KEY,
    template_id INTEGER NOT NULL REFERENCES checklist_templates (id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 0,
    kind        TEXT    NOT NULL DEFAULT 'item' CHECK (kind IN ('item', 'h1', 'h2', 'h3', 'h4', 'text', 'bullet', 'numbered', 'quote', 'code', 'divider')),
    indent      INTEGER NOT NULL DEFAULT 0 CHECK (indent BETWEEN 0 AND 4),
    lang        TEXT    NOT NULL DEFAULT '',
    note_color  TEXT    NOT NULL DEFAULT '' CHECK (note_color IN ('', 'green', 'orange', 'red', 'indigo'))
);
CREATE INDEX idx_template_items_template_id ON template_items (template_id, position);

CREATE TABLE checklists (
    id          INTEGER PRIMARY KEY,
    project_id  INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    -- origin reference only; contents are copied when the template is applied
    template_id INTEGER REFERENCES checklist_templates (id) ON DELETE SET NULL,
    position    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_checklists_project_id ON checklists (project_id, position);
CREATE INDEX idx_checklists_template_id ON checklists (template_id);

CREATE TABLE checklist_items (
    id           INTEGER PRIMARY KEY,
    checklist_id INTEGER  NOT NULL REFERENCES checklists (id) ON DELETE CASCADE,
    title        TEXT     NOT NULL,
    description  TEXT     NOT NULL DEFAULT '',
    position     INTEGER  NOT NULL DEFAULT 0,
    assignee_id  INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    due_date     TEXT,
    done_at      DATETIME,
    done_by      INTEGER  REFERENCES users (id) ON DELETE SET NULL,
    -- block type; only 'item' (checkbox) counts towards progress
    kind         TEXT     NOT NULL DEFAULT 'item' CHECK (kind IN ('item', 'h1', 'h2', 'h3', 'h4', 'text', 'bullet', 'numbered', 'quote', 'code', 'divider')),
    -- indentation (0-4) and language for code blocks
    indent       INTEGER  NOT NULL DEFAULT 0 CHECK (indent BETWEEN 0 AND 4),
    lang         TEXT     NOT NULL DEFAULT '',
    -- color marker of the note (description); title/description are sanitized HTML
    note_color   TEXT     NOT NULL DEFAULT '' CHECK (note_color IN ('', 'green', 'orange', 'red', 'indigo'))
);
CREATE INDEX idx_checklist_items_checklist_id ON checklist_items (checklist_id, position);
CREATE INDEX idx_checklist_items_assignee_id ON checklist_items (assignee_id);
CREATE INDEX idx_checklist_items_done_by ON checklist_items (done_by);

-- +goose Down
DROP TABLE checklist_items;
DROP TABLE checklists;
DROP TABLE template_items;
DROP TABLE checklist_templates;
DROP TABLE projects;
DROP TABLE memberships;
DROP TABLE customers;
DROP TABLE sessions;
DROP TABLE users;
