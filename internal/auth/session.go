package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

const CookieName = "pl_session"

// Identity ist eine geladene, gültige Session.
type Identity struct {
	User      authz.User
	SessionID string
	CSRFToken string
}

// Sessions verwaltet serverseitige Sessions. In der DB wird nur der
// SHA-256-Hash des Tokens gespeichert; der Klartext existiert nur im Cookie.
// Der CSRF-Token wird ebenfalls aus dem Token abgeleitet, ist aber nicht
// umkehrbar und für fremde Seiten nicht lesbar (Cookie ist httpOnly).
type Sessions struct {
	Q      *sqlc.Queries
	TTL    time.Duration
	Secure bool
	Now    func() time.Time
}

func (s *Sessions) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func sessionID(token string) string {
	sum := sha256.Sum256([]byte("session|" + token))
	return hex.EncodeToString(sum[:])
}

func csrfToken(token string) string {
	sum := sha256.Sum256([]byte("csrf|" + token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// Create legt eine Session an, setzt das Cookie und gibt den CSRF-Token zurück.
func (s *Sessions) Create(ctx context.Context, w http.ResponseWriter, userID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := s.now().Add(s.TTL)
	if err := s.Q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID: sessionID(token), UserID: userID, ExpiresAt: expires,
	}); err != nil {
		return "", err
	}
	s.setCookie(w, token, expires)
	return csrfToken(token), nil
}

// Load liest die Session aus dem Request. (nil, nil) = nicht angemeldet.
// Läuft die Session bald ab (weniger als halbe TTL), wird sie verlängert.
func (s *Sessions) Load(ctx context.Context, w http.ResponseWriter, r *http.Request) (*Identity, error) {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" || len(c.Value) > 100 {
		return nil, nil
	}
	id := sessionID(c.Value)
	now := s.now()
	row, err := s.Q.GetSessionWithUser(ctx, sqlc.GetSessionWithUserParams{ID: id, ExpiresAt: now})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if row.ExpiresAt.Sub(now) < s.TTL/2 {
		expires := now.Add(s.TTL)
		if err := s.Q.ExtendSession(ctx, sqlc.ExtendSessionParams{ExpiresAt: expires, ID: id}); err == nil {
			s.setCookie(w, c.Value, expires)
		}
	}
	return &Identity{
		User:      authz.User{ID: row.ID, Email: row.Email, Name: row.Name, Role: row.Role},
		SessionID: id,
		CSRFToken: csrfToken(c.Value),
	}, nil
}

// Destroy löscht die Session und das Cookie.
func (s *Sessions) Destroy(ctx context.Context, w http.ResponseWriter, sessionID string) error {
	s.setCookie(w, "", time.Unix(0, 0))
	return s.Q.DeleteSession(ctx, sessionID)
}

// Cleanup entfernt abgelaufene Sessions.
func (s *Sessions) Cleanup(ctx context.Context) (int64, error) {
	return s.Q.DeleteExpiredSessions(ctx, s.now())
}

func (s *Sessions) setCookie(w http.ResponseWriter, value string, expires time.Time) {
	c := &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.Secure,
		SameSite: http.SameSiteLaxMode,
	}
	if value == "" {
		c.MaxAge = -1
	}
	http.SetCookie(w, c)
}
