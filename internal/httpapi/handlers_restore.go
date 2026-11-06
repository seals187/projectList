package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
)

// maxRestoreSize limits uploaded backups (they contain project files as well).
const maxRestoreSize = 4 << 30

// one restore at a time
var restoreMu sync.Mutex

type restoreResp struct {
	Version    int64       `json:"version"`     // schema version of the backup
	SafetyCopy string      `json:"safety_copy"` // file name of the copy made before the restore
	Relogin    bool        `json:"relogin"`     // the admin does not exist in the backup → sign in again
	User       *authz.User `json:"user,omitempty"`
	CSRFToken  string      `json:"csrf_token,omitempty"`
}

// POST /api/admin/restore (admin, multipart field "file"): replaces all data
// with an uploaded backup (as downloaded via GET /api/admin/backup).
//
//  1. the upload is streamed into the data directory and validated
//     (SQLite, integrity, ProjectList schema not newer than this build, an admin)
//  2. a safety copy of the current data is written next to the database
//     ("pre-restore-<time>.db"; it stays there)
//  3. the backup is copied into the open database (SQLite backup API, no restart)
//  4. migrations bring older backups up to date
//  5. if the restoring admin (same e-mail) is an admin in the backup, they get a
//     new session; otherwise they have to sign in again
func (s *Server) restore(w http.ResponseWriter, r *http.Request, u authz.User) error {
	if !restoreMu.TryLock() {
		return &httpError{Status: http.StatusConflict, Msg: "restore_running"}
	}
	defer restoreMu.Unlock()

	// large backups on slow connections: more time than the server-wide read timeout
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(30 * time.Minute))
	r.Body = http.MaxBytesReader(w, r.Body, maxRestoreSize+1<<20)

	dir := filepath.Dir(s.dbPath)
	stamp := strings.ReplaceAll(time.Now().UTC().Format("20060102-150405.000"), ".", "-")
	tmp := filepath.Join(dir, ".restore-"+stamp+".db")
	defer os.Remove(tmp)
	if err := s.receiveBackup(r, tmp); err != nil {
		return err
	}

	version, err := db.CheckBackup(r.Context(), tmp)
	switch {
	case errors.Is(err, db.ErrNotSQLite):
		return errBadRequest("restore_not_sqlite")
	case errors.Is(err, db.ErrCorrupt):
		return errBadRequest("restore_corrupt")
	case errors.Is(err, db.ErrNotProjectList):
		return errBadRequest("restore_not_projectlist")
	case errors.Is(err, db.ErrNewerVersion):
		return errBadRequest("restore_newer")
	case errors.Is(err, db.ErrNoAdmin):
		return errBadRequest("restore_no_admin")
	case err != nil:
		return err
	}

	safety := "pre-restore-" + stamp + ".db"
	if err := db.VacuumInto(r.Context(), s.db, filepath.Join(dir, safety)); err != nil {
		return fmt.Errorf("sicherheitskopie: %w", err)
	}
	if err := db.Restore(r.Context(), s.db, tmp); err != nil {
		s.log.Error("wiederherstellung fehlgeschlagen", "err", err, "sicherheitskopie", safety)
		return err
	}
	if err := db.Migrate(r.Context(), s.db, s.log); err != nil {
		s.log.Error("migration nach wiederherstellung fehlgeschlagen", "err", err, "sicherheitskopie", safety)
		return err
	}
	s.log.Info("backup wiederhergestellt", "admin", u.Email, "version", version, "sicherheitskopie", safety)

	resp := restoreResp{Version: version, SafetyCopy: safety, Relogin: true}
	if me, err := s.q.GetUserByEmail(r.Context(), u.Email); err == nil && me.Role == authz.RoleAdmin {
		csrf, err := s.sessions.Create(r.Context(), w, me.ID)
		if err == nil {
			resp.Relogin = false
			resp.User = &authz.User{ID: me.ID, Email: me.Email, Name: me.Name, Role: me.Role}
			resp.CSRFToken = csrf
		}
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

// receiveBackup streams the multipart field "file" to dest.
func (s *Server) receiveBackup(r *http.Request, dest string) error {
	mr, err := r.MultipartReader()
	if err != nil {
		return errBadRequest("restore_no_file")
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return errBadRequest("restore_no_file")
		}
		if err != nil {
			return errBadRequest("restore_upload_failed")
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(f, part)
		closeErr := f.Close()
		part.Close()
		var tooBig *http.MaxBytesError
		if errors.As(copyErr, &tooBig) {
			return &httpError{Status: http.StatusRequestEntityTooLarge, Msg: "restore_too_large"}
		}
		if copyErr != nil {
			return errBadRequest("restore_upload_failed")
		}
		return closeErr
	}
}
