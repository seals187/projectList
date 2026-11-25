package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/service"
)

// Project page extras: activity log, files, watchers, duplicating a project
// and the Easy Redmine base URL (global setting).

// ---- Activity ------------------------------------------------------------------

// Activity kinds (also used by the web app for the texts).
const (
	actProjectCreated   = service.ActProjectCreated
	actStatus           = service.ActStatus
	actChecklistCreated = service.ActChecklistCreated
	actChecklistDone    = service.ActChecklistDone
	actChecklistDeleted = service.ActChecklistDeleted
	actItemDone         = service.ActItemDone
	actFileAdded        = service.ActFileAdded
	actFileDeleted      = service.ActFileDeleted
	actTableCreated     = service.ActTableCreated
	actTableDeleted     = service.ActTableDeleted
)

// logActivity records an entry in the project's activity log. A failure is
// only logged: the change itself has already been saved.
func (s *Server) logActivity(ctx context.Context, projectID, userID int64, kind, subject, detail string) {
	s.life.LogActivity(ctx, projectID, service.UserActor(userID, ""), kind, subject, detail)
}

const maxActivity = 200

// GET /api/projects/{id}/activity?limit=n: newest first.
func (s *Server) projectActivity(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	limit := int64(50)
	if n, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64); err == nil && n > 0 {
		limit = min(n, maxActivity)
	}
	rows, err := s.q.ListProjectActivity(r.Context(), sqlc.ListProjectActivityParams{ProjectID: id, Limit: limit})
	if err != nil {
		return err
	}
	if rows == nil {
		rows = []sqlc.ListProjectActivityRow{}
	}
	writeJSON(w, http.StatusOK, rows)
	return nil
}

// ---- Files ---------------------------------------------------------------------------

const maxFileSize = 25 << 20 // 25 MB per file

// GET /api/projects/{id}/files
func (s *Server) listProjectFiles(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	files, err := s.q.ListProjectFiles(r.Context(), id)
	if err != nil {
		return err
	}
	if files == nil {
		files = []sqlc.ListProjectFilesRow{}
	}
	writeJSON(w, http.StatusOK, files)
	return nil
}

// cleanFileName keeps the base name without control characters (max. 200 characters).
func cleanFileName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if r := []rune(name); len(r) > 200 {
		name = string(r[:200])
	}
	if name == "" || name == "." || name == "/" {
		name = "file"
	}
	return name
}

// POST /api/projects/{id}/files (multipart/form-data, field "file", max. 25 MB).
func (s *Server) uploadProjectFile(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+1<<20) // + room for the multipart framing
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return errBadRequest("file_too_large")
		}
		return errBadRequest("invalid_request")
	}
	defer r.MultipartForm.RemoveAll()
	f, hdr, err := r.FormFile("file")
	if err != nil {
		return errBadRequest("file_missing")
	}
	defer f.Close()
	if hdr.Size > maxFileSize {
		return errBadRequest("file_too_large")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if err != nil {
		return err
	}
	if len(data) > maxFileSize {
		return errBadRequest("file_too_large")
	}
	name := cleanFileName(hdr.Filename)
	ctype := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if ctype == "" {
		ctype = http.DetectContentType(data)
	}
	var fileID int64
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		fileID, err = q.CreateProjectFile(r.Context(), sqlc.CreateProjectFileParams{
			ProjectID: id, Name: name, Mime: ctype, Size: int64(len(data)), UploadedBy: &u.ID, CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}
		return q.SetProjectFileData(r.Context(), sqlc.SetProjectFileDataParams{FileID: fileID, Data: data})
	})
	if err != nil {
		return err
	}
	s.logActivity(r.Context(), id, u.ID, actFileAdded, name, "")
	writeJSON(w, http.StatusCreated, map[string]any{"id": fileID, "name": name, "mime": ctype, "size": len(data)})
	return nil
}

// GET /api/files/{id}: download. Always as attachment and with a sandbox
// policy, so uploaded HTML/SVG can never run in the app's origin.
func (s *Server) downloadProjectFile(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	f, err := s.loadFile(r.Context(), u, id)
	if err != nil {
		return err
	}
	data, err := s.q.GetProjectFileData(r.Context(), id)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", f.Mime)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Name}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "", f.CreatedAt, bytes.NewReader(data))
	return nil
}

// DELETE /api/files/{id}
func (s *Server) deleteProjectFile(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	f, err := s.loadFile(r.Context(), u, id)
	if err != nil {
		return err
	}
	if err := s.q.DeleteProjectFile(r.Context(), id); err != nil {
		return err
	}
	s.logActivity(r.Context(), f.ProjectID, u.ID, actFileDeleted, f.Name, "")
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---- Watchers ----------------------------------------------------------------------------

const maxWatchers = 50

// PUT /api/projects/{id}/watchers {"user_ids": [..]}: only people with access to the customer.
func (s *Server) setProjectWatchers(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	p, err := s.loadProject(r.Context(), u, id)
	if err != nil {
		return err
	}
	var req struct {
		UserIDs []int64 `json:"user_ids"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if len(req.UserIDs) > maxWatchers {
		return errBadRequest("too_many")
	}
	allowed, err := s.q.ListUsersWithAccessToCustomer(r.Context(), p.CustomerID)
	if err != nil {
		return err
	}
	ok := make(map[int64]bool, len(allowed))
	for _, a := range allowed {
		ok[a.ID] = true
	}
	for _, uid := range req.UserIDs {
		if !ok[uid] {
			return &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"user_ids": "invalid"}}
		}
	}
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		if err := q.ClearProjectWatchers(r.Context(), id); err != nil {
			return err
		}
		for _, uid := range req.UserIDs {
			if err := q.AddProjectWatcher(r.Context(), sqlc.AddProjectWatcherParams{ProjectID: id, UserID: uid}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	watchers, err := s.q.ListProjectWatchers(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, nonNil(watchers))
	return nil
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// ---- Duplicate ----------------------------------------------------------------------------

// POST /api/projects/{id}/duplicate: copy of the project in the same customer
// with description, person responsible, labels, all tables and all checklists (items not
// checked off, without due dates). Status: planning.
func (s *Server) duplicateProject(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	src, err := s.loadProject(r.Context(), u, id)
	if err != nil {
		return err
	}
	ctx := r.Context()
	// optional {"name"}: the web app sends the name in its UI language
	var req struct {
		Name string `json:"name"`
	}
	if r.ContentLength != 0 {
		if err := decode(w, r, &req); err != nil {
			return err
		}
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = src.Name + " (" + copySuffix(r) + ")"
	}
	if r := []rune(name); len(r) > 200 {
		name = string(r[:200])
	}
	p, err := s.life.Duplicate(ctx, actor(u), src, name)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, p)
	return nil
}

// copySuffix: "copy" in the language of the request (Accept-Language), as the
// new name is stored, not translated on display.
func copySuffix(r *http.Request) string {
	switch lang := strings.ToLower(r.Header.Get("Accept-Language")); {
	case strings.HasPrefix(lang, "de"):
		return "Kopie"
	case strings.HasPrefix(lang, "fr"):
		return "copie"
	default:
		return "copy"
	}
}

// ---- App settings (Easy Redmine) ------------------------------------------------------

const settingRedmineURL = "redmine_url"

// redmineURL returns the configured Easy Redmine base URL (” = not set).
func (s *Server) redmineURL(ctx context.Context) (string, error) {
	v, err := s.q.GetAppSetting(ctx, settingRedmineURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}
