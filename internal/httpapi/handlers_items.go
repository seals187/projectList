package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// loadItem loads an item and checks access via the customer.
func (s *Server) loadItem(ctx context.Context, u authz.User, id int64) (sqlc.GetItemWithCustomerRow, error) {
	it, err := s.q.GetItemWithCustomer(ctx, id)
	if err != nil {
		return it, err
	}
	return it, s.requireCustomer(ctx, u, it.CustomerID)
}

func (s *Server) itemResponse(w http.ResponseWriter, r *http.Request, status int, id int64) error {
	row, err := s.q.GetItemFull(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, status, toItemDTO(sqlc.ListItemsByChecklistRow(row)))
	return nil
}

// validateAssignee ensures that the assignee has access to the customer.
func (s *Server) validateAssignee(ctx context.Context, v validator, assigneeID, customerID int64) error {
	user, err := s.q.GetUserByID(ctx, assigneeID)
	if err != nil {
		v["assignee_id"] = "unknown_user"
		return nil //nolint:nilerr // reported as a validation error
	}
	ok, err := authz.CanAccessCustomer(ctx, s.q, authz.User{ID: user.ID, Role: user.Role}, customerID)
	if err != nil {
		return err
	}
	if !ok {
		v["assignee_id"] = "no_customer_access"
	}
	return nil
}

type itemReq struct {
	Kind        Opt[string] `json:"kind"`
	Title       Opt[string] `json:"title"`
	Description Opt[string] `json:"description"`
	NoteColor   Opt[string] `json:"note_color"`
	BgColor     Opt[string] `json:"bg_color"`
	Indent      Opt[int64]  `json:"indent"`
	Lang        Opt[string] `json:"lang"`
	AssigneeID  Opt[int64]  `json:"assignee_id"`
	DueDate     Opt[string] `json:"due_date"`
	Status      Opt[string] `json:"status"` // free text, entries only
	// Position: insert before the entry at this position (0-based).
	// If missing, it is appended at the end. Only on create.
	Position Opt[int64] `json:"position"`
}

// apply copies the fields that are set onto b.
func (req itemReq) apply(b *block) {
	if req.Kind.Set {
		b.Kind = req.Kind.V
	}
	if req.Title.Set {
		b.Title = req.Title.V
	}
	if req.Description.Set {
		b.Description = req.Description.V
	}
	if req.BgColor.Set {
		b.BgColor = req.BgColor.V
	}
	if req.NoteColor.Set {
		b.NoteColor = req.NoteColor.V
	}
	if req.Indent.Set {
		b.Indent = req.Indent.V
	}
	if req.Lang.Set {
		b.Lang = req.Lang.V
	}
}

func (s *Server) createItem(w http.ResponseWriter, r *http.Request, u authz.User) error {
	checklistID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	cl, err := s.loadChecklist(r.Context(), u, checklistID)
	if err != nil {
		return err
	}
	var req itemReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	b := block{Kind: kindItem}
	req.apply(&b)
	v.normalize(&b)
	p := sqlc.CreateItemParams{
		ChecklistID: checklistID, Kind: b.Kind, Title: b.Title, Description: b.Description,
		NoteColor: b.NoteColor, BgColor: b.BgColor, Indent: b.Indent, Lang: b.Lang,
	}
	if req.Status.Set && !req.Status.Null {
		p.Status = v.status(req.Status.V)
	}
	if p.Kind == kindItem {
		if req.AssigneeID.Set && !req.AssigneeID.Null {
			if err := s.validateAssignee(r.Context(), v, req.AssigneeID.V, cl.CustomerID); err != nil {
				return err
			}
			p.AssigneeID = &req.AssigneeID.V
		}
		if req.DueDate.Set && !req.DueDate.Null && req.DueDate.V != "" {
			d := v.date("due_date", req.DueDate.V)
			p.DueDate = &d
		}
	}
	if req.Position.Set && !req.Position.Null && req.Position.V < 0 {
		v["position"] = "invalid_value"
	}
	if err := v.err(); err != nil {
		return err
	}
	var id int64
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		next, err := q.NextItemPosition(r.Context(), checklistID)
		if err != nil {
			return err
		}
		p.Position = next
		if req.Position.Set && !req.Position.Null && req.Position.V < next {
			p.Position = req.Position.V
			if err := q.ShiftItemPositions(r.Context(), sqlc.ShiftItemPositionsParams{ChecklistID: checklistID, Position: p.Position}); err != nil {
				return err
			}
		}
		id, err = q.CreateItem(r.Context(), p)
		return err
	})
	if err != nil {
		return err
	}
	return s.itemResponse(w, r, http.StatusCreated, id)
}

