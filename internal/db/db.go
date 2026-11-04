// Package db encapsulates opening the SQLite database, migrations and transactions.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // registers the "sqlite" driver (pure Go, no CGO)

	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/migrations"
)

// pragmas apply to every new connection in the pool.
const pragmas = "_pragma=foreign_keys(1)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=busy_timeout(5000)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=cache_size(-2000)" + // max. ~2 MB page cache per connection
	"&_txlock=immediate" + // write transactions lock immediately → no upgrade deadlocks
	"&_time_format=sqlite"

// Open opens (or creates) the database file.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("datenverzeichnis anlegen: %w", err)
		}
	}
	u := url.URL{Scheme: "file", Opaque: path, RawQuery: pragmas}
	d, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(4)
	d.SetMaxIdleConns(2)
	d.SetConnMaxIdleTime(5 * time.Minute)
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

// OpenMemory opens an in-memory database (for tests). A single
// connection, since every :memory: connection would be its own database.
func OpenMemory() (*sql.DB, error) {
	d, err := sql.Open("sqlite", "file::memory:?"+pragmas)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1)
	return d, d.Ping()
}

// Migrate runs all pending goose migrations.
func Migrate(ctx context.Context, d *sql.DB, log *slog.Logger) error {
	p, err := goose.NewProvider(goose.DialectSQLite3, d, migrations.FS)
	if err != nil {
		return err
	}
	results, err := p.Up(ctx)
	if err != nil {
		return fmt.Errorf("migration: %w", err)
	}
	for _, r := range results {
		log.Info("migration angewendet", "version", r.Source.Version, "dauer", r.Duration.String())
	}
	return nil
}

// WithTx runs fn in a transaction; on error it is rolled back.
func WithTx(ctx context.Context, d *sql.DB, fn func(q *sqlc.Queries) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit
	if err := fn(sqlc.New(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// VacuumInto creates a consistent copy of the database (also while running).
func VacuumInto(ctx context.Context, d *sql.DB, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("zieldatei existiert bereits: %s", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	_, err := d.ExecContext(ctx, "VACUUM INTO ?", dest)
	return err
}
