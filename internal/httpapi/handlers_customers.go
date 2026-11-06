package httpapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/dbtypes"
)

// customerListItem: customer with latest activity (customer overview, sorting).
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
	Name         *string          `json:"name"`
	Notes        *string          `json:"notes"`
	Location     *string          `json:"location"`
	ContactName  *string          `json:"contact_name"`
	ContactEmail *string          `json:"contact_email"`
	Phones       *[]customerPhone `json:"phones"`
}

// customerPhone is one phone number of a customer ("Zentrale", "+41 41 123 45 67").
type customerPhone struct {
	Label  string `json:"label"`
	Number string `json:"number"`
}

var phoneRe = regexp.MustCompile(`^[0-9+()/. -]{3,40}$`)

// phonesField validates the phone list (max. 10) and returns it as JSON.
func (v validator) phonesField(list []customerPhone) dbtypes.JSONText {
	if len(list) > 10 {
		v["phones"] = "too_many"
		return nil
	}
	out := make([]customerPhone, 0, len(list))
	for _, p := range list {
		n := strings.TrimSpace(p.Number)
		l := strings.TrimSpace(p.Label)
		if n == "" && l == "" {
			continue // empty row of the form
		}
		if !phoneRe.MatchString(n) {
			v["phones"] = "invalid_phone"
			return nil
		}
		if utf8.RuneCountInString(l) > 40 {
			v["phones"] = "length:0:40"
			return nil
		}
		out = append(out, customerPhone{Label: l, Number: n})
	}
	b, _ := json.Marshal(out)
	return b
}

// contactFields checks location and contact person; an empty email is allowed.
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
	p.Phones = dbtypes.JSONText(`[]`)
	if req.Phones != nil {
		p.Phones = v.phonesField(*req.Phones)
	}
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
	p := sqlc.UpdateCustomerParams{ID: id, Name: c.Name, Notes: c.Notes, Location: c.Location, ContactName: c.ContactName, ContactEmail: c.ContactEmail, Phones: c.Phones}
	if req.Name != nil {
		p.Name = v.text("name", *req.Name, 1, 200)
	}
	if req.Notes != nil {
		p.Notes = v.text("notes", *req.Notes, 0, 10000)
	}
	v.contactFields(req, &p.Location, &p.ContactName, &p.ContactEmail)
	if req.Phones != nil {
		p.Phones = v.phonesField(*req.Phones)
	}
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