func (s *Server) updateItem(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	it, err := s.loadItem(r.Context(), u, id)
	if err != nil {
		return err
	}
	var req itemReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	cur, err := s.q.GetItemFull(r.Context(), id)
	if err != nil {
		return err
	}
	v := validator{}
	if req.Position.Set {
		v["position"] = "invalid_value" // order only via the sort endpoint
	}
	b := block{Kind: cur.Kind, Title: cur.Title, Description: cur.Description, NoteColor: cur.NoteColor, BgColor: cur.BgColor, Indent: cur.Indent, Lang: cur.Lang}
	req.apply(&b)
	v.normalize(&b)
	p := sqlc.UpdateItemParams{
		ID: id, Kind: b.Kind, Title: b.Title, Description: b.Description, NoteColor: b.NoteColor, BgColor: b.BgColor,
		Status: cur.Status, Indent: b.Indent, Lang: b.Lang, AssigneeID: cur.AssigneeID, DueDate: cur.DueDate,
	}
	if req.Status.Set {
		p.Status = ""
		if !req.Status.Null {
			p.Status = v.status(req.Status.V)
		}
	}
	if req.AssigneeID.Set {
		if req.AssigneeID.Null {
			p.AssigneeID = nil
		} else {
			if err := s.validateAssignee(r.Context(), v, req.AssigneeID.V, it.CustomerID); err != nil {
				return err
			}
			p.AssigneeID = &req.AssigneeID.V
		}
	}
	if req.DueDate.Set {
		if req.DueDate.Null || req.DueDate.V == "" {
			p.DueDate = nil
		} else {
			d := v.date("due_date", req.DueDate.V)
			p.DueDate = &d
		}
	}
	if err := v.err(); err != nil {
		return err
	}
	if err := s.q.UpdateItem(r.Context(), p); err != nil {
		return err
	}
	return s.itemResponse(w, r, http.StatusOK, id)
}

// setItemDone sets the status idempotently (good for optimistic updates):
// {"done": true} checks off and records who/when, {"done": false} unchecks.
func (s *Server) setItemDone(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	it, err := s.loadItem(r.Context(), u, id)
	if err != nil {
		return err
	}
	if it.Kind != kindItem {
		return errBadRequest("heading_not_checkable")
	}
	var req struct {
		Done *bool `json:"done"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if req.Done == nil {
		return errBadRequest("invalid_request")
	}
	p := sqlc.SetItemDoneParams{ID: id}
	if *req.Done {
		if it.DoneAt != nil { // already done → overwrite nothing
			return s.itemResponse(w, r, http.StatusOK, id)
		}
		now := time.Now().UTC()
		p.DoneAt, p.DoneBy = &now, &u.ID
	}
	if err := s.q.SetItemDone(r.Context(), p); err != nil {
		return err
	}
	return s.itemResponse(w, r, http.StatusOK, id)
}

func (s *Server) deleteItem(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadItem(r.Context(), u, id); err != nil {
		return err
	}
	if err := s.q.DeleteItem(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) orderItems(w http.ResponseWriter, r *http.Request, u authz.User) error {
	checklistID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadChecklist(r.Context(), u, checklistID); err != nil {
		return err
	}
	var req orderReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return err
	}
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		for i, id := range req.IDs {
			n, err := q.SetItemPosition(r.Context(), sqlc.SetItemPositionParams{Position: int64(i), ID: id, ChecklistID: checklistID})
			if err != nil {
				return err
			}
			if n != 1 {
				return errBadRequest("invalid_order")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
