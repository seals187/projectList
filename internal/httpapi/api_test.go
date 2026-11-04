package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/httpapi"
	"github.com/seals187/projectList/internal/testutil"
)

// ---- Test infrastructure -------------------------------------------------------

type env struct {
	t   *testing.T
	srv *httptest.Server
	q   *sqlc.Queries
}

func newEnv(t *testing.T) *env {
	d, q := testutil.DB(t)
	h := httpapi.New(httpapi.Options{
		DB:       d,
		DBPath:   t.TempDir() + "/test.db",
		Sessions: &auth.Sessions{Q: q, TTL: time.Hour},
		Logger:   testutil.Logger(),
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, q: q}
}

type client struct {
	e    *env
	http *http.Client
	csrf string
}

func (e *env) anon() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, http: &http.Client{Jar: jar}}
}

// user creates a user and logs them in.
func (e *env) user(email, role string, customerIDs ...int64) *client {
	e.t.Helper()
	u := testutil.CreateUser(e.t, e.q, email, "passwort-123", role)
	for _, cid := range customerIDs {
		if err := e.q.AddMembership(context.Background(), sqlc.AddMembershipParams{UserID: u.ID, CustomerID: cid}); err != nil {
			e.t.Fatal(err)
		}
	}
	c := e.anon()
	var res struct {
		CSRFToken string `json:"csrf_token"`
	}
	if st := c.do("POST", "/api/auth/login", map[string]string{"email": email, "password": "passwort-123"}, &res); st != 200 {
		e.t.Fatalf("login %s: status %d", email, st)
	}
	c.csrf = res.CSRFToken
	return c
}

func (c *client) do(method, path string, body any, out any) int {
	c.e.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			c.e.t.Fatalf("%s %s: ungültiges JSON %q", method, path, data)
		}
	}
	return resp.StatusCode
}

