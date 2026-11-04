// Package authz enthält die Rechteprüfung (Rollen + Kundenzuordnung).
package authz

import (
	"context"

	"github.com/seals187/projectList/internal/db/sqlc"
)

const (
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

func ValidRole(r string) bool {
	return r == RoleAdmin || r == RoleMember || r == RoleViewer
}

// User ist der angemeldete Benutzer (ohne Passwort-Hash).
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// CanWrite: Admins und Member dürfen ändern, Viewer nur lesen.
func (u User) CanWrite() bool { return u.Role == RoleAdmin || u.Role == RoleMember }

// VisibilityFilter liefert den Parameter für die "user_id IS NULL OR …"-Abfragen:
// nil für Admins (alles sichtbar), sonst die eigene ID.
func (u User) VisibilityFilter() any {
	if u.IsAdmin() {
		return nil
	}
	return u.ID
}

// CanAccessCustomer prüft, ob u den Kunden sehen darf.
func CanAccessCustomer(ctx context.Context, q *sqlc.Queries, u User, customerID int64) (bool, error) {
	if u.IsAdmin() {
		return true, nil
	}
	return q.HasMembership(ctx, sqlc.HasMembershipParams{UserID: u.ID, CustomerID: customerID})
}
