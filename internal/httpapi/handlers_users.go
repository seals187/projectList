package httpapi

import (
	"net/http"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	users, err := s.q.ListUsers(r.Context())
	if err != nil {
		return err
	}
	ms, err := s.q.ListAllMemberships(r.Context())
	if err != nil {
		return err
	}
	byUser := make(map[int64][]int64)
	for _, m := range ms {
		byUser[m.UserID] = append(byUser[m.UserID], m.CustomerID)
	}
	out := make([]userDTO, len(users))
	for i, u := range users {
		out[i] = toUserDTO(u, byUser[u.ID])
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func validatePassword(v validator, pw string) {
	if l := len([]rune(pw)); l < auth.MinPasswordLen || l > 200 {
		v["password"] = "length:8:200"
	}
}

func setMemberships(r *http.Request, q *sqlc.Queries, userID int64, customerIDs []int64) error {
	if err := q.DeleteMembershipsForUser(r.Context(), userID); err != nil {
		return err
	}
	for _, cid := range customerIDs {
		if err := q.AddMembership(r.Context(), sqlc.AddMembershipParams{UserID: userID, CustomerID: cid}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	var req struct {
		Email       string  `json:"email"`
		Name        string  `json:"name"`
		Role        string  `json:"role"`
		Password    string  `json:"password"`
		CustomerIDs []int64 `json:"customer_ids"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	email := v.email("email", req.Email)
	name := v.text("name", req.Name, 1, 100)
	role := v.oneOf("role", req.Role, authz.RoleAdmin, authz.RoleMember, authz.RoleViewer)
	validatePassword(v, req.Password)
	if err := v.err(); err != nil {
		return err
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return err
	}
	var user sqlc.User
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		var err error
		user, err = q.CreateUser(r.Context(), sqlc.CreateUserParams{Email: email, Name: name, PasswordHash: hash, Role: role})
		if err != nil {
			return err
		}
		return setMemberships(r, q, user.ID, req.CustomerIDs)
	})
	if isUniqueViolation(err) {
		return errConflict("email_taken")
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, toUserDTO(user, req.CustomerIDs))
	return nil
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request, me authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Email       *string  `json:"email"`
		Name        *string  `json:"name"`
		Role        *string  `json:"role"`
		Password    *string  `json:"password"`
		CustomerIDs *[]int64 `json:"customer_ids"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	user, err := s.q.GetUserByID(r.Context(), id)
	if err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateUserParams{ID: id, Email: user.Email, Name: user.Name, Role: user.Role}
	if req.Email != nil {
		p.Email = v.email("email", *req.Email)
	}
	if req.Name != nil {
		p.Name = v.text("name", *req.Name, 1, 100)
	}
	if req.Role != nil {
		p.Role = v.oneOf("role", *req.Role, authz.RoleAdmin, authz.RoleMember, authz.RoleViewer)
	}
	var hash string
	if req.Password != nil {
		validatePassword(v, *req.Password)
	}
	if err := v.err(); err != nil {
		return err
	}
	if user.Role == authz.RoleAdmin && p.Role != authz.RoleAdmin {
		if id == me.ID {
			return errBadRequest("self_demote")
		}
		n, err := s.q.CountAdmins(r.Context())
		if err != nil {
			return err
		}
		if n <= 1 {
			return errBadRequest("last_admin")
		}
	}
	if req.Password != nil {
		if hash, err = auth.HashPassword(*req.Password); err != nil {
			return err
		}
	}

	var updated sqlc.User
	var customerIDs []int64
	err = db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		var err error
		if updated, err = q.UpdateUser(r.Context(), p); err != nil {
			return err
		}
		if hash != "" {
			if err := q.UpdateUserPassword(r.Context(), sqlc.UpdateUserPasswordParams{PasswordHash: hash, ID: id}); err != nil {
				return err
			}
			if id != me.ID { // fremde Sessions nach Passwort-Reset beenden
				if err := q.DeleteSessionsForUser(r.Context(), id); err != nil {
					return err
				}
			}
		}
		if req.CustomerIDs != nil {
			if err := setMemberships(r, q, id, *req.CustomerIDs); err != nil {
				return err
			}
		}
		customerIDs, err = q.ListMembershipsForUser(r.Context(), id)
		return err
	})
	if isUniqueViolation(err) {
		return errConflict("email_taken")
	}
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, toUserDTO(updated, customerIDs))
	return nil
}

func (s *Server) setUserCustomers(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		CustomerIDs []int64 `json:"customer_ids"`
	}
	if err := decode(w, r, &req); err != nil {
		return err
	}
	if _, err := s.q.GetUserByID(r.Context(), id); err != nil {
		return err
	}
	if err := db.WithTx(r.Context(), s.db, func(q *sqlc.Queries) error {
		return setMemberships(r, q, id, req.CustomerIDs)
	}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, me authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if id == me.ID {
		return errBadRequest("self_delete")
	}
	if _, err := s.q.GetUserByID(r.Context(), id); err != nil {
		return err
	}
	if err := s.q.DeleteUser(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
