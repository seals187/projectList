package httpapi

import (
	"net/http"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/service"
)

// instantiate handles POST /api/projects/{id}/{checklists|tables}/from-template
// {template_id, title, values}: the template (of the given kind) is applied to the project.
func (s *Server) instantiate(w http.ResponseWriter, r *http.Request, u authz.User, kind string) (service.Instance, error) {
	projectID, err := pathID(r, "id")
	if err != nil {
		return service.Instance{}, err
	}
	if _, err := s.loadProject(r.Context(), u, projectID); err != nil {
		return service.Instance{}, err
	}
	var req struct {
		TemplateID int64             `json:"template_id"`
		Title      string            `json:"title"`
		Values     map[string]string `json:"values"` // custom variables ({{key}})
	}
	if err := decode(w, r, &req); err != nil {
		return service.Instance{}, err
	}
	v := validator{}
	title := v.text("title", req.Title, 0, 200)
	if req.TemplateID <= 0 {
		v["template_id"] = "required"
	}
	if err := v.err(); err != nil {
		return service.Instance{}, err
	}
	if len(req.Values) > 100 {
		return service.Instance{}, errBadRequest("too_many_values")
	}
	return s.life.Instantiate(r.Context(), actor(u), service.InstantiateReq{
		ProjectID: projectID, TemplateID: req.TemplateID, Kind: kind, Title: title, Values: req.Values,
	})
}

func (s *Server) createChecklistFromTemplate(w http.ResponseWriter, r *http.Request, u authz.User) error {
	inst, err := s.instantiate(w, r, u, service.KindChecklist)
	if err != nil {
		return err
	}
	dto, err := s.checklistDTO(r.Context(), *inst.Checklist)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, dto)
	return nil
}

func (s *Server) createTableFromTemplate(w http.ResponseWriter, r *http.Request, u authz.User) error {
	inst, err := s.instantiate(w, r, u, service.KindTable)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toTableDTO(*inst.Table, u.Name))
	return nil
}
