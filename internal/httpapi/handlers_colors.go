package httpapi

import (
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Customer color (shared by all users, set by members and admins). With
// "everywhere" it wins over the personal card colors (user pref "card_colors")
// in all views. Must match CARD_COLORS in web/src/lib/cardColors.ts; each color
// also as "<name>-strong" (stronger tint).
var cardColors = func() map[string]bool {
	m := map[string]bool{"": true}
	for _, c := range []string{"red", "orange", "amber", "yellow", "lime", "green", "teal", "cyan", "blue", "indigo", "violet", "pink", "gray"} {
		m[c], m[c+"-strong"] = true, true
	}
	return m
}()

// PUT /api/customers/{id}/colors {"color"?, "everywhere"?}: only the given fields change.
func (s *Server) setCustomerColors(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, id); err != nil {
		return err
	}
	var req struct {
		Color      *string `json:"color"`
		Everywhere *bool   `json:"everywhere"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	c, err := s.q.GetCustomer(r.Context(), id)
	if err != nil {
		return err
	}
	p := sqlc.SetCustomerColorParams{ID: id, Color: c.Color, ColorEverywhere: c.ColorEverywhere}
	if req.Color != nil {
		if !cardColors[*req.Color] {
			return &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"color": "invalid"}}
		}
		p.Color = *req.Color
	}
	if req.Everywhere != nil {
		p.ColorEverywhere = *req.Everywhere
	}
	if err := s.q.SetCustomerColor(r.Context(), p); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"color": p.Color, "color_everywhere": p.ColorEverywhere})
	return nil
}
