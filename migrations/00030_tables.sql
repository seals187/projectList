-- +goose Up
-- Tables (spreadsheets) in projects, next to the checklists: a workbook with
-- one or more sheets, cells with formulas and formatting (Fortune-sheet in the
-- web app). data: JSON array of sheets as the web app stores them (sparse
-- "celldata", see web/src/lib/sheetData.ts); the server does not interpret it.
-- version: optimistic locking, every save increments it (a save with an older
-- version is rejected so that nobody overwrites the changes of someone else).
CREATE TABLE project_tables (
    id          INTEGER PRIMARY KEY,
    project_id  INTEGER NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    -- origin reference only; the contents are copied when the template is applied
    template_id INTEGER REFERENCES checklist_templates (id) ON DELETE SET NULL,
    position    INTEGER NOT NULL DEFAULT 0,
    data        TEXT    NOT NULL DEFAULT '[]',
    version     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL DEFAULT '',
    updated_at  TEXT    NOT NULL DEFAULT '',
    updated_by  INTEGER REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX idx_project_tables_project_id ON project_tables (project_id, position);
CREATE INDEX idx_project_tables_template_id ON project_tables (template_id);

-- Templates are checklists or tables. Table templates keep their workbook in
-- sheet_data (same format as project_tables.data) and have no template_items.
ALTER TABLE checklist_templates ADD COLUMN kind TEXT NOT NULL DEFAULT 'checklist' CHECK (kind IN ('checklist', 'table'));
ALTER TABLE checklist_templates ADD COLUMN sheet_data TEXT NOT NULL DEFAULT '';
-- an application of a table template creates a table
ALTER TABLE template_uses ADD COLUMN table_id INTEGER REFERENCES project_tables (id) ON DELETE SET NULL;

-- +goose Down
-- template_uses.table_id stays (SQLite cannot drop a column with a foreign key; it is NULL then)
DELETE FROM checklist_templates WHERE kind = 'table';
ALTER TABLE checklist_templates DROP COLUMN sheet_data;
ALTER TABLE checklist_templates DROP COLUMN kind;
DROP TABLE project_tables;
