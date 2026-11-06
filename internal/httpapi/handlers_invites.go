package httpapi

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

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/mail"
)

// Invitations and password reset links: the admin creates a one-time link
// (sent by e-mail if SMTP is configured, otherwise copied by hand); the user
// opens /invite?token=… and sets a password, which signs them in.

const (
	purposeInvite = "invite"
	purposeReset  = "reset"
	inviteTTL     = 7 * 24 * time.Hour
	resetTTL      = 24 * time.Hour
	seenInterval  = 5 * time.Minute
)

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte("user-token|" + token))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// touchSeen records the user's last activity (at most every few minutes).
func (s *Server) touchSeen(ctx context.Context, userID int64) {
	now := s.now().UTC()
	before := now.Add(-seenInterval)
	if err := s.q.TouchUserSeen(ctx, sqlc.TouchUserSeenParams{Now: &now, ID: userID, Before: &before}); err != nil {
		s.log.Warn("last_seen", "user", userID, "err", err)
	}
}

// baseURL: APP_URL or scheme and host of the request.
func (s *Server) baseURL(r *http.Request) string {
	if s.appURL != "" {
		return s.appURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// newUserToken replaces all tokens of the user by a new one and returns the link.
func (s *Server) newUserToken(ctx context.Context, q *sqlc.Queries, r *http.Request, userID int64, purpose string, ttl time.Duration) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := s.now().UTC().Add(ttl)
	if err := q.DeleteUserTokens(ctx, userID); err != nil {
		return "", time.Time{}, err
	}
	if err := q.CreateUserToken(ctx, sqlc.CreateUserTokenParams{ID: tokenHash(token), UserID: userID, Purpose: purpose, ExpiresAt: expires}); err != nil {
		return "", time.Time{}, err
	}
	return s.baseURL(r) + "/invite?token=" + token, expires, nil
}

// sendMail sends if configured; failures are logged (the admin still gets the link).
func (s *Server) sendMail(ctx context.Context, m mail.Message) bool {
	if !s.mailer.Enabled() {
		return false
	}
	if err := s.mailer.Send(ctx, m); err != nil {
		s.log.Error("e-mail konnte nicht gesendet werden", "to", m.To, "err", err)
		return false
	}
	return true
}

type linkResp struct {
	Link    string `json:"link"`
	Emailed bool   `json:"emailed"`
	Purpose string `json:"purpose"`
}

// POST /api/users/invite {name, email, role, customer_ids, lang} (admin)
func (s *Server) inviteUser(w http.ResponseWriter, r *http.Request, me authz.User) error {
	var req struct {
		Email       string  `json:"email"`
		Name        string  `json:"name"`
		Role        string  `json:"role"`
		CustomerIDs []int64 `json:"customer_ids"`
		Lang        string  `json:"lang"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	email := v.email("email", req.Email)
	name := v.text("name", req.Name, 1, 100)
	role := v.oneOf("role", req.Role, authz.RoleAdmin, authz.RoleMember, authz.RoleViewer)
	if err := v.err(); err != nil {
		return err
	}
	// unusable password until the invitation is accepted
	pw, err := randomToken()
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	var user sqlc.User
	var link string
	var expires time.Time
	now := s.now().UTC()
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		var err error
		if user, err = q.CreateUser(r.Context(), sqlc.CreateUserParams{Email: email, Name: name, PasswordHash: hash, Role: role}); err != nil {
			return err
		}
		if err := q.SetUserInvited(r.Context(), sqlc.SetUserInvitedParams{InvitedAt: &now, ID: user.ID}); err != nil {
			return err
		}
		user.InvitedAt = &now
		if err := setMemberships(r, q, user.ID, req.CustomerIDs); err != nil {
			return err
		}
		link, expires, err = s.newUserToken(r.Context(), q, r, user.ID, purposeInvite, inviteTTL)
		return err
	})
	if isUniqueViolation(err) {
		return errConflict("email_taken")
	}
	if err != nil {
		return err
	}
	emailed := s.sendMail(r.Context(), mail.Invite(req.Lang, user.Email, user.Name, me.Name, link, expires))
	writeJSON(w, http.StatusCreated, struct {
		User    userDTO `json:"user"`
		Link    string  `json:"link"`
		Emailed bool    `json:"emailed"`
	}{toUserDTO(user, req.CustomerIDs), link, emailed})
	return nil
}

// POST /api/users/{id}/reset-link {lang} (admin): new invitation link for
// invited users, otherwise a password reset link. Older links stop working.
func (s *Server) resetLink(w http.ResponseWriter, r *http.Request, me authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Lang string `json:"lang"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	user, err := s.q.GetUserByID(r.Context(), id)
	if err != nil {
		return err
	}
	purpose, ttl := purposeReset, resetTTL
	if user.InvitedAt != nil {
		purpose, ttl = purposeInvite, inviteTTL
	}
	var link string
	var expires time.Time
	if err := db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		var err error
		link, expires, err = s.newUserToken(r.Context(), q, r, id, purpose, ttl)
		return err
	}); err != nil {
		return err
	}
	m := mail.Reset(req.Lang, user.Email, user.Name, link, expires)
	if purpose == purposeInvite {
		m = mail.Invite(req.Lang, user.Email, user.Name, me.Name, link, expires)
	}
	writeJSON(w, http.StatusOK, linkResp{Link: link, Emailed: s.sendMail(r.Context(), m), Purpose: purpose})
	return nil
}

