package httpapi_test

import (
	"fmt"
	"testing"
)

type favRow struct {
	Kind     string `json:"kind"`
	ID       int64  `json:"id"`
	GroupID  *int64 `json:"group_id"`
	Position int64  `json:"position"`
}

type favGroup struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Position  int64  `json:"position"`
	Collapsed bool   `json:"collapsed"`
}

type favSide struct {
	Favorites      []favRow   `json:"favorites"`
	FavoriteGroups []favGroup `json:"favorite_groups"`
}

func (c *client) favSidebar() favSide {
	c.e.t.Helper()
	var s favSide
	if st := c.do("GET", "/api/sidebar", nil, &s); st != 200 {
		c.e.t.Fatalf("sidebar: %d", st)
	}
	return s
}

// layout returns "kind:id" per group (key 0 = without group) in display order.
func (s favSide) layout() map[int64][]string {
	out := map[int64][]string{}
	for _, f := range s.Favorites { // already ordered by position
		var g int64
		if f.GroupID != nil {
			g = *f.GroupID
		}
		out[g] = append(out[g], fmt.Sprintf("%s:%d", f.Kind, f.ID))
	}
	return out
}

func TestFavoriteGroups(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var a, p1, p2, tpl idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &a)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]string{"name": "P1"}, &p1)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]string{"name": "P2"}, &p2)
	expect(t, "template", admin.do("POST", "/api/templates", map[string]any{"name": "T"}, &tpl), 201)
	member := e.user("member@example.com", "member", a.ID)

	// empty: favorites [] and favorite_groups [] (not null)
	s := admin.favSidebar()
	if s.Favorites == nil || s.FavoriteGroups == nil || len(s.Favorites)+len(s.FavoriteGroups) != 0 {
		t.Fatalf("leere listen erwartet: %+v", s)
	}

	url := func(kind string, id int64) string { return fmt.Sprintf("/api/favorites/%s/%d", kind, id) }
	for _, u := range []string{url("project", p1.ID), url("project", p2.ID), url("customer", a.ID), url("template", tpl.ID)} {
		expect(t, "favorit "+u, admin.do("PUT", u, nil, nil), 204)
	}
	expect(t, "template gibt es nicht", admin.do("PUT", url("template", 9999), nil, nil), 404)
	expect(t, "member: template-favorit", member.do("PUT", url("template", tpl.ID), nil, nil), 204)
	s = admin.favSidebar()
	if got := fmt.Sprint(s.layout()[0]); got != fmt.Sprintf("[project:%d project:%d customer:%d template:%d]", p1.ID, p2.ID, a.ID, tpl.ID) {
		t.Fatalf("reihenfolge neuer favoriten: %s", got)
	}
	for _, f := range s.Favorites {
		if f.GroupID != nil {
			t.Fatalf("neue favoriten ohne gruppe: %+v", f)
		}
	}

	// groups: create, rename, collapse; foreign user 404
	var g1, g2 favGroup
	expect(t, "gruppe 1", admin.do("POST", "/api/favorite-groups", map[string]string{"name": "  Kunden  "}, &g1), 201)
	expect(t, "gruppe 2", admin.do("POST", "/api/favorite-groups", map[string]string{"name": "Projekte"}, &g2), 201)
	if g1.Name != "Kunden" || g1.Position != 0 || g2.Position != 1 || g1.Collapsed {
		t.Fatalf("gruppen: %+v %+v", g1, g2)
	}
	expect(t, "leerer name", admin.do("POST", "/api/favorite-groups", map[string]string{"name": " "}, nil), 400)
	var upd favGroup
	expect(t, "einklappen", admin.do("PATCH", fmt.Sprintf("/api/favorite-groups/%d", g1.ID), map[string]any{"collapsed": true}, &upd), 200)
	if !upd.Collapsed || upd.Name != "Kunden" {
		t.Fatalf("patch: %+v", upd)
	}
	expect(t, "umbenennen", admin.do("PATCH", fmt.Sprintf("/api/favorite-groups/%d", g1.ID), map[string]any{"name": "Wichtig"}, &upd), 200)
	if !upd.Collapsed || upd.Name != "Wichtig" {
		t.Fatalf("collapsed bleibt beim umbenennen: %+v", upd)
	}
	expect(t, "fremde gruppe patch", member.do("PATCH", fmt.Sprintf("/api/favorite-groups/%d", g1.ID), map[string]any{"name": "x"}, nil), 404)
	expect(t, "fremde gruppe delete", member.do("DELETE", fmt.Sprintf("/api/favorite-groups/%d", g1.ID), nil, nil), 404)
	s = admin.favSidebar()
	if len(s.FavoriteGroups) != 2 || !s.FavoriteGroups[0].Collapsed || s.FavoriteGroups[0].Name != "Wichtig" {
		t.Fatalf("gruppen in sidebar: %+v", s.FavoriteGroups)
	}
	if len(member.favSidebar().FavoriteGroups) != 0 {
		t.Fatalf("gruppen sind pro benutzer")
	}

	// order: groups swapped, items moved between groups, foreign items ignored
	type item struct {
		Kind    string `json:"kind"`
		ID      int64  `json:"id"`
		GroupID *int64 `json:"group_id"`
	}
	gid := func(id int64) *int64 { return &id }
	order := map[string]any{
		"groups": []int64{g2.ID, g1.ID},
		"items": []item{
			{"template", tpl.ID, nil},
			{"project", p2.ID, gid(g2.ID)},
			{"project", p1.ID, gid(g2.ID)},
			{"customer", a.ID, gid(g1.ID)},
			{"project", 424242, nil}, // not a favorite: ignored
		},
	}
	expect(t, "reihenfolge", admin.do("PUT", "/api/favorites/order", order, nil), 204)
	s = admin.favSidebar()
	l := s.layout()
	if fmt.Sprint(l[0]) != fmt.Sprintf("[template:%d]", tpl.ID) ||
		fmt.Sprint(l[g2.ID]) != fmt.Sprintf("[project:%d project:%d]", p2.ID, p1.ID) ||
		fmt.Sprint(l[g1.ID]) != fmt.Sprintf("[customer:%d]", a.ID) {
		t.Fatalf("layout nach reihenfolge: %v", l)
	}
	if s.FavoriteGroups[0].ID != g2.ID || s.FavoriteGroups[1].ID != g1.ID {
		t.Fatalf("gruppenreihenfolge: %+v", s.FavoriteGroups)
	}
	// foreign or unknown group → 400, nothing changes
	var mg favGroup
	member.do("POST", "/api/favorite-groups", map[string]string{"name": "M"}, &mg)
	expect(t, "fremde gruppe in groups", admin.do("PUT", "/api/favorites/order", map[string]any{"groups": []int64{mg.ID}, "items": []item{}}, nil), 400)
	expect(t, "fremde gruppe in items", admin.do("PUT", "/api/favorites/order", map[string]any{"groups": []int64{}, "items": []item{{"template", tpl.ID, gid(mg.ID)}}}, nil), 400)
	if fmt.Sprint(admin.favSidebar().layout()) != fmt.Sprint(l) {
		t.Fatalf("ungültige reihenfolge hat etwas verändert")
	}

	// delete a group: its favorites go to the end of the list without group (order kept)
	expect(t, "gruppe löschen", admin.do("DELETE", fmt.Sprintf("/api/favorite-groups/%d", g2.ID), nil, nil), 204)
	s = admin.favSidebar()
	if got := fmt.Sprint(s.layout()[0]); got != fmt.Sprintf("[template:%d project:%d project:%d]", tpl.ID, p2.ID, p1.ID) {
		t.Fatalf("nach gruppe löschen: %s", got)
	}
	if len(s.FavoriteGroups) != 1 {
		t.Fatalf("gruppe nicht gelöscht: %+v", s.FavoriteGroups)
	}
	// a new favorite comes after them
	var cl idResp
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p1.ID), map[string]string{"title": "CL"}, &cl)
	admin.do("PUT", url("checklist", cl.ID), nil, nil)
	if l0 := admin.favSidebar().layout()[0]; l0[len(l0)-1] != fmt.Sprintf("checklist:%d", cl.ID) {
		t.Fatalf("neuer favorit am ende: %v", l0)
	}

	// deleting the template removes it from all favorites (trigger)
	expect(t, "template löschen", admin.do("DELETE", fmt.Sprintf("/api/templates/%d", tpl.ID), nil, nil), 204)
	var n int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM user_favorites WHERE kind = 'template'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("template-favoriten übrig: %d (%v)", n, err)
	}
	// deleting a project (trigger still works after the table rebuild)
	expect(t, "projekt löschen", admin.do("DELETE", fmt.Sprintf("/api/projects/%d", p2.ID), nil, nil), 204)
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM user_favorites WHERE kind = 'project' AND target_id = ?`, p2.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("projekt-favorit übrig: %d (%v)", n, err)
	}
}
