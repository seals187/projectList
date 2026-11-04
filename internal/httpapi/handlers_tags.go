package httpapi

import (
	"context"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Custom status names and project tags. Both are global and are delivered
// with /api/sidebar, so every page has them without an extra request.

var hexColor = regexp.MustCompile(`^#[0-9a-f]{6}$`)

const maxTagsPerProject = 20

// color validates a color in the form #rrggbb (stored lowercase).
func (v validator) color(field, s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if !hexColor.MatchString(s) {
		v[field] = "invalid_color"
	}
	return s
}

// statusSettings are the custom names, colors and WIP limits per status
// (only statuses that have one).
type statusSettings struct {
	Labels map[string]string
	Colors map[string]string
	WIP    map[string]int64
}

func (s *Server) statusSettings(ctx context.Context) (statusSettings, error) {
	out := statusSettings{Labels: map[string]string{}, Colors: map[string]string{}, WIP: map[string]int64{}}
	rows, err := s.q.ListStatusLabels(ctx)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.Label != "" {
			out.Labels[r.Status] = r.Label
		}
		if r.Color != "" {
			out.Colors[r.Status] = r.Color
		}
		if r.WipLimit > 0 {
			out.WIP[r.Status] = r.WipLimit
		}
	}
	return out, nil
}

// PUT /api/status-labels/{status} {"label", "color", "wip_limit"} (admin): name,
// color and WIP limit of one status for everyone. Empty values / 0 fall back to
// the default name, the default color and no limit.
func (s *Server) setStatusLabel(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		Label    string `json:"label"`
		Color    string `json:"color"`
		WIPLimit int64  `json:"wip_limit"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	status := v.oneOf("status", r.PathValue("status"), projectStatuses...)
	label := v.text("label", req.Label, 0, 40)
	color := ""
	if strings.TrimSpace(req.Color) != "" {
		color = v.color("color", req.Color)
	}
	if req.WIPLimit < 0 || req.WIPLimit > 999 {
		v["wip_limit"] = "invalid"
	}
	if err := v.err(); err != nil {
		return err
	}
	ctx := r.Context()
	var err error
	if label == "" && color == "" && req.WIPLimit == 0 {
		err = s.q.DeleteStatusLabel(ctx, status)
	} else {
		err = s.q.UpsertStatusLabel(ctx, sqlc.UpsertStatusLabelParams{Status: status, Label: label, Color: color, WipLimit: req.WIPLimit})
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "label": label, "color": color, "wip_limit": req.WIPLimit})
	return nil
}

type tagReq struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

// POST /api/tags {"name", "color"} (members and admins).
func (s *Server) createTag(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req tagReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.CreateTagParams{Name: v.text("name", deref(req.Name), 1, 40), Color: v.color("color", deref(req.Color))}
	if err := v.err(); err != nil {
		return err
	}
	t, err := s.q.CreateTag(r.Context(), p)
	if isUniqueViolation(err) {
		return errConflict("tag_exists")
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, t)
	return nil
}

// PATCH /api/tags/{id} {"name"?, "color"?} (admin).
func (s *Server) updateTag(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	cur, err := s.q.GetTag(r.Context(), id)
	if err != nil {
		return err
	}
	var req tagReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateTagParams{ID: id, Name: cur.Name, Color: cur.Color}
	if req.Name != nil {
		p.Name = v.text("name", *req.Name, 1, 40)
	}
	if req.Color != nil {
		p.Color = v.color("color", *req.Color)
	}
	if err := v.err(); err != nil {
		return err
	}
	t, err := s.q.UpdateTag(r.Context(), p)
	if isUniqueViolation(err) {
		return errConflict("tag_exists")
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, t)
	return nil
}

// DELETE /api/tags/{id} (admin): also removes it from all projects.
func (s *Server) deleteTag(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetTag(r.Context(), id); err != nil {
		return err
	}
	if err := s.q.DeleteTag(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// PUT /api/projects/{id}/tags {"tag_ids": [..]}: replaces the project's tags.
func (s *Server) setProjectTags(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	ctx := r.Context()
	if _, err := s.loadProject(ctx, u, id); err != nil {
		return err
	}
	var req struct {
		TagIDs []int64 `json:"tag_ids"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if len(req.TagIDs) > maxTagsPerProject {
		return errBadRequest("too_many_tags")
	}
	names := make([]string, 0, len(req.TagIDs))
	for _, t := range req.TagIDs {
		tag, err := s.q.GetTag(ctx, t)
		if err != nil {
			return errBadRequest("unknown_tag")
		}
		names = append(names, tag.Name)
	}
	prev, err := s.q.ListTagIDsByProject(ctx, id)
	if err != nil {
		return err
	}
	err = db.WithTx(ctx, s.db, func(q *sqlc.Queries) error {
		if err := q.ClearProjectTags(ctx, id); err != nil {
			return err
		}
		for _, t := range req.TagIDs {
			if err := q.AddProjectTag(ctx, sqlc.AddProjectTagParams{ProjectID: id, TagID: t}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	ids, err := s.q.ListTagIDsByProject(ctx, id)
	if err != nil {
		return err
	}
	if !slices.Equal(prev, ids) {
		s.logActivity(ctx, id, u.ID, actTags, "", strings.Join(names, ", "))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tag_ids": ids})
	return nil
}
