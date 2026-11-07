package httpapi_test

import (
	"fmt"
	"strings"
	"testing"
)

type libTemplate struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	CategoryID *int64 `json:"category_id"`
	UsedCount  int64  `json:"used_count"`
	LastUsed   string `json:"last_used"`
}

func TestTemplateLibrary(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, p struct {
		ID int64 `json:"id"`
	}
	admin.do("POST", "/api/customers", map[string]any{"name": "Kunde A"}, &c)
	member := e.user("member@example.com", "member", c.ID)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]any{"name": "Projekt D"}, &p)

	// categories: admins only
	var cat struct {
		ID int64 `json:"id"`
	}
	expect(t, "kategorie member", member.do("POST", "/api/template-categories", map[string]any{"name": "Security"}, nil), 403)
	expect(t, "kategorie farbe", admin.do("POST", "/api/template-categories", map[string]any{"name": "X", "color": "pink-ish"}, nil), 400)
	expect(t, "kategorie", admin.do("POST", "/api/template-categories", map[string]any{"name": "Security", "color": "red"}, &cat), 201)

	// template with variables; members may set the category
	var tpl struct {
		ID int64 `json:"id"`
	}
	admin.do("POST", "/api/templates", map[string]any{
		"name": "Audit {{customer.name}}",
		"items": []map[string]string{
			{"title": "Server {{server.hostname}} prüfen"},
			{"title": "Lizenz {{lizenz.nummer}} · {{unbekannt}}"},
			{"title": "Fällig {{ stichtag }} für <{{project.name}}>"},
		},
	}, &tpl)
	tu := fmt.Sprintf("/api/templates/%d", tpl.ID)
	expect(t, "kategorie setzen", member.do("PATCH", tu, map[string]any{"category_id": cat.ID}, nil), 200)
	expect(t, "kategorie ungültig", member.do("PATCH", tu, map[string]any{"category_id": 999}, nil), 400)

	// custom variables: admins only, names validated, system names reserved
	expect(t, "variable member", member.do("POST", "/api/template-variables", map[string]any{"name": "server.hostname"}, nil), 403)
	expect(t, "variable name", admin.do("POST", "/api/template-variables", map[string]any{"name": "Server Host"}, nil), 400)
	expect(t, "variable system", admin.do("POST", "/api/template-variables", map[string]any{"name": "heute"}, nil), 400)
	expect(t, "variable system en", admin.do("POST", "/api/template-variables", map[string]any{"name": "customer.name"}, nil), 400)
	expect(t, "variable quelle", admin.do("POST", "/api/template-variables", map[string]any{"name": "x.y", "source": "jira"}, nil), 400)
	expect(t, "variable", admin.do("POST", "/api/template-variables", map[string]any{"name": "server.hostname", "description": "Hostname"}, nil), 201)
	expect(t, "variable doppelt", admin.do("POST", "/api/template-variables", map[string]any{"name": "server.hostname"}, nil), 409)
	expect(t, "variable default", admin.do("POST", "/api/template-variables", map[string]any{"name": "lizenz.nummer", "default_value": "L-1"}, nil), 201)

	var used []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
		UsedIn int    `json:"used_in"`
	}
	member.do("GET", "/api/template-variables", nil, &used)
	counts := map[string]int{}
	for _, v := range used {
		counts[v.Name+"/"+v.Source] = v.UsedIn
	}
	if counts["customer.name/system"] != 1 || counts["server.hostname/custom"] != 1 || counts["today/system"] != 0 {
		t.Fatalf("verwendung: %v", counts)
	}
	var tv []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	}
	member.do("GET", tu+"/variables", nil, &tv)
	if len(tv) != 6 || tv[len(tv)-1].Name != "unbekannt" || tv[len(tv)-1].Source != "unknown" {
		t.Fatalf("variablen des templates: %+v", tv)
	}

	// apply: variables replaced (values HTML-escaped in items), use counted
	admin.do("PATCH", fmt.Sprintf("/api/projects/%d", p.ID), map[string]any{"due_date": "2026-11-20"}, nil)
	var cl struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	expect(t, "anwenden", member.do("POST", fmt.Sprintf("/api/projects/%d/checklists/from-template", p.ID),
		map[string]any{"template_id": tpl.ID, "values": map[string]string{"server.hostname": "srv<01>"}}, &cl), 201)
	if cl.Title != "Audit Kunde A" {
		t.Fatalf("titel: %q", cl.Title)
	}
	var detail struct {
		Checklist struct {
			Items []struct {
				Title string `json:"title"`
			} `json:"items"`
		} `json:"checklist"`
	}
	member.do("GET", fmt.Sprintf("/api/checklists/%d", cl.ID), nil, &detail)
	var titles []string
	for _, it := range detail.Checklist.Items {
		titles = append(titles, it.Title)
	}
	got := strings.Join(titles, " | ")
	for _, want := range []string{"srv&lt;01&gt;", "L-1", "{{unbekannt}}", "20.11.2026", "Projekt D"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ersetzung %q fehlt: %s", want, got)
		}
	}
	var list []libTemplate
	member.do("GET", "/api/templates", nil, &list)
	if list[0].UsedCount != 1 || list[0].LastUsed == "" || list[0].CategoryID == nil || *list[0].CategoryID != cat.ID {
		t.Fatalf("bibliothek: %+v", list[0])
	}
	// deleting the category keeps the template (uncategorized)
	expect(t, "kategorie löschen", admin.do("DELETE", fmt.Sprintf("/api/template-categories/%d", cat.ID), nil, nil), 204)
	member.do("GET", "/api/templates", nil, &list)
	if list[0].CategoryID != nil {
		t.Fatalf("kategorie nicht entfernt: %+v", list[0])
	}
}

