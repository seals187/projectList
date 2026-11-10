package httpapi

import (
	"context"
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/automation"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// projectStatuses: phases on the board (overview) plus "archived" for hiding.
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
	StartDate   Opt[string] `json:"start_date"` // YYYY-MM-DD, null/"" = none (timeline: creation date)
	DueDate     Opt[string] `json:"due_date"`   // YYYY-MM-DD, null/"" = none
	// whether the project counts towards the customer's overall progress
	IncludeInCustomerProgress *bool `json:"include_in_customer_progress"`
	// person responsible for the project, null = nobody
	OwnerID Opt[int64] `json:"owner_id"`
	// Easy Redmine ticket (number or full URL), "" = none
	RedmineTicket *string `json:"redmine_ticket"`
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
	p := sqlc.CreateProjectParams{CustomerID: customerID, Status: "planning"} // new projects always start in planning
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
	s.logActivity(r.Context(), pr.ID, u.ID, actProjectCreated, "", "")
	s.fire(r.Context(), automation.ProjectCreated, pr, nil)
	writeJSON(w, http.StatusCreated, pr)
	return nil
}

// loadProject loads a project and checks access.
func (s *Server) loadProject(ctx context.Context, u authz.User, id int64) (sqlc.Project, error) {
	p, err := s.q.GetProject(ctx, id)
	if err != nil {
		return p, err
	}
	return p, s.requireCustomer(ctx, u, p.CustomerID)
}

// projectChecklists loads all checklists incl. items with exactly two queries.
// getProject returns the project with its checklists and tables; progress and
// due date per checklist are aggregated in SQL (no items loaded).
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
	tables, err := s.q.ListTablesByProject(r.Context(), id)
	if err != nil {
		return err
	}
	tagIDs, err := s.q.ListTagIDsByProject(r.Context(), id)
	if err != nil {
		return err
	}
	// people who can be made responsible (access to the customer)
	assignees, err := s.q.ListUsersWithAccessToCustomer(r.Context(), p.CustomerID)
	if err != nil {
		return err
	}
	watchers, err := s.q.ListProjectWatchers(r.Context(), id)
	if err != nil {
		return err
	}
	files, err := s.q.CountProjectFiles(r.Context(), id)
	if err != nil {
		return err
	}
	redmine, err := s.redmineURL(r.Context())
	if err != nil {
		return err
	}
	var prog progress
	for _, c := range checklists {
		prog.Total += c.Total
		prog.Done += c.Done
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project":     p,
		"customer":    map[string]any{"id": customer.ID, "name": customer.Name},
		"checklists":  checklists,
		"tables":      tables,
		"progress":    prog,
		"tag_ids":     tagIDs,
		"assignees":   assignees,
		"watchers":    nonNil(watchers),
		"files_count": files,
		"redmine_url": redmine,
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
	up := sqlc.UpdateProjectParams{ID: id, Name: p.Name, Description: p.Description, Status: p.Status, StartDate: p.StartDate, DueDate: p.DueDate, IncludeInCustomerProgress: p.IncludeInCustomerProgress, OwnerID: p.OwnerID, RedmineTicket: p.RedmineTicket}
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
	up.StartDate = v.optDate("start_date", req.StartDate, up.StartDate)
	if up.StartDate != nil && up.DueDate != nil && *up.StartDate > *up.DueDate && v["start_date"] == "" && v["due_date"] == "" {
		v["start_date"] = "start_after_due"
	}
	if req.IncludeInCustomerProgress != nil {
		up.IncludeInCustomerProgress = *req.IncludeInCustomerProgress
	}
	if req.OwnerID.Set {
		up.OwnerID = nil
		if !req.OwnerID.Null {
			owner := req.OwnerID.V
			ov := validator{}
			if err := s.validateAssignee(r.Context(), ov, owner, p.CustomerID); err != nil {
				return err
			}
			if msg, bad := ov["assignee_id"]; bad {
				v["owner_id"] = msg
			}
			up.OwnerID = &owner
		}
	}
	if req.RedmineTicket != nil {
		up.RedmineTicket = v.text("redmine_ticket", *req.RedmineTicket, 0, 300)
	}
	if err := v.err(); err != nil {
		return err
	}
	prev := p
	p, err = s.q.UpdateProject(r.Context(), up)
	if err != nil {
		return err
	}
	if !samePtr(p.OwnerID, prev.OwnerID) {
		// the new person responsible takes over the checklists without a person and those of the previous one
		if err := s.q.AssignChecklistsToOwner(r.Context(), sqlc.AssignChecklistsToOwnerParams{NewOwner: p.OwnerID, ProjectID: p.ID, OldOwner: prev.OwnerID}); err != nil {
			return err
		}
	}
	s.logProjectChanges(r.Context(), u, prev, p)
	if p.Status != prev.Status {
		s.fire(r.Context(), automation.ProjectStatus, p, nil)
	}
	writeJSON(w, http.StatusOK, p)
	return nil
}

// logProjectChanges writes the changed properties into the activity log.
func (s *Server) logProjectChanges(ctx context.Context, u authz.User, prev, p sqlc.Project) {
	if p.Name != prev.Name {
		s.logActivity(ctx, p.ID, u.ID, actRenamed, prev.Name, p.Name)
	}
	if p.Status != prev.Status {
		s.logActivity(ctx, p.ID, u.ID, actStatus, prev.Status, p.Status)
	}
	if dateOrEmpty(p.StartDate) != dateOrEmpty(prev.StartDate) || dateOrEmpty(p.DueDate) != dateOrEmpty(prev.DueDate) {
		s.logActivity(ctx, p.ID, u.ID, actDates, dateOrEmpty(p.StartDate), dateOrEmpty(p.DueDate))
	}
	if !samePtr(p.OwnerID, prev.OwnerID) {
		name := ""
		if p.OwnerID != nil {
			if o, err := s.q.GetUserByID(ctx, *p.OwnerID); err == nil {
				name = o.Name
			}
		}
		s.logActivity(ctx, p.ID, u.ID, actOwner, "", name)
	}
	if p.RedmineTicket != prev.RedmineTicket {
		s.logActivity(ctx, p.ID, u.ID, actRedmine, "", p.RedmineTicket)
	}
}

func samePtr[T comparable](a, b *T) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
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

// GET /api/customers/{id}/assignees: users who can be made responsible for
// a project or assigned to checklists of this customer.
func (s *Server) customerAssignees(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, id); err != nil {
		return err
	}
	rows, err := s.q.ListUsersWithAccessToCustomer(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rows)
	return nil
}
