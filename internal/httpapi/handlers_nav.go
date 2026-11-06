package httpapi

import (
	"context"
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Favorites of the sidebar – customers, projects or checklists, stored per user
// in the DB so they are the same on all devices. They are read via /api/sidebar
// as [{kind, id}] in the order they were added.

type favoriteDTO struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
}

func toFavoriteDTOs(rows []sqlc.ListFavoritesRow) []favoriteDTO {
	out := make([]favoriteDTO, len(rows))
	for i, r := range rows {
		out[i] = favoriteDTO{Kind: r.Kind, ID: r.TargetID}
	}
	return out
}

// favoriteTarget reads {kind}/{id} from the path and checks that the target
// exists and the user may see it (404 otherwise).
func (s *Server) favoriteTarget(ctx context.Context, r *http.Request, u authz.User, check bool) (string, int64, error) {
	kind := r.PathValue("kind")
	id, err := pathID(r, "id")
	if err != nil {
		return "", 0, err
	}
	if kind != "customer" && kind != "project" && kind != "checklist" {
		return "", 0, errNotFound()
	}
	if !check {
		return kind, id, nil
	}
	switch kind {
	case "customer":
		if _, err := s.q.GetCustomer(ctx, id); err != nil {
			return "", 0, err
		}
		err = s.requireCustomer(ctx, u, id)
	case "project":
		_, err = s.loadProject(ctx, u, id)
	case "checklist":
		_, err = s.loadChecklist(ctx, u, id)
	}
	return kind, id, err
}

// PUT /api/favorites/{kind}/{id}: mark a customer, project or checklist as
// favorite (only with access).
func (s *Server) addFavorite(w http.ResponseWriter, r *http.Request, u authz.User) error {
	kind, id, err := s.favoriteTarget(r.Context(), r, u, true)
	if err != nil {
		return err
	}
	if err := s.q.AddFavorite(r.Context(), sqlc.AddFavoriteParams{UserID: u.ID, Kind: kind, TargetID: id}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// DELETE /api/favorites/{kind}/{id}: remove favorite (own list, no further check needed).
func (s *Server) removeFavorite(w http.ResponseWriter, r *http.Request, u authz.User) error {
	kind, id, err := s.favoriteTarget(r.Context(), r, u, false)
	if err != nil {
		return err
	}
	if err := s.q.RemoveFavorite(r.Context(), sqlc.RemoveFavoriteParams{UserID: u.ID, Kind: kind, TargetID: id}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
