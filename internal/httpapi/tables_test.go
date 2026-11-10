package httpapi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type tableResp struct {
	ID        int64           `json:"id"`
	ProjectID int64           `json:"project_id"`
	Title     string          `json:"title"`
	Version   int64           `json:"version"`
	Data      json.RawMessage `json:"data"`
}

// Tables in a project: create, load, save with optimistic locking, rename,
// order, delete, access and the activity log.
func TestTables(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, other, p, q idResp
	admin.do("POST", "/api/customers", map[string]any{"name": "A"}, &c)
	admin.do("POST", "/api/customers", map[string]any{"name": "B"}, &other)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]any{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", other.ID), map[string]any{"name": "Q"}, &q)
	member := e.user("member@example.com", "member", c.ID)
	viewer := e.user("viewer@example.com", "viewer", c.ID)
	stranger := e.user("stranger@example.com", "member", other.ID)

	// empty table: one empty sheet
	var tb tableResp
	expect(t, "anlegen", member.do("POST", fmt.Sprintf("/api/projects/%d/tables", p.ID), map[string]any{"title": "Kalkulation"}, &tb), 201)
	if tb.Version != 1 || !strings.Contains(string(tb.Data), `"name":"Sheet1"`) {
		t.Fatalf("neue tabelle: %+v %s", tb, tb.Data)
	}
	expect(t, "ohne titel", member.do("POST", fmt.Sprintf("/api/projects/%d/tables", p.ID), map[string]any{"title": " "}, nil), 400)
	expect(t, "viewer anlegen", viewer.do("POST", fmt.Sprintf("/api/projects/%d/tables", p.ID), map[string]any{"title": "X"}, nil), 403)
	expect(t, "fremdes projekt", stranger.do("POST", fmt.Sprintf("/api/projects/%d/tables", p.ID), map[string]any{"title": "X"}, nil), 404)

	// imported workbook (XLSX import in the web app)
	wb := []map[string]any{
		{"name": "Hardware", "celldata": []map[string]any{{"r": 0, "c": 0, "v": map[string]any{"v": 2}}, {"r": 0, "c": 1, "v": map[string]any{"f": "=A1*3", "v": 6}}}},
		{"name": "Budget", "celldata": []any{}},
	}
	var imp tableResp
	expect(t, "import", member.do("POST", fmt.Sprintf("/api/projects/%d/tables", p.ID), map[string]any{"title": "Import", "data": wb}, &imp), 201)
	for name, bad := range map[string]any{
		"kein array":  map[string]any{"name": "x"},
		"leer":        []any{},
		"ohne name":   []map[string]any{{"celldata": []any{}}},
		"zu viele":    make([]map[string]any, 51),
		"kein objekt": []any{1, 2},
	} {
		expect(t, "import "+name, member.do("POST", fmt.Sprintf("/api/projects/%d/tables", p.ID), map[string]any{"title": "X", "data": bad}, nil), 400)
	}

	// load: data, project and customer; viewer reads, a stranger does not
	var got struct {
		Table    tableResp      `json:"table"`
		Project  map[string]any `json:"project"`
		Customer map[string]any `json:"customer"`
	}
	expect(t, "laden viewer", viewer.do("GET", fmt.Sprintf("/api/tables/%d", imp.ID), nil, &got), 200)
	if got.Table.Title != "Import" || !strings.Contains(string(got.Table.Data), `"f":"=A1*3"`) || got.Project["name"] != "P" || got.Customer["name"] != "A" {
		t.Fatalf("laden: %+v", got)
	}
	expect(t, "laden fremd", stranger.do("GET", fmt.Sprintf("/api/tables/%d", imp.ID), nil, nil), 404)

	// save: version check (two people on the same version → the second gets 409)
	path := fmt.Sprintf("/api/tables/%d/data", tb.ID)
	var saved struct {
		Version int64 `json:"version"`
	}
	expect(t, "speichern", member.do("PUT", path, map[string]any{"version": 1, "data": wb}, &saved), 200)
	if saved.Version != 2 {
		t.Fatalf("version nach speichern: %d", saved.Version)
	}
	expect(t, "konflikt", admin.do("PUT", path, map[string]any{"version": 1, "data": wb}, nil), 409)
	expect(t, "neue version", admin.do("PUT", path, map[string]any{"version": 2, "data": wb[:1]}, &saved), 200)
	expect(t, "viewer speichern", viewer.do("PUT", path, map[string]any{"version": 3, "data": wb}, nil), 403)
	expect(t, "ungültig speichern", member.do("PUT", path, map[string]any{"version": 3, "data": "x"}, nil), 400)
	member.do("GET", fmt.Sprintf("/api/tables/%d", tb.ID), nil, &got)
	if got.Table.Version != 3 || strings.Contains(string(got.Table.Data), "Budget") {
		t.Fatalf("gespeicherter stand: v%d %s", got.Table.Version, got.Table.Data)
	}
	// too large
	big := []map[string]any{{"name": "x", "note": strings.Repeat("a", 9<<20)}}
	expect(t, "zu gross", member.do("PUT", path, map[string]any{"version": 3, "data": big}, nil), 400)

	// rename, order, list in the project and in the sidebar
	expect(t, "umbenennen", member.do("PATCH", fmt.Sprintf("/api/tables/%d", tb.ID), map[string]any{"title": "Kalkulation HW"}, nil), 200)
	expect(t, "reihenfolge", member.do("PUT", fmt.Sprintf("/api/projects/%d/tables/order", p.ID), map[string]any{"ids": []int64{imp.ID, tb.ID}}, nil), 204)
	expect(t, "reihenfolge fremd", member.do("PUT", fmt.Sprintf("/api/projects/%d/tables/order", p.ID), map[string]any{"ids": []int64{imp.ID, 9999}}, nil), 400)
	var detail struct {
		Tables []struct {
			ID     int64  `json:"id"`
			Title  string `json:"title"`
			Sheets int64  `json:"sheets"`
		} `json:"tables"`
	}
	member.do("GET", fmt.Sprintf("/api/projects/%d", p.ID), nil, &detail)
	if len(detail.Tables) != 2 || detail.Tables[0].ID != imp.ID || detail.Tables[0].Sheets != 2 || detail.Tables[1].Title != "Kalkulation HW" {
		t.Fatalf("projekt tabellen: %+v", detail.Tables)
	}
	var sb struct {
		Customers []struct {
			Projects []struct {
				Tables []struct {
					ID int64 `json:"id"`
				} `json:"tables"`
			} `json:"projects"`
		} `json:"customers"`
	}
	stranger.do("GET", "/api/sidebar", nil, &sb)
	for _, cu := range sb.Customers {
		for _, pr := range cu.Projects {
			if len(pr.Tables) != 0 {
				t.Fatal("sidebar: fremde tabellen sichtbar")
			}
		}
	}

	// duplicate a project: the tables are copied
	var dup idResp
	expect(t, "duplizieren", member.do("POST", fmt.Sprintf("/api/projects/%d/duplicate", p.ID), nil, &dup), 201)
	member.do("GET", fmt.Sprintf("/api/projects/%d", dup.ID), nil, &detail)
	if len(detail.Tables) != 2 {
		t.Fatalf("kopierte tabellen: %+v", detail.Tables)
	}

	// delete, activity
	expect(t, "viewer löschen", viewer.do("DELETE", fmt.Sprintf("/api/tables/%d", tb.ID), nil, nil), 403)
	expect(t, "löschen", member.do("DELETE", fmt.Sprintf("/api/tables/%d", tb.ID), nil, nil), 204)
	expect(t, "gelöscht", member.do("GET", fmt.Sprintf("/api/tables/%d", tb.ID), nil, nil), 404)
	var act []struct {
		Kind    string `json:"kind"`
		Subject string `json:"subject"`
	}
	member.do("GET", fmt.Sprintf("/api/projects/%d/activity", p.ID), nil, &act)
	kinds := map[string]bool{}
	for _, a := range act {
		kinds[a.Kind+":"+a.Subject] = true
	}
	if !kinds["table_created:Kalkulation"] || !kinds["table_deleted:Kalkulation HW"] {
		t.Fatalf("aktivität: %+v", act)
	}
}

