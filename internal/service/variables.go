package service

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/db/sqlc"
)

// Variables in templates: "{{key}}" in the template name, item titles and
// descriptions is replaced when the template is applied. System variables come
// from the project, its customer and the date; custom variables (table
// template_variables) are asked when applying (default value prefilled) or, for
// integrations (source "easyredmine", "otobo", …), delivered with the event.
// Unknown keys stay as they are.

// SystemVariable is a built-in variable.
type SystemVariable struct {
	Name string // key in {{…}}
	Desc string // i18n key of the description in the web app
}

// SystemVariables lists the built-in variables in display order.
var SystemVariables = []SystemVariable{
	{"customer.name", "var.customerName"},
	{"customer.location", "var.customerLocation"},
	{"customer.contact", "var.customerContact"},
	{"customer.email", "var.customerEmail"},
	{"project.name", "var.projectName"},
	{"project.ticket", "var.projectTicket"},
	{"project.start", "var.projectStart"},
	{"project.due", "var.dueDate"},
	{"today", "var.today"},
	{"user.name", "var.userName"},
}

// LegacyNames: the first (German) names of the system variables → current name.
// Migration 00027 rewrote them in the templates; they still work when applying.
var LegacyNames = map[string]string{
	"kunde.name": "customer.name", "kunde.ort": "customer.location", "kunde.ansprechpartner": "customer.contact",
	"kunde.email": "customer.email", "projekt.name": "project.name", "projekt.ticket": "project.ticket",
	"projekt.start": "project.start", "stichtag": "project.due", "heute": "today", "benutzer.name": "user.name",
}

// IsSystemVariable reports whether name is built in (also an old name).
func IsSystemVariable(name string) bool {
	if _, ok := LegacyNames[name]; ok {
		return true
	}
	for _, v := range SystemVariables {
		if v.Name == name {
			return true
		}
	}
	return false
}

// VariableSources: where the value of a custom variable comes from.
var VariableSources = []string{"custom", "easyredmine", "otobo"}

var (
	varRef      = regexp.MustCompile(`\{\{\s*([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z0-9_]+)*)\s*\}\}`)
	varNameRule = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z0-9_]+)*$`)
)

// ValidVariableName: lower case, digits, "_" and dots ("server.hostname").
func ValidVariableName(name string) bool {
	return len(name) <= 60 && varNameRule.MatchString(name)
}

// VariablesIn returns the variable names used in the texts (sorted, unique).
func VariablesIn(texts ...string) []string {
	seen := map[string]bool{}
	for _, t := range texts {
		for _, m := range varRef.FindAllStringSubmatch(t, -1) {
			seen[m[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Substitute replaces the known variables in text.
func Substitute(text string, vals map[string]string) string {
	if !strings.Contains(text, "{{") {
		return text
	}
	return varRef.ReplaceAllStringFunc(text, func(m string) string {
		name := varRef.FindStringSubmatch(m)[1]
		if v, ok := vals[name]; ok {
			return v
		}
		return m
	})
}

// swissDate: "2026-11-20" → "20.11.2026" ("" stays "").
func swissDate(iso string) string {
	if len(iso) >= 10 {
		if t, err := time.Parse("2006-01-02", iso[:10]); err == nil {
			return t.Format("02.01.2006")
		}
	}
	return iso
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SystemValues returns the values of the system variables for a project
// (projectID 0: only date and user).
func SystemValues(ctx context.Context, q *sqlc.Queries, projectID int64, userName string, now time.Time) (map[string]string, error) {
	vals := map[string]string{"today": now.Format("02.01.2006"), "user.name": userName}
	defer func() {
		for old, cur := range LegacyNames {
			if v, ok := vals[cur]; ok {
				vals[old] = v
			}
		}
	}()
	if projectID == 0 {
		return vals, nil
	}
	c, err := q.VariableContext(ctx, projectID)
	if err != nil {
		return nil, err
	}
	vals["customer.name"] = c.CustomerName
	vals["customer.location"] = c.Location
	vals["customer.contact"] = c.ContactName
	vals["customer.email"] = c.ContactEmail
	vals["project.name"] = c.ProjectName
	vals["project.ticket"] = c.RedmineTicket
	vals["project.start"] = swissDate(deref(c.StartDate))
	vals["project.due"] = swissDate(deref(c.DueDate))
	return vals, nil
}

// ResolveValues: system values of the project, then the defaults of the custom
// variables, then the given values (custom and integration variables only; the
// system values cannot be overridden).
func ResolveValues(ctx context.Context, q *sqlc.Queries, projectID int64, userName string, given map[string]string, now time.Time) (map[string]string, error) {
	vals, err := SystemValues(ctx, q, projectID, userName, now)
	if err != nil {
		return nil, err
	}
	custom, err := q.ListTemplateVariables(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range custom {
		if v.DefaultValue != "" {
			vals[v.Name] = v.DefaultValue
		}
		if g, ok := given[v.Name]; ok && strings.TrimSpace(g) != "" {
			vals[v.Name] = strings.TrimSpace(g)
		}
	}
	return vals, nil
}
