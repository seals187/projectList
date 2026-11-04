package httpapi

import (
	"net/http"
	"strings"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
)

// customerListItem: Kunde mit letzter Aktivität (Kundenübersicht, Sortierung).
type customerListItem struct {
	sqlc.Customer
	LastActivity string `json:"last_activity"`
}

func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request, u authz.User) error {
	ctx, filter := r.Context(), u.VisibilityFilter()
	cs, err := s.q.ListCustomers(ctx, filter)
	if err != nil {
		return err
	}
	act, err := s.q.ListCustomerActivity(ctx, filter)
	if err != nil {
		return err
	}
	last := make(map[int64]string, len(act))
	for _, a := range act {
		last[a.CustomerID] = a.LastActivity
	}
	out := make([]customerListItem, len(cs))
	for i, c := range cs {
		out[i] = customerListItem{Customer: c, LastActivity: last[c.ID]}
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

type customerReq struct {
	Name         *string `json:"name"`
	Notes        *string `json:"notes"`
	Location     *string `json:"location"`
	ContactName  *string `json:"contact_name"`
	ContactEmail *string `json:"contact_email"`
}

// contactFields prüft Ort und Ansprechpartner; leere E-Mail ist erlaubt.
func (v validator) contactFields(req customerReq, location, name, email *string) {
	if req.Location != nil {
		*location = v.text("location", *req.Location, 0, 100)
	}
	if req.ContactName != nil {
		*name = v.text("contact_name", *req.ContactName, 0, 100)
	}
	if req.ContactEmail != nil {
		*email = strings.TrimSpace(*req.ContactEmail)
		if *email != "" {
			*email = v.email("contact_email", *email)
		}
	}
}

func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	var req customerReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.CreateCustomerParams{}
	p.Name = v.text("name", deref(req.Name), 1, 200)
	p.Notes = v.text("notes", deref(req.Notes), 0, 10000)
	v.contactFields(req, &p.Location, &p.ContactName, &p.ContactEmail)
	if err := v.err(); err != nil {
		return err
	}
	c, err := s.q.CreateCustomer(r.Context(), p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, c)
	return nil
}

func (s *Server) getCustomer(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, id); err != nil {
		return err
	}
	c, err := s.q.GetCustomer(r.Context(), id)
	if err != nil {
		return err
	}
	projects, err := s.q.ListProjectsByCustomer(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"customer": c, "projects": projects})
	return nil
}

func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, id); err != nil {
		return err
	}
	var req customerReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	c, err := s.q.GetCustomer(r.Context(), id)
	if err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateCustomerParams{ID: id, Name: c.Name, Notes: c.Notes, Location: c.Location, ContactName: c.ContactName, ContactEmail: c.ContactEmail}
	if req.Name != nil {
		p.Name = v.text("name", *req.Name, 1, 200)
	}
	if req.Notes != nil {
		p.Notes = v.text("notes", *req.Notes, 0, 10000)
	}
	v.contactFields(req, &p.Location, &p.ContactName, &p.ContactEmail)
	if err := v.err(); err != nil {
		return err
	}
	c, err = s.q.UpdateCustomer(r.Context(), p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, c)
	return nil
}

func (s *Server) deleteCustomer(w http.ResponseWriter, r *http.Request, _ authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.q.GetCustomer(r.Context(), id); err != nil {
		return err
	}
	if err := s.q.DeleteCustomer(r.Context(), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
