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
	StartDate  *string            `json:"start_date"`
	DueDate    *string            `json:"due_date"`
	CreatedAt  string             `json:"created_at"` // YYYY-MM-DD
	InProgress bool               `json:"include_in_customer_progress"`
	TagIDs     []int64            `json:"tag_ids"`
	OwnerID    *int64             `json:"owner_id"`
	OwnerName  string             `json:"owner_name"`
	BoardPos   int64              `json:"board_position"`
	Background string             `json:"background"`
	Total      int64              `json:"total"`
	Done       int64              `json:"done"`
	Checklists []sidebarChecklist `json:"checklists"`
}

type sidebarCustomer struct {
	ID         int64            `json:"id"`
	Name       string           `json:"name"`
	Location   string           `json:"location"`
	Background string           `json:"background"` // "", "g<n>" or "img:<id>"
	Projects   []sidebarProject `json:"projects"`
}

// sidebar returns the tree customers → projects → checklists incl. progress
// plus the number of the user's own open tasks, favorites, saved settings,
// the tag catalog and the custom status names.
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
	projectTags, err := s.q.ListProjectTags(ctx, filter)
	if err != nil {
		return err
	}
	tags, err := s.q.ListTags(ctx)
	if err != nil {
		return err
	}
	statuses, err := s.statusSettings(ctx)
	if err != nil {
		return err
	}
	prefs, err := s.clientPrefs(ctx, u.ID)
	if err != nil {
		return err
	}
	boardHidden, err := s.boardHidden(ctx, u.ID)
	if err != nil {
		return err
	}
	tagsByProject := make(map[int64][]int64, len(projects))
	for _, pt := range projectTags {
		tagsByProject[pt.ProjectID] = append(tagsByProject[pt.ProjectID], pt.TagID)
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
		tagIDs := tagsByProject[p.ID]
		if tagIDs == nil {
			tagIDs = []int64{}
		}
		byCustomer[p.CustomerID] = append(byCustomer[p.CustomerID], sidebarProject{
			ID: p.ID, Name: p.Name, Status: p.Status, StartDate: p.StartDate, DueDate: p.DueDate, CreatedAt: p.CreatedAt.Format("2006-01-02"), InProgress: p.IncludeInCustomerProgress, TagIDs: tagIDs, OwnerID: p.OwnerID, OwnerName: p.OwnerName, BoardPos: p.BoardPosition, Background: p.Background, Total: p.Total, Done: p.Done, Checklists: cls,
		})
	}
	out := make([]sidebarCustomer, len(customers))
	for i, c := range customers {
		ps := byCustomer[c.ID]
		if ps == nil {
			ps = []sidebarProject{}
		}
		out[i] = sidebarCustomer{ID: c.ID, Name: c.Name, Location: c.Location, Background: c.Background, Projects: ps}
	}
	favorites, err := s.q.ListFavorites(ctx, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"customers": out, "my_tasks": myTasks, "favorites": toFavoriteDTOs(favorites),
		"tags": tags, "status_labels": statuses.Labels, "status_colors": statuses.Colors, "status_wip": statuses.WIP,
		"board_hidden": boardHidden, "prefs": prefs,
	})
	return nil
}

type overviewPerson struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type overviewProject struct {
	Team       []overviewPerson            `json:"team"`
	NextStep   string                      `json:"next_step"`  // rich-text HTML of the first open entry
	UpdatedAt  *time.Time                  `json:"updated_at"` // latest activity, null = none logged
	Checklists map[int64]overviewChecklist `json:"checklists"` // per checklist ID
}

type overviewChecklist struct {
	AssigneeID   *int64 `json:"assignee_id"`
	AssigneeName string `json:"assignee_name"`
	Overdue      int64  `json:"overdue"` // open items with a due date before today
}

// overview supplements the sidebar data for the overview: per project the team
// (assigned people), the next open step, the latest activity and per checklist
// the person responsible and overdue items. ?today=YYYY-MM-DD: the client's
// date (time zone), default: today in UTC.
func (s *Server) overview(w http.ResponseWriter, r *http.Request, u authz.User) error {
	ctx, filter := r.Context(), u.VisibilityFilter()
	today := r.URL.Query().Get("today")
	if _, err := time.Parse("2006-01-02", today); err != nil {
		today = time.Now().UTC().Format("2006-01-02")
	}
	teams, err := s.q.ListProjectTeams(ctx, filter)
	if err != nil {
		return err
	}
	next, err := s.q.ListProjectNextSteps(ctx, filter)
	if err != nil {
		return err
	}
	checklists, err := s.q.ListOverviewChecklists(ctx, sqlc.ListOverviewChecklistsParams{Today: &today, UserID: filter})
	if err != nil {
		return err
	}
	last, err := s.q.ListProjectLastActivity(ctx, filter)
	if err != nil {
		return err
	}
	out := map[int64]*overviewProject{}
	get := func(id int64) *overviewProject {
		if out[id] == nil {
			out[id] = &overviewProject{Team: []overviewPerson{}, Checklists: map[int64]overviewChecklist{}}
		}
		return out[id]
	}
	for _, n := range next {
		get(n.ProjectID).NextStep = n.NextStep
	}
	for _, t := range teams {
		p := get(t.ProjectID)
		p.Team = append(p.Team, overviewPerson{ID: t.UserID, Name: t.UserName})
	}
	for _, c := range checklists {
		get(c.ProjectID).Checklists[c.ID] = overviewChecklist{AssigneeID: c.AssigneeID, AssigneeName: c.AssigneeName, Overdue: c.Overdue}
	}
	for _, l := range last {
		at := l.CreatedAt
		get(l.ProjectID).UpdatedAt = &at
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