func (c *client) raw(method, path string) string {
	req, _ := http.NewRequest(method, c.e.srv.URL+path, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

type idResp struct {
	ID int64 `json:"id"`
}

func expect(t *testing.T, what string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: status %d, erwartet %d", what, got, want)
	}
}

// ---- Auth --------------------------------------------------------------------

func TestLoginLogoutAndMe(t *testing.T) {
	e := newEnv(t)
	testutil.CreateUser(t, e.q, "alice@example.com", "passwort-123", "member")
	c := e.anon()

	expect(t, "me ohne Login", c.do("GET", "/api/auth/me", nil, nil), 401)
	expect(t, "falsches Passwort", c.do("POST", "/api/auth/login", map[string]string{"email": "alice@example.com", "password": "nope"}, nil), 401)
	expect(t, "unbekannte E-Mail", c.do("POST", "/api/auth/login", map[string]string{"email": "x@example.com", "password": "nope"}, nil), 401)

	var res struct {
		User      map[string]any `json:"user"`
		CSRFToken string         `json:"csrf_token"`
	}
	// Email is case-insensitive
	expect(t, "login", c.do("POST", "/api/auth/login", map[string]string{"email": "ALICE@example.com", "password": "passwort-123"}, &res), 200)
	if res.CSRFToken == "" || res.User["email"] != "alice@example.com" {
		t.Fatalf("login-antwort unvollständig: %+v", res)
	}
	if _, leaked := res.User["password_hash"]; leaked {
		t.Fatal("password_hash in Antwort!")
	}
	expect(t, "me", c.do("GET", "/api/auth/me", nil, nil), 200)

	// Logout without CSRF token is rejected
	expect(t, "logout ohne csrf", c.do("POST", "/api/auth/logout", nil, nil), 403)
	c.csrf = res.CSRFToken
	expect(t, "logout", c.do("POST", "/api/auth/logout", nil, nil), 204)
	expect(t, "me nach logout", c.do("GET", "/api/auth/me", nil, nil), 401)
}

func TestLoginRateLimit(t *testing.T) {
	e := newEnv(t)
	testutil.CreateUser(t, e.q, "bob@example.com", "passwort-123", "member")
	c := e.anon()
	for i := range 10 {
		expect(t, fmt.Sprintf("fehlversuch %d", i), c.do("POST", "/api/auth/login", map[string]string{"email": "bob@example.com", "password": "falsch"}, nil), 401)
	}
	expect(t, "gesperrt", c.do("POST", "/api/auth/login", map[string]string{"email": "bob@example.com", "password": "passwort-123"}, nil), 429)
}

func TestCSRFAndOrigin(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")

	token := admin.csrf
	admin.csrf = "falsch"
	expect(t, "falscher csrf", admin.do("POST", "/api/customers", map[string]string{"name": "X"}, nil), 403)
	admin.csrf = token
	expect(t, "korrekter csrf", admin.do("POST", "/api/customers", map[string]string{"name": "X"}, nil), 201)

	// Foreign origin
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/customers", strings.NewReader(`{"name":"Y"}`))
	req.Header.Set("X-CSRF-Token", token)
	req.Header.Set("Origin", "https://evil.example")
	resp, err := admin.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	expect(t, "fremder origin", resp.StatusCode, 403)
}

// ---- Permissions -------------------------------------------------------------

func TestPermissions(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")

	var c1, c2 idResp
	expect(t, "kunde 1", admin.do("POST", "/api/customers", map[string]string{"name": "Kunde Eins"}, &c1), 201)
	expect(t, "kunde 2", admin.do("POST", "/api/customers", map[string]string{"name": "Kunde Zwei"}, &c2), 201)

	member := e.user("member@example.com", "member", c1.ID)
	viewer := e.user("viewer@example.com", "viewer", c1.ID)

	// Visibility
	var list []idResp
	expect(t, "member kunden", member.do("GET", "/api/customers", nil, &list), 200)
	if len(list) != 1 || list[0].ID != c1.ID {
		t.Fatalf("member sieht falsche Kunden: %+v", list)
	}
	expect(t, "admin kunden", admin.do("GET", "/api/customers", nil, &list), 200)
	if len(list) != 2 {
		t.Fatalf("admin sollte alle Kunden sehen: %+v", list)
	}
	expect(t, "fremder kunde", member.do("GET", fmt.Sprintf("/api/customers/%d", c2.ID), nil, nil), 404)

	// Admin endpoints
	expect(t, "member legt kunden an", member.do("POST", "/api/customers", map[string]string{"name": "X"}, nil), 403)
	expect(t, "member löscht kunden", member.do("DELETE", fmt.Sprintf("/api/customers/%d", c1.ID), nil, nil), 403)
	expect(t, "member listet benutzer", member.do("GET", "/api/users", nil, nil), 403)
	expect(t, "member backup", member.do("GET", "/api/admin/backup", nil, nil), 403)

	// Member: project/checklist/item in own customer
	var p1, cl, item idResp
	expect(t, "member projekt", member.do("POST", fmt.Sprintf("/api/customers/%d/projects", c1.ID), map[string]string{"name": "P1"}, &p1), 201)
	expect(t, "member projekt fremd", member.do("POST", fmt.Sprintf("/api/customers/%d/projects", c2.ID), map[string]string{"name": "P"}, nil), 404)
	expect(t, "checkliste", member.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p1.ID), map[string]string{"title": "CL"}, &cl), 201)
	expect(t, "item", member.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "Aufgabe"}, &item), 201)

	// Checking off with done_by/done_at
	var done struct {
		DoneAt     *string `json:"done_at"`
		DoneByName *string `json:"done_by_name"`
	}
	expect(t, "abhaken", member.do("POST", fmt.Sprintf("/api/items/%d/done", item.ID), map[string]bool{"done": true}, &done), 200)
	if done.DoneAt == nil || done.DoneByName == nil || *done.DoneByName != "member@example.com" {
		t.Fatalf("done_at/done_by nicht gesetzt: %+v", done)
	}

	// Viewer: read yes, write no
	expect(t, "viewer liest projekt", viewer.do("GET", fmt.Sprintf("/api/projects/%d", p1.ID), nil, nil), 200)
	expect(t, "viewer hakt ab", viewer.do("POST", fmt.Sprintf("/api/items/%d/done", item.ID), map[string]bool{"done": false}, nil), 403)
	expect(t, "viewer projekt", viewer.do("POST", fmt.Sprintf("/api/customers/%d/projects", c1.ID), map[string]string{"name": "X"}, nil), 403)
	expect(t, "viewer template", viewer.do("POST", "/api/templates", map[string]string{"name": "X"}, nil), 403)

	// Admin project in customer 2: invisible to the member
	var p2, cl2, item2 idResp
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c2.ID), map[string]string{"name": "Geheim"}, &p2)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p2.ID), map[string]string{"title": "CL"}, &cl2)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl2.ID), map[string]string{"title": "X"}, &item2)
	expect(t, "fremdes projekt", member.do("GET", fmt.Sprintf("/api/projects/%d", p2.ID), nil, nil), 404)
	expect(t, "fremde checkliste", member.do("PATCH", fmt.Sprintf("/api/checklists/%d", cl2.ID), map[string]string{"title": "x"}, nil), 404)
	expect(t, "fremdes item", member.do("POST", fmt.Sprintf("/api/items/%d/done", item2.ID), map[string]bool{"done": true}, nil), 404)
	expect(t, "fremdes item löschen", member.do("DELETE", fmt.Sprintf("/api/items/%d", item2.ID), nil, nil), 404)
	expect(t, "fremdes template anwenden", member.do("POST", fmt.Sprintf("/api/projects/%d/checklists/from-template", p2.ID), map[string]int64{"template_id": 1}, nil), 404)

	// Foreign items must not be sorted into the own checklist
	expect(t, "sortieren mit fremder id", member.do("PUT", fmt.Sprintf("/api/checklists/%d/items/order", cl.ID), map[string][]int64{"ids": {item.ID, item2.ID}}, nil), 400)

	// Assignee without access to the customer is rejected
	other := testutil.CreateUser(t, e.q, "other@example.com", "passwort-123", "member")
	expect(t, "assignee ohne zugriff", member.do("PATCH", fmt.Sprintf("/api/items/%d", item.ID), map[string]int64{"assignee_id": other.ID}, nil), 400)

	// Search & sidebar respect visibility
	var sr struct {
		Projects []idResp `json:"projects"`
	}
	expect(t, "suche", member.do("GET", "/api/search?q=geheim", nil, &sr), 200)
	if len(sr.Projects) != 0 {
		t.Fatal("Suche liefert Projekt eines fremden Kunden")
	}
	expect(t, "suche admin", admin.do("GET", "/api/search?q=GEHEIM", nil, &sr), 200)
	if len(sr.Projects) != 1 {
		t.Fatal("Admin findet Projekt nicht")
	}
	var sbResp struct {
		Customers []struct {
			ID       int64 `json:"id"`
			Projects []struct {
				Total      int64 `json:"total"`
				Done       int64 `json:"done"`
				Checklists []struct {
					ID int64 `json:"id"`
				} `json:"checklists"`
			} `json:"projects"`
		} `json:"customers"`
	}
	expect(t, "sidebar", member.do("GET", "/api/sidebar", nil, &sbResp), 200)
	sb := sbResp.Customers
	if len(sb) != 1 || len(sb[0].Projects) != 1 || sb[0].Projects[0].Total != 1 || sb[0].Projects[0].Done != 1 ||
		len(sb[0].Projects[0].Checklists) != 1 {
		t.Fatalf("sidebar falsch: %+v", sb)
	}

	// Remove assignment → access gone immediately
	var users []struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	admin.do("GET", "/api/users", nil, &users)
	var memberID int64
	for _, u := range users {
		if u.Email == "member@example.com" {
			memberID = u.ID
		}
	}
	expect(t, "zuordnung entfernen", admin.do("PUT", fmt.Sprintf("/api/users/%d/customers", memberID), map[string][]int64{"customer_ids": {}}, nil), 204)
	expect(t, "nach entzug", member.do("GET", fmt.Sprintf("/api/projects/%d", p1.ID), nil, nil), 404)
}

