package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Template kinds: a checklist template has items, a table template a workbook (sheet_data).
const (
	KindChecklist = "checklist"
	KindTable     = "table"
)

// ErrTemplateKind: the template is of the other kind (a table template where a
// checklist is created or the other way round).
var ErrTemplateKind = errors.New("wrong template kind")

// EmptyWorkbook is the data of a new table: one empty sheet.
const EmptyWorkbook = `[{"name":"Sheet1","order":0,"status":1,"celldata":[]}]`

// SubstituteJSON replaces the variables ({{key}}) inside the strings of a JSON
// document (a workbook): the values are JSON-escaped, so the document stays valid.
func SubstituteJSON(doc string, vals map[string]string) string {
	if len(vals) == 0 || !strings.Contains(doc, "{{") {
		return doc
	}
	escaped := make(map[string]string, len(vals))
	for k, v := range vals {
		b, _ := json.Marshal(v)
		escaped[k] = string(b[1 : len(b)-1]) // without the quotes
	}
	return Substitute(doc, escaped)
}

// ApplyTableTemplate creates a new table in the project as a SNAPSHOT of the
// table template (like ApplyTemplate for checklists). If title is empty, the
// template name is used; with Values the variables in the title and in the
// cells are replaced. userID: the person creating it (nil for automations).
func ApplyTableTemplate(ctx context.Context, d *sql.DB, projectID, templateID int64, title string, userID *int64, opts ...ApplyOptions) (sqlc.ProjectTable, error) {
	var o ApplyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	var t sqlc.ProjectTable
	err := db.WithTx(ctx, d, func(q *sqlc.Queries) error {
		tmpl, err := q.GetTemplate(ctx, templateID)
		if err != nil {
			return err
		}
		if tmpl.Kind != KindTable {
			return ErrTemplateKind
		}
		if title == "" {
			title = tmpl.Name
		}
		data := tmpl.SheetData
		if data == "" {
			data = EmptyWorkbook
		}
		pos, err := q.NextTablePosition(ctx, projectID)
		if err != nil {
			return err
		}
		now := o.Now.UTC().Format(time.RFC3339)
		t, err = q.CreateTable(ctx, sqlc.CreateTableParams{
			ProjectID: projectID, Title: Substitute(title, o.Values), TemplateID: &templateID, Position: pos,
			Data: SubstituteJSON(data, o.Values), CreatedAt: now, UpdatedAt: now, UpdatedBy: userID,
		})
		if err != nil {
			return err
		}
		pid, tid := projectID, t.ID
		return q.AddTableTemplateUse(ctx, sqlc.AddTableTemplateUseParams{
			TemplateID: templateID, ProjectID: &pid, TableID: &tid, RuleID: o.RuleID, UsedAt: now,
		})
	})
	return t, err
}
