package httpapi

import (
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Favorites of the sidebar – stored per user in the DB so they are the same
// on all devices. They are read via /api/sidebar.

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
