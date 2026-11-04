package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/testutil"
)

func TestHashAndVerifyPassword(t *testing.T) {
	h, err := auth.HashPassword("geheim-123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unerwartetes Hash-Format: %s", h)
	}
	if ok, err := auth.VerifyPassword("geheim-123", h); err != nil || !ok {
		t.Fatalf("korrektes Passwort abgelehnt: ok=%v err=%v", ok, err)
	}
	if ok, _ := auth.VerifyPassword("falsch", h); ok {
		t.Fatal("falsches Passwort akzeptiert")
	}
	h2, _ := auth.HashPassword("geheim-123")
	if h == h2 {
		t.Fatal("Salt wird nicht verwendet (identische Hashes)")
	}
	if _, err := auth.VerifyPassword("x", "kaputt"); err == nil {
		t.Fatal("ungültiger Hash sollte Fehler liefern")
	}
}

func TestSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	_, q := testutil.DB(t)
	u := testutil.CreateUser(t, q, "a@example.com", "passwort1", "member")

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := &auth.Sessions{Q: q, TTL: time.Hour, Now: func() time.Time { return now }}

	rec := httptest.NewRecorder()
	csrf, err := s.Create(ctx, rec, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("Cookie-Attribute falsch: %+v", cookies)
	}
	token := cookies[0].Value

	load := func() *auth.Identity {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
		id, err := s.Load(ctx, httptest.NewRecorder(), req)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	id := load()
	if id == nil || id.User.ID != u.ID || id.CSRFToken != csrf {
		t.Fatalf("Session nicht korrekt geladen: %+v", id)
	}
	if id.SessionID == token {
		t.Fatal("Token darf nicht im Klartext als Session-ID gespeichert werden")
	}

	// Gefälschter Token → keine Session
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "gefaelscht"})
	if id, _ := s.Load(ctx, httptest.NewRecorder(), req); id != nil {
		t.Fatal("gefälschter Token akzeptiert")
	}

	// Abgelaufen
	now = now.Add(2 * time.Hour)
	if load() != nil {
		t.Fatal("abgelaufene Session akzeptiert")
	}
	if n, err := s.Cleanup(ctx); err != nil || n != 1 {
		t.Fatalf("Cleanup: n=%d err=%v", n, err)
	}

	// Logout
	now = now.Add(time.Hour)
	rec = httptest.NewRecorder()
	if _, err := s.Create(ctx, rec, u.ID); err != nil {
		t.Fatal(err)
	}
	token = rec.Result().Cookies()[0].Value
	id = load()
	if err := s.Destroy(ctx, httptest.NewRecorder(), id.SessionID); err != nil {
		t.Fatal(err)
	}
	if load() != nil {
		t.Fatal("Session nach Logout noch gültig")
	}
}

func TestEnsureAdmin(t *testing.T) {
	ctx := context.Background()
	_, q := testutil.DB(t)
	log := testutil.Logger()

	if err := auth.EnsureAdmin(ctx, q, "Admin@Example.com", "kurz", "A", log); err == nil {
		t.Fatal("zu kurzes Passwort sollte abgelehnt werden")
	}
	if err := auth.EnsureAdmin(ctx, q, "admin@example.com", "langes-passwort", "Admin", log); err != nil {
		t.Fatal(err)
	}
	u, err := q.GetUserByEmail(ctx, "admin@example.com")
	if err != nil || u.Role != "admin" {
		t.Fatalf("Admin nicht angelegt: %+v %v", u, err)
	}
	// Zweiter Aufruf: keine Änderung, kein Fehler
	if err := auth.EnsureAdmin(ctx, q, "other@example.com", "langes-passwort", "X", log); err != nil {
		t.Fatal(err)
	}
	if n, _ := q.CountUsers(ctx); n != 1 {
		t.Fatalf("erwartet 1 Benutzer, gefunden %d", n)
	}
}

func TestLoginLimiter(t *testing.T) {
	l := auth.NewLoginLimiter(3, time.Minute)
	for range 3 {
		if !l.Allowed("k") {
			t.Fatal("zu früh gesperrt")
		}
		l.Fail("k")
	}
	if l.Allowed("k") {
		t.Fatal("nach 3 Fehlversuchen nicht gesperrt")
	}
	if !l.Allowed("anderer") {
		t.Fatal("anderer Schlüssel fälschlich gesperrt")
	}
	l.Reset("k")
	if !l.Allowed("k") {
		t.Fatal("Reset wirkungslos")
	}
}
