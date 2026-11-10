package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/seals187/projectList/internal/service"
)

// ---- Errors ------------------------------------------------------------------

type httpError struct {
	Status int               `json:"-"`
	Msg    string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func (e *httpError) Error() string { return e.Msg }

func errBadRequest(msg string) error { return &httpError{Status: http.StatusBadRequest, Msg: msg} }
func errNotFound() error             { return &httpError{Status: http.StatusNotFound, Msg: "not_found"} }
func errForbidden() error            { return &httpError{Status: http.StatusForbidden, Msg: "forbidden"} }
func errConflict(msg string) error   { return &httpError{Status: http.StatusConflict, Msg: msg} }

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func isFKViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var he *httpError
	switch {
	case errors.As(err, &he):
	case errors.Is(err, sql.ErrNoRows):
		he = errNotFound().(*httpError)
	case errors.Is(err, service.ErrTemplateKind):
		he = errBadRequest("wrong_template_kind").(*httpError)
	case isFKViolation(err):
		he = errBadRequest("reference_missing").(*httpError)
	default:
		s.log.Error("interner Fehler", "method", r.Method, "path", r.URL.Path, "err", err)
		he = &httpError{Status: http.StatusInternalServerError, Msg: "internal"}
	}
	writeJSON(w, he.Status, he)
}

// ---- JSON --------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, `{"error":"Interner Fehler"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errBadRequest("invalid_request")
	}
	return nil
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, errNotFound()
	}
	return id, nil
}

// Opt distinguishes "field missing" from "field is null" in PATCH requests.
type Opt[T any] struct {
	Set  bool
	Null bool
	V    T
}

func (o *Opt[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Null = true
		return nil
	}
	return json.Unmarshal(b, &o.V)
}

// ---- Validation --------------------------------------------------------------

type validator map[string]string

// text trims s and checks its length in characters.
func (v validator) text(field, s string, min, max int) string {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	switch {
	case !utf8.ValidString(s):
		v[field] = "invalid_chars"
	case n < min && n == 0:
		v[field] = "required"
	case n < min:
		v[field] = fmt.Sprintf("min:%d", min)
	case n > max:
		v[field] = fmt.Sprintf("max:%d", max)
	}
	return s
}

func (v validator) email(field, s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || len(s) > 254 {
		v[field] = "invalid_email"
	}
	return s
}

func (v validator) oneOf(field, s string, allowed ...string) string {
	for _, a := range allowed {
		if s == a {
			return s
		}
	}
	v[field] = "invalid_value"
	return s
}

func (v validator) date(field, s string) string {
	s = strings.TrimSpace(s)
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		v[field] = "invalid_date"
	}
	return s
}

func (v validator) err() error {
	if len(v) == 0 {
		return nil
	}
	return &httpError{Status: http.StatusBadRequest, Msg: "validation", Fields: v}
}

// orderReq is the body for sort endpoints.
type orderReq struct {
	IDs []int64 `json:"ids"`
}

func (o orderReq) validate() error {
	if len(o.IDs) == 0 || len(o.IDs) > 1000 {
		return errBadRequest("invalid_order")
	}
	seen := make(map[int64]struct{}, len(o.IDs))
	for _, id := range o.IDs {
		if _, dup := seen[id]; dup {
			return errBadRequest("invalid_order")
		}
		seen[id] = struct{}{}
	}
	return nil
}

// optDate evaluates an optional date field: not sent → cur,
// null or "" → nil, otherwise a validated YYYY-MM-DD.
func (v validator) optDate(field string, o Opt[string], cur *string) *string {
	if !o.Set {
		return cur
	}
	if o.Null || strings.TrimSpace(o.V) == "" {
		return nil
	}
	d := v.date(field, strings.TrimSpace(o.V))
	return &d
}
