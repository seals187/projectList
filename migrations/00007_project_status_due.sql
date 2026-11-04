-- +goose Up
-- Projektstatus erweitert (Planung, Aktiv, Review, Blockiert, Fertig, Archiviert)
-- und Fälligkeitsdatum für Projekte.
--
-- Kein Neuaufbau der Tabelle: DROP TABLE projects würde wegen ON DELETE CASCADE
-- alle Checklisten löschen. Stattdessen neue Spalte ohne CHECK anlegen, Werte
-- kopieren, alte Spalte (samt CHECK) entfernen und umbenennen. Die erlaubten
-- Werte prüft der Server (httpapi.projectStatuses).
ALTER TABLE projects ADD COLUMN status_new TEXT NOT NULL DEFAULT 'active';
UPDATE projects SET status_new = status;
ALTER TABLE projects DROP COLUMN status;
ALTER TABLE projects RENAME COLUMN status_new TO status;
ALTER TABLE projects ADD COLUMN due_date TEXT; -- YYYY-MM-DD oder NULL

-- +goose Down
ALTER TABLE projects DROP COLUMN due_date;
ALTER TABLE projects ADD COLUMN status_old TEXT NOT NULL DEFAULT 'active' CHECK (status_old IN ('active', 'done', 'archived'));
UPDATE projects SET status_old = CASE status WHEN 'done' THEN 'done' WHEN 'archived' THEN 'archived' ELSE 'active' END;
ALTER TABLE projects DROP COLUMN status;
ALTER TABLE projects RENAME COLUMN status_old TO status;
