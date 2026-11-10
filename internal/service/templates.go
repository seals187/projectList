// Package service contains business logic that bundles several queries into
// one transaction.
package service

import (
	"context"
	"database/sql"
	"html"
	"time"

	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// ApplyTemplate creates a new checklist in the project as a SNAPSHOT of the
// template: entries and headings are copied; later template changes
// do not affect the checklist. template_id is only an origin
// reference. If title is empty, the template name is used. With Values, the
// variables ({{key}}) in title, items and descriptions are replaced. Every
// application is logged in template_uses ("used" in the library).
func ApplyTemplate(ctx context.Context, d *sql.DB, projectID, templateID int64, title string, opts ...ApplyOptions) (sqlc.Checklist, error) {
	var o ApplyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	sub := func(s string) string { return Substitute(s, o.Values) }
	// item texts are rich text (HTML), except code blocks
	escaped := make(map[string]string, len(o.Values))
	for k, v := range o.Values {
		escaped[k] = html.EscapeString(v)
	}
	subItem := func(kind, s string) string {
		if kind == "code" {
			return sub(s)
		}
		return Substitute(s, escaped)
	}
	var cl sqlc.Checklist
	err := db.WithTx(ctx, d, func(q *sqlc.Queries) error {
		tmpl, err := q.GetTemplate(ctx, templateID)
		if err != nil {
			return err
		}
		if tmpl.Kind == KindTable {
			return ErrTemplateKind
		}
		items, err := q.ListTemplateItems(ctx, templateID)
		if err != nil {
			return err
		}
		if title == "" {
			title = tmpl.Name
		}
		title = sub(title)
		pos, err := q.NextChecklistPosition(ctx, projectID)
		if err != nil {
			return err
		}
		cl, err = q.CreateChecklist(ctx, sqlc.CreateChecklistParams{
			ProjectID: projectID, Title: title, TemplateID: &templateID, Position: pos,
		})
		if err != nil {
			return err
		}
		for i, it := range items {
			if _, err := q.CreateItem(ctx, sqlc.CreateItemParams{
				ChecklistID: cl.ID, Kind: it.Kind, Title: subItem(it.Kind, it.Title), Description: subItem("", it.Description),
				NoteColor: it.NoteColor, BgColor: it.BgColor, Indent: it.Indent, Lang: it.Lang, Position: int64(i),
			}); err != nil {
				return err
			}
		}
		pid, cid := projectID, cl.ID
		return q.AddTemplateUse(ctx, sqlc.AddTemplateUseParams{
			TemplateID: templateID, ProjectID: &pid, ChecklistID: &cid, RuleID: o.RuleID, UsedAt: o.Now.UTC().Format(time.RFC3339),
		})
	})
	return cl, err
}

// SaveChecklistAsTemplate creates a new template with the titles, headings and
// notes of the checklist (without status, assignees, due dates).
func SaveChecklistAsTemplate(ctx context.Context, d *sql.DB, checklistID int64, name, description string) (sqlc.ChecklistTemplate, error) {
	var tmpl sqlc.ChecklistTemplate
	err := db.WithTx(ctx, d, func(q *sqlc.Queries) error {
		items, err := q.ListItemsByChecklist(ctx, checklistID)
		if err != nil {
			return err
		}
		tmpl, err = q.CreateTemplate(ctx, sqlc.CreateTemplateParams{Name: name, Description: description})
		if err != nil {
			return err
		}
		for i, it := range items {
			if _, err := q.CreateTemplateItem(ctx, sqlc.CreateTemplateItemParams{
				TemplateID: tmpl.ID, Kind: it.Kind, Title: it.Title, Description: it.Description,
				NoteColor: it.NoteColor, BgColor: it.BgColor, Indent: it.Indent, Lang: it.Lang, Position: int64(i),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return tmpl, err
}

// ApplyOptions of ApplyTemplate.
type ApplyOptions struct {
	Values map[string]string // variable values (see ResolveValues); nil = no replacement
	RuleID *int64            // automation rule that applied the template
	Now    time.Time         // zero = time.Now
}