func TestUserAdminSafeguards(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	admin.do("GET", "/api/auth/me", nil, &me)

	expect(t, "selbst herabstufen", admin.do("PATCH", fmt.Sprintf("/api/users/%d", me.User.ID), map[string]string{"role": "member"}, nil), 400)
	expect(t, "selbst löschen", admin.do("DELETE", fmt.Sprintf("/api/users/%d", me.User.ID), nil, nil), 400)

	var created struct {
		ID           int64  `json:"id"`
		PasswordHash string `json:"password_hash"`
	}
	body := map[string]any{"email": "neu@example.com", "name": "Neu", "role": "viewer", "password": "passwort-123"}
	expect(t, "benutzer anlegen", admin.do("POST", "/api/users", body, &created), 201)
	if created.PasswordHash != "" {
		t.Fatal("password_hash in Antwort!")
	}
	expect(t, "doppelte e-mail", admin.do("POST", "/api/users", body, nil), 409)
	expect(t, "ungültige rolle", admin.do("POST", "/api/users", map[string]any{"email": "x@example.com", "name": "X", "role": "root", "password": "passwort-123"}, nil), 400)
	expect(t, "kurzes passwort", admin.do("POST", "/api/users", map[string]any{"email": "y@example.com", "name": "Y", "role": "member", "password": "kurz"}, nil), 400)
	expect(t, "löschen", admin.do("DELETE", fmt.Sprintf("/api/users/%d", created.ID), nil, nil), 204)
}

func TestValidation(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	expect(t, "leerer name", admin.do("POST", "/api/customers", map[string]string{"name": "   "}, nil), 400)
	expect(t, "unbekanntes feld", admin.do("POST", "/api/customers", map[string]string{"name": "A", "foo": "b"}, nil), 400)
	var c, p, cl, it idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &c)
	expect(t, "ungültiger status", admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P", "status": "kaputt"}, nil), 400)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "I"}, &it)
	expect(t, "ungültiges datum", admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]string{"due_date": "31.12.2026"}, nil), 400)
	expect(t, "gültiges datum", admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]string{"due_date": "2026-12-31"}, nil), 200)
	expect(t, "datum entfernen", admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]any{"due_date": nil}, nil), 200)
	expect(t, "ungültige id", admin.do("GET", "/api/projects/abc", nil, nil), 404)
	expect(t, "unbekannter endpunkt", admin.do("GET", "/api/gibtsnicht", nil, nil), 404)
}

func TestTemplateFlowViaAPI(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, p, tmpl idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &c)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P"}, &p)
	expect(t, "template", admin.do("POST", "/api/templates", map[string]any{
		"name":  "Go-Live",
		"items": []map[string]string{{"title": "DNS"}, {"title": "TLS"}},
	}, &tmpl), 201)

	var cl struct {
		ID         int64 `json:"id"`
		TemplateID int64 `json:"template_id"`
		Total      int   `json:"total"`
	}
	expect(t, "aus template", admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists/from-template", p.ID), map[string]any{"template_id": tmpl.ID}, &cl), 201)
	if cl.Total != 2 || cl.TemplateID != tmpl.ID {
		t.Fatalf("checkliste aus template falsch: %+v", cl)
	}

	var saved idResp
	expect(t, "als template speichern", admin.do("POST", fmt.Sprintf("/api/checklists/%d/save-as-template", cl.ID), map[string]string{"name": "Kopie"}, &saved), 201)
	var detail struct {
		Items []struct {
			Title string `json:"title"`
		} `json:"items"`
	}
	admin.do("GET", fmt.Sprintf("/api/templates/%d", saved.ID), nil, &detail)
	if len(detail.Items) != 2 || detail.Items[0].Title != "DNS" {
		t.Fatalf("gespeichertes template falsch: %+v", detail)
	}

	// Project detail: progress via aggregation
	var pd struct {
		Progress struct{ Total, Done int } `json:"progress"`
	}
	admin.do("GET", fmt.Sprintf("/api/projects/%d", p.ID), nil, &pd)
	if pd.Progress.Total != 2 || pd.Progress.Done != 0 {
		t.Fatalf("fortschritt falsch: %+v", pd)
	}
}

func TestHealthz(t *testing.T) {
	e := newEnv(t)
	if body := e.anon().raw("GET", "/healthz"); !strings.Contains(body, `"ok"`) {
		t.Fatalf("healthz: %s", body)
	}
}

