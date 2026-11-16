package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// archivedProject is one row of the archive (page and customer tab).
// ArchivedAt / ArchivedBy are null for projects archived before the activity
// log existed.
type archivedProject struct {
	ID             int64      `json:"id"`
	CustomerID     int64      `json:"customer_id"`
	Name           string     `json:"name"`
	CustomerName   string     `json:"customer_name"`
	ArchivedAt     *time.Time `json:"archived_at"`
	ArchivedBy     *int64     `json:"archived_by"`
	ArchivedByName string     `json:"archived_by_name"`
	Total          int64      `json:"total"`
	Done           int64      `json:"done"`
}

// GET /api/archive[?customer_id=n]: archived projects of all visible
// customers (or of one), newest archiving first.
func (s *Server) listArchive(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var customer any
	if v := r.URL.Query().Get("customer_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return errBadRequest("invalid customer_id")
		}
		if err := s.requireCustomer(r.Context(), u, id); err != nil {
			return err
		}
		customer = id
	}
	rows, err := s.q.ListArchivedProjects(r.Context(), sqlc.ListArchivedProjectsParams{CustomerID: customer, UserID: u.VisibilityFilter()})
	if err != nil {
		return err
	}
	out := make([]archivedProject, 0, len(rows))
	for _, p := range rows {
		a := archivedProject{
			ID: p.ID, CustomerID: p.CustomerID, Name: p.Name, CustomerName: p.CustomerName,
			ArchivedBy: p.ArchivedBy, ArchivedByName: p.ArchivedByName.String, Total: p.Total, Done: p.Done,
		}
		if p.ArchivedAt.Valid {
			t := p.ArchivedAt.Time
			a.ArchivedAt = &t
		}
		out = append(out, a)
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

// searchProject is a project hit of the global search.
type searchProject struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	CustomerID   int64      `json:"customer_id"`
	CustomerName string     `json:"customer_name"`
	Status       string     `json:"status"`
	ArchivedAt   *time.Time `json:"archived_at"`
}

func toSearchProjects(rows []sqlc.SearchProjectsRow) []searchProject {
	out := make([]searchProject, 0, len(rows))
	for _, p := range rows {
		sp := searchProject{ID: p.ID, Name: p.Name, CustomerID: p.CustomerID, CustomerName: p.CustomerName, Status: p.Status}
		if p.ArchivedAt.Valid {
			t := p.ArchivedAt.Time
			sp.ArchivedAt = &t
		}
		out = append(out, sp)
	}
	return out
}
