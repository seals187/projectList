package httpapi_test

import (
	"fmt"
	"testing"
	"time"
)

func TestArchiveAndSearchScope(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")

	var a, b idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "Kunde A"}, &a)
	admin.do("POST", "/api/customers", map[string]string{"name": "Kunde B"}, &b)
	var live, old, other idResp
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]string{"name": "Migration live"}, &live)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]string{"name": "Migration alt"}, &old)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", b.ID), map[string]string{"name": "Migration B"}, &other)
	var cl, item idResp
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", old.ID), map[string]string{"title": "Migration Cutover"}, &cl)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "x"}, &item)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "y"}, nil)
	admin.do("POST", fmt.Sprintf("/api/items/%d/done", item.ID), map[string]bool{"done": true}, nil)

	for _, id := range []int64{old.ID, other.ID} {
		expect(t, "archivieren", admin.do("PATCH", fmt.Sprintf("/api/projects/%d", id), map[string]string{"status": "archived"}, nil), 200)
	}

	type row struct {
		ID             int64      `json:"id"`
		CustomerName   string     `json:"customer_name"`
		ArchivedAt     *time.Time `json:"archived_at"`
		ArchivedBy     *int64     `json:"archived_by"`
		ArchivedByName string     `json:"archived_by_name"`
		Total          int64      `json:"total"`
		Done           int64      `json:"done"`
	}
	var all []row
	expect(t, "archiv", admin.do("GET", "/api/archive", nil, &all), 200)
	if len(all) != 2 {
		t.Fatalf("archiv: %+v", all)
	}
	for _, r := range all {
		if r.ArchivedAt == nil || r.ArchivedBy == nil || r.ArchivedByName == "" {
			t.Fatalf("archiviert am/von fehlt: %+v", r)
		}
	}
	var mine []row
	expect(t, "archiv kunde", admin.do("GET", fmt.Sprintf("/api/archive?customer_id=%d", a.ID), nil, &mine), 200)
	if len(mine) != 1 || mine[0].ID != old.ID || mine[0].Total != 2 || mine[0].Done != 1 || mine[0].CustomerName != "Kunde A" {
		t.Fatalf("archiv kunde: %+v", mine)
	}
	expect(t, "archiv kunde ungültig", admin.do("GET", "/api/archive?customer_id=x", nil, nil), 400)

	// restoring removes it from the archive
	admin.do("PATCH", fmt.Sprintf("/api/projects/%d", other.ID), map[string]string{"status": "active"}, nil)
	expect(t, "archiv", admin.do("GET", "/api/archive", nil, &all), 200)
	if len(all) != 1 {
		t.Fatalf("wiederhergestellt noch im archiv: %+v", all)
	}

	// members only see the archive of their customers
	member := e.user("member@example.com", "member", b.ID)
	expect(t, "archiv member", member.do("GET", "/api/archive", nil, &all), 200)
	if len(all) != 0 {
		t.Fatalf("member sieht fremdes archiv: %+v", all)
	}
	expect(t, "archiv fremder kunde", member.do("GET", fmt.Sprintf("/api/archive?customer_id=%d", a.ID), nil, nil), 404)

	// search scope
	type sres struct {
		Projects []struct {
			ID         int64      `json:"id"`
			Status     string     `json:"status"`
			ArchivedAt *time.Time `json:"archived_at"`
		} `json:"projects"`
		Checklists []struct {
			ID            int64  `json:"id"`
			ProjectStatus string `json:"project_status"`
		} `json:"checklists"`
	}
	var sr sres
	expect(t, "suche alle", admin.do("GET", "/api/search?q=migration", nil, &sr), 200)
	if len(sr.Projects) != 3 || sr.Projects[len(sr.Projects)-1].ID != old.ID || sr.Projects[len(sr.Projects)-1].ArchivedAt == nil {
		t.Fatalf("suche alle (archivierte zuletzt, mit datum): %+v", sr.Projects)
	}
	if len(sr.Checklists) != 1 || sr.Checklists[0].ProjectStatus != "archived" {
		t.Fatalf("suche checklisten: %+v", sr.Checklists)
	}
	sr = sres{}
	expect(t, "suche aktiv", admin.do("GET", "/api/search?q=migration&scope=active", nil, &sr), 200)
	if len(sr.Projects) != 2 || len(sr.Checklists) != 0 {
		t.Fatalf("suche aktiv: %+v", sr)
	}
	sr = sres{}
	expect(t, "suche archiviert", admin.do("GET", "/api/search?q=migration&scope=archived", nil, &sr), 200)
	if len(sr.Projects) != 1 || sr.Projects[0].ID != old.ID || len(sr.Checklists) != 1 {
		t.Fatalf("suche archiviert: %+v", sr)
	}
	expect(t, "suche scope ungültig", admin.do("GET", "/api/search?q=x&scope=foo", nil, nil), 400)
}