func TestHeadingsNotesAndInsertPosition(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, p, cl, h, a, b, mid idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &c)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	items := fmt.Sprintf("/api/checklists/%d/items", cl.ID)

	expect(t, "heading", admin.do("POST", items, map[string]string{"kind": "h1", "title": "Hardware"}, &h), 201)
	admin.do("POST", items, map[string]string{"title": "A", "description": "Note A", "note_color": "green"}, &a)
	admin.do("POST", items, map[string]string{"title": "B"}, &b)
	expect(t, "ungültige kind", admin.do("POST", items, map[string]string{"kind": "x", "title": "X"}, nil), 400)
	expect(t, "ungültige farbe", admin.do("POST", items, map[string]string{"title": "X", "note_color": "pink"}, nil), 400)

	// Insert between A and B (position 2)
	expect(t, "einfügen", admin.do("POST", items, map[string]any{"title": "Mitte", "position": 2}, &mid), 201)

	// Heading cannot be checked off
	expect(t, "heading abhaken", admin.do("POST", fmt.Sprintf("/api/items/%d/done", h.ID), map[string]bool{"done": true}, nil), 400)
	admin.do("POST", fmt.Sprintf("/api/items/%d/done", a.ID), map[string]bool{"done": true}, nil)

	var detail struct {
		Checklist struct {
			Total int `json:"total"`
			Done  int `json:"done"`
			Items []struct {
				ID        int64  `json:"id"`
				Kind      string `json:"kind"`
				NoteColor string `json:"note_color"`
			} `json:"items"`
		} `json:"checklist"`
		Project struct {
			ID int64 `json:"id"`
		} `json:"project"`
	}
	expect(t, "checkliste", admin.do("GET", fmt.Sprintf("/api/checklists/%d", cl.ID), nil, &detail), 200)
	got := []int64{}
	for _, it := range detail.Checklist.Items {
		got = append(got, it.ID)
	}
	if fmt.Sprint(got) != fmt.Sprint([]int64{h.ID, a.ID, mid.ID, b.ID}) {
		t.Fatalf("reihenfolge falsch: %v", got)
	}
	if detail.Checklist.Total != 3 || detail.Checklist.Done != 1 || detail.Project.ID != p.ID {
		t.Fatalf("fortschritt darf überschriften nicht zählen: %+v", detail.Checklist)
	}
	if detail.Checklist.Items[1].NoteColor != "green" || detail.Checklist.Items[0].Kind != "h1" {
		t.Fatalf("kind/note_color falsch: %+v", detail.Checklist.Items)
	}

	// Project and sidebar aggregation do not count headings either
	var pd struct {
		Progress   struct{ Total, Done int } `json:"progress"`
		Checklists []struct {
			Total   int    `json:"total"`
			DueDate string `json:"due_date"`
		} `json:"checklists"`
	}
	admin.do("PATCH", fmt.Sprintf("/api/items/%d", b.ID), map[string]string{"due_date": "2026-12-01"}, nil)
	admin.do("GET", fmt.Sprintf("/api/projects/%d", p.ID), nil, &pd)
	if pd.Progress.Total != 3 || pd.Progress.Done != 1 || pd.Checklists[0].DueDate != "2026-12-01" {
		t.Fatalf("projekt-aggregation falsch: %+v", pd)
	}

	// Snapshot carries over headings and notes
	var tmpl idResp
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/save-as-template", cl.ID), map[string]string{"name": "T"}, &tmpl)
	var cl2 struct {
		Total int `json:"total"`
		Items []struct {
			Kind        string `json:"kind"`
			Description string `json:"description"`
		} `json:"items"`
	}
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists/from-template", p.ID), map[string]int64{"template_id": tmpl.ID}, &cl2)
	if cl2.Total != 3 || len(cl2.Items) != 4 || cl2.Items[0].Kind != "h1" || cl2.Items[1].Description != "Note A" {
		t.Fatalf("snapshot mit überschriften falsch: %+v", cl2)
	}
}

func TestMyTasks(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, p, cl, i1, i2 idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &c)
	member := e.user("member@example.com", "member", c.ID)
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	member.do("GET", "/api/auth/me", nil, &me)

	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	items := fmt.Sprintf("/api/checklists/%d/items", cl.ID)
	admin.do("POST", items, map[string]any{"title": "Später", "assignee_id": me.User.ID, "due_date": "2026-12-24"}, &i1)
	admin.do("POST", items, map[string]any{"title": "Bald", "assignee_id": me.User.ID, "due_date": "2026-11-05"}, &i2)
	admin.do("POST", items, map[string]any{"title": "Ohne Zuständigen"}, nil)

	var tasks []struct {
		ID           int64  `json:"id"`
		CustomerName string `json:"customer_name"`
	}
	expect(t, "my tasks", member.do("GET", "/api/my-tasks", nil, &tasks), 200)
	if len(tasks) != 2 || tasks[0].ID != i2.ID || tasks[0].CustomerName != "A" {
		t.Fatalf("my tasks falsch (sortiert nach fälligkeit erwartet): %+v", tasks)
	}
	member.do("POST", fmt.Sprintf("/api/items/%d/done", i2.ID), map[string]bool{"done": true}, nil)
	var sb struct {
		MyTasks int `json:"my_tasks"`
	}
	member.do("GET", "/api/sidebar", nil, &sb)
	if sb.MyTasks != 1 {
		t.Fatalf("my_tasks-zähler: %d, erwartet 1", sb.MyTasks)
	}
}

