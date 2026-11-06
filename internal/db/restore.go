package db

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	sqlite "modernc.org/sqlite"

	"github.com/seals187/projectList/migrations"
)

// Reasons why a file cannot be restored (shown to the admin, see the httpapi codes).
var (
	ErrNotSQLite       = errors.New("not a sqlite database")
	ErrCorrupt         = errors.New("database is damaged")
	ErrNotProjectList  = errors.New("not a projectlist database")
	ErrNewerVersion    = errors.New("backup is from a newer version")
	ErrNoAdmin         = errors.New("backup has no admin")
	ErrRestoreFailed   = errors.New("restore failed")
	sqliteHeader       = []byte("SQLite format 3\x00")
	restoreBusyRetries = 20
)

// LatestVersion is the highest migration version this build knows.
func LatestVersion() (int64, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return 0, err
	}
	var max int64
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		num, _, _ := strings.Cut(name, "_")
		if v, err := strconv.ParseInt(num, 10, 64); err == nil && v > max {
			max = v
		}
	}
	return max, nil
}

// CheckBackup validates an uploaded backup file before it replaces any data:
// SQLite header, integrity, a ProjectList schema (goose version table and
// users) not newer than this build, and at least one admin (otherwise nobody
// could sign in afterwards). Returns the schema version of the backup.
func CheckBackup(ctx context.Context, path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	head := make([]byte, len(sqliteHeader))
	_, err = io.ReadFull(f, head)
	f.Close()
	if err != nil || !bytes.Equal(head, sqliteHeader) {
		return 0, ErrNotSQLite
	}

	u := url.URL{Scheme: "file", Opaque: path, RawQuery: "mode=ro&_pragma=busy_timeout(5000)"}
	d, err := sql.Open("sqlite", u.String())
	if err != nil {
		return 0, ErrNotSQLite
	}
	defer d.Close()
	d.SetMaxOpenConns(1)

	var check string
	if err := d.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check); err != nil || check != "ok" {
		return 0, ErrCorrupt
	}
	var version sql.NullInt64
	if err := d.QueryRowContext(ctx, "SELECT MAX(version_id) FROM goose_db_version WHERE is_applied").Scan(&version); err != nil || !version.Valid || version.Int64 < 1 {
		return 0, ErrNotProjectList
	}
	latest, err := LatestVersion()
	if err != nil {
		return 0, err
	}
	if version.Int64 > latest {
		return version.Int64, ErrNewerVersion
	}
	var admins int
	if err := d.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&admins); err != nil {
		return version.Int64, ErrNotProjectList
	}
	if admins == 0 {
		return version.Int64, ErrNoAdmin
	}
	return version.Int64, nil
}

type restorer interface {
	NewRestore(srcURI string) (*sqlite.Backup, error)
}

// Restore replaces the content of the open database with the backup file
// (SQLite online backup API: all connections of the pool see the new data,
// no restart needed). Run Migrate afterwards for backups of older versions.
// The caller must not hold another connection of a single-connection pool.
func Restore(ctx context.Context, d *sql.DB, src string) error {
	conn, err := d.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(dc any) error {
		r, ok := dc.(restorer)
		if !ok {
			return fmt.Errorf("%w: driver without backup api", ErrRestoreFailed)
		}
		b, err := r.NewRestore(src)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrRestoreFailed, err)
		}
		var stepErr error
		for i := 0; ; i++ {
			more, err := b.Step(-1)
			if err == nil && !more {
				break
			}
			if err != nil {
				// other connections may briefly hold a lock: retry a few times
				msg := strings.ToLower(err.Error())
				if i < restoreBusyRetries && (strings.Contains(msg, "busy") || strings.Contains(msg, "locked")) {
					time.Sleep(250 * time.Millisecond)
					continue
				}
				stepErr = err
				break
			}
		}
		finErr := b.Finish()
		if stepErr != nil {
			return fmt.Errorf("%w: %v", ErrRestoreFailed, stepErr)
		}
		if finErr != nil {
			return fmt.Errorf("%w: %v", ErrRestoreFailed, finErr)
		}
		return nil
	})
}
