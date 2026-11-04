package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

const MinPasswordLen = 8

// EnsureAdmin creates an admin from ADMIN_EMAIL/ADMIN_PASSWORD on first start
// (empty users table). If users already exist, nothing happens.
func EnsureAdmin(ctx context.Context, q *sqlc.Queries, email, password, name string, log *slog.Logger) error {
	n, err := q.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if email == "" || password == "" {
		log.Warn("keine Benutzer vorhanden und ADMIN_EMAIL/ADMIN_PASSWORD nicht gesetzt – Login nicht möglich")
		return nil
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return fmt.Errorf("ADMIN_EMAIL ist keine gültige E-Mail-Adresse")
	}
	if len([]rune(password)) < MinPasswordLen {
		return fmt.Errorf("ADMIN_PASSWORD muss mindestens %d Zeichen lang sein", MinPasswordLen)
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if _, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		Email: strings.ToLower(email), Name: name, PasswordHash: hash, Role: authz.RoleAdmin,
	}); err != nil {
		return err
	}
	log.Info("erster Admin angelegt", "email", strings.ToLower(email))
	return nil
}
