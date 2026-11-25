package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Siblings names a list of ordered children of one parent.
type Siblings int

const (
	ChecklistsOfProject Siblings = iota // parent: project
	TablesOfProject                     // parent: project
	ItemsOfChecklist                    // parent: checklist
	ItemsOfTemplate                     // parent: template
)

// ErrInvalidOrder: an id is not a child of the parent (or the list is malformed).
var ErrInvalidOrder = errors.New("invalid order")

// Reorder sets the positions of the children of parentID to the order of ids
// (0, 1, …) in one transaction. Every id must belong to the parent, otherwise
// nothing changes and ErrInvalidOrder is returned. Children not listed keep
// their position.
func Reorder(ctx context.Context, d *sql.DB, kind Siblings, parentID int64, ids []int64) error {
	return db.WithTx(ctx, d, func(q *sqlc.Queries) error {
		for i, id := range ids {
			n, err := setPosition(ctx, q, kind, parentID, id, int64(i))
			if err != nil {
				return err
			}
			if n != 1 {
				return ErrInvalidOrder
			}
		}
		return nil
	})
}

func setPosition(ctx context.Context, q *sqlc.Queries, kind Siblings, parentID, id, pos int64) (int64, error) {
	switch kind {
	case ChecklistsOfProject:
		return q.SetChecklistPosition(ctx, sqlc.SetChecklistPositionParams{Position: pos, ID: id, ProjectID: parentID})
	case TablesOfProject:
		return q.SetTablePosition(ctx, sqlc.SetTablePositionParams{Position: pos, ID: id, ProjectID: parentID})
	case ItemsOfChecklist:
		return q.SetItemPosition(ctx, sqlc.SetItemPositionParams{Position: pos, ID: id, ChecklistID: parentID})
	case ItemsOfTemplate:
		return q.SetTemplateItemPosition(ctx, sqlc.SetTemplateItemPositionParams{Position: pos, ID: id, TemplateID: parentID})
	}
	return 0, errors.New("unknown siblings kind")
}

// InsertItem creates a checklist entry at position at (nil or beyond the end:
// appended), shifting the following entries down. p.Position is ignored.
func InsertItem(ctx context.Context, d *sql.DB, p sqlc.CreateItemParams, at *int64) (int64, error) {
	var id int64
	err := db.WithTx(ctx, d, func(q *sqlc.Queries) error {
		pos, err := insertPosition(at,
			func() (int64, error) { return q.NextItemPosition(ctx, p.ChecklistID) },
			func(pos int64) error {
				return q.ShiftItemPositions(ctx, sqlc.ShiftItemPositionsParams{ChecklistID: p.ChecklistID, Position: pos})
			})
		if err != nil {
			return err
		}
		p.Position = pos
		id, err = q.CreateItem(ctx, p)
		return err
	})
	return id, err
}

// InsertTemplateItem is InsertItem for template entries.
func InsertTemplateItem(ctx context.Context, d *sql.DB, p sqlc.CreateTemplateItemParams, at *int64) (sqlc.TemplateItem, error) {
	var it sqlc.TemplateItem
	err := db.WithTx(ctx, d, func(q *sqlc.Queries) error {
		pos, err := insertPosition(at,
			func() (int64, error) { return q.NextTemplateItemPosition(ctx, p.TemplateID) },
			func(pos int64) error {
				return q.ShiftTemplateItemPositions(ctx, sqlc.ShiftTemplateItemPositionsParams{TemplateID: p.TemplateID, Position: pos})
			})
		if err != nil {
			return err
		}
		p.Position = pos
		it, err = q.CreateTemplateItem(ctx, p)
		return err
	})
	return it, err
}

// insertPosition: the end of the list, or at (shifting the rest) if it lies inside.
func insertPosition(at *int64, next func() (int64, error), shift func(int64) error) (int64, error) {
	end, err := next()
	if err != nil {
		return 0, err
	}
	if at == nil || *at >= end {
		return end, nil
	}
	return *at, shift(*at)
}