func TestErrorCodes(t *testing.T) {
	e := newEnv(t)
	c := e.anon()
	var res struct {
		Error string `json:"error"`
	}
	c.do("POST", "/api/auth/login", map[string]string{"email": "x@example.com", "password": "y"}, &res)
	if res.Error != "invalid_credentials" {
		t.Fatalf("fehlercode: %q", res.Error)
	}
	admin := e.user("admin@example.com", "admin")
	var v struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	admin.do("POST", "/api/customers", map[string]string{"name": ""}, &v)
	if v.Error != "validation" || v.Fields["name"] != "required" {
		t.Fatalf("validierungscodes: %+v", v)
	}
}

func TestBlocksRichTextAndKinds(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, p, cl, it, code idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &c)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	items := fmt.Sprintf("/api/checklists/%d/items", cl.ID)

	// HTML is sanitized (whitelist)
	var got struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Kind        string `json:"kind"`
		Indent      int    `json:"indent"`
		Lang        string `json:"lang"`
	}
	expect(t, "rich", admin.do("POST", items, map[string]any{
		"title":       `<b>fett</b><script>alert(1)</script><span data-color="red" onclick="x">rot</span>`,
		"description": `<a href="javascript:alert(1)">böse</a> <a href="https://ok.example">gut</a>`,
	}, &got), 201)
	if got.Title != `<b>fett</b><span data-color="red">rot</span>` {
		t.Fatalf("titel nicht bereinigt: %q", got.Title)
	}
	if strings.Contains(got.Description, "javascript") || !strings.Contains(got.Description, `href="https://ok.example"`) {
		t.Fatalf("notiz nicht bereinigt: %q", got.Description)
	}

	// Code block stores plain text unchanged
	src := "New-ADUser -Name \"Laura\" `\n  -Enabled $true <tag>"
	expect(t, "code", admin.do("POST", items, map[string]any{"kind": "code", "lang": "powershell", "title": src, "indent": 1}, &code), 201)
	admin.do("GET", fmt.Sprintf("/api/checklists/%d", cl.ID), nil, nil)
	var detail struct {
		Checklist struct {
			Total int `json:"total"`
			Items []struct {
				Title  string `json:"title"`
				Kind   string `json:"kind"`
				Lang   string `json:"lang"`
				Indent int    `json:"indent"`
			} `json:"items"`
		} `json:"checklist"`
	}
	admin.do("GET", fmt.Sprintf("/api/checklists/%d", cl.ID), nil, &detail)
	if len(detail.Checklist.Items) != 2 || detail.Checklist.Items[1].Title != src || detail.Checklist.Items[1].Lang != "powershell" || detail.Checklist.Items[1].Indent != 1 {
		t.Fatalf("codeblock falsch: %+v", detail.Checklist.Items)
	}
	if detail.Checklist.Total != 1 {
		t.Fatalf("nur Checklisten-Einträge zählen: total=%d", detail.Checklist.Total)
	}

	// Change type (entry → heading), reject invalid values
	admin.do("POST", items, map[string]string{"title": "x"}, &it)
	expect(t, "typ wechseln", admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]any{"kind": "h2"}, &got), 200)
	if got.Kind != "h2" {
		t.Fatalf("kind: %q", got.Kind)
	}
	expect(t, "überschrift abhaken", admin.do("POST", fmt.Sprintf("/api/items/%d/done", it.ID), map[string]bool{"done": true}, nil), 400)
	expect(t, "ungültige sprache", admin.do("PATCH", fmt.Sprintf("/api/items/%d", code.ID), map[string]string{"lang": "cobol"}, nil), 400)
	expect(t, "ungültige einrückung", admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]int{"indent": 9}, nil), 400)
	expect(t, "leerer titel erlaubt", admin.do("POST", items, map[string]string{"kind": "text", "title": ""}, nil), 201)

	// Section color: store valid values, reject unknown ones, template carries them over
	var colored struct {
		BgColor string `json:"bg_color"`
	}
	expect(t, "abschnittsfarbe", admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]string{"bg_color": "blue"}, &colored), 200)
	if colored.BgColor != "blue" {
		t.Fatalf("bg_color: %q", colored.BgColor)
	}
	expect(t, "ungültige abschnittsfarbe", admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]string{"bg_color": "pink"}, nil), 400)
	var tpl idResp
	expect(t, "als template", admin.do("POST", fmt.Sprintf("/api/checklists/%d/save-as-template", cl.ID), map[string]string{"name": "T"}, &tpl), 201)
	var tplDetail struct {
		Items []struct {
			Kind    string `json:"kind"`
			BgColor string `json:"bg_color"`
		} `json:"items"`
	}
	admin.do("GET", fmt.Sprintf("/api/templates/%d", tpl.ID), nil, &tplDetail)
	found := false
	for _, ti := range tplDetail.Items {
		found = found || (ti.Kind == "h2" && ti.BgColor == "blue")
	}
	if !found {
		t.Fatalf("template ohne abschnittsfarbe: %+v", tplDetail.Items)
	}
}

