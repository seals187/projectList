package httpapi

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Favorites of the sidebar – customers, projects, checklists or templates,
// stored per user in the DB (same on all devices), optionally in user-defined
// groups. Read via /api/sidebar: favorites [{kind, id, group_id, position}] and
// favorite_groups [{id, name, position, collapsed}]. Display order: favorites
// without group first, then the groups; inside each by position.

type favoriteDTO struct {
	Kind     string `json:"kind"`
	ID       int64  `json:"id"`
	GroupID  *int64 `json:"group_id"`
	Position int64  `json:"position"`
}

type favoriteGroupDTO struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Position  int64  `json:"position"`
	Collapsed bool   `json:"collapsed"`
}

func toFavoriteDTOs(rows []sqlc.ListFavoritesRow) []favoriteDTO {
	out := make([]favoriteDTO, len(rows))
	for i, r := range rows {
		out[i] = favoriteDTO{Kind: r.Kind, ID: r.TargetID, GroupID: r.GroupID, Position: r.Position}
	}
	return out
}

func toFavoriteGroupDTOs(rows []sqlc.ListFavoriteGroupsRow) []favoriteGroupDTO {
	out := make([]favoriteGroupDTO, len(rows))
	for i, g := range rows {
		out[i] = favoriteGroupDTO(g)
	}
	return out
}

func validFavoriteKind(kind string) bool {
	return kind == "customer" || kind == "project" || kind == "checklist" || kind == "template"
}

// favoriteTarget reads {kind}/{id} from the path and checks that the target
// exists and the user may see it (404 otherwise).
func (s *Server) favoriteTarget(ctx context.Context, r *http.Request, u authz.User, check bool) (string, int64, error) {
	kind := r.PathValue("kind")
	id, err := pathID(r, "id")
	if err != nil {
		return "", 0, err
	}
	if !validFavoriteKind(kind) {
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
	case "template": // templates are visible to every signed-in user
		_, err = s.q.GetTemplate(ctx, id)
	}
	return kind, id, err
}

// PUT /api/favorites/{kind}/{id}: mark as favorite (only with access); new
// favorites go to the end of the list without group. Idempotent.
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

func groupName(v validator, name string) string {
	n := strings.TrimSpace(name)
	if l := utf8.RuneCountInString(n); l < 1 || l > 60 {
		v["name"] = "length:1:60"
	}
	return n
}

// POST /api/favorite-groups {name}: new group at the end.
func (s *Server) createFavoriteGroup(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	name := groupName(v, req.Name)
	if err := v.err(); err != nil {
		return err
	}
	g, err := s.q.CreateFavoriteGroup(r.Context(), sqlc.CreateFavoriteGroupParams{UserID: u.ID, Name: name})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, favoriteGroupDTO(g))
	return nil
}

// PATCH /api/favorite-groups/{id} {name?, collapsed?}: only own groups.
func (s *Server) updateFavoriteGroup(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Name      *string `json:"name"`
		Collapsed *bool   `json:"collapsed"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	g, err := s.q.GetFavoriteGroup(r.Context(), sqlc.GetFavoriteGroupParams{ID: id, UserID: u.ID})
	if err != nil {
		return err
	}
	p := sqlc.UpdateFavoriteGroupParams{ID: id, UserID: u.ID, Name: g.Name, Collapsed: g.Collapsed}
	v := validator{}
	if req.Name != nil {
		p.Name = groupName(v, *req.Name)
	}
	if req.Collapsed != nil {
		p.Collapsed = *req.Collapsed
	}
	if err := v.err(); err != nil {
		return err
	}
	out, err := s.q.UpdateFavoriteGroup(r.Context(), p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, favoriteGroupDTO(out))
	return nil
}

// DELETE /api/favorite-groups/{id}: its favorites move to the end of the list
// without group (relative order kept).
func (s *Server) deleteFavoriteGroup(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetFavoriteGroup(r.Context(), sqlc.GetFavoriteGroupParams{ID: id, UserID: u.ID}); err != nil {
		return err
	}
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		next, err := q.NextUngroupedPosition(r.Context(), u.ID)
		if err != nil {
			return err
		}
		if err := q.UngroupFavorites(r.Context(), sqlc.UngroupFavoritesParams{Offset: next, UserID: u.ID, GroupID: &id}); err != nil {
			return err
		}
		return q.DeleteFavoriteGroup(r.Context(), sqlc.DeleteFavoriteGroupParams{ID: id, UserID: u.ID})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// PUT /api/favorites/order {groups: [id…], items: [{kind, id, group_id|null}…]}:
// group positions = index in groups; each favorite gets its group and the index
// among the items of the same group. Unknown or foreign groups → 400; items that
// are not favorites of the user are ignored. One transaction.
func (s *Server) orderFavorites(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		Groups []int64 `json:"groups"`
		Items  []struct {
			Kind    string `json:"kind"`
			ID      int64  `json:"id"`
			GroupID *int64 `json:"group_id"`
		} `json:"items"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	own, err := s.q.ListFavoriteGroups(r.Context(), u.ID)
	if err != nil {
		return err
	}
	mine := make(map[int64]bool, len(own))
	for _, g := range own {
		mine[g.ID] = true
	}
	for _, id := range req.Groups {
		if !mine[id] {
			return &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"groups": "unknown_group"}}
		}
	}
	for _, it := range req.Items {
		if it.GroupID != nil && !mine[*it.GroupID] {
			return &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"items": "unknown_group"}}
		}
	}

	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		for i, id := range req.Groups {
			if err := q.SetFavoriteGroupPosition(r.Context(), sqlc.SetFavoriteGroupPositionParams{Position: int64(i), ID: id, UserID: u.ID}); err != nil {
				return err
			}
		}
		counters := map[int64]int64{} // group id (0 = without group) → next position
		for _, it := range req.Items {
			if !validFavoriteKind(it.Kind) {
				continue
			}
			var key int64
			if it.GroupID != nil {
				key = *it.GroupID
			}
			pos := counters[key]
			counters[key] = pos + 1
			// rows that are not favorites of the user are simply not updated
			if err := q.SetFavoritePlacement(r.Context(), sqlc.SetFavoritePlacementParams{
				GroupID: it.GroupID, Position: pos, UserID: u.ID, Kind: it.Kind, TargetID: it.ID,
			}); err != nil {
				return err
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