// Table templates: save a table as template, apply it (variables replaced in the
// cells), edit the template workbook, and keep the kinds apart.
func TestTableTemplates(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, p idResp
	admin.do("POST", "/api/customers", map[string]any{"name": "Spital"}, &c)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]any{"name": "Migration"}, &p)
	expect(t, "variable", admin.do("POST", "/api/template-variables", map[string]any{"name": "server.hostname", "default_value": "srv01"}, nil), 201)

	wb := []map[string]any{{"name": "Kalkulation", "celldata": []map[string]any{
		{"r": 0, "c": 0, "v": map[string]any{"v": "Kunde {{customer.name}}", "m": "Kunde {{customer.name}}"}},
		{"r": 1, "c": 0, "v": map[string]any{"v": `Host "{{server.hostname}}"`}},
	}}}
	var tb tableResp
	admin.do("POST", fmt.Sprintf("/api/projects/%d/tables", p.ID), map[string]any{"title": "Kalkulation", "data": wb}, &tb)

	var tpl struct {
		ID   int64  `json:"id"`
		Kind string `json:"kind"`
	}
	expect(t, "als vorlage", admin.do("POST", fmt.Sprintf("/api/tables/%d/save-as-template", tb.ID), map[string]any{"name": "Kalkulation {{project.name}}"}, &tpl), 201)
	if tpl.Kind != "table" {
		t.Fatalf("vorlage: %+v", tpl)
	}
	var list []struct {
		ID   int64  `json:"id"`
		Kind string `json:"kind"`
	}
	admin.do("GET", "/api/templates", nil, &list)
	if len(list) != 1 || list[0].Kind != "table" {
		t.Fatalf("bibliothek: %+v", list)
	}
	// the variables of the cells are found (apply dialog)
	var vars []struct {
		Name string `json:"name"`
	}
	admin.do("GET", fmt.Sprintf("/api/templates/%d/variables", tpl.ID), nil, &vars)
	names := []string{}
	for _, v := range vars {
		names = append(names, v.Name)
	}
	if strings.Join(names, ",") != "customer.name,project.name,server.hostname" {
		t.Fatalf("variablen: %v", names)
	}

	// apply: variables replaced (JSON stays valid with quotes in the value)
	var applied tableResp
	expect(t, "anwenden", admin.do("POST", fmt.Sprintf("/api/projects/%d/tables/from-template", p.ID),
		map[string]any{"template_id": tpl.ID, "values": map[string]string{"server.hostname": `db "01"`}}, &applied), 201)
	if applied.Title != "Kalkulation Migration" {
		t.Fatalf("titel: %q", applied.Title)
	}
	var full struct {
		Table tableResp `json:"table"`
	}
	admin.do("GET", fmt.Sprintf("/api/tables/%d", applied.ID), nil, &full)
	var sheets []struct {
		Celldata []struct {
			V struct {
				V string `json:"v"`
			} `json:"v"`
		} `json:"celldata"`
	}
	if err := json.Unmarshal(full.Table.Data, &sheets); err != nil {
		t.Fatalf("daten ungültig: %v %s", err, full.Table.Data)
	}
	if sheets[0].Celldata[0].V.V != "Kunde Spital" || sheets[0].Celldata[1].V.V != `Host "db "01""` {
		t.Fatalf("ersetzt: %+v", sheets)
	}

	// edit the workbook of the template; the applied table stays as it is (snapshot)
	expect(t, "vorlage blatt", admin.do("PUT", fmt.Sprintf("/api/templates/%d/sheet", tpl.ID), map[string]any{"data": []map[string]any{{"name": "Neu"}}}, nil), 204)
	var tget struct {
		Template struct {
			SheetData string `json:"sheet_data"`
		} `json:"template"`
	}
	admin.do("GET", fmt.Sprintf("/api/templates/%d", tpl.ID), nil, &tget)
	if !strings.Contains(tget.Template.SheetData, `"Neu"`) {
		t.Fatalf("vorlage gespeichert: %s", tget.Template.SheetData)
	}

	// new empty table template from the library
	var empty struct {
		ID        int64  `json:"id"`
		Kind      string `json:"kind"`
		SheetData string `json:"sheet_data"`
	}
	expect(t, "leere tabellenvorlage", admin.do("POST", "/api/templates", map[string]any{"name": "Leer", "kind": "table"}, &empty), 201)
	if empty.Kind != "table" || !strings.Contains(empty.SheetData, "Sheet1") {
		t.Fatalf("leere vorlage: %+v", empty)
	}
	expect(t, "falsche art", admin.do("POST", "/api/templates", map[string]any{"name": "X", "kind": "slides"}, nil), 400)

	// kinds stay apart
	var cl idResp
	admin.do("POST", "/api/templates", map[string]any{"name": "Checkliste"}, &cl)
	expect(t, "checkliste aus tabellenvorlage", admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists/from-template", p.ID), map[string]any{"template_id": tpl.ID}, nil), 400)
	expect(t, "tabelle aus checklistenvorlage", admin.do("POST", fmt.Sprintf("/api/projects/%d/tables/from-template", p.ID), map[string]any{"template_id": cl.ID}, nil), 400)
	expect(t, "eintrag in tabellenvorlage", admin.do("POST", fmt.Sprintf("/api/templates/%d/items", tpl.ID), map[string]any{"title": "x"}, nil), 400)
	expect(t, "blatt in checklistenvorlage", admin.do("PUT", fmt.Sprintf("/api/templates/%d/sheet", cl.ID), map[string]any{"data": wb}, nil), 400)
	expect(t, "automation mit tabellenvorlage", admin.do("POST", "/api/automations", map[string]any{"trigger_kind": "project_created", "template_id": tpl.ID}, nil), 400)

	// the template counts as used
	var lib []struct {
		ID        int64 `json:"id"`
		UsedCount int64 `json:"used_count"`
	}
	admin.do("GET", "/api/templates", nil, &lib)
	for _, l := range lib {
		if l.ID == tpl.ID && l.UsedCount != 1 {
			t.Fatalf("benutzt: %+v", l)
		}
	}
}
