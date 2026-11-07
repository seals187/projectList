-- +goose Up
-- Reporting: creation time of checklists and items (RFC 3339, UTC; "" = unknown,
-- i.e. created before this version) and scheduled report e-mails.
ALTER TABLE checklists ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
ALTER TABLE checklist_items ADD COLUMN created_at TEXT NOT NULL DEFAULT '';
-- checklists from templates: the time of the application is known
UPDATE checklists SET created_at = COALESCE(
    (SELECT MIN(u.used_at) FROM template_uses u WHERE u.checklist_id = checklists.id AND u.used_at <> ''), '');
UPDATE checklist_items SET created_at = (SELECT c.created_at FROM checklists c WHERE c.id = checklist_items.checklist_id)
WHERE checklist_id IN (SELECT id FROM checklists WHERE created_at <> '');

-- Report e-mails: sent to recipient_id (with the recipient's access) weekly or
-- monthly. filters: JSON {"period","customers","assignees","tags"} of the report.
CREATE TABLE report_schedules (
    id           INTEGER PRIMARY KEY,
    created_by   INTEGER REFERENCES users (id) ON DELETE SET NULL,
    recipient_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    freq         TEXT    NOT NULL CHECK (freq IN ('weekly', 'monthly')),
    weekday      INTEGER NOT NULL DEFAULT 1, -- weekly: 1 = Monday … 7 = Sunday
    day          INTEGER NOT NULL DEFAULT 1, -- monthly: day of the month (beyond the end: last day)
    filters      TEXT    NOT NULL DEFAULT '{}',
    lang         TEXT    NOT NULL DEFAULT 'de',
    last_sent_on TEXT    NOT NULL DEFAULT '',
    created_at   TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX idx_report_schedules_recipient ON report_schedules (recipient_id);

-- +goose Down
DROP TABLE report_schedules;
ALTER TABLE checklist_items DROP COLUMN created_at;
ALTER TABLE checklists DROP COLUMN created_at;
