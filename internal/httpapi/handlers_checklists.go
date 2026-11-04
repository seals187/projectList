package httpapi

import (
	"context"
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/service"
)

// loadChecklist lädt eine Checkliste und prüft den Zugriff über den Kunden.
func (s *Server) loadChecklist(ctx context.Context, u authz.User, id int64) (sqlc.GetChecklistWithCustomerRow, error) {
	c, err := s.q.GetChecklistWithCustomer(ctx, id)
	if err != nil {
		return c, err
	}
	return c, s.requireCustomer(ctx, u, c.CustomerID)
}

func checklistFromRow(c sqlc.GetChecklistWithCustomerRow) sqlc.Checklist {
	return sqlc.Checklist{ID: c.ID, ProjectID: c.ProjectID, Title: c.Title, TemplateID: c.TemplateID, Position: c.Position, AssigneeID: c.AssigneeID}
}

func (s *Server) checklistDTO(ctx context.Context, c sqlc.Checklist) (checklistDTO, error) {
	rows, err := s.q.ListItemsByChecklist(ctx, c.ID)
	if err != nil {
		return checklistDTO{}, err
	}
	items := make([]itemDTO, len(rows))
	for i, row := range rows {
		items[i] = toItemDTO(row)
	}
	return toChecklistDTO(c, items), nil
}

func (s *Server) listChecklists(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	cls, err := s.q.ListChecklistsByProject(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, cls)
	return nil
}

func (s *Server) createChecklist(w http.ResponseWriter, r *http.Request, u authz.User) error {
	projectID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, projectID); err != nil {
		return err
	}
	var req struct {
		Title string `json:"title"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	title := v.text("title", req.Title, 1, 200)
	if err := v.err(); err != nil {
		return err
	}
	var cl sqlc.Checklist
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		pos, err := q.NextChecklistPosition(r.Context(), projectID)
		if err != nil {
			return err
		}
		cl, err = q.CreateChecklist(r.Context(), sqlc.CreateChecklistParams{ProjectID: projectID, Title: title, Position: pos})
		return err
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toChecklistDTO(cl, nil))
	return nil
}

func (s *Server) createChecklistFromTemplate(w http.ResponseWriter, r *http.Request, u authz.User) error {
	projectID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, projectID); err != nil {
		return err
	}
	var req struct {
		TemplateID int64  `json:"template_id"`
		Title      string `json:"title"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	title := v.text("title", req.Title, 0, 200)
	if req.TemplateID <= 0 {
		v["template_id"] = "required"
	}
	if err := v.err(); err != nil {
		return err
	}
	cl, err := service.ApplyTemplate(r.Context(), s.db, projectID, req.TemplateID, title)
	if err != nil {
		return err
	}
	dto, err := s.checklistDTO(r.Context(), cl)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, dto)
	return nil
}

func (s *Server) orderChecklists(w http.ResponseWriter, r *http.Request, u authz.User) error {
	projectID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, projectID); err != nil {
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
			n, err := q.SetChecklistPosition(r.Context(), sqlc.SetChecklistPositionParams{Position: int64(i), ID: id, ProjectID: projectID})
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

func (s *Server) getChecklist(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	c, err := s.loadChecklist(r.Context(), u, id)
	if err != nil {
		return err
	}
	dto, err := s.checklistDTO(r.Context(), checklistFromRow(c))
	if err != nil {
		return err
	}
	if c.AssigneeName != "" {
		dto.AssigneeName = &c.AssigneeName
	}
	p, err := s.q.GetProject(r.Context(), c.ProjectID)
	if err != nil {
		return err
	}
	customer, err := s.q.GetCustomer(r.Context(), c.CustomerID)
	if err != nil {
		return err
	}
	assignees, err := s.q.ListUsersWithAccessToCustomer(r.Context(), c.CustomerID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"checklist": dto,
		"project":   map[string]any{"id": p.ID, "name": p.Name, "status": p.Status},
		"customer":  map[string]any{"id": customer.ID, "name": customer.Name},
		"assignees": assignees,
	})
	return nil
}

func (s *Server) updateChecklist(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	cur, err := s.loadChecklist(r.Context(), u, id)
	if err != nil {
		return err
	}
	// Teil-Update: nur gesetzte Felder ändern; assignee_id null hebt die Zuweisung auf.
	var req struct {
		Title      Opt[string] `json:"title"`
		AssigneeID Opt[int64]  `json:"assignee_id"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	var title string
	if req.Title.Set {
		title = v.text("title", req.Title.V, 1, 200)
	}
	if req.AssigneeID.Set && !req.AssigneeID.Null {
		if err := s.validateAssignee(r.Context(), v, req.AssigneeID.V, cur.CustomerID); err != nil {
			return err
		}
	}
	if err := v.err(); err != nil {
		return err
	}
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		if req.Title.Set {
			if _, err := q.UpdateChecklistTitle(r.Context(), sqlc.UpdateChecklistTitleParams{Title: title, ID: id}); err != nil {
				return err
			}
		}
		if req.AssigneeID.Set {
			var a *int64
			if !req.AssigneeID.Null {
				a = &req.AssigneeID.V
			}
			return q.SetChecklistAssignee(r.Context(), sqlc.SetChecklistAssigneeParams{AssigneeID: a, ID: id})
		}
		return nil
	})
	if err != nil {
		return err
	}
	c, err := s.q.GetChecklistWithCustomer(r.Context(), id)
	if err != nil {
		return err
	}
	var name *string
	if c.AssigneeName != "" {
		name = &c.AssigneeName
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": c.ID, "project_id": c.ProjectID, "title": c.Title, "template_id": c.TemplateID,
		"position": c.Position, "assignee_id": c.AssigneeID, "assignee_name": name,
	})
	return nil
}

func (s *Server) deleteChecklist(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadChecklist(r.Context(), u, id); err != nil {
		return err
	}
	if err := s.q.DeleteChecklist(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) saveChecklistAsTemplate(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadChecklist(r.Context(), u, id); err != nil {
		return err
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	name := v.text("name", req.Name, 1, 200)
	desc := v.text("description", req.Description, 0, 10000)
	if err := v.err(); err != nil {
		return err
	}
	t, err := service.SaveChecklistAsTemplate(r.Context(), s.db, id, name, desc)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, t)
	return nil
}
