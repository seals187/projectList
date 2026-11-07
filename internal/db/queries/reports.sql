-- Reporting: raw data, aggregated in Go (internal/report). Visibility: the
-- caller passes the customers the user may see (user_id NULL = all).

-- name: ReportItems :many
SELECT i.id, i.title, i.assignee_id, i.due_date, i.done_at, i.created_at,
       c.id AS checklist_id, c.title AS checklist_title, c.template_id, c.assignee_id AS checklist_assignee_id,
       c.created_at AS checklist_created_at,
       p.id AS project_id, p.customer_id
FROM checklist_items i
JOIN checklists c ON c.id = i.checklist_id
JOIN projects p ON p.id = c.project_id
WHERE i.kind = 'item' AND p.status <> 'archived'
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)));

-- name: ReportProjects :many
SELECT p.id, p.customer_id, p.name, p.status, p.due_date, p.include_in_customer_progress
FROM projects p
WHERE p.status <> 'archived'
  AND (sqlc.narg(user_id) IS NULL
       OR p.customer_id IN (SELECT m.customer_id FROM memberships m WHERE m.user_id = sqlc.narg(user_id)));

-- name: ReportProjectsDone :many
-- status changes to "done" (projects done in a period)
SELECT a.project_id, a.created_at FROM project_activity a WHERE a.kind = 'status' AND a.detail = 'done';

-- name: ReportTemplateUses :many
SELECT u.template_id, u.used_at, u.rule_id FROM template_uses u WHERE u.used_at <> '';

-- name: ReportAutomationRuns :many
SELECT r.status, r.created_at FROM automation_runs r;

-- name: ReportUsers :many
SELECT id, name FROM users ORDER BY name COLLATE NOCASE;

-- name: ListReportSchedules :many
SELECT * FROM report_schedules ORDER BY id;

-- name: GetReportSchedule :one
SELECT * FROM report_schedules WHERE id = ?;

-- name: CreateReportSchedule :one
INSERT INTO report_schedules (created_by, recipient_id, freq, weekday, day, filters, lang, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateReportSchedule :one
UPDATE report_schedules SET recipient_id = ?, freq = ?, weekday = ?, day = ?, filters = ?, lang = ? WHERE id = ? RETURNING *;

-- name: DeleteReportSchedule :exec
DELETE FROM report_schedules WHERE id = ?;

-- name: MarkReportSent :exec
UPDATE report_schedules SET last_sent_on = ? WHERE id = ?;