func TestChecklistAssignee(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var a, b, p, cl idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &a)
	admin.do("POST", "/api/customers", map[string]string{"name": "B"}, &b)
	member := e.user("member@example.com", "member", a.ID)
	other := e.user("other@example.com", "member", b.ID)
	var me, otherMe struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	member.do("GET", "/api/auth/me", nil, &me)
	other.do("GET", "/api/auth/me", nil, &otherMe)

	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "x"}, nil)
	url := fmt.Sprintf("/api/checklists/%d", cl.ID)

	// not assignable without customer access
	expect(t, "fremder benutzer", admin.do("PATCH", url, map[string]any{"assignee_id": otherMe.User.ID}, nil), 400)

	var res struct {
		Title        string  `json:"title"`
		AssigneeID   *int64  `json:"assignee_id"`
		AssigneeName *string `json:"assignee_name"`
	}
	expect(t, "zuweisen", member.do("PATCH", url, map[string]any{"assignee_id": me.User.ID}, &res), 200)
	if res.AssigneeID == nil || *res.AssigneeID != me.User.ID || res.AssigneeName == nil || res.Title != "CL" {
		t.Fatalf("zuweisung falsch (titel muss bleiben): %+v", res)
	}

	var mine []struct {
		ID           int64  `json:"id"`
		CustomerName string `json:"customer_name"`
		Total        int    `json:"total"`
	}
	expect(t, "meine checklisten", member.do("GET", "/api/my-checklists", nil, &mine), 200)
	if len(mine) != 1 || mine[0].ID != cl.ID || mine[0].CustomerName != "A" || mine[0].Total != 1 {
		t.Fatalf("my-checklists falsch: %+v", mine)
	}
	var detail struct {
		Checklist struct {
			AssigneeName *string `json:"assignee_name"`
		} `json:"checklist"`
	}
	admin.do("GET", url, nil, &detail)
	if detail.Checklist.AssigneeName == nil {
		t.Fatal("assignee_name fehlt in der Checkliste")
	}

	// Changing the title keeps the assignment; null removes it
	member.do("PATCH", url, map[string]any{"title": "Neu"}, &res)
	if res.AssigneeID == nil || res.Title != "Neu" {
		t.Fatalf("titel-update hat zuweisung verändert: %+v", res)
	}
	expect(t, "aufheben", member.do("PATCH", url, map[string]any{"assignee_id": nil}, &res), 200)
	if res.AssigneeID != nil {
		t.Fatalf("zuweisung nicht aufgehoben: %+v", res)
	}
	member.do("GET", "/api/my-checklists", nil, &mine)
	if len(mine) != 0 {
		t.Fatalf("my-checklists nach aufheben: %+v", mine)
	}
}

func TestItemStatus(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var c, p, cl, it idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &c)
	viewer := e.user("viewer@example.com", "viewer", c.ID)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "x"}, &it)
	url := fmt.Sprintf("/api/items/%d", it.ID)

	var got struct {
		Status string `json:"status"`
		Title  string `json:"title"`
	}
	expect(t, "status setzen", admin.do("PATCH", url, map[string]string{"status": "  Erfüllt <b>ok</b> "}, &got), 200)
	if got.Status != "Erfüllt <b>ok</b>" || got.Title != "x" {
		t.Fatalf("status falsch (klartext, getrimmt): %+v", got)
	}
	expect(t, "zu lang", admin.do("PATCH", url, map[string]string{"status": strings.Repeat("x", 61)}, nil), 400)
	expect(t, "zeilenumbruch", admin.do("PATCH", url, map[string]string{"status": "a\nb"}, nil), 400)
	expect(t, "viewer darf nicht", viewer.do("PATCH", url, map[string]string{"status": "x"}, nil), 403)
	admin.do("PATCH", url, map[string]string{"title": "y"}, &got)
	if got.Status != "Erfüllt <b>ok</b>" {
		t.Fatalf("titel-update hat status verändert: %+v", got)
	}
	expect(t, "status leeren", admin.do("PATCH", url, map[string]any{"status": nil}, &got), 200)
	if got.Status != "" {
		t.Fatalf("status nicht geleert: %+v", got)
	}

	// Row color "none" (explicitly no background, even in a colored section)
	var bg struct {
		BgColor string `json:"bg_color"`
	}
	expect(t, "ohne farbe", admin.do("PATCH", url, map[string]string{"bg_color": "none"}, &bg), 200)
	if bg.BgColor != "none" {
		t.Fatalf("bg_color none nicht gespeichert: %+v", bg)
	}
	// strong shades (like Word/Excel); the server rejects unknown values
	expect(t, "kräftige farbe", admin.do("PATCH", url, map[string]string{"bg_color": "yellow-strong"}, &bg), 200)
	if bg.BgColor != "yellow-strong" {
		t.Fatalf("bg_color yellow-strong: %+v", bg)
	}
	expect(t, "unbekannte farbe", admin.do("PATCH", url, map[string]string{"bg_color": "pink-strong"}, nil), 400)
}

