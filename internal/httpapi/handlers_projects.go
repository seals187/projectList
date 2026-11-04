package httpapi

import (
	"context"
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// projectStatuses: Phasen auf dem Board (Übersicht) plus "archived" zum Ausblenden.
var projectStatuses = []string{"planning", "active", "review", "blocked", "done", "archived"}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, id); err != nil {
		return err
	}
	ps, err := s.q.ListProjectsByCustomer(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, ps)
	return nil
}

type projectReq struct {
	Name        *string     `json:"name"`
	Description *string     `json:"description"`
	Status      *string     `json:"status"`
	DueDate     Opt[string] `json:"due_date"` // YYYY-MM-DD, null/"" = keins
	// ob das Projekt in den Gesamtfortschritt des Kunden einfliesst
	IncludeInCustomerProgress *bool `json:"include_in_customer_progress"`
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request, u authz.User) error {
	customerID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, customerID); err != nil {
		return err
	}
	var req projectReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.CreateProjectParams{CustomerID: customerID, Status: "active"}
	p.Name = v.text("name", deref(req.Name), 1, 200)
	p.Description = v.text("description", deref(req.Description), 0, 10000)
	if req.Status != nil {
		p.Status = v.oneOf("status", *req.Status, projectStatuses...)
	}
	p.DueDate = v.optDate("due_date", req.DueDate, nil)
	if err := v.err(); err != nil {
		return err
	}
	pr, err := s.q.CreateProject(r.Context(), p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, pr)
	return nil
}

// loadProject lädt ein Projekt und prüft den Zugriff.
func (s *Server) loadProject(ctx context.Context, u authz.User, id int64) (sqlc.Project, error) {
	p, err := s.q.GetProject(ctx, id)
	if err != nil {
		return p, err
	}
	return p, s.requireCustomer(ctx, u, p.CustomerID)
}

// projectChecklists lädt alle Checklisten samt Items mit genau zwei Abfragen.
// getProject liefert das Projekt mit seinen Checklisten; Fortschritt und
// Fälligkeit je Checkliste werden per SQL aggregiert (keine Items geladen).
func (s *Server) getProject(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	p, err := s.loadProject(r.Context(), u, id)
	if err != nil {
		return err
	}
	customer, err := s.q.GetCustomer(r.Context(), p.CustomerID)
	if err != nil {
		return err
	}
	checklists, err := s.q.ListChecklistsByProject(r.Context(), id)
	if err != nil {
		return err
	}
	var prog progress
	for _, c := range checklists {
		prog.Total += c.Total
		prog.Done += c.Done
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project":    p,
		"customer":   map[string]any{"id": customer.ID, "name": customer.Name},
		"checklists": checklists,
		"progress":   prog,
	})
	return nil
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	p, err := s.loadProject(r.Context(), u, id)
	if err != nil {
		return err
	}
	var req projectReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	up := sqlc.UpdateProjectParams{ID: id, Name: p.Name, Description: p.Description, Status: p.Status, DueDate: p.DueDate, IncludeInCustomerProgress: p.IncludeInCustomerProgress}
	if req.Name != nil {
		up.Name = v.text("name", *req.Name, 1, 200)
	}
	if req.Description != nil {
		up.Description = v.text("description", *req.Description, 0, 10000)
	}
	if req.Status != nil {
		up.Status = v.oneOf("status", *req.Status, projectStatuses...)
	}
	up.DueDate = v.optDate("due_date", req.DueDate, up.DueDate)
	if req.IncludeInCustomerProgress != nil {
		up.IncludeInCustomerProgress = *req.IncludeInCustomerProgress
	}
	if err := v.err(); err != nil {
		return err
	}
	p, err = s.q.UpdateProject(r.Context(), up)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, p)
	return nil
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	if err := s.q.DeleteProject(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
