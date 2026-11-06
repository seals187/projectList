package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

type sessionResp struct {
	User      authz.User `json:"user"`
	CSRFToken string     `json:"csrf_token"`
	Subtitle  string     `json:"app_subtitle"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" || len(req.Password) > 1024 || len(email) > 254 {
		s.writeError(w, r, errBadRequest("credentials_required"))
		return
	}
	key := clientIP(r) + "|" + email
	if !s.limiter.Allowed(key) {
		writeJSON(w, http.StatusTooManyRequests, &httpError{Msg: "too_many_attempts"})
		return
	}

	user, err := s.q.GetUserByEmail(r.Context(), email)
	ok := false
	switch {
	case errors.Is(err, sql.ErrNoRows):
		auth.VerifyDummy(req.Password)
	case err != nil:
		s.writeError(w, r, err)
		return
	default:
		ok, err = auth.VerifyPassword(req.Password, user.PasswordHash)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
	}
	if !ok {
		s.limiter.Fail(key)
		s.log.Warn("login fehlgeschlagen", "email", email, "remote", clientIP(r))
		writeJSON(w, http.StatusUnauthorized, &httpError{Msg: "invalid_credentials"})
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
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id := identityFrom(r.Context())
	if err := s.sessions.Destroy(r.Context(), w, id.SessionID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, u authz.User) error {
	writeJSON(w, http.StatusOK, sessionResp{User: u, CSRFToken: identityFrom(r.Context()).CSRFToken, Subtitle: s.subtitle})
	return nil
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	if l := len([]rune(req.NewPassword)); l < auth.MinPasswordLen || l > 200 {
		v["new_password"] = "length:8:200"
	}
	if err := v.err(); err != nil {
		return err
	}
	user, err := s.q.GetUserByID(r.Context(), u.ID)
	if err != nil {
		return err
	}
	ok, err := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return &httpError{Status: http.StatusBadRequest, Msg: "wrong_password",
			Fields: map[string]string{"current_password": "wrong"}}
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err := s.q.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{PasswordHash: hash, ID: u.ID}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
