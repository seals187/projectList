package httpapi

import (
	"context"
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Favoriten und „Zuletzt besucht“ der Seitenleiste – pro Benutzer in der DB,
// damit sie auf allen Geräten gleich sind. Gelesen werden beide über /api/sidebar.

const maxRecent = 20

type recentEntry struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

// recentItem ist ein aufgelöster Eintrag für die Seitenleiste: Titel und
// übergeordnetes Element (Kunde eines Projekts, Projekt einer Checkliste).
type recentItem struct {
	Kind   string `json:"kind"`
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Parent string `json:"parent"`
}

// sidebarNames: Namen der für den Benutzer sichtbaren Elemente (aus den Sidebar-Daten).
type sidebarNames struct {
	customers  map[int64]string
	projects   map[int64][2]string // Name, Kundenname
	checklists map[int64][2]string // Titel, Projektname
}

// navLists lädt Favoriten (Projekt-IDs) und die zuletzt besuchten Seiten,
// aufgelöst zu Titeln. Nicht mehr vorhandene oder nicht sichtbare Ziele fallen weg.
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
			if templates == nil { // nur bei Bedarf laden
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

// PUT /api/favorites/{id}: Projekt als Favorit merken (nur mit Zugriff).
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

// DELETE /api/favorites/{id}: Favorit entfernen (eigene Liste, keine weitere Prüfung nötig).
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

// POST /api/recent {"kind": "customer|project|checklist", "id": n}: Besuch merken.
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
	if err := s.q.TouchRecent(ctx, sqlc.TouchRecentParams{UserID: u.ID, Kind: req.Kind, TargetID: req.ID}); err != nil {
		return err
	}
	// auf die neuesten maxRecent kürzen
	rows, err := s.q.ListRecent(ctx, u.ID)
	if err != nil {
		return err
	}
	if len(rows) == maxRecent {
		if err := s.q.DeleteRecentBefore(ctx, sqlc.DeleteRecentBeforeParams{UserID: u.ID, VisitedAt: rows[maxRecent-1].VisitedAt}); err != nil {
			return err
		}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DELETE /api/recent: „Zuletzt besucht“ des Benutzers leeren.
func (s *Server) clearRecent(w http.ResponseWriter, r *http.Request, u authz.User) error {
	if err := s.q.ClearRecent(r.Context(), u.ID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