func TestProjectStatusDueAndOverview(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var a, b, p, q, cl idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &a)
	admin.do("POST", "/api/customers", map[string]string{"name": "B"}, &b)
	member := e.user("member@example.com", "member", a.ID)
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	member.do("GET", "/api/auth/me", nil, &me)

	type proj struct {
		Status  string  `json:"status"`
		DueDate *string `json:"due_date"`
	}
	var pr proj
	expect(t, "projekt mit status+datum", admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID),
		map[string]any{"name": "P", "status": "planning", "due_date": "2026-11-20"}, &p), 201)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", b.ID), map[string]any{"name": "Q"}, &q)
	url := fmt.Sprintf("/api/projects/%d", p.ID)

	for _, s := range []string{"active", "review", "blocked", "done", "archived"} {
		expect(t, "status "+s, admin.do("PATCH", url, map[string]any{"status": s}, &pr), 200)
		if pr.Status != s || pr.DueDate == nil || *pr.DueDate != "2026-11-20" {
			t.Fatalf("status %s: datum muss bleiben: %+v", s, pr)
		}
	}
	expect(t, "ungültiger status", admin.do("PATCH", url, map[string]any{"status": "paused"}, nil), 400)
	expect(t, "ungültiges datum", admin.do("PATCH", url, map[string]any{"due_date": "20.11.2026"}, nil), 400)
	expect(t, "datum löschen", admin.do("PATCH", url, map[string]any{"due_date": nil}, &pr), 200)
	if pr.DueDate != nil {
		t.Fatalf("datum nicht gelöscht: %+v", pr)
	}

	// Team and next step
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	var first idResp
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "erster"}, &first)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]any{"title": "zweiter", "assignee_id": me.User.ID}, nil)
	admin.do("POST", fmt.Sprintf("/api/items/%d/done", first.ID), map[string]bool{"done": true}, nil)

	var ov map[string]struct {
		Team []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		NextStep string `json:"next_step"`
	}
	expect(t, "overview", member.do("GET", "/api/overview", nil, &ov), 200)
	got, ok := ov[fmt.Sprint(p.ID)]
	if !ok || got.NextStep != "zweiter" || len(got.Team) != 1 || got.Team[0].ID != me.User.ID {
		t.Fatalf("overview falsch: %+v", ov)
	}
	if _, leaked := ov[fmt.Sprint(q.ID)]; leaked {
		t.Fatalf("fremdes projekt sichtbar: %+v", ov)
	}
}

func TestFavoritesAndRecent(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var a, b, p, q, cl idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &a)
	admin.do("POST", "/api/customers", map[string]string{"name": "B"}, &b)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", b.ID), map[string]string{"name": "Q"}, &q)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	member := e.user("member@example.com", "member", a.ID)

	type side struct {
		Favorites []int64 `json:"favorites"`
		Recent    []struct {
			Kind string `json:"kind"`
			ID   int64  `json:"id"`
		} `json:"recent"`
	}
	var s side
	member.do("GET", "/api/sidebar", nil, &s)
	if s.Favorites == nil || s.Recent == nil || len(s.Favorites) != 0 || len(s.Recent) != 0 {
		t.Fatalf("anfangs leere listen erwartet (nicht null): %+v", s)
	}

	fav := fmt.Sprintf("/api/favorites/%d", p.ID)
	expect(t, "favorit", member.do("PUT", fav, nil, nil), 204)
	expect(t, "favorit doppelt", member.do("PUT", fav, nil, nil), 204)
	expect(t, "fremdes projekt", member.do("PUT", fmt.Sprintf("/api/favorites/%d", q.ID), nil, nil), 404)

	expect(t, "besuch kunde", member.do("POST", "/api/recent", map[string]any{"kind": "customer", "id": a.ID}, nil), 204)
	expect(t, "besuch checkliste", member.do("POST", "/api/recent", map[string]any{"kind": "checklist", "id": cl.ID}, nil), 204)
	expect(t, "besuch kunde erneut", member.do("POST", "/api/recent", map[string]any{"kind": "customer", "id": a.ID}, nil), 204)
	expect(t, "fremder kunde", member.do("POST", "/api/recent", map[string]any{"kind": "customer", "id": b.ID}, nil), 404)
	expect(t, "falsche art", member.do("POST", "/api/recent", map[string]any{"kind": "user", "id": 1}, nil), 400)

	member.do("GET", "/api/sidebar", nil, &s)
	if len(s.Favorites) != 1 || s.Favorites[0] != p.ID {
		t.Fatalf("favoriten falsch: %+v", s.Favorites)
	}
	if len(s.Recent) != 2 || s.Recent[0].Kind != "customer" || s.Recent[1].Kind != "checklist" {
		t.Fatalf("zuletzt falsch (neuester zuerst, ohne duplikate): %+v", s.Recent)
	}
	// Admin sees their own (empty) lists
	var as side
	admin.do("GET", "/api/sidebar", nil, &as)
	if len(as.Favorites) != 0 || len(as.Recent) != 0 {
		t.Fatalf("listen sind pro benutzer: %+v", as)
	}

	// limited to 20
	for i := 0; i < 25; i++ {
		var c idResp
		admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": fmt.Sprint("X", i)}, &c)
		member.do("POST", "/api/recent", map[string]any{"kind": "checklist", "id": c.ID}, nil)
	}
	member.do("GET", "/api/sidebar", nil, &s)
	if len(s.Recent) != 20 {
		t.Fatalf("zuletzt nicht auf 20 gekürzt: %d", len(s.Recent))
	}

	var tpl idResp
	admin.do("POST", "/api/templates", map[string]string{"name": "Vorlage"}, &tpl)
	expect(t, "besuch template", member.do("POST", "/api/recent", map[string]any{"kind": "template", "id": tpl.ID}, nil), 204)
	var named struct {
		Recent []struct {
			Kind   string `json:"kind"`
			Title  string `json:"title"`
			Parent string `json:"parent"`
		} `json:"recent"`
	}
	member.do("GET", "/api/sidebar", nil, &named)
	if named.Recent[0].Kind != "template" || named.Recent[0].Title != "Vorlage" || named.Recent[1].Title != "X24" || named.Recent[1].Parent != "P" {
		t.Fatalf("aufgelöste titel falsch: %+v", named.Recent[:2])
	}
	expect(t, "zuletzt leeren", member.do("DELETE", "/api/recent", nil, nil), 204)
	member.do("GET", "/api/sidebar", nil, &s)
	if len(s.Recent) != 0 {
		t.Fatalf("nicht geleert: %+v", s.Recent)
	}

	expect(t, "favorit entfernen", member.do("DELETE", fav, nil, nil), 204)
	// deleted project disappears from favorites
	admin.do("PUT", fav, nil, nil)
	admin.do("DELETE", fmt.Sprintf("/api/projects/%d", p.ID), nil, nil)
	admin.do("GET", "/api/sidebar", nil, &as)
	member.do("GET", "/api/sidebar", nil, &s)
	if len(as.Favorites) != 0 || len(s.Favorites) != 0 {
		t.Fatalf("favoriten nach löschen: %+v / %+v", as.Favorites, s.Favorites)
	}
}

