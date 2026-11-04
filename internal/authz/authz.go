// Package authz contains the permission checks (roles + customer assignment).
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

// User is the logged-in user (without password hash).
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// CanWrite: admins and members may modify, viewers may only read.
func (u User) CanWrite() bool { return u.Role == RoleAdmin || u.Role == RoleMember }

// VisibilityFilter returns the parameter for the "user_id IS NULL OR …" queries:
// nil for admins (everything visible), otherwise the user's own ID.
func (u User) VisibilityFilter() any {
	if u.IsAdmin() {
		return nil
	}
	return u.ID
}

// CanAccessCustomer checks whether u may see the customer.
func CanAccessCustomer(ctx context.Context, q *sqlc.Queries, u User, customerID int64) (bool, error) {
	if u.IsAdmin() {
		return true, nil
	}
	return q.HasMembership(ctx, sqlc.HasMembershipParams{UserID: u.ID, CustomerID: customerID})
}
