-- +goose Up
-- Ob ein Projekt in den Gesamtfortschritt seines Kunden einfliesst
-- (z. B. abgeschlossene oder interne Projekte ausblenden).
ALTER TABLE projects ADD COLUMN include_in_customer_progress BOOLEAN NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE projects DROP COLUMN include_in_customer_progress;
