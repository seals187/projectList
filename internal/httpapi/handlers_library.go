package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/automation"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/service"
)

// Template library: categories (admins), variables (admins) and automations
// (admins). Everybody logged in may read them.

// ---- Categories ------------------------------------------------------------------

type categoryReq struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (v validator) categoryColor(s string) string {
	if s != "" && !cardColors[s] {
		v["color"] = "invalid_value"
	}
	return s
}

func (s *Server) listTemplateCategories(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	cs, err := s.q.ListTemplateCategories(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, cs)
	return nil
}

func (s *Server) createTemplateCategory(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	var req categoryReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.CreateTemplateCategoryParams{Name: v.text("name", deref(req.Name), 1, 60), Color: v.categoryColor(deref(req.Color))}
	if err := v.err(); err != nil {
		return err
	}
	c, err := s.q.CreateTemplateCategory(r.Context(), p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, c)
	return nil
}

func (s *Server) updateTemplateCategory(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req categoryReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	c, err := s.q.GetTemplateCategory(r.Context(), id)
	if err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateTemplateCategoryParams{ID: id, Name: c.Name, Color: c.Color}
	if req.Name != nil {
		p.Name = v.text("name", *req.Name, 1, 60)
	}
	if req.Color != nil {
		p.Color = v.categoryColor(*req.Color)
	}
	if err := v.err(); err != nil {
		return err
	}
	if c, err = s.q.UpdateTemplateCategory(r.Context(), p); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, c)
	return nil
}

