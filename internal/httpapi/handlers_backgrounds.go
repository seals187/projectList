package httpapi

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Backgrounds: a shared library of uploaded images (max. 5 MB, raster images
// only; the web app scales larger images down before uploading) plus gradient
// presets defined in the web app. Users pick their own background (pref
// "background"), admins/members set one per customer or project.

const maxBackgroundSize = 5 << 20

// allowed image types (detected from the content, not from the file name)
var backgroundTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true}

// background values: "" (none), "g<n>" (gradient preset) or "img:<id>"
var backgroundValue = regexp.MustCompile(`^(|g[0-9]{1,3}|img:[0-9]{1,18})$`)

// GET /api/backgrounds: uploaded images (without content).
func (s *Server) listBackgrounds(w http.ResponseWriter, r *http.Request, u authz.User) error {
	rows, err := s.q.ListBackgrounds(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, nonNil(rows))
	return nil
}

// POST /api/backgrounds (multipart/form-data, field "file", max. 5 MB).
func (s *Server) uploadBackground(w http.ResponseWriter, r *http.Request, u authz.User) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBackgroundSize+1<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return errBadRequest("image_too_large")
		}
		return errBadRequest("invalid_request")
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("file")
	if err != nil {
		return errBadRequest("file_missing")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBackgroundSize+1))
	if err != nil {
		return err
	}
	if len(data) > maxBackgroundSize {
		return errBadRequest("image_too_large")
	}
	mime := http.DetectContentType(data)
	if !backgroundTypes[mime] {
		return errBadRequest("not_an_image")
	}
	id, err := s.q.CreateBackground(r.Context(), sqlc.CreateBackgroundParams{
		Mime: mime, Size: int64(len(data)), UploadedBy: &u.ID, CreatedAt: time.Now().UTC(), Data: data,
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "mime": mime, "size": len(data)})
	return nil
}

// GET /api/backgrounds/{id}: the image (cacheable: images never change).
func (s *Server) getBackground(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	b, err := s.q.GetBackground(r.Context(), id)
	if err != nil {
		return err
	}
	data, err := s.q.GetBackgroundData(r.Context(), id)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", b.Mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeContent(w, r, "", b.CreatedAt, bytes.NewReader(data))
	return nil
}

// DELETE /api/backgrounds/{id}: uploader or admin. Customers and projects
// using it fall back to no background.
func (s *Server) deleteBackground(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	b, err := s.q.GetBackground(r.Context(), id)
	if err != nil {
		return err
	}
	if !u.IsAdmin() && (b.UploadedBy == nil || *b.UploadedBy != u.ID) {
		return errForbidden()
	}
	ref := "img:" + strconv.FormatInt(id, 10)
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		if err := q.ClearBackgroundRefs(r.Context(), ref); err != nil {
			return err
		}
		if err := q.ClearProjectBackgroundRefs(r.Context(), ref); err != nil {
			return err
		}
		return q.DeleteBackground(r.Context(), id)
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// validBackground checks a background value; an image must exist.
func (s *Server) validBackground(r *http.Request, value string) (string, error) {
	value = strings.TrimSpace(value)
	if !backgroundValue.MatchString(value) {
		return "", &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"background": "invalid"}}
	}
	if id, ok := strings.CutPrefix(value, "img:"); ok {
		n, _ := strconv.ParseInt(id, 10, 64)
		if _, err := s.q.GetBackground(r.Context(), n); err != nil {
			return "", &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: map[string]string{"background": "invalid"}}
		}
	}
	return value, nil
}

// PUT /api/customers/{id}/background {"background"} (members and admins with access).
func (s *Server) setCustomerBackground(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, id); err != nil {
		return err
	}
	var req struct {
		Background string `json:"background"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	value, err := s.validBackground(r, req.Background)
	if err != nil {
		return err
	}
	if err := s.q.SetCustomerBackground(r.Context(), sqlc.SetCustomerBackgroundParams{Background: value, ID: id}); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"background": value})
	return nil
}

// PUT /api/projects/{id}/background {"background"}
func (s *Server) setProjectBackground(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.loadProject(r.Context(), u, id); err != nil {
		return err
	}
	var req struct {
		Background string `json:"background"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	value, err := s.validBackground(r, req.Background)
	if err != nil {
		return err
	}
	if err := s.q.SetProjectBackground(r.Context(), sqlc.SetProjectBackgroundParams{Background: value, ID: id}); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"background": value})
	return nil
}
