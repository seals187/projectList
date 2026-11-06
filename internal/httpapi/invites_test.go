package httpapi_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/httpapi"
	"github.com/seals187/projectList/internal/mail"
	"github.com/seals187/projectList/internal/testutil"
)

// fakeMailer captures e-mails instead of sending them.
type fakeMailer struct {
	mu   sync.Mutex
	sent []mail.Message
	fail bool
}

func (f *fakeMailer) Enabled() bool { return true }
func (f *fakeMailer) Send(_ context.Context, m mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return fmt.Errorf("smtp down")
	}
	f.sent = append(f.sent, m)
	return nil
}
func (f *fakeMailer) last() mail.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sent[len(f.sent)-1]
}

// clock is a server clock that tests can move forward.
type clock struct {
	mu  sync.Mutex
	add time.Duration
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Add(c.add)
}
func (c *clock) advance(d time.Duration) { c.mu.Lock(); c.add += d; c.mu.Unlock() }

func newEnvWith(t *testing.T, mod func(*httpapi.Options)) *env {
	d, q := testutil.DB(t)
	o := httpapi.Options{
		DB:       d,
		DBPath:   t.TempDir() + "/test.db",
		Sessions: &auth.Sessions{Q: q, TTL: time.Hour},
		Logger:   testutil.Logger(),
	}
	mod(&o)
	srv := httptest.NewServer(httpapi.New(o))
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, q: q}
}

