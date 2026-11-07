// Package automation runs the automation rules of the template library:
// a trigger (event inside ProjectList, schedule or – later – an event of an
// integration such as Easy Redmine or OTOBO) creates a checklist from a
// template in a project.
//
// Integrations plug in as a Source: Poll is called regularly and reports what
// happened in the external system as Events (Kind "external_project" or
// "external_ticket", Source = the integration's name, Fields = its attributes,
// which conditions match and which also fill the integration variables).
// External customers are mapped to ProjectList customers in the table
// integration_customers (Engine.CustomerFor).
package automation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/service"
)

// Trigger kinds (automation_rules.trigger_kind).
const (
	ProjectCreated  = "project_created"  // fields: name, customer, status
	ProjectTag      = "project_tag"      // a tag was added; fields: tag, name, customer, status
	ProjectStatus   = "project_status"   // the status changed; fields: status, name, customer
	Scheduled       = "schedule"         // calendar (rule.schedule), in rule.target_project_id
	ExternalProject = "external_project" // integration: new project in the external system
	ExternalTicket  = "external_ticket"  // integration: new ticket in the external system
)

// SourceApp: events from ProjectList itself.
const SourceApp = "app"

// Triggers lists the trigger kinds with their source ("" = app).
var Triggers = []string{ProjectCreated, ProjectTag, ProjectStatus, Scheduled, ExternalProject, ExternalTicket}

// Integrations whose events are prepared (no Source implemented yet).
var Integrations = []string{"easyredmine", "otobo"}

// IsExternal reports whether a trigger kind comes from an integration.
func IsExternal(kind string) bool { return kind == ExternalProject || kind == ExternalTicket }

// Event is something that happened and may trigger rules.
type Event struct {
	Kind      string
	Source    string            // SourceApp or the integration
	ProjectID int64             // 0 = no project (e.g. external customer not mapped)
	Fields    map[string]string // matched by the conditions; integration variable values
	Ref       string            // external reference for the log ("Easy Redmine project #4820")
}

// Condition of a rule: Field Op Value. Ops: "equals", "contains" (case-insensitive).
type Condition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

// Matches: all conditions hold (no conditions: always).
func Matches(conds []Condition, fields map[string]string) bool {
	for _, c := range conds {
		got := strings.ToLower(strings.TrimSpace(fields[c.Field]))
		want := strings.ToLower(strings.TrimSpace(c.Value))
		switch c.Op {
		case "contains":
			if !strings.Contains(got, want) {
				return false
			}
		default: // equals
			if got != want {
				return false
			}
		}
	}
	return true
}

// Schedule of a "schedule" rule. Freq: "yearly" (Month, Day), "monthly" (Day;
// beyond the end of the month: last day), "weekly" (Weekday 1 = Monday … 7 = Sunday).
type Schedule struct {
	Freq    string `json:"freq"`
	Month   int    `json:"month,omitempty"`
	Day     int    `json:"day,omitempty"`
	Weekday int    `json:"weekday,omitempty"`
}

// Valid reports whether the schedule is complete.
func (s Schedule) Valid() bool {
	switch s.Freq {
	case "yearly":
		return s.Month >= 1 && s.Month <= 12 && s.Day >= 1 && s.Day <= 31
	case "monthly":
		return s.Day >= 1 && s.Day <= 31
	case "weekly":
		return s.Weekday >= 1 && s.Weekday <= 7
	}
	return false
}

// Due reports whether the schedule runs on the given day.
func (s Schedule) Due(day time.Time) bool {
	lastDay := time.Date(day.Year(), day.Month()+1, 0, 0, 0, 0, 0, day.Location()).Day()
	dayOK := day.Day() == s.Day || (s.Day > lastDay && day.Day() == lastDay)
	switch s.Freq {
	case "yearly":
		return int(day.Month()) == s.Month && dayOK
	case "monthly":
		return dayOK
	case "weekly":
		wd := int(day.Weekday())
		if wd == 0 {
			wd = 7
		}
		return wd == s.Weekday
	}
	return false
}

// Source is an integration (Easy Redmine, OTOBO, …) that reports its events.
type Source interface {
	Name() string
	// Poll fetches what changed since the last call and calls e.Fire per event.
	Poll(ctx context.Context, e *Engine) error
}

// Engine evaluates the rules.
type Engine struct {
	DB      *sql.DB
	Q       *sqlc.Queries
	Log     *slog.Logger
	Now     func() time.Time
	Sources []Source
}

// New returns an engine without integrations.
func New(d *sql.DB, log *slog.Logger) *Engine {
	return &Engine{DB: d, Q: sqlc.New(d), Log: log, Now: time.Now}
}

