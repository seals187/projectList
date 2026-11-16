package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"

	"github.com/seals187/projectList/internal/db/sqlc"
)

// Config: global reporting settings (Settings → Reporting, admins). Stored as
// JSON in app_settings under ConfigKey; missing or invalid values fall back to
// the defaults, which reproduce the behavior before the settings existed. The
// reporting page, the print view and the report e-mails all use it (Build).
type Config struct {
	Views         []string `json:"views"`          // enabled views, in display order
	DefaultView   string   `json:"default_view"`   // view the reporting page opens with
	Periods       []int    `json:"periods"`        // selectable periods (days)
	DefaultPeriod int      `json:"default_period"` // period without a choice
	// Progress of customers and projects: "checklists" (completed / all
	// checklists) or "items" (checked / all items).
	ProgressBasis string `json:"progress_basis"`
	// Due date and "overdue" of a checklist: "checklist" = latest due date of its
	// items, overdue when that day has passed and the checklist is not complete;
	// "item" = earliest due date of an open item, overdue as soon as one open item is.
	DueBasis    string `json:"due_basis"`
	DueSoonDays int    `json:"due_soon_days"` // window "due soon" (dashboard, customer report)
	// Checklists of projects with the status "done" count (open, overdue,
	// workload, customers).
	IncludeDone bool `json:"include_done"`
	// Projects excluded from the customer progress are left out of the
	// customer progress in the reports, too.
	RespectCustomerProgress bool `json:"respect_customer_progress"`
	// Workload: a person counts as overloaded from this many open checklists (0 = off).
	OverloadThreshold int `json:"overload_threshold"`
	// Key figures on the dashboard (and in the e-mail), in display order.
	KPIs []string `json:"kpis"`
}

// ConfigKey: app_settings key of the reporting settings.
const ConfigKey = "reporting"

var (
	AllViews   = []string{"dashboard", "customers", "workload", "templates"}
	AllPeriods = []int{7, 14, 30, 60, 90, 180, 365}
	AllKPIs    = []string{"completed", "open", "overdue", "projects_done"}
)

// DefaultConfig: the behavior without settings.
func DefaultConfig() Config {
	return Config{
		Views:                   slices.Clone(AllViews),
		DefaultView:             "dashboard",
		Periods:                 []int{7, 30, 90, 365},
		DefaultPeriod:           30,
		ProgressBasis:           "checklists",
		DueBasis:                "checklist",
		DueSoonDays:             14,
		IncludeDone:             true,
		RespectCustomerProgress: true,
		OverloadThreshold:       0,
		KPIs:                    slices.Clone(AllKPIs),
	}
}

// Validate returns the invalid fields (field → code), empty when valid.
func (c Config) Validate() map[string]string {
	v := map[string]string{}
	subset := func(field string, xs []string, all []string, min int) {
		seen := map[string]bool{}
		for _, x := range xs {
			if !slices.Contains(all, x) || seen[x] {
				v[field] = "invalid_value"
				return
			}
			seen[x] = true
		}
		if len(xs) < min {
			v[field] = "required"
		}
	}
	subset("views", c.Views, AllViews, 1)
	subset("kpis", c.KPIs, AllKPIs, 0)
	if !slices.Contains(c.Views, c.DefaultView) {
		v["default_view"] = "invalid_value"
	}
	seen := map[int]bool{}
	for _, p := range c.Periods {
		if !slices.Contains(AllPeriods, p) || seen[p] {
			v["periods"] = "invalid_value"
		}
		seen[p] = true
	}
	if len(c.Periods) == 0 {
		v["periods"] = "required"
	}
	if !slices.Contains(c.Periods, c.DefaultPeriod) {
		v["default_period"] = "invalid_value"
	}
	if c.ProgressBasis != "checklists" && c.ProgressBasis != "items" {
		v["progress_basis"] = "invalid_value"
	}
	if c.DueBasis != "checklist" && c.DueBasis != "item" {
		v["due_basis"] = "invalid_value"
	}
	if c.DueSoonDays < 1 || c.DueSoonDays > 90 {
		v["due_soon_days"] = "invalid_value"
	}
	if c.OverloadThreshold < 0 || c.OverloadThreshold > 999 {
		v["overload_threshold"] = "invalid_value"
	}
	return v
}

// normalized: periods in ascending order, views and KPIs as given.
func (c Config) normalized() Config {
	c.Periods = slices.Clone(c.Periods)
	slices.Sort(c.Periods)
	return c
}

// LoadConfig reads the settings; defaults when none are stored or they are invalid.
func LoadConfig(ctx context.Context, q *sqlc.Queries) (Config, error) {
	raw, err := q.GetAppSetting(ctx, ConfigKey)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && raw == "") {
		return DefaultConfig(), nil
	}
	if err != nil {
		return DefaultConfig(), err
	}
	c := DefaultConfig() // fields missing in older JSON keep their default
	if json.Unmarshal([]byte(raw), &c) != nil || len(c.Validate()) > 0 {
		return DefaultConfig(), nil
	}
	return c.normalized(), nil
}

// SaveConfig stores valid settings (the caller validates).
func SaveConfig(ctx context.Context, q *sqlc.Queries, c Config) error {
	b, err := json.Marshal(c.normalized())
	if err != nil {
		return err
	}
	return q.SetAppSetting(ctx, sqlc.SetAppSettingParams{Key: ConfigKey, Value: string(b)})
}

// period: the requested period if selectable, otherwise the default period.
func (c Config) period(days int) int {
	if slices.Contains(c.Periods, days) {
		return days
	}
	return c.DefaultPeriod
}

// HasView: whether a view is enabled.
func (c Config) HasView(v string) bool { return slices.Contains(c.Views, v) }