func TestAutomations(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, tpl, tag struct {
		ID int64 `json:"id"`
	}
	admin.do("POST", "/api/customers", map[string]any{"name": "Kunde A"}, &c)
	member := e.user("member@example.com", "member", c.ID)
	admin.do("POST", "/api/templates", map[string]any{"name": "Go-Live {{project.name}}", "items": []map[string]string{{"title": "Start"}}}, &tpl)
	admin.do("POST", "/api/tags", map[string]any{"name": "Go-Live", "color": "#22c55e"}, &tag)

	// rules: admins only, validated
	rule := map[string]any{"trigger_kind": "project_created", "template_id": tpl.ID,
		"conditions": []map[string]string{{"field": "name", "op": "contains", "value": "migration"}}}
	expect(t, "regel member", member.do("POST", "/api/automations", rule, nil), 403)
	expect(t, "regel ohne template", admin.do("POST", "/api/automations", map[string]any{"trigger_kind": "project_created"}, nil), 400)
	expect(t, "regel trigger", admin.do("POST", "/api/automations", map[string]any{"trigger_kind": "magic", "template_id": tpl.ID}, nil), 400)
	expect(t, "regel extern quelle", admin.do("POST", "/api/automations", map[string]any{"trigger_kind": "external_ticket", "source": "app", "template_id": tpl.ID}, nil), 400)
	expect(t, "regel zeitplan", admin.do("POST", "/api/automations", map[string]any{"trigger_kind": "schedule", "template_id": tpl.ID, "schedule": map[string]any{"freq": "yearly", "month": 13, "day": 1}}, nil), 400)
	expect(t, "regel", admin.do("POST", "/api/automations", rule, nil), 201)
	var tagRule struct {
		ID int64 `json:"id"`
	}
	expect(t, "tag-regel", admin.do("POST", "/api/automations", map[string]any{"trigger_kind": "project_tag", "template_id": tpl.ID,
		"conditions": []map[string]string{{"field": "tag", "op": "equals", "value": fmt.Sprint(tag.ID)}}}, &tagRule), 201)

	newProject := func(name string) int64 {
		var p struct {
			ID int64 `json:"id"`
		}
		member.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]any{"name": name}, &p)
		return p.ID
	}
	checklists := func(id int64) []string {
		var cls []struct {
			Title string `json:"title"`
		}
		member.do("GET", fmt.Sprintf("/api/projects/%d/checklists", id), nil, &cls)
		var out []string
		for _, x := range cls {
			out = append(out, x.Title)
		}
		return out
	}
	p1 := newProject("Server Migration 7.2")
	p2 := newProject("Website")
	if got := checklists(p1); len(got) != 1 || got[0] != "Go-Live Server Migration 7.2" {
		t.Fatalf("projekt erstellt: %v", got)
	}
	if got := checklists(p2); len(got) != 0 {
		t.Fatalf("bedingung ignoriert: %v", got)
	}
	// tag added → once per project
	tags := fmt.Sprintf("/api/projects/%d/tags", p2)
	member.do("PUT", tags, map[string]any{"tag_ids": []int64{tag.ID}}, nil)
	member.do("PUT", tags, map[string]any{"tag_ids": []int64{}}, nil)
	member.do("PUT", tags, map[string]any{"tag_ids": []int64{tag.ID}}, nil)
	if got := checklists(p2); len(got) != 1 {
		t.Fatalf("tag-regel: %v", got)
	}
	// paused rules do not run
	expect(t, "pausieren", admin.do("PATCH", fmt.Sprintf("/api/automations/%d", tagRule.ID), map[string]any{"enabled": false}, nil), 200)
	p3 := newProject("Shop")
	member.do("PUT", fmt.Sprintf("/api/projects/%d/tags", p3), map[string]any{"tag_ids": []int64{tag.ID}}, nil)
	if got := checklists(p3); len(got) != 0 {
		t.Fatalf("pausierte regel lief: %v", got)
	}

	var out struct {
		Rules []struct {
			RunCount     int64  `json:"run_count"`
			TemplateName string `json:"template_name"`
			Enabled      bool   `json:"enabled"`
		} `json:"rules"`
		Runs []struct {
			Status string `json:"status"`
			Target string `json:"target"`
		} `json:"runs"`
	}
	member.do("GET", "/api/automations", nil, &out)
	if len(out.Rules) != 2 || out.Rules[0].RunCount != 1 || out.Rules[1].RunCount != 1 || out.Rules[1].Enabled || out.Rules[0].TemplateName != "Go-Live {{project.name}}" {
		t.Fatalf("regeln: %+v", out.Rules)
	}
	if len(out.Runs) != 2 || out.Runs[0].Status != "ok" || out.Runs[0].Target != "Kunde A › Website" {
		t.Fatalf("läufe: %+v", out.Runs)
	}
}