type tokenUser struct {
	ID         int64      `json:"id"`
	Invited    bool       `json:"invited"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	Role       string     `json:"role"`
}

func tokenOf(t *testing.T, link string) string {
	t.Helper()
	_, tok, ok := strings.Cut(link, "/invite?token=")
	if !ok || tok == "" {
		t.Fatalf("link without token: %q", link)
	}
	return tok
}

func findUser(t *testing.T, admin *client, id int64) tokenUser {
	t.Helper()
	var us []tokenUser
	expect(t, "users", admin.do("GET", "/api/users", nil, &us), 200)
	for _, u := range us {
		if u.ID == id {
			return u
		}
	}
	t.Fatalf("user %d not found", id)
	return tokenUser{}
}

func TestInviteFlow(t *testing.T) {
	mailer := &fakeMailer{}
	clk := &clock{}
	e := newEnvWith(t, func(o *httpapi.Options) {
		o.Mailer = mailer
		o.Now = clk.now
		o.AppURL = "https://pl.example.com"
	})
	admin := e.user("admin@example.com", "admin")
	member := e.user("member@example.com", "member")

	body := map[string]any{"name": "Markus S.", "email": "markus@example.com", "role": "viewer", "customer_ids": []int64{}, "lang": "de"}
	expect(t, "invite member", member.do("POST", "/api/users/invite", body, nil), 403)
	var inv struct {
		User    tokenUser `json:"user"`
		Link    string    `json:"link"`
		Emailed bool      `json:"emailed"`
	}
	expect(t, "invite", admin.do("POST", "/api/users/invite", body, &inv), 201)
	if !inv.Emailed || !inv.User.Invited || inv.User.Role != "viewer" || !strings.HasPrefix(inv.Link, "https://pl.example.com/invite?token=") {
		t.Fatalf("invite response: %+v", inv)
	}
	m := mailer.last()
	if m.To != "markus@example.com" || !strings.Contains(m.Body, inv.Link) || !strings.Contains(m.Body, "admin@example.com") || m.Subject != "Einladung zu ProjectList" {
		t.Fatalf("mail: %+v", m)
	}
	expect(t, "invite twice", admin.do("POST", "/api/users/invite", body, nil), 409)
	expect(t, "invite invalid", admin.do("POST", "/api/users/invite", map[string]any{"name": "", "email": "x", "role": "viewer"}, nil), 400)

	// not possible to sign in before accepting
	anon := e.anon()
	expect(t, "login invited", anon.do("POST", "/api/auth/login", map[string]string{"email": "markus@example.com", "password": "passwort-123"}, nil), 401)

	tok := tokenOf(t, inv.Link)
	var info map[string]string
	expect(t, "token info", anon.do("GET", "/api/auth/token/"+tok, nil, &info), 200)
	if info["email"] != "markus@example.com" || info["name"] != "Markus S." || info["purpose"] != "invite" {
		t.Fatalf("token info: %v", info)
	}
	expect(t, "unknown token", anon.do("GET", "/api/auth/token/nope", nil, nil), 404)
	expect(t, "short password", anon.do("POST", "/api/auth/token", map[string]string{"token": tok, "password": "kurz"}, nil), 400)

	var sess struct {
		CSRFToken string `json:"csrf_token"`
		User      struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	expect(t, "accept", anon.do("POST", "/api/auth/token", map[string]string{"token": tok, "password": "neues-passwort-1"}, &sess), 200)
	if sess.CSRFToken == "" || sess.User.Email != "markus@example.com" {
		t.Fatalf("accept response: %+v", sess)
	}
	expect(t, "me after accept", anon.do("GET", "/api/auth/me", nil, nil), 200)
	expect(t, "single use", e.anon().do("POST", "/api/auth/token", map[string]string{"token": tok, "password": "neues-passwort-2"}, nil), 404)
	expect(t, "token gone", e.anon().do("GET", "/api/auth/token/"+tok, nil, nil), 404)
	expect(t, "login with new password", e.anon().do("POST", "/api/auth/login", map[string]string{"email": "markus@example.com", "password": "neues-passwort-1"}, nil), 200)

	u := findUser(t, admin, inv.User.ID)
	if u.Invited || u.LastSeenAt == nil {
		t.Fatalf("after accept: %+v", u)
	}

	// expired invitation: 7 days
	expect(t, "invite 2", admin.do("POST", "/api/users/invite", map[string]any{"name": "Old", "email": "old@example.com", "role": "member", "lang": "en"}, &inv), 201)
	tok2 := tokenOf(t, inv.Link)
	clk.advance(7*24*time.Hour + time.Minute)
	expect(t, "expired token info", e.anon().do("GET", "/api/auth/token/"+tok2, nil, nil), 404)
	expect(t, "expired accept", e.anon().do("POST", "/api/auth/token", map[string]string{"token": tok2, "password": "neues-passwort-1"}, nil), 404)

	// a new link for the invited user is again an invitation
	var link struct {
		Link    string `json:"link"`
		Emailed bool   `json:"emailed"`
		Purpose string `json:"purpose"`
	}
	expect(t, "new invite link", admin.do("POST", fmt.Sprintf("/api/users/%d/reset-link", inv.User.ID), map[string]string{"lang": "en"}, &link), 200)
	if link.Purpose != "invite" || !link.Emailed || mailer.last().Subject != "Invitation to ProjectList" {
		t.Fatalf("new invite link: %+v", link)
	}
}

func TestResetLinkEndsSessions(t *testing.T) {
	mailer := &fakeMailer{fail: true} // sending fails: the admin still gets the link
	e := newEnvWith(t, func(o *httpapi.Options) { o.Mailer = mailer })
	admin := e.user("admin@example.com", "admin")
	member := e.user("member@example.com", "member")
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	member.do("GET", "/api/auth/me", nil, &me)

	u := findUser(t, admin, me.User.ID)
	if u.LastSeenAt == nil || u.Invited {
		t.Fatalf("last_seen_at after login: %+v", u)
	}

	var link struct {
		Link    string `json:"link"`
		Emailed bool   `json:"emailed"`
		Purpose string `json:"purpose"`
	}
	expect(t, "reset member", member.do("POST", fmt.Sprintf("/api/users/%d/reset-link", me.User.ID), map[string]string{"lang": "fr"}, nil), 403)
	expect(t, "reset link", admin.do("POST", fmt.Sprintf("/api/users/%d/reset-link", me.User.ID), map[string]string{"lang": "fr"}, &link), 200)
	if link.Purpose != "reset" || link.Emailed || !strings.Contains(link.Link, "/invite?token=") {
		t.Fatalf("reset link: %+v", link)
	}
	// a second link replaces the first
	first := tokenOf(t, link.Link)
	expect(t, "reset link 2", admin.do("POST", fmt.Sprintf("/api/users/%d/reset-link", me.User.ID), map[string]string{}, &link), 200)
	expect(t, "first link invalid", e.anon().do("GET", "/api/auth/token/"+first, nil, nil), 404)

	expect(t, "old session still valid", member.do("GET", "/api/auth/me", nil, nil), 200)
	expect(t, "accept reset", e.anon().do("POST", "/api/auth/token", map[string]string{"token": tokenOf(t, link.Link), "password": "anderes-pw-123"}, nil), 200)
	expect(t, "old session ended", member.do("GET", "/api/auth/me", nil, nil), 401)
	expect(t, "old password", e.anon().do("POST", "/api/auth/login", map[string]string{"email": "member@example.com", "password": "passwort-123"}, nil), 401)
	expect(t, "new password", e.anon().do("POST", "/api/auth/login", map[string]string{"email": "member@example.com", "password": "anderes-pw-123"}, nil), 200)
}

func TestAppSettingsMailAndRedmineTest(t *testing.T) {
	e := newEnv(t) // no mailer configured
	admin := e.user("admin@example.com", "admin")
	member := e.user("member@example.com", "member")

	type settings struct {
		RedmineURL       string `json:"redmine_url"`
		MailEnabled      bool   `json:"mail_enabled"`
		RedmineCheckedAt string `json:"redmine_checked_at"`
		RedmineOK        bool   `json:"redmine_ok"`
	}
	var s settings
	expect(t, "settings", member.do("GET", "/api/settings/app", nil, &s), 200)
	if s.MailEnabled || s.RedmineURL != "" || s.RedmineCheckedAt != "" {
		t.Fatalf("settings: %+v", s)
	}

	// invitation without e-mail: link only
	var inv struct {
		Link    string `json:"link"`
		Emailed bool   `json:"emailed"`
	}
	expect(t, "invite without mail", admin.do("POST", "/api/users/invite", map[string]any{"name": "N", "email": "n@example.com", "role": "member"}, &inv), 201)
	if inv.Emailed || !strings.HasPrefix(inv.Link, e.srv.URL+"/invite?token=") {
		t.Fatalf("invite without mail: %+v", inv)
	}

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/down" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusUnauthorized) // login required still means reachable
	}))
	defer up.Close()

	expect(t, "save url", admin.do("PUT", "/api/settings/app", map[string]string{"redmine_url": up.URL + "/"}, &s), 200)
	if s.RedmineURL != up.URL || s.RedmineCheckedAt != "" {
		t.Fatalf("saved: %+v", s)
	}
	type testRes struct {
		OK        bool   `json:"ok"`
		Status    int    `json:"status"`
		Error     string `json:"error"`
		CheckedAt string `json:"checked_at"`
	}
	var tr testRes
	expect(t, "test member", member.do("POST", "/api/settings/redmine/test", map[string]string{}, nil), 403)
	expect(t, "test stored", admin.do("POST", "/api/settings/redmine/test", map[string]string{}, &tr), 200)
	if !tr.OK || tr.Status != 401 || tr.CheckedAt == "" {
		t.Fatalf("test stored: %+v", tr)
	}
	expect(t, "settings after test", member.do("GET", "/api/settings/app", nil, &s), 200)
	if !s.RedmineOK || s.RedmineCheckedAt != tr.CheckedAt {
		t.Fatalf("result not stored: %+v", s)
	}

	// another URL: tested but not stored
	expect(t, "test other", admin.do("POST", "/api/settings/redmine/test", map[string]string{"redmine_url": up.URL + "/down"}, &tr), 200)
	if tr.OK || tr.Status != 502 {
		t.Fatalf("test 502: %+v", tr)
	}
	member.do("GET", "/api/settings/app", nil, &s)
	if !s.RedmineOK {
		t.Fatalf("other URL changed the stored result: %+v", s)
	}
	expect(t, "test unreachable", admin.do("POST", "/api/settings/redmine/test", map[string]string{"redmine_url": "http://127.0.0.1:1"}, &tr), 200)
	if tr.OK || tr.Status != 0 || tr.Error == "" {
		t.Fatalf("unreachable: %+v", tr)
	}
	expect(t, "test invalid", admin.do("POST", "/api/settings/redmine/test", map[string]string{"redmine_url": "ftp://x"}, nil), 400)

	// changing the URL clears the result
	expect(t, "change url", admin.do("PUT", "/api/settings/app", map[string]string{"redmine_url": "https://other.example.com"}, &s), 200)
	if s.RedmineOK || s.RedmineCheckedAt != "" {
		t.Fatalf("result not cleared: %+v", s)
	}

	// appearance settings are synced via the "ui" pref
	expect(t, "ui pref", member.do("PUT", "/api/prefs/ui", map[string]any{"accent": "teal", "density": "compact", "glass": false}, nil), 204)
	var pr struct {
		Prefs map[string]any `json:"prefs"`
	}
	member.do("GET", "/api/sidebar", nil, &pr)
	if ui, ok := pr.Prefs["ui"].(map[string]any); !ok || ui["accent"] != "teal" {
		t.Fatalf("ui pref: %+v", pr.Prefs)
	}
	// order and folders of the sidebar favorites
	layout := []any{map[string]any{"t": "f", "id": "x", "name": "Kunden", "open": true, "items": []string{"customer:1"}}}
	expect(t, "fav_layout pref", member.do("PUT", "/api/prefs/fav_layout", layout, nil), 204)
	member.do("GET", "/api/sidebar", nil, &pr)
	if fl, ok := pr.Prefs["fav_layout"].([]any); !ok || len(fl) != 1 {
		t.Fatalf("fav_layout pref: %+v", pr.Prefs)
	}
	expect(t, "views_layout pref", member.do("PUT", "/api/prefs/views_layout", layout, nil), 204)
}
