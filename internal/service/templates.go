// Package service contains business logic that bundles several queries into
// one transaction.
package service

import (
	"context"
	"database/sql"

	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// ApplyTemplate creates a new checklist in the project as a SNAPSHOT of the
// template: entries and headings are copied; later template changes
// do not affect the checklist. template_id is only an origin
// reference. If title is empty, the template name is used.
func ApplyTemplate(ctx context.Context, d *sql.DB, projectID, templateID int64, title string) (sqlc.Checklist, error) {
	var cl sqlc.Checklist
	err := db.WithTx(ctx, d, func(q *sqlc.Queries) error {
		tmpl, err := q.GetTemplate(ctx, templateID)
		if err != nil {
			return err
		}
		items, err := q.ListTemplateItems(ctx, templateID)
		if err != nil {
			return err
		}
		if title == "" {
			title = tmpl.Name
		}
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
				ChecklistID: cl.ID, Kind: it.Kind, Title: it.Title, Description: it.Description,
				NoteColor: it.NoteColor, BgColor: it.BgColor, Indent: it.Indent, Lang: it.Lang, Position: int64(i),
			}); err != nil {
				return err
			}
		}
		return nil
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
