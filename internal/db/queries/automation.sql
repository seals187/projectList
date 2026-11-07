-- Automations: rules (trigger -> template) and their runs.

-- name: ListAutomationRules :many
SELECT * FROM automation_rules ORDER BY position, id;

-- name: ListEnabledRulesByTrigger :many
SELECT * FROM automation_rules WHERE enabled = 1 AND trigger_kind = ? AND source = ? ORDER BY position, id;

-- name: GetAutomationRule :one
SELECT * FROM automation_rules WHERE id = ?;

-- name: CreateAutomationRule :one
INSERT INTO automation_rules (trigger_kind, source, conditions, schedule, template_id, target_project_id, checklist_title, enabled, position)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM automation_rules))
RETURNING *;

-- name: UpdateAutomationRule :one
UPDATE automation_rules
SET trigger_kind = ?, source = ?, conditions = ?, schedule = ?, template_id = ?, target_project_id = ?, checklist_title = ?, enabled = ?
WHERE id = ?
RETURNING *;

-- name: DeleteAutomationRule :exec
DELETE FROM automation_rules WHERE id = ?;

-- name: MarkRuleRun :exec
UPDATE automation_rules SET run_count = run_count + 1, last_run_on = ? WHERE id = ?;

-- name: SetRuleLastRunOn :exec
UPDATE automation_rules SET last_run_on = ? WHERE id = ?;

-- name: RuleRanForProject :one
SELECT CAST(EXISTS (SELECT 1 FROM automation_runs WHERE rule_id = ? AND project_id = ? AND status = 'ok') AS BOOLEAN);

-- name: AddAutomationRun :exec
INSERT INTO automation_runs (rule_id, project_id, checklist_id, status, code, template_name, target, detail, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAutomationRuns :many
SELECT * FROM automation_runs ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: CustomerForExternal :one
SELECT customer_id FROM integration_customers WHERE source = ? AND external_id = ?;
