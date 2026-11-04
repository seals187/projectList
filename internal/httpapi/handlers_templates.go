package httpapi

import (
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Templates sind global: alle angemeldeten Benutzer dürfen sie lesen,
// Admins und Member dürfen sie bearbeiten.

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	ts, err := s.q.ListTemplates(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, ts)
	return nil
}

type templateItemReq struct {
	Kind        *string `json:"kind"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	NoteColor   *string `json:"note_color"`
	BgColor     *string `json:"bg_color"`
	Indent      *int64  `json:"indent"`
	Lang        *string `json:"lang"`
	Position    *int64  `json:"position"` // nur beim Anlegen: einfügen vor dieser Position
}

// apply überträgt gesetzte Felder auf b.
func (it templateItemReq) apply(b *block) {
	if it.Kind != nil {
		b.Kind = *it.Kind
	}
	if it.Title != nil {
		b.Title = *it.Title
	}
	if it.Description != nil {
		b.Description = *it.Description
	}
	if it.BgColor != nil {
		b.BgColor = *it.BgColor
	}
	if it.NoteColor != nil {
		b.NoteColor = *it.NoteColor
	}
	if it.Indent != nil {
		b.Indent = *it.Indent
	}
	if it.Lang != nil {
		b.Lang = *it.Lang
	}
}

// fill validiert die Felder für das Anlegen eines Template-Eintrags.
func (it templateItemReq) fill(v validator, p *sqlc.CreateTemplateItemParams, _ string) {
	b := block{Kind: kindItem}
	it.apply(&b)
	v.normalize(&b)
	p.Kind, p.Title, p.Description, p.NoteColor, p.BgColor, p.Indent, p.Lang = b.Kind, b.Title, b.Description, b.NoteColor, b.BgColor, b.Indent, b.Lang
}

func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	var req struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Items       []templateItemReq `json:"items"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	name := v.text("name", req.Name, 1, 200)
	desc := v.text("description", req.Description, 0, 10000)
	if len(req.Items) > 1000 {
		v["items"] = "too_many"
	}
	items := make([]sqlc.CreateTemplateItemParams, len(req.Items))
	for i, it := range req.Items {
		it.fill(v, &items[i], "items")
		items[i].Position = int64(i)
	}
	if err := v.err(); err != nil {
		return err
	}
	var t sqlc.ChecklistTemplate
	err := db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		var err error
		if t, err = q.CreateTemplate(r.Context(), sqlc.CreateTemplateParams{Name: name, Description: desc}); err != nil {
			return err
		}
		for _, it := range items {
			it.TemplateID = t.ID
			if _, err := q.CreateTemplateItem(r.Context(), it); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, t)
	return nil
}

func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	t, err := s.q.GetTemplate(r.Context(), id)
	if err != nil {
		return err
	}
	items, err := s.q.ListTemplateItems(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"template": t, "items": items})
	return nil
}

func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	t, err := s.q.GetTemplate(r.Context(), id)
	if err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateTemplateParams{ID: id, Name: t.Name, Description: t.Description}
	if req.Name != nil {
		p.Name = v.text("name", *req.Name, 1, 200)
	}
	if req.Description != nil {
		p.Description = v.text("description", *req.Description, 0, 10000)
	}
	if err := v.err(); err != nil {
		return err
	}
	t, err = s.q.UpdateTemplate(r.Context(), p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, t)
	return nil
}

func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetTemplate(r.Context(), id); err != nil {
		return err
	}
	// Bestehende Checklisten bleiben erhalten (template_id → NULL).
	if err := s.q.DeleteTemplate(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) createTemplateItem(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	templateID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetTemplate(r.Context(), templateID); err != nil {
		return err
	}
	var req templateItemReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.CreateTemplateItemParams{TemplateID: templateID}
	req.fill(v, &p, "title")
	if req.Position != nil && *req.Position < 0 {
		v["position"] = "invalid_value"
	}
	if err := v.err(); err != nil {
		return err
	}
	var it sqlc.TemplateItem
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		next, err := q.NextTemplateItemPosition(r.Context(), templateID)
		if err != nil {
			return err
		}
		p.Position = next
		if req.Position != nil && *req.Position < next {
			p.Position = *req.Position
			if err := q.ShiftTemplateItemPositions(r.Context(), sqlc.ShiftTemplateItemPositionsParams{TemplateID: templateID, Position: p.Position}); err != nil {
				return err
			}
		}
		it, err = q.CreateTemplateItem(r.Context(), p)
		return err
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, it)
	return nil
}

func (s *Server) updateTemplateItem(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req templateItemReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	it, err := s.q.GetTemplateItem(r.Context(), id)
	if err != nil {
		return err
	}
	v := validator{}
	if req.Position != nil {
		v["position"] = "invalid_value"
	}
	b := block{Kind: it.Kind, Title: it.Title, Description: it.Description, NoteColor: it.NoteColor, BgColor: it.BgColor, Indent: it.Indent, Lang: it.Lang}
	req.apply(&b)
	v.normalize(&b)
	p := sqlc.UpdateTemplateItemParams{
		ID: id, Kind: b.Kind, Title: b.Title, Description: b.Description, NoteColor: b.NoteColor, BgColor: b.BgColor, Indent: b.Indent, Lang: b.Lang,
	}
	if err := v.err(); err != nil {
		return err
	}
	it, err = s.q.UpdateTemplateItem(r.Context(), p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, it)
	return nil
}

func (s *Server) deleteTemplateItem(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetTemplateItem(r.Context(), id); err != nil {
		return err
	}
	if err := s.q.DeleteTemplateItem(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) orderTemplateItems(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	templateID, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req orderReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return err
	}
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		for i, id := range req.IDs {
			n, err := q.SetTemplateItemPosition(r.Context(), sqlc.SetTemplateItemPositionParams{Position: int64(i), ID: id, TemplateID: templateID})
			if err != nil {
				return err
			}
			if n != 1 {
				return errBadRequest("invalid_order")
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
