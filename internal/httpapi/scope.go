package httpapi

import (
	"context"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// Customer scope: every customer-owned entity (project, checklist, table,
// entry, file) is visible only to admins and to users assigned to its customer.
// All single-entity reads go through the loaders below; an entity that does not
// exist and one the user may not see both give 404, so existence is not
// revealed. Lists use authz.User.VisibilityFilter in SQL instead.
//
// The role gate (viewer/member/admin) is separate: the authed/writer/admin
// wrappers in server.go. Templates, tags and the template library are global
// (visible to every signed-in user) and are not scoped.

// scoped loads a row with get and checks access to the customer it belongs to.
func scoped[T any](ctx context.Context, s *Server, u authz.User, id int64,
	get func(context.Context, int64) (T, error), customerOf func(T) int64) (T, error) {
	row, err := get(ctx, id)
	if err != nil {
		return row, err
	}
	return row, s.requireCustomer(ctx, u, customerOf(row))
}

// requireCustomer checks access to a customer (404 without access).
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

// loadCustomer checks that the customer exists and u may see it.
func (s *Server) loadCustomer(ctx context.Context, u authz.User, id int64) (sqlc.Customer, error) {
	return scoped(ctx, s, u, id, s.q.GetCustomer, func(c sqlc.Customer) int64 { return c.ID })
}

func (s *Server) loadProject(ctx context.Context, u authz.User, id int64) (sqlc.Project, error) {
	return scoped(ctx, s, u, id, s.q.GetProject, func(p sqlc.Project) int64 { return p.CustomerID })
}

func (s *Server) loadChecklist(ctx context.Context, u authz.User, id int64) (sqlc.GetChecklistWithCustomerRow, error) {
	return scoped(ctx, s, u, id, s.q.GetChecklistWithCustomer, func(c sqlc.GetChecklistWithCustomerRow) int64 { return c.CustomerID })
}

func (s *Server) loadTable(ctx context.Context, u authz.User, id int64) (sqlc.GetTableWithCustomerRow, error) {
	return scoped(ctx, s, u, id, s.q.GetTableWithCustomer, func(t sqlc.GetTableWithCustomerRow) int64 { return t.CustomerID })
}

func (s *Server) loadItem(ctx context.Context, u authz.User, id int64) (sqlc.GetItemWithCustomerRow, error) {
	return scoped(ctx, s, u, id, s.q.GetItemWithCustomer, func(it sqlc.GetItemWithCustomerRow) int64 { return it.CustomerID })
}

// loadFile loads a project file; access via its project.
func (s *Server) loadFile(ctx context.Context, u authz.User, id int64) (sqlc.ProjectFile, error) {
	f, err := s.q.GetProjectFile(ctx, id)
	if err != nil {
		return f, err
	}
	_, err = s.loadProject(ctx, u, f.ProjectID)
	return f, err
}

// requireScope checks an entity named by kind ("customer", "project",
// "checklist", "table", "item", "file"); unknown kinds: 404.
func (s *Server) requireScope(ctx context.Context, u authz.User, kind string, id int64) error {
	var err error
	switch kind {
	case "customer":
		_, err = s.loadCustomer(ctx, u, id)
	case "project":
		_, err = s.loadProject(ctx, u, id)
	case "checklist":
		_, err = s.loadChecklist(ctx, u, id)
	case "table":
		_, err = s.loadTable(ctx, u, id)
	case "item":
		_, err = s.loadItem(ctx, u, id)
	case "file":
		_, err = s.loadFile(ctx, u, id)
	default:
		err = errNotFound()
	}
	return err
}
