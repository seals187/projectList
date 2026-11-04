// Package testutil provides a migrated in-memory SQLite database for tests.
package testutil

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

func DB(t *testing.T) (*sql.DB, *sqlc.Queries) {
	t.Helper()
	d, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(context.Background(), d, Logger()); err != nil {
		t.Fatal(err)
	}
	return d, sqlc.New(d)
}

func Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func CreateUser(t *testing.T, q *sqlc.Queries, email, password, role string) sqlc.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	u, err := q.CreateUser(context.Background(), sqlc.CreateUserParams{Email: email, Name: email, PasswordHash: hash, Role: role})
	if err != nil {
		t.Fatal(err)
	}
	return u
}
