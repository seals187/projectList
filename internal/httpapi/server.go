// Package httpapi provides the JSON API and routing.
package httpapi

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/mail"
)

type Server struct {
	db            *sql.DB
	q             *sqlc.Queries
	sessions      *auth.Sessions
	limiter       *auth.LoginLimiter
	log           *slog.Logger
	dbPath        string
	subtitle      string
	mailer        mail.Sender
	appURL        string
	now           func() time.Time
	redmineClient *http.Client // nil = default (tests may replace)
}

type Options struct {
	DB       *sql.DB
	DBPath   string // for backups (target directory for the temporary file)
	Sessions *auth.Sessions
	Logger   *slog.Logger
	Static   http.Handler     // SPA; nil = no static files
	Subtitle string           // APP_SUBTITLE, included in /api/auth/me
	Mailer   mail.Sender      // nil = e-mail disabled (links are shown to the admin)
	AppURL   string           // APP_URL: base for links in e-mails; "" = from the request
	Now      func() time.Time // nil = time.Now (tests)
}

// handlerFunc is an API handler for logged-in users.
type handlerFunc func(w http.ResponseWriter, r *http.Request, u authz.User) error

type ctxKey struct{}

func identityFrom(ctx context.Context) *auth.Identity {
	id, _ := ctx.Value(ctxKey{}).(*auth.Identity)
	return id
}