// Run evaluates the schedules and polls the integrations every interval until
// ctx is done.
func (e *Engine) Run(ctx context.Context, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		e.RunSchedules(ctx)
		for _, src := range e.Sources {
			if err := src.Poll(ctx, e); err != nil {
				e.Log.Warn("integration nicht abgefragt", "source", src.Name(), "err", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// CustomerFor maps a customer of an integration to a ProjectList customer.
func (e *Engine) CustomerFor(ctx context.Context, source, externalID string) (int64, bool) {
	id, err := e.Q.CustomerForExternal(ctx, sqlc.CustomerForExternalParams{Source: source, ExternalID: externalID})
	return id, err == nil
}

// Fire runs the enabled rules matching the event. Event rules run at most once
// per project. Errors are logged (and written to the run log), not returned.
func (e *Engine) Fire(ctx context.Context, ev Event) {
	if ev.Source == "" {
		ev.Source = SourceApp
	}
	rules, err := e.Q.ListEnabledRulesByTrigger(ctx, sqlc.ListEnabledRulesByTriggerParams{TriggerKind: ev.Kind, Source: ev.Source})
	if err != nil {
		e.Log.Warn("automationen nicht geladen", "err", err)
		return
	}
	for _, r := range rules {
		var conds []Condition
		if json.Unmarshal([]byte(r.Conditions), &conds) != nil || !Matches(conds, ev.Fields) {
			continue
		}
		if ev.ProjectID == 0 {
			e.logRun(ctx, r, nil, nil, "warning", "no_project", ev.Ref)
			continue
		}
		ran, err := e.Q.RuleRanForProject(ctx, sqlc.RuleRanForProjectParams{RuleID: &r.ID, ProjectID: &ev.ProjectID})
		if err != nil || ran {
			continue
		}
		e.apply(ctx, r, ev.ProjectID, ev.Fields, ev.Ref)
	}
}

// RunSchedules runs the scheduled rules that are due today (once per day).
func (e *Engine) RunSchedules(ctx context.Context) {
	now := e.Now()
	today := now.Format("2006-01-02")
	rules, err := e.Q.ListEnabledRulesByTrigger(ctx, sqlc.ListEnabledRulesByTriggerParams{TriggerKind: Scheduled, Source: SourceApp})
	if err != nil {
		e.Log.Warn("automationen nicht geladen", "err", err)
		return
	}
	for _, r := range rules {
		var s Schedule
		if json.Unmarshal([]byte(r.Schedule), &s) != nil || !s.Due(now) || r.LastRunOn == today {
			continue
		}
		if r.TargetProjectID == nil {
			e.logRun(ctx, r, nil, nil, "warning", "no_project", "")
			if err := e.Q.SetRuleLastRunOn(ctx, sqlc.SetRuleLastRunOnParams{LastRunOn: today, ID: r.ID}); err != nil {
				e.Log.Warn("automation nicht markiert", "rule", r.ID, "err", err)
			}
			continue
		}
		e.apply(ctx, r, *r.TargetProjectID, nil, "")
	}
}

// apply creates the checklist of the rule in the project and logs the run.
func (e *Engine) apply(ctx context.Context, r sqlc.AutomationRule, projectID int64, given map[string]string, ref string) {
	now := e.Now()
	pid := projectID
	vals, err := service.ResolveValues(ctx, e.Q, projectID, "Automation", given, now)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			e.logRun(ctx, r, nil, nil, "warning", "no_project", ref)
			return
		}
		e.logRun(ctx, r, &pid, nil, "error", "failed", err.Error())
		return
	}
	ruleID := r.ID
	cl, err := service.ApplyTemplate(ctx, e.DB, projectID, r.TemplateID, r.ChecklistTitle, service.ApplyOptions{Values: vals, RuleID: &ruleID, Now: now})
	if err != nil {
		e.logRun(ctx, r, &pid, nil, "error", "failed", err.Error())
		return
	}
	if err := e.Q.MarkRuleRun(ctx, sqlc.MarkRuleRunParams{LastRunOn: now.Format("2006-01-02"), ID: r.ID}); err != nil {
		e.Log.Warn("automation nicht markiert", "rule", r.ID, "err", err)
	}
	if err := e.Q.AddProjectActivity(ctx, sqlc.AddProjectActivityParams{
		ProjectID: projectID, Kind: "checklist_created", Subject: cl.Title, Detail: "automation", CreatedAt: now.UTC(),
	}); err != nil {
		e.Log.Warn("aktivität nicht gespeichert", "project", projectID, "err", err)
	}
	clID := cl.ID
	e.logRun(ctx, r, &pid, &clID, "ok", "created", ref)
}

// logRun writes a line of the run log ("Recent runs").
func (e *Engine) logRun(ctx context.Context, r sqlc.AutomationRule, projectID, checklistID *int64, status, code, detail string) {
	tmplName, target := "", ""
	if t, err := e.Q.GetTemplate(ctx, r.TemplateID); err == nil {
		tmplName = t.Name
	}
	if projectID != nil {
		if c, err := e.Q.VariableContext(ctx, *projectID); err == nil {
			target = c.CustomerName + " › " + c.ProjectName
		}
	}
	ruleID := r.ID
	if err := e.Q.AddAutomationRun(ctx, sqlc.AddAutomationRunParams{
		RuleID: &ruleID, ProjectID: projectID, ChecklistID: checklistID, Status: status, Code: code,
		TemplateName: tmplName, Target: target, Detail: detail, CreatedAt: e.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		e.Log.Warn("automationslauf nicht gespeichert", "rule", r.ID, "err", err)
	}
	if status != "ok" {
		e.Log.Warn("automation", "rule", r.ID, "status", status, "code", code, "detail", detail)
	}
}
