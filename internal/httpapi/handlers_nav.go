package httpapi

import (
	"context"
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Favorites and "recently visited" of the sidebar – stored per user in the DB
// so they are the same on all devices. Both are read via /api/sidebar.

const maxRecent = 20

type recentEntry struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

// recentItem is a resolved entry for the sidebar: title and
// parent element (customer of a project, project of a checklist).
type recentItem struct {
	Kind   string `json:"kind"`
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Parent string `json:"parent"`
}

// sidebarNames: names of the elements visible to the user (from the sidebar data).
type sidebarNames struct {
	customers  map[int64]string
	projects   map[int64][2]string // name, customer name
	checklists map[int64][2]string // title, project name
}

// navLists loads favorites (project IDs) and the recently visited pages,
// resolved to titles. Targets that no longer exist or are not visible are dropped.
func (s *Server) navLists(ctx context.Context, u authz.User, n sidebarNames) ([]int64, []recentItem, error) {
	favs, err := s.q.ListFavorites(ctx, u.ID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.q.ListRecent(ctx, u.ID)
	if err != nil {
		return nil, nil, err
	}
	var templates map[int64]string
	recent := make([]recentItem, 0, len(rows))
	for _, r := range rows {
		it := recentItem{Kind: r.Kind, ID: r.TargetID}
		ok := false
		switch r.Kind {
		case "customer":
			it.Title, ok = n.customers[r.TargetID]
		case "project":
			var p [2]string
			if p, ok = n.projects[r.TargetID]; ok {
				it.Title, it.Parent = p[0], p[1]
			}
		case "checklist":
			var c [2]string
			if c, ok = n.checklists[r.TargetID]; ok {
				it.Title, it.Parent = c[0], c[1]
			}
		case "template":
			if templates == nil { // load only when needed
				ts, err := s.q.ListTemplates(ctx)
				if err != nil {
					return nil, nil, err
				}
				templates = make(map[int64]string, len(ts))
				for _, t := range ts {
					templates[t.ID] = t.Name
				}
			}
			it.Title, ok = templates[r.TargetID]
		}
		if ok {
			recent = append(recent, it)
		}
	}
	return favs, recent, nil
}

// PUT /api/favorites/{id}: mark project as favorite (only with access).
func (s *Server) addFavorite(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	if err := s.q.AddFavorite(r.Context(), sqlc.AddFavoriteParams{UserID: u.ID, ProjectID: id}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DELETE /api/favorites/{id}: remove favorite (own list, no further check needed).
func (s *Server) removeFavorite(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.q.RemoveFavorite(r.Context(), sqlc.RemoveFavoriteParams{UserID: u.ID, ProjectID: id}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// POST /api/recent {"kind": "customer|project|checklist", "id": n}: record a visit.
func (s *Server) touchRecent(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req recentEntry
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	v.oneOf("kind", req.Kind, "customer", "project", "checklist", "template")
	if req.ID <= 0 {
		v["id"] = "required"
	}
	if err := v.err(); err != nil {
		return err
	}
	ctx := r.Context()
	var err error
	switch req.Kind {
	case "customer":
		if _, err = s.q.GetCustomer(ctx, req.ID); err == nil {
			err = s.requireCustomer(ctx, u, req.ID)
		}
	case "project":
		_, err = s.loadProject(ctx, u, req.ID)
	case "checklist":
		_, err = s.loadChecklist(ctx, u, req.ID)
	case "template":
		_, err = s.q.GetTemplate(ctx, req.ID)
	}
	if err != nil {
		return err
	}
	err = db.WithTx(ctx, s.db, func(q *sqlc.Queries) error {
		// Re-insert instead of updating: the new rowid orders visits within the same millisecond.
		if err := q.DeleteRecentEntry(ctx, sqlc.DeleteRecentEntryParams{UserID: u.ID, Kind: req.Kind, TargetID: req.ID}); err != nil {
			return err
		}
		if err := q.TouchRecent(ctx, sqlc.TouchRecentParams{UserID: u.ID, Kind: req.Kind, TargetID: req.ID}); err != nil {
			return err
		}
		// trim to the newest maxRecent
		rows, err := q.ListRecent(ctx, u.ID)
		if err != nil {
			return err
		}
		if len(rows) == maxRecent {
			return q.DeleteRecentBefore(ctx, sqlc.DeleteRecentBeforeParams{UserID: u.ID, VisitedAt: rows[maxRecent-1].VisitedAt})
		}
		return nil
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DELETE /api/recent: clear the user's "recently visited" list.
func (s *Server) clearRecent(w http.ResponseWriter, r *http.Request, u authz.User) error {
	if err := s.q.ClearRecent(r.Context(), u.ID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
