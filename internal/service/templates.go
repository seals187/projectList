// Package service enthält Geschäftslogik, die mehrere Abfragen in einer
// Transaktion bündelt.
package service

import (
	"context"
	"database/sql"

	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// ApplyTemplate erzeugt eine neue Checkliste im Projekt als SNAPSHOT des
// Templates: Einträge und Überschriften werden kopiert; spätere
// Template-Änderungen wirken sich nicht auf die Checkliste aus. template_id ist
// nur eine Herkunftsreferenz. Ist title leer, wird der Template-Name verwendet.
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

// SaveChecklistAsTemplate legt ein neues Template mit Titeln, Überschriften und
// Notizen der Checkliste an (ohne Status, Zuständige, Fälligkeit).
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