func (s *Server) deleteTemplateCategory(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetTemplateCategory(r.Context(), id); err != nil {
		return err
	}
	// templates of the category become uncategorized (ON DELETE SET NULL)
	if err := s.q.DeleteTemplateCategory(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---- Variables -------------------------------------------------------------------

type variableDTO struct {
	ID           int64   `json:"id,omitempty"` // 0 = system variable
	Name         string  `json:"name"`
	Source       string  `json:"source"` // "system", "custom" or an integration
	Description  string  `json:"description"`
	DescKey      string  `json:"desc_key,omitempty"` // system: i18n key of the description
	DefaultValue string  `json:"default_value"`
	Example      string  `json:"example"`   // system: value for the newest project
	UsedIn       int     `json:"used_in"`   // number of templates using it
	Templates    []int64 `json:"templates"` // their IDs
}

// templateVariableUse: variable name → IDs of the templates using it.
func (s *Server) templateVariableUse(ctx context.Context) (map[string][]int64, error) {
	rows, err := s.q.ListAllTemplateTexts(ctx)
	if err != nil {
		return nil, err
	}
	use := map[string][]int64{}
	for _, row := range rows {
		for _, name := range service.VariablesIn(row.Text) {
			if cur, ok := service.LegacyNames[name]; ok {
				name = cur // old (German) name counts for the current one
			}
			if !slices.Contains(use[name], row.TemplateID) {
				use[name] = append(use[name], row.TemplateID)
			}
		}
	}
	return use, nil
}

// GET /api/template-variables: system and custom variables with usage.
func (s *Server) listTemplateVariables(w http.ResponseWriter, r *http.Request, u authz.User) error {
	ctx := r.Context()
	use, err := s.templateVariableUse(ctx)
	if err != nil {
		return err
	}
	latest, err := s.q.LatestProjectID(ctx)
	if err != nil {
		return err
	}
	examples, err := service.SystemValues(ctx, s.q, latest, u.Name, s.now())
	if err != nil {
		examples, _ = service.SystemValues(ctx, s.q, 0, u.Name, s.now())
	}
	out := []variableDTO{}
	for _, sv := range service.SystemVariables {
		ids := orEmpty(use[sv.Name])
		out = append(out, variableDTO{Name: sv.Name, Source: "system", DescKey: sv.Desc, Example: examples[sv.Name], UsedIn: len(ids), Templates: ids})
	}
	custom, err := s.q.ListTemplateVariables(ctx)
	if err != nil {
		return err
	}
	for _, cv := range custom {
		ids := orEmpty(use[cv.Name])
		out = append(out, variableDTO{
			ID: cv.ID, Name: cv.Name, Source: cv.Source, Description: cv.Description, DefaultValue: cv.DefaultValue, UsedIn: len(ids), Templates: ids,
		})
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func orEmpty(ids []int64) []int64 {
	if ids == nil {
		return []int64{}
	}
	return ids
}

type variableReq struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	DefaultValue *string `json:"default_value"`
	Source       *string `json:"source"`
}

func (v validator) variableName(name string) string {
	switch {
	case !service.ValidVariableName(name):
		v["name"] = "invalid_value"
	case service.IsSystemVariable(name):
		v["name"] = "reserved"
	}
	return name
}

func (s *Server) createTemplateVariable(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	var req variableReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.CreateTemplateVariableParams{
		Name:         v.variableName(deref(req.Name)),
		Description:  v.text("description", deref(req.Description), 0, 200),
		DefaultValue: v.text("default_value", deref(req.DefaultValue), 0, 500),
		Source:       "custom",
	}
	if req.Source != nil {
		p.Source = v.oneOf("source", *req.Source, service.VariableSources...)
	}
	if err := v.err(); err != nil {
		return err
	}
	c, err := s.q.CreateTemplateVariable(r.Context(), p)
	if err != nil {
		if isUniqueViolation(err) {
			return &httpError{Status: http.StatusConflict, Msg: "validation", Fields: map[string]string{"name": "taken"}}
		}
		return err
	}
	writeJSON(w, http.StatusCreated, c)
	return nil
}

func (s *Server) updateTemplateVariable(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req variableReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	c, err := s.q.GetTemplateVariable(r.Context(), id)
	if err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateTemplateVariableParams{ID: id, Name: c.Name, Description: c.Description, DefaultValue: c.DefaultValue, Source: c.Source}
	if req.Name != nil {
		p.Name = v.variableName(*req.Name)
	}
	if req.Description != nil {
		p.Description = v.text("description", *req.Description, 0, 200)
	}
	if req.DefaultValue != nil {
		p.DefaultValue = v.text("default_value", *req.DefaultValue, 0, 500)
	}
	if req.Source != nil {
		p.Source = v.oneOf("source", *req.Source, service.VariableSources...)
	}
	if err := v.err(); err != nil {
		return err
	}
	if c, err = s.q.UpdateTemplateVariable(r.Context(), p); err != nil {
		if isUniqueViolation(err) {
			return &httpError{Status: http.StatusConflict, Msg: "validation", Fields: map[string]string{"name": "taken"}}
		}
		return err
	}
	writeJSON(w, http.StatusOK, c)
	return nil
}

func (s *Server) deleteTemplateVariable(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetTemplateVariable(r.Context(), id); err != nil {
		return err
	}
	if err := s.q.DeleteTemplateVariable(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// GET /api/templates/{id}/variables: the variables a template uses (apply dialog:
// custom ones are asked, with their default value).
func (s *Server) templateVariables(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	ctx := r.Context()
	t, err := s.q.GetTemplate(ctx, id)
	if err != nil {
		return err
	}
	items, err := s.q.ListTemplateItems(ctx, id)
	if err != nil {
		return err
	}
	texts := []string{t.Name}
	if t.Kind == service.KindTable {
		texts = append(texts, t.SheetData)
	}
	for _, it := range items {
		texts = append(texts, it.Title, it.Description)
	}
	custom, err := s.q.ListTemplateVariables(ctx)
	if err != nil {
		return err
	}
	out := []variableDTO{}
	for _, name := range service.VariablesIn(texts...) {
		d := variableDTO{Name: name, Source: "unknown"}
		if service.IsSystemVariable(name) {
			d.Source = "system"
		}
		for _, cv := range custom {
			if cv.Name == name {
				d = variableDTO{ID: cv.ID, Name: name, Source: cv.Source, Description: cv.Description, DefaultValue: cv.DefaultValue}
			}
		}
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

// ---- Automations -----------------------------------------------------------------

type ruleDTO struct {
	sqlc.AutomationRule
	Conditions   []automation.Condition `json:"conditions"`
	Schedule     *automation.Schedule   `json:"schedule"`
	TemplateName string                 `json:"template_name"`
	TargetName   string                 `json:"target_name"` // "Customer › Project" (schedule)
}

func (s *Server) ruleDTO(ctx context.Context, r sqlc.AutomationRule) ruleDTO {
	d := ruleDTO{AutomationRule: r, Conditions: []automation.Condition{}}
	_ = json.Unmarshal([]byte(r.Conditions), &d.Conditions)
	if r.TriggerKind == automation.Scheduled {
		var sc automation.Schedule
		if json.Unmarshal([]byte(r.Schedule), &sc) == nil {
			d.Schedule = &sc
		}
	}
	if t, err := s.q.GetTemplate(ctx, r.TemplateID); err == nil {
		d.TemplateName = t.Name
	}
	if r.TargetProjectID != nil {
		if c, err := s.q.VariableContext(ctx, *r.TargetProjectID); err == nil {
			d.TargetName = c.CustomerName + " › " + c.ProjectName
		}
	}
	return d
}

// GET /api/automations: rules and the latest runs.
func (s *Server) listAutomations(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	ctx := r.Context()
	rules, err := s.q.ListAutomationRules(ctx)
	if err != nil {
		return err
	}
	out := make([]ruleDTO, len(rules))
	for i, rule := range rules {
		out[i] = s.ruleDTO(ctx, rule)
	}
	runs, err := s.q.ListAutomationRuns(ctx, 15)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": out, "runs": runs, "integrations": automation.Integrations})
	return nil
}

type ruleReq struct {
	Trigger         *string                 `json:"trigger_kind"`
	Source          *string                 `json:"source"`
	Conditions      *[]automation.Condition `json:"conditions"`
	Schedule        *automation.Schedule    `json:"schedule"`
	TemplateID      *int64                  `json:"template_id"`
	TargetProjectID Opt[int64]              `json:"target_project_id"`
	ChecklistTitle  *string                 `json:"checklist_title"`
	Enabled         *bool                   `json:"enabled"`
}

// fill applies the request to p (current values) and validates the result.
func (s *Server) fillRule(ctx context.Context, v validator, req ruleReq, p *sqlc.UpdateAutomationRuleParams) {
	if req.Trigger != nil {
		p.TriggerKind = v.oneOf("trigger_kind", *req.Trigger, automation.Triggers...)
	}
	if req.Source != nil {
		p.Source = *req.Source
	}
	if automation.IsExternal(p.TriggerKind) {
		v.oneOf("source", p.Source, automation.Integrations...)
	} else {
		p.Source = automation.SourceApp
	}
	if req.Conditions != nil {
		conds := *req.Conditions
		if len(conds) > 10 {
			v["conditions"] = "too_many"
		}
		for i := range conds {
			conds[i].Field = v.text("conditions", conds[i].Field, 1, 60)
			conds[i].Op = v.oneOf("conditions", conds[i].Op, "equals", "contains")
			conds[i].Value = v.text("conditions", conds[i].Value, 0, 200)
		}
		b, _ := json.Marshal(conds)
		p.Conditions = string(b)
	}
	if req.Schedule != nil {
		b, _ := json.Marshal(req.Schedule)
		p.Schedule = string(b)
	}
	if p.TriggerKind == automation.Scheduled {
		var sc automation.Schedule
		if json.Unmarshal([]byte(p.Schedule), &sc) != nil || !sc.Valid() {
			v["schedule"] = "invalid_value"
		}
	}
	if req.TemplateID != nil {
		p.TemplateID = *req.TemplateID
	}
	if tmpl, err := s.q.GetTemplate(ctx, p.TemplateID); err != nil {
		v["template_id"] = "required"
	} else if tmpl.Kind != service.KindChecklist {
		v["template_id"] = "invalid_value" // automations create checklists only
	}
	if req.TargetProjectID.Set {
		p.TargetProjectID = nil
		if !req.TargetProjectID.Null {
			id := req.TargetProjectID.V
			if _, err := s.q.GetProject(ctx, id); err != nil {
				v["target_project_id"] = "invalid_value"
			}
			p.TargetProjectID = &id
		}
	}
	if p.TriggerKind == automation.Scheduled && p.TargetProjectID == nil {
		v["target_project_id"] = "required"
	}
	if req.ChecklistTitle != nil {
		p.ChecklistTitle = v.text("checklist_title", *req.ChecklistTitle, 0, 200)
	}
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
}

func (s *Server) createAutomation(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	var req ruleReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	ctx := r.Context()
	v := validator{}
	p := sqlc.UpdateAutomationRuleParams{Source: automation.SourceApp, Conditions: "[]", Schedule: "{}", Enabled: true}
	if req.Trigger == nil {
		v["trigger_kind"] = "required"
	}
	s.fillRule(ctx, v, req, &p)
	if err := v.err(); err != nil {
		return err
	}
	rule, err := s.q.CreateAutomationRule(ctx, sqlc.CreateAutomationRuleParams{
		TriggerKind: p.TriggerKind, Source: p.Source, Conditions: p.Conditions, Schedule: p.Schedule, TemplateID: p.TemplateID,
		TargetProjectID: p.TargetProjectID, ChecklistTitle: p.ChecklistTitle, Enabled: p.Enabled,
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, s.ruleDTO(ctx, rule))
	return nil
}

func (s *Server) updateAutomation(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req ruleReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	ctx := r.Context()
	cur, err := s.q.GetAutomationRule(ctx, id)
	if err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateAutomationRuleParams{
		ID: id, TriggerKind: cur.TriggerKind, Source: cur.Source, Conditions: cur.Conditions, Schedule: cur.Schedule, TemplateID: cur.TemplateID,
		TargetProjectID: cur.TargetProjectID, ChecklistTitle: cur.ChecklistTitle, Enabled: cur.Enabled,
	}
	s.fillRule(ctx, v, req, &p)
	if err := v.err(); err != nil {
		return err
	}
	rule, err := s.q.UpdateAutomationRule(ctx, p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, s.ruleDTO(ctx, rule))
	return nil
}

func (s *Server) deleteAutomation(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetAutomationRule(r.Context(), id); err != nil {
		return err
	}
	if err := s.q.DeleteAutomationRule(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---- Events from the app ---------------------------------------------------------

// fire runs the automations of an event inside ProjectList (after the change is saved).
func (s *Server) fire(ctx context.Context, kind string, p sqlc.Project, extra map[string]string) {
	fields := map[string]string{"name": p.Name, "customer": strconv.FormatInt(p.CustomerID, 10), "status": p.Status}
	for k, v := range extra {
		fields[k] = v
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	s.auto.Fire(ctx, automation.Event{Kind: kind, Source: automation.SourceApp, ProjectID: p.ID, Fields: fields})
}
