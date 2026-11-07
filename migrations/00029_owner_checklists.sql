-- +goose Up
-- The person responsible for a project gets its checklists: once for the
-- existing ones without a person (from now on: CreateChecklist / updateProject).
UPDATE checklists
SET assignee_id = (SELECT p.owner_id FROM projects p WHERE p.id = checklists.project_id)
WHERE assignee_id IS NULL
  AND (SELECT p.owner_id FROM projects p WHERE p.id = checklists.project_id) IS NOT NULL;

-- +goose Down
SELECT 1;
