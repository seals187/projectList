package httpapi_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/seals187/projectList/internal/db/sqlc"
)

// TestCustomerScope pins the access rule for every customer-owned entity:
// admins and assigned users see it, everyone else gets 404 (as for an id that
// does not exist), independent of the role.
func TestCustomerScope(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	c, _ := e.q.CreateCustomer(ctx, sqlc.CreateCustomerParams{Name: "Kunde"})
	p, _ := e.q.CreateProject(ctx, sqlc.CreateProjectParams{CustomerID: c.ID, Name: "P", Status: "active"})
	cl, _ := e.q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: p.ID, Title: "L"})
	tbl, err := e.q.CreateTable(ctx, sqlc.CreateTableParams{ProjectID: p.ID, Title: "T", Data: "[]", CreatedAt: "x", UpdatedAt: "x"})
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{
		fmt.Sprintf("/api/customers/%d", c.ID),
		fmt.Sprintf("/api/customers/%d/projects", c.ID),
		fmt.Sprintf("/api/projects/%d", p.ID),
		fmt.Sprintf("/api/projects/%d/checklists", p.ID),
		fmt.Sprintf("/api/projects/%d/tables", p.ID),
		fmt.Sprintf("/api/projects/%d/activity", p.ID),
		fmt.Sprintf("/api/projects/%d/files", p.ID),
		fmt.Sprintf("/api/checklists/%d", cl.ID),
		fmt.Sprintf("/api/tables/%d", tbl.ID),
	}
	cases := []struct {
		name      string
		role      string
		assigned  bool
		wantFound bool
	}{
		{"admin", "admin", false, true},
		{"member assigned", "member", true, true},
		{"viewer assigned", "viewer", true, true},
		{"member not assigned", "member", false, false},
		{"viewer not assigned", "viewer", false, false},
	}
	for i, tc := range cases {
		var ids []int64
		if tc.assigned {
			ids = []int64{c.ID}
		}
		u := e.user(fmt.Sprintf("u%d@example.com", i), tc.role, ids...)
		for _, path := range paths {
			want := 404
			if tc.wantFound {
				want = 200
			}
			expect(t, tc.name+" GET "+path, u.do("GET", path, nil, nil), want)
		}
		// favorites resolve their target through the same scope
		want := 404
		if tc.wantFound {
			want = 204
		}
		expect(t, tc.name+" favorite project", u.do("PUT", fmt.Sprintf("/api/favorites/project/%d", p.ID), nil, nil), want)
	}
}

// TestBoardMoveLogsAndTriggersStatusChange: moving a card into another board
// column is a status change like PATCH (activity log entry).
func TestBoardMoveLogsStatusChange(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	c, _ := e.q.CreateCustomer(ctx, sqlc.CreateCustomerParams{Name: "Kunde"})
	p, _ := e.q.CreateProject(ctx, sqlc.CreateProjectParams{CustomerID: c.ID, Name: "P", Status: "active"})
	u := e.user("m@example.com", "member", c.ID)

	expect(t, "board move", u.do("PUT", "/api/board/order", map[string]any{"status": "review", "ids": []int64{p.ID}}, nil), 204)
	var act []struct {
		Kind    string `json:"kind"`
		Subject string `json:"subject"`
		Detail  string `json:"detail"`
	}
	expect(t, "activity", u.do("GET", fmt.Sprintf("/api/projects/%d/activity", p.ID), nil, &act), 200)
	if len(act) != 1 || act[0].Kind != "status" || act[0].Subject != "active" || act[0].Detail != "review" {
		t.Errorf("activity after board move: %+v", act)
	}
	// reordering inside the same column is not a change
	expect(t, "same column", u.do("PUT", "/api/board/order", map[string]any{"status": "review", "ids": []int64{p.ID}}, nil), 204)
	expect(t, "activity", u.do("GET", fmt.Sprintf("/api/projects/%d/activity", p.ID), nil, &act), 200)
	if len(act) != 1 {
		t.Errorf("reorder in the same column logged: %+v", act)
	}
	expect(t, "viewer may not move", e.user("v@example.com", "viewer", c.ID).do("PUT", "/api/board/order", map[string]any{"status": "done", "ids": []int64{p.ID}}, nil), 403)
}