func New(o Options) http.Handler {
	s := &Server{
		db:       o.DB,
		q:        sqlc.New(o.DB),
		sessions: o.Sessions,
		limiter:  auth.NewLoginLimiter(10, 15*time.Minute),
		log:      o.Logger,
		dbPath:   o.DBPath,
		subtitle: o.Subtitle,
		mailer:   o.Mailer,
		appURL:   o.AppURL,
		now:      o.Now,
	}
	if s.mailer == nil {
		s.mailer = mail.Disabled{}
	}
	if s.now == nil {
		s.now = time.Now
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)

	// Auth
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.Handle("POST /api/auth/logout", s.authed(s.logout))
	mux.Handle("GET /api/auth/me", s.authed(s.me))
	mux.Handle("PUT /api/auth/password", s.authed(s.changePassword))
	mux.HandleFunc("GET /api/auth/token/{token}", s.getToken)
	mux.HandleFunc("POST /api/auth/token", s.acceptToken)

	// User management (admin)
	mux.Handle("GET /api/users", s.admin(s.listUsers))
	mux.Handle("POST /api/users", s.admin(s.createUser))
	mux.Handle("POST /api/users/invite", s.admin(s.inviteUser))
	mux.Handle("POST /api/users/{id}/reset-link", s.admin(s.resetLink))
	mux.Handle("PATCH /api/users/{id}", s.admin(s.updateUser))
	mux.Handle("DELETE /api/users/{id}", s.admin(s.deleteUser))
	mux.Handle("PUT /api/users/{id}/customers", s.admin(s.setUserCustomers))

	// Customers
	mux.Handle("GET /api/customers", s.authed(s.listCustomers))
	mux.Handle("POST /api/customers", s.admin(s.createCustomer))
	mux.Handle("GET /api/customers/{id}", s.authed(s.getCustomer))
	mux.Handle("PATCH /api/customers/{id}", s.writer(s.updateCustomer))
	mux.Handle("DELETE /api/customers/{id}", s.admin(s.deleteCustomer))

	// Projects
	mux.Handle("GET /api/customers/{id}/projects", s.authed(s.listProjects))
	mux.Handle("GET /api/customers/{id}/assignees", s.authed(s.customerAssignees))
	mux.Handle("POST /api/customers/{id}/projects", s.writer(s.createProject))
	mux.Handle("GET /api/projects/{id}", s.authed(s.getProject))
	mux.Handle("PATCH /api/projects/{id}", s.writer(s.updateProject))
	mux.Handle("DELETE /api/projects/{id}", s.writer(s.deleteProject))
	mux.Handle("PUT /api/projects/{id}/tags", s.writer(s.setProjectTags))
	mux.Handle("PUT /api/projects/{id}/watchers", s.writer(s.setProjectWatchers))
	mux.Handle("POST /api/projects/{id}/duplicate", s.writer(s.duplicateProject))
	mux.Handle("GET /api/projects/{id}/activity", s.authed(s.projectActivity))
	mux.Handle("GET /api/projects/{id}/files", s.authed(s.listProjectFiles))
	mux.Handle("POST /api/projects/{id}/files", s.writer(s.uploadProjectFile))
	mux.Handle("GET /api/files/{id}", s.authed(s.downloadProjectFile))
	mux.Handle("DELETE /api/files/{id}", s.writer(s.deleteProjectFile))
	mux.Handle("GET /api/settings/app", s.authed(s.getAppSettings))
	mux.Handle("PUT /api/settings/app", s.admin(s.setAppSettings))
	mux.Handle("POST /api/settings/redmine/test", s.admin(s.testRedmine))
	mux.Handle("GET /api/backgrounds", s.authed(s.listBackgrounds))
	mux.Handle("POST /api/backgrounds", s.authed(s.uploadBackground))
	mux.Handle("GET /api/backgrounds/{id}", s.authed(s.getBackground))
	mux.Handle("DELETE /api/backgrounds/{id}", s.authed(s.deleteBackground))
	mux.Handle("PUT /api/customers/{id}/background", s.writer(s.setCustomerBackground))
	mux.Handle("PUT /api/projects/{id}/background", s.writer(s.setProjectBackground))

	// Tags and status names (catalog: members create, admins rename/delete)
	mux.Handle("POST /api/tags", s.writer(s.createTag))
	mux.Handle("PATCH /api/tags/{id}", s.admin(s.updateTag))
	mux.Handle("DELETE /api/tags/{id}", s.admin(s.deleteTag))
	mux.Handle("PUT /api/status-labels/{status}", s.admin(s.setStatusLabel))

	// Checklists
	mux.Handle("GET /api/projects/{id}/checklists", s.authed(s.listChecklists))
	mux.Handle("POST /api/projects/{id}/checklists", s.writer(s.createChecklist))
	mux.Handle("POST /api/projects/{id}/checklists/from-template", s.writer(s.createChecklistFromTemplate))
	mux.Handle("PUT /api/projects/{id}/checklists/order", s.writer(s.orderChecklists))
	mux.Handle("GET /api/checklists/{id}", s.authed(s.getChecklist))
	mux.Handle("PATCH /api/checklists/{id}", s.writer(s.updateChecklist))
	mux.Handle("DELETE /api/checklists/{id}", s.writer(s.deleteChecklist))
	mux.Handle("POST /api/checklists/{id}/save-as-template", s.writer(s.saveChecklistAsTemplate))

	// Items
	mux.Handle("POST /api/checklists/{id}/items", s.writer(s.createItem))
	mux.Handle("PUT /api/checklists/{id}/items/order", s.writer(s.orderItems))
	mux.Handle("PATCH /api/items/{id}", s.writer(s.updateItem))
	mux.Handle("POST /api/items/{id}/done", s.writer(s.setItemDone))
	mux.Handle("DELETE /api/items/{id}", s.writer(s.deleteItem))

	// Templates
	mux.Handle("GET /api/templates", s.authed(s.listTemplates))
	mux.Handle("POST /api/templates", s.writer(s.createTemplate))
	mux.Handle("GET /api/templates/{id}", s.authed(s.getTemplate))
	mux.Handle("PATCH /api/templates/{id}", s.writer(s.updateTemplate))
	mux.Handle("DELETE /api/templates/{id}", s.writer(s.deleteTemplate))
	mux.Handle("POST /api/templates/{id}/items", s.writer(s.createTemplateItem))
	mux.Handle("PUT /api/templates/{id}/items/order", s.writer(s.orderTemplateItems))
	mux.Handle("PATCH /api/template-items/{id}", s.writer(s.updateTemplateItem))
	mux.Handle("DELETE /api/template-items/{id}", s.writer(s.deleteTemplateItem))

	// Miscellaneous
	mux.Handle("GET /api/sidebar", s.authed(s.sidebar))
	mux.Handle("GET /api/search", s.authed(s.search))
	mux.Handle("GET /api/my-tasks", s.authed(s.myTasks))
	mux.Handle("GET /api/my-checklists", s.authed(s.myChecklists))
	mux.Handle("GET /api/overview", s.authed(s.overview))
	mux.Handle("PUT /api/board/order", s.writer(s.orderBoard))
	mux.Handle("PUT /api/prefs/board-hidden", s.authed(s.setBoardHidden))
	mux.Handle("PUT /api/prefs/{key}", s.authed(s.setClientPref))
	mux.Handle("PUT /api/favorites/{kind}/{id}", s.authed(s.addFavorite))
	mux.Handle("DELETE /api/favorites/{kind}/{id}", s.authed(s.removeFavorite))
	mux.Handle("GET /api/admin/backup", s.admin(s.backup))
	mux.Handle("POST /api/admin/restore", s.admin(s.restore))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, &httpError{Msg: "unknown_endpoint"})
	})
	if o.Static != nil {
		mux.Handle("/", o.Static)
	}

	return s.recoverer(s.logRequests(securityHeaders(s.originCheck(mux))))
}