func (s *Server) lookupToken(ctx context.Context, q *sqlc.Queries, token string) (sqlc.GetUserTokenRow, error) {
	if token == "" || len(token) > 100 {
		return sqlc.GetUserTokenRow{}, sql.ErrNoRows
	}
	return q.GetUserToken(ctx, sqlc.GetUserTokenParams{ID: tokenHash(token), ExpiresAt: s.now().UTC()})
}

var errInvalidToken = &httpError{Status: http.StatusNotFound, Msg: "invalid_token"}

// GET /api/auth/token/{token} (public): who the link is for.
func (s *Server) getToken(w http.ResponseWriter, r *http.Request) {
	t, err := s.lookupToken(r.Context(), s.q, r.PathValue("token"))
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errInvalidToken)
		return
	}
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": t.Name, "email": t.Email, "purpose": t.Purpose})
}

// POST /api/auth/token {token, password} (public): sets the password, ends all
// other sessions and signs in (same response as the login).
func (s *Server) acceptToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	key := "token|" + clientIP(r)
	if !s.limiter.Allowed(key) {
		writeJSON(w, http.StatusTooManyRequests, &httpError{Msg: "too_many_attempts"})
		return
	}
	v := validator{}
	validatePassword(v, req.Password)
	if err := v.err(); err != nil {
		s.writeError(w, r, err)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var user sqlc.User
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		t, err := s.lookupToken(r.Context(), q, req.Token)
		if err != nil {
			return err
		}
		if err := q.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{PasswordHash: hash, ID: t.UserID}); err != nil {
			return err
		}
		if err := q.SetUserInvited(r.Context(), sqlc.SetUserInvitedParams{InvitedAt: nil, ID: t.UserID}); err != nil {
			return err
		}
		if err := q.DeleteUserTokens(r.Context(), t.UserID); err != nil {
			return err
		}
		if err := q.DeleteSessionsForUser(r.Context(), t.UserID); err != nil {
			return err
		}
		user, err = q.GetUserByID(r.Context(), t.UserID)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		s.limiter.Fail(key)
		writeJSON(w, http.StatusNotFound, errInvalidToken)
		return
	}
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.limiter.Reset(key)
	csrf, err := s.sessions.Create(r.Context(), w, user.ID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.touchSeen(r.Context(), user.ID)
	writeJSON(w, http.StatusOK, sessionResp{
		User:      authz.User{ID: user.ID, Email: user.Email, Name: user.Name, Role: user.Role},
		CSRFToken: csrf,
		Subtitle:  s.subtitle,
		Version:   s.version,
	})
}
