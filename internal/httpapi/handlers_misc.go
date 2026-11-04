package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type sidebarChecklist struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Total   int64  `json:"total"`
	Done    int64  `json:"done"`
	NextDue string `json:"next_due"` // next due date of open entries, "" = none
}

type sidebarProject struct {
	ID         int64              `json:"id"`
	Name       string             `json:"name"`
	Status     string             `json:"status"`
	DueDate    *string            `json:"due_date"`
	InProgress bool               `json:"include_in_customer_progress"`
	Total      int64              `json:"total"`
	Done       int64              `json:"done"`
	Checklists []sidebarChecklist `json:"checklists"`
}

type sidebarCustomer struct {
	ID       int64            `json:"id"`
	Name     string           `json:"name"`
	Projects []sidebarProject `json:"projects"`
}

// sidebar returns the tree customers → projects → checklists incl. progress
// plus the number of the user's own open tasks – with four queries in total.
func (s *Server) sidebar(w http.ResponseWriter, r *http.Request, u authz.User) error {
	ctx, filter := r.Context(), u.VisibilityFilter()
	customers, err := s.q.ListCustomers(ctx, filter)
	if err != nil {
		return err
	}
	projects, err := s.q.ListProjectsWithProgress(ctx, filter)
	if err != nil {
		return err
	}
	checklists, err := s.q.ListChecklistsWithProgress(ctx, filter)
	if err != nil {
		return err
	}
	myTasks, err := s.q.CountMyOpenTasks(ctx, sqlc.CountMyOpenTasksParams{AssigneeID: &u.ID, UserID: filter})
	if err != nil {
		return err
	}

	byProject := make(map[int64][]sidebarChecklist, len(projects))
	for _, c := range checklists {
		byProject[c.ProjectID] = append(byProject[c.ProjectID], sidebarChecklist{ID: c.ID, Title: c.Title, Total: c.Total, Done: c.Done, NextDue: c.NextDue})
	}
	byCustomer := make(map[int64][]sidebarProject, len(customers))
	for _, p := range projects {
		cls := byProject[p.ID]
		if cls == nil {
			cls = []sidebarChecklist{}
		}
		byCustomer[p.CustomerID] = append(byCustomer[p.CustomerID], sidebarProject{
			ID: p.ID, Name: p.Name, Status: p.Status, DueDate: p.DueDate, InProgress: p.IncludeInCustomerProgress, Total: p.Total, Done: p.Done, Checklists: cls,
		})
	}
	out := make([]sidebarCustomer, len(customers))
	for i, c := range customers {
		ps := byCustomer[c.ID]
		if ps == nil {
			ps = []sidebarProject{}
		}
		out[i] = sidebarCustomer{ID: c.ID, Name: c.Name, Projects: ps}
	}
	names := sidebarNames{customers: map[int64]string{}, projects: map[int64][2]string{}, checklists: map[int64][2]string{}}
	for _, c := range customers {
		names.customers[c.ID] = c.Name
	}
	for _, p := range projects {
		names.projects[p.ID] = [2]string{p.Name, names.customers[p.CustomerID]}
	}
	for _, c := range checklists {
		names.checklists[c.ID] = [2]string{c.Title, names.projects[c.ProjectID][0]}
	}
	favorites, recent, err := s.navLists(ctx, u, names)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"customers": out, "my_tasks": myTasks, "favorites": favorites, "recent": recent})
	return nil
}

type overviewPerson struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type overviewProject struct {
	Team     []overviewPerson `json:"team"`
	NextStep string           `json:"next_step"` // rich-text HTML of the first open entry
}

// overview supplements the sidebar data for the overview: per project the team
// (assigned people) and the next open step.
func (s *Server) overview(w http.ResponseWriter, r *http.Request, u authz.User) error {
	ctx, filter := r.Context(), u.VisibilityFilter()
	teams, err := s.q.ListProjectTeams(ctx, filter)
	if err != nil {
		return err
	}
	next, err := s.q.ListProjectNextSteps(ctx, filter)
	if err != nil {
		return err
	}
	out := make(map[int64]*overviewProject, len(next))
	for _, n := range next {
		out[n.ProjectID] = &overviewProject{Team: []overviewPerson{}, NextStep: n.NextStep}
	}
	for _, t := range teams {
		if p := out[t.ProjectID]; p != nil {
			p.Team = append(p.Team, overviewPerson{ID: t.UserID, Name: t.UserName})
		}
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

// myTasks returns the open entries assigned to the user.
// ?done=1: the user's own done entries from the last 30 days.
func (s *Server) myTasks(w http.ResponseWriter, r *http.Request, u authz.User) error {
	if r.URL.Query().Get("done") == "1" {
		rows, err := s.q.ListMyDoneTasks(r.Context(), sqlc.ListMyDoneTasksParams{AssigneeID: &u.ID, UserID: u.VisibilityFilter()})
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, rows)
		return nil
	}
	rows, err := s.q.ListMyTasks(r.Context(), sqlc.ListMyTasksParams{AssigneeID: &u.ID, UserID: u.VisibilityFilter()})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rows)
	return nil
}

// myChecklists: checklists assigned to the logged-in user.
func (s *Server) myChecklists(w http.ResponseWriter, r *http.Request, u authz.User) error {
	rows, err := s.q.ListMyChecklists(r.Context(), sqlc.ListMyChecklistsParams{AssigneeID: &u.ID, UserID: u.VisibilityFilter()})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rows)
	return nil
}

func (s *Server) search(w http.ResponseWriter, r *http.Request, u authz.User) error {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" || utf8.RuneCountInString(q) > 100 {
		writeJSON(w, http.StatusOK, map[string]any{"customers": []any{}, "projects": []any{}, "checklists": []any{}})
		return nil
	}
	filter := u.VisibilityFilter()
	customers, err := s.q.SearchCustomers(r.Context(), sqlc.SearchCustomersParams{Needle: q, UserID: filter})
	if err != nil {
		return err
	}
	projects, err := s.q.SearchProjects(r.Context(), sqlc.SearchProjectsParams{Needle: q, UserID: filter})
	if err != nil {
		return err
	}
	checklists, err := s.q.SearchChecklists(r.Context(), sqlc.SearchChecklistsParams{Needle: q, UserID: filter})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"customers": customers, "projects": projects, "checklists": checklists})
	return nil
}

// backup creates a consistent copy via VACUUM INTO and serves it as a download.
func (s *Server) backup(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	name := fmt.Sprintf("projectlist-%s.db", time.Now().UTC().Format("20060102-150405"))
	tmp := filepath.Join(filepath.Dir(s.dbPath), ".backup-"+name)
	if err := db.VacuumInto(r.Context(), s.db, tmp); err != nil {
		return err
	}
	defer os.Remove(tmp)
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/vnd.sqlite3")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Content-Length", fmt.Sprint(st.Size()))
	w.Header().Set("Cache-Control", "no-store")
	_, err = io.Copy(w, f)
	if err != nil {
		s.log.Warn("backup-download abgebrochen", "err", err)
	}
	return nil
}