// ---- Auth wrappers -----------------------------------------------------------

func isUnsafe(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

// authed requires a valid session and, for write requests, a valid
// CSRF token in the X-CSRF-Token header.
func (s *Server) authed(h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := s.sessions.Load(r.Context(), w, r)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		if id == nil {
			writeJSON(w, http.StatusUnauthorized, &httpError{Msg: "unauthorized"})
			return
		}
		if isUnsafe(r.Method) && r.Header.Get("X-CSRF-Token") != id.CSRFToken {
			writeJSON(w, http.StatusForbidden, &httpError{Msg: "csrf"})
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, id))
		s.touchSeen(r.Context(), id.User.ID)
		if err := h(w, r, id.User); err != nil {
			s.writeError(w, r, err)
		}
	})
}

func (s *Server) admin(h handlerFunc) http.Handler {
	return s.authed(func(w http.ResponseWriter, r *http.Request, u authz.User) error {
		if !u.IsAdmin() {
			return errForbidden()
		}
		return h(w, r, u)
	})
}

func (s *Server) writer(h handlerFunc) http.Handler {
	return s.authed(func(w http.ResponseWriter, r *http.Request, u authz.User) error {
		if !u.CanWrite() {
			return errForbidden()
		}
		return h(w, r, u)
	})
}

// requireCustomer checks access to a customer. If read permission is missing,
// 404 is returned (do not reveal existence).
func (s *Server) requireCustomer(ctx context.Context, u authz.User, customerID int64) error {
	ok, err := authz.CanAccessCustomer(ctx, s.q, u, customerID)
	if err != nil {
		return err
	}
	if !ok {
		return errNotFound()
	}
	return nil
}

// ---- Middleware --------------------------------------------------------------

// originCheck rejects write requests with a foreign origin (in addition to
// SameSite=Lax and the CSRF token; also protects the login against login CSRF).
func (s *Server) originCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isUnsafe(r.Method) {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host {
					writeJSON(w, http.StatusForbidden, &httpError{Msg: "origin"})
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeJSON(w, http.StatusForbidden, &httpError{Msg: "origin"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy", "frame-ancestors 'none'; base-uri 'self'; object-src 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		level := slog.LevelInfo
		if r.URL.Path == "/healthz" {
			level = slog.LevelDebug
		}
		if rec.status >= 500 {
			level = slog.LevelError
		}
		s.log.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"dur_ms", float64(time.Since(start).Microseconds())/1000,
			"remote", clientIP(r),
		)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.Error("panic", "err", v, "stack", string(debug.Stack()))
				writeJSON(w, http.StatusInternalServerError, &httpError{Msg: "internal"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
