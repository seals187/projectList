package httpapi_test

import (
	"fmt"
	"testing"
)

// The person responsible for a project gets its checklists: new ones, and on a
// change those without a person and those of the previous person responsible.
func TestOwnerTakesChecklists(t *testing.T) {
	e := newEnv(t)
	admin := e.user("admin@example.com", "admin")
	type idOnly struct {
		ID int64 `json:"id"`
	}
	var c, p, tpl idOnly
	admin.do("POST", "/api/customers", map[string]any{"name": "A"}, &c)
	anna := e.user("anna@example.com", "member", c.ID)
	ben := e.user("ben@example.com", "member", c.ID)
	userID := func(cl *client) int64 {
		var me struct {
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
		}
		cl.do("GET", "/api/auth/me", nil, &me)
		return me.User.ID
	}
	annaID, benID := userID(anna), userID(ben)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", c.ID), map[string]any{"name": "P"}, &p)
	newList := func(title string) int64 {
		var cl idOnly
		admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p.ID), map[string]any{"title": title}, &cl)
		return cl.ID
	}
	assignee := func(id int64) int64 {
		var d struct {
			Checklist struct {
				AssigneeID *int64 `json:"assignee_id"`
			} `json:"checklist"`
		}
		admin.do("GET", fmt.Sprintf("/api/checklists/%d", id), nil, &d)
		if d.Checklist.AssigneeID == nil {
			return 0
		}
		return *d.Checklist.AssigneeID
	}
	free := newList("frei")
	bens := newList("ben")
	expect(t, "ben zuweisen", admin.do("PATCH", fmt.Sprintf("/api/checklists/%d", bens), map[string]any{"assignee_id": benID}, nil), 200)
	if assignee(free) != 0 {
		t.Fatal("ohne zuständige person: niemand")
	}

	// Anna becomes responsible: takes the free checklist, Ben's stays
	expect(t, "anna zuständig", admin.do("PATCH", fmt.Sprintf("/api/projects/%d", p.ID), map[string]any{"owner_id": annaID}, nil), 200)
	if assignee(free) != annaID || assignee(bens) != benID {
		t.Fatalf("übernahme: frei=%d ben=%d", assignee(free), assignee(bens))
	}
	// new checklists (empty and from a template) belong to Anna
	if assignee(newList("neu")) != annaID {
		t.Fatal("neue checkliste")
	}
	admin.do("POST", "/api/templates", map[string]any{"name": "T"}, &tpl)
	var fromTpl idOnly
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists/from-template", p.ID), map[string]any{"template_id": tpl.ID}, &fromTpl)
	if assignee(fromTpl.ID) != annaID {
		t.Fatal("checkliste aus template")
	}
	// Ben takes over: Anna's checklists move to him
	expect(t, "ben zuständig", admin.do("PATCH", fmt.Sprintf("/api/projects/%d", p.ID), map[string]any{"owner_id": benID}, nil), 200)
	if assignee(free) != benID || assignee(fromTpl.ID) != benID {
		t.Fatal("wechsel der zuständigen person")
	}
	// nobody responsible: the checklists of the previous person become free
	expect(t, "niemand", admin.do("PATCH", fmt.Sprintf("/api/projects/%d", p.ID), map[string]any{"owner_id": nil}, nil), 200)
	if assignee(free) != 0 || assignee(bens) != 0 {
		t.Fatal("ohne zuständige person")
	}
}