func TestCustomerContactAndActivity(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	type cust struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		Location     string `json:"location"`
		ContactName  string `json:"contact_name"`
		ContactEmail string `json:"contact_email"`
		LastActivity string `json:"last_activity"`
	}
	var c cust
	expect(t, "anlegen", admin.do("POST", "/api/customers", map[string]string{
		"name": "(KA) Kunde-A", "location": " Musterstadt ", "contact_name": "Erika Muster", "contact_email": "Erika.Muster@Kunde-A.example",
	}, &c), 201)
	if c.Location != "Musterstadt" || c.ContactName != "Erika Muster" || c.ContactEmail != "erika.muster@kunde-a.example" {
		t.Fatalf("felder falsch: %+v", c)
	}
	url := fmt.Sprintf("/api/customers/%d", c.ID)
	expect(t, "ungültige mail", admin.do("PATCH", url, map[string]string{"contact_email": "kein-mail"}, nil), 400)
	expect(t, "mail leeren", admin.do("PATCH", url, map[string]string{"contact_email": ""}, &c), 200)
	if c.ContactEmail != "" || c.Location != "Musterstadt" {
		t.Fatalf("patch darf andere felder nicht ändern: %+v", c)
	}

	var list []cust
	admin.do("GET", "/api/customers", nil, &list)
	if len(list) != 1 || list[0].LastActivity == "" || list[0].Location != "Musterstadt" {
		t.Fatalf("liste: %+v", list)
	}
	before := list[0].LastActivity
	var p, cl, it idResp
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]string{"title": "x"}, &it)
	admin.do("POST", fmt.Sprintf("/api/items/%d/done", it.ID), map[string]bool{"done": true}, nil)
	admin.do("GET", "/api/customers", nil, &list)
	if list[0].LastActivity < before {
		t.Fatalf("letzte aktivität darf nicht zurückgehen: %q < %q", list[0].LastActivity, before)
	}
}

func TestProjectProgressFlagAndMyDoneTasks(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	var a, p, cl, it idResp
	admin.do("POST", "/api/customers", map[string]string{"name": "A"}, &a)
	member := e.user("member@example.com", "member", a.ID)
	viewer := e.user("viewer@example.com", "viewer", a.ID)
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	member.do("GET", "/api/auth/me", nil, &me)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]string{"name": "P"}, &p)
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]string{"title": "CL"}, &cl)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]any{"title": "x", "assignee_id": me.User.ID, "due_date": "2026-11-20"}, &it)
	admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]any{"title": "y", "due_date": "2026-11-10"}, nil)

	type side struct {
		Customers []struct {
			Projects []struct {
				InProgress bool `json:"include_in_customer_progress"`
				Checklists []struct {
					NextDue string `json:"next_due"`
				} `json:"checklists"`
			} `json:"projects"`
		} `json:"customers"`
	}
	var s side
	member.do("GET", "/api/sidebar", nil, &s)
	pr := s.Customers[0].Projects[0]
	if !pr.InProgress || pr.Checklists[0].NextDue != "2026-11-10" {
		t.Fatalf("standard/nächste fälligkeit falsch: %+v", pr)
	}

	url := fmt.Sprintf("/api/projects/%d", p.ID)
	expect(t, "viewer darf nicht", viewer.do("PATCH", url, map[string]any{"include_in_customer_progress": false}, nil), 403)
	var got struct {
		Name       string `json:"name"`
		InProgress bool   `json:"include_in_customer_progress"`
	}
	expect(t, "member schaltet aus", member.do("PATCH", url, map[string]any{"include_in_customer_progress": false}, &got), 200)
	if got.InProgress || got.Name != "P" {
		t.Fatalf("flag nicht gesetzt: %+v", got)
	}
	expect(t, "anderes feld ändern", member.do("PATCH", url, map[string]any{"status": "review"}, &got), 200)
	if got.InProgress {
		t.Fatalf("flag darf durch anderes patch nicht zurückspringen")
	}

	var tasks []struct {
		ID int64 `json:"id"`
	}
	member.do("GET", "/api/my-tasks?done=1", nil, &tasks)
	if len(tasks) != 0 {
		t.Fatalf("noch nichts erledigt: %+v", tasks)
	}
	member.do("POST", fmt.Sprintf("/api/items/%d/done", it.ID), map[string]bool{"done": true}, nil)
	member.do("GET", "/api/my-tasks?done=1", nil, &tasks)
	if len(tasks) != 1 || tasks[0].ID != it.ID {
		t.Fatalf("erledigte aufgaben falsch: %+v", tasks)
	}
	member.do("GET", "/api/my-tasks", nil, &tasks)
	if len(tasks) != 0 {
		t.Fatalf("offene aufgaben falsch: %+v", tasks)
	}
}
