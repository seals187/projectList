-- +goose Up
-- Template library: categories, usage log, custom variables and automations.

-- Categories of templates (admins manage them). color: a card color name
-- (web/src/lib/cardColors.ts), "" = gray.
CREATE TABLE template_categories (
    id       INTEGER PRIMARY KEY,
    name     TEXT    NOT NULL,
    color    TEXT    NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0
);
ALTER TABLE checklist_templates ADD COLUMN category_id INTEGER REFERENCES template_categories (id) ON DELETE SET NULL;

-- Automation rules: trigger (+ conditions) → create a checklist from a template.
-- trigger_kind: project_created | project_tag | project_status | schedule |
--          external_project | external_ticket (integrations, see internal/automation)
-- source:  "app" for triggers inside ProjectList, otherwise the integration
-- conditions: JSON [{"field","op","value"}]; schedule: JSON {"freq","month","day","weekday"}
-- target_project_id: project of scheduled rules (event rules use the project of the event)
CREATE TABLE automation_rules (
    id                INTEGER PRIMARY KEY,
    trigger_kind      TEXT    NOT NULL,
    source            TEXT    NOT NULL DEFAULT 'app',
    conditions        TEXT    NOT NULL DEFAULT '[]',
    schedule          TEXT    NOT NULL DEFAULT '{}',
    template_id       INTEGER NOT NULL REFERENCES checklist_templates (id) ON DELETE CASCADE,
    target_project_id INTEGER REFERENCES projects (id) ON DELETE SET NULL,
    checklist_title   TEXT    NOT NULL DEFAULT '',
    enabled           BOOLEAN NOT NULL DEFAULT 1,
    run_count         INTEGER NOT NULL DEFAULT 0,
    last_run_on       TEXT    NOT NULL DEFAULT '',
    position          INTEGER NOT NULL DEFAULT 0
);

-- Every application of a template (manually or by an automation): "used" and
-- "last used" in the library. used_at: RFC 3339 (UTC), "" = unknown (backfill).
CREATE TABLE template_uses (
    id           INTEGER PRIMARY KEY,
    template_id  INTEGER NOT NULL REFERENCES checklist_templates (id) ON DELETE CASCADE,
    project_id   INTEGER REFERENCES projects (id) ON DELETE SET NULL,
    checklist_id INTEGER REFERENCES checklists (id) ON DELETE SET NULL,
    rule_id      INTEGER REFERENCES automation_rules (id) ON DELETE SET NULL,
    used_at      TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX idx_template_uses_template_id ON template_uses (template_id, used_at);
INSERT INTO template_uses (template_id, project_id, checklist_id, used_at)
SELECT template_id, project_id, id, '' FROM checklists WHERE template_id IS NOT NULL;

-- Variables in templates ({{key}}): system variables are built in (Go code);
-- this table holds the custom ones. source: "custom" (value asked when applying,
-- default_value prefilled) or an integration ("easyredmine", "otobo", …) that
-- delivers the value with its events.
CREATE TABLE template_variables (
    id            INTEGER PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE, -- the key in {{…}}, e.g. "server.hostname"
    description   TEXT NOT NULL DEFAULT '',
    default_value TEXT NOT NULL DEFAULT '',
    source        TEXT NOT NULL DEFAULT 'custom'
);

-- Log of automation runs ("Recent runs"). status: ok | warning | error;
-- code: created | no_project | failed; detail: e.g. the external reference.
CREATE TABLE automation_runs (
    id            INTEGER PRIMARY KEY,
    rule_id       INTEGER REFERENCES automation_rules (id) ON DELETE SET NULL,
    project_id    INTEGER REFERENCES projects (id) ON DELETE SET NULL,
    checklist_id  INTEGER REFERENCES checklists (id) ON DELETE SET NULL,
    status        TEXT    NOT NULL,
    code          TEXT    NOT NULL,
    template_name TEXT    NOT NULL DEFAULT '',
    target        TEXT    NOT NULL DEFAULT '',
    detail        TEXT    NOT NULL DEFAULT '',
    created_at    TEXT    NOT NULL
);
CREATE INDEX idx_automation_runs_rule ON automation_runs (rule_id, project_id);

-- Integrations (prepared): external customer IDs → ProjectList customers.
CREATE TABLE integration_customers (
    source      TEXT    NOT NULL,
    external_id TEXT    NOT NULL,
    customer_id INTEGER NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    PRIMARY KEY (source, external_id)
);

-- +goose Down
DROP TABLE integration_customers;
DROP TABLE automation_runs;
DROP TABLE template_uses;
DROP TABLE automation_rules;
DROP TABLE template_variables;
ALTER TABLE checklist_templates DROP COLUMN category_id;
DROP TABLE template_categories;
