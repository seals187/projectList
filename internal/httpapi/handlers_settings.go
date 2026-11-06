package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Global settings (app_settings): Easy Redmine base URL and the result of the
// last connection test; whether e-mail (SMTP) is configured.

const (
	settingRedmineOK      = "redmine_ok"
	settingRedmineChecked = "redmine_checked_at"
)

type appSettingsResp struct {
	RedmineURL       string `json:"redmine_url"`
	MailEnabled      bool   `json:"mail_enabled"`
	RedmineCheckedAt string `json:"redmine_checked_at"`
	RedmineOK        bool   `json:"redmine_ok"`
}

func (s *Server) appSetting(ctx context.Context, key string) (string, error) {
	v, err := s.q.GetAppSetting(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func (s *Server) appSettings(ctx context.Context) (appSettingsResp, error) {
	out := appSettingsResp{MailEnabled: s.mailer.Enabled()}
	var err error
	if out.RedmineURL, err = s.appSetting(ctx, settingRedmineURL); err != nil {
		return out, err
	}
	if out.RedmineCheckedAt, err = s.appSetting(ctx, settingRedmineChecked); err != nil {
		return out, err
	}
	ok, err := s.appSetting(ctx, settingRedmineOK)
	out.RedmineOK = ok == "1"
	return out, err
}

// GET /api/settings/app (all users: the URL is needed to build ticket links).
func (s *Server) getAppSettings(w http.ResponseWriter, r *http.Request, u authz.User) error {
	out, err := s.appSettings(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func validRedmineURL(raw string) (string, bool) {
	u := strings.TrimRight(strings.TrimSpace(raw), "/")
	if u == "" {
		return "", true
	}
	if !(strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")) || len(u) > 500 {
		return u, false
	}
	return u, true
}

// PUT /api/settings/app {"redmine_url"} (admin): http(s) URL or empty. A new
// URL clears the result of the last connection test.
func (s *Server) setAppSettings(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		RedmineURL string `json:"redmine_url"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	newURL, ok := validRedmineURL(req.RedmineURL)
	if !ok {
		return &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"redmine_url": "invalid_url"}}
	}
	old, err := s.appSetting(r.Context(), settingRedmineURL)
	if err != nil {
		return err
	}
	if err := db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		if err := q.SetAppSetting(r.Context(), sqlc.SetAppSettingParams{Key: settingRedmineURL, Value: newURL}); err != nil {
			return err
		}
		if newURL == old {
			return nil
		}
		for _, k := range []string{settingRedmineOK, settingRedmineChecked} {
			if err := q.SetAppSetting(r.Context(), sqlc.SetAppSettingParams{Key: k, Value: ""}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return s.getAppSettings(w, r, u)
}

// redmineHTTP: 5 s timeout, at most 3 redirects.
func (s *Server) redmineHTTP() *http.Client {
	if s.redmineClient != nil {
		return s.redmineClient
	}
	return &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

// POST /api/settings/redmine/test {"redmine_url"?} (admin): checks whether the
// address answers (any HTTP status below 500 counts as reachable). The result is
// stored when the tested URL is the saved one.
func (s *Server) testRedmine(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req struct {
		RedmineURL string `json:"redmine_url"`
	}
	if r.ContentLength != 0 {
		if err := decode(w, r, &req); err != nil {
			return err
		}
	}
	stored, err := s.appSetting(r.Context(), settingRedmineURL)
	if err != nil {
		return err
	}
	target, ok := validRedmineURL(req.RedmineURL)
	if target == "" {
		target = stored
	}
	if parsed, err := url.Parse(target); !ok || target == "" || err != nil || parsed.Host == "" {
		return &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"redmine_url": "invalid_url"}}
	}

	res := struct {
		OK        bool   `json:"ok"`
		Status    int    `json:"status"`
		Error     string `json:"error"`
		CheckedAt string `json:"checked_at"`
	}{CheckedAt: s.now().UTC().Format(time.RFC3339)}

	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	hreq, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		res.Error = err.Error()
	} else {
		hreq.Header.Set("User-Agent", "ProjectList")
		resp, err := s.redmineHTTP().Do(hreq)
		if err != nil {
			res.Error = shortNetErr(err)
		} else {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			resp.Body.Close()
			res.Status = resp.StatusCode
			res.OK = resp.StatusCode < 500
			if !res.OK {
				res.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
		}
	}

	if target == stored {
		okVal := "0"
		if res.OK {
			okVal = "1"
		}
		if err := db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
			if err := q.SetAppSetting(r.Context(), sqlc.SetAppSettingParams{Key: settingRedmineOK, Value: okVal}); err != nil {
				return err
			}
			return q.SetAppSetting(r.Context(), sqlc.SetAppSettingParams{Key: settingRedmineChecked, Value: res.CheckedAt})
		}); err != nil {
			return err
		}
	}
	writeJSON(w, http.StatusOK, res)
	return nil
}

// shortNetErr strips the request URL from client errors ("Get \"…\": dial tcp: …").
func shortNetErr(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	msg := err.Error()
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}
