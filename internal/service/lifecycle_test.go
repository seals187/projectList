package service_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/service"
	"github.com/seals187/projectList/internal/testutil"
)

type event struct {
	kind   string
	status string
	extra  map[string]string
}

// lifecycle returns a Lifecycle on a fresh database that records its events.
func lifecycle(t *testing.T) (*service.Lifecycle, *sqlc.Queries, *[]event) {
	t.Helper()
	d, q := testutil.DB(t)
	var events []event
	l := &service.Lifecycle{DB: d, Q: q, Log: testutil.Logger(), Emit: func(_ context.Context, kind string, p sqlc.Project, extra map[string]string) {
		events = append(events, event{kind, p.Status, extra})
	}}
	return l, q, &events
}

func activityKinds(t *testing.T, q *sqlc.Queries, projectID int64) []string {
	t.Helper()
	rows, err := q.ListProjectActivity(context.Background(), sqlc.ListProjectActivityParams{ProjectID: projectID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, r := range rows {
		kinds = append(kinds, r.Kind+":"+r.Subject+">"+r.Detail)
	}
	slices.Sort(kinds)
	return kinds
}

func TestMoveOnBoardHasStatusSideEffects(t *testing.T) {
	ctx := context.Background()
	l, q, events := lifecycle(t)
	p := setupProject(t, q) // status "active"
	other, _ := q.CreateProject(ctx, sqlc.CreateProjectParams{CustomerID: p.CustomerID, Name: "B", Status: "review"})
	user := testutil.CreateUser(t, q, "a@example.com", "pw-123456789", "member")

	if err := l.MoveOnBoard(ctx, service.UserActor(user.ID, user.Name), "review", []int64{p.ID, other.ID}); err != nil {
		t.Fatal(err)
	}
	got, _ := q.GetProject(ctx, p.ID)
	if got.Status != "review" || got.BoardPosition != 0 {
		t.Fatalf("project not moved: %+v", got)
	}
	if k := activityKinds(t, q, p.ID); !slices.Equal(k, []string{"status:active>review"}) {
		t.Errorf("activity of the moved project: %v", k)
	}
	if k := activityKinds(t, q, other.ID); len(k) != 0 {
		t.Errorf("project that stayed in its column must not log: %v", k)
	}
	if len(*events) != 1 || (*events)[0].kind != service.EventProjectStatus || (*events)[0].status != "review" {
		t.Errorf("events: %+v", *events)
	}
}

func TestMoveOnBoardUnknownProjectChangesNothing(t *testing.T) {
	ctx := context.Background()
	l, q, events := lifecycle(t)
	p := setupProject(t, q)
	err := l.MoveOnBoard(ctx, service.Automation, "done", []int64{p.ID, 99999})
	if err == nil {
		t.Fatal("expected an error")
	}
	got, _ := q.GetProject(ctx, p.ID)
	if got.Status != "active" || len(*events) != 0 {
		t.Errorf("must be atomic: status %q, events %v", got.Status, *events)
	}
}

func TestUpdateProjectLogsAndEmitsOnlyChanges(t *testing.T) {
	ctx := context.Background()
	l, q, events := lifecycle(t)
	p := setupProject(t, q)
	user := testutil.CreateUser(t, q, "a@example.com", "pw-123456789", "member")
	a := service.UserActor(user.ID, user.Name)

	up := sqlc.UpdateProjectParams{Name: "Neu", Description: p.Description, Status: p.Status, IncludeInCustomerProgress: true}
	if _, err := l.UpdateProject(ctx, a, p, up); err != nil {
		t.Fatal(err)
	}
	if len(*events) != 0 {
		t.Errorf("rename must not emit a status event: %v", *events)
	}
	p, _ = q.GetProject(ctx, p.ID)
	up.Status = "done"
	if _, err := l.UpdateProject(ctx, a, p, up); err != nil {
		t.Fatal(err)
	}
	want := []string{"renamed:Projekt>Neu", "status:active>done"}
	if k := activityKinds(t, q, p.ID); !slices.Equal(k, want) {
		t.Errorf("activity %v, want %v", k, want)
	}
	if len(*events) != 1 || (*events)[0].kind != service.EventProjectStatus {
		t.Errorf("events: %+v", *events)
	}
}

func TestSetTagsEmitsAddedTagsOnly(t *testing.T) {
	ctx := context.Background()
	l, q, events := lifecycle(t)
	p := setupProject(t, q)
	t1, _ := q.CreateTag(ctx, sqlc.CreateTagParams{Name: "eins", Color: "red"})
	t2, _ := q.CreateTag(ctx, sqlc.CreateTagParams{Name: "zwei", Color: "blue"})

	if _, err := l.SetTags(ctx, service.Automation, p.ID, []int64{t1.ID}, []string{"eins"}); err != nil {
		t.Fatal(err)
	}
	ids, err := l.SetTags(ctx, service.Automation, p.ID, []int64{t1.ID, t2.ID}, []string{"eins", "zwei"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || len(*events) != 2 || (*events)[1].extra["tag"] == (*events)[0].extra["tag"] {
		t.Fatalf("ids %v events %+v", ids, *events)
	}
	if _, err := l.SetTags(ctx, service.Automation, p.ID, []int64{t1.ID, t2.ID}, nil); err != nil {
		t.Fatal(err)
	}
	if len(*events) != 2 || len(activityKinds(t, q, p.ID)) != 2 {
		t.Errorf("unchanged tags must neither log nor emit")
	}
}

func TestInstantiateChecklistAndTable(t *testing.T) {
	ctx := context.Background()
	l, q, _ := lifecycle(t)
	p := setupProject(t, q)
	ct, _ := q.CreateTemplate(ctx, sqlc.CreateTemplateParams{Name: "Liste {{project.name}}"})
	if _, err := q.CreateTemplateItem(ctx, sqlc.CreateTemplateItemParams{TemplateID: ct.ID, Kind: "item", Title: "x"}); err != nil {
		t.Fatal(err)
	}
	tt, err := q.CreateTemplate(ctx, sqlc.CreateTemplateParams{Name: "Tabelle"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.DB.ExecContext(ctx, `UPDATE checklist_templates SET kind = 'table' WHERE id = ?`, tt.ID); err != nil {
		t.Fatal(err)
	}

	inst, err := l.Instantiate(ctx, service.Automation, service.InstantiateReq{ProjectID: p.ID, TemplateID: ct.ID})
	if err != nil || inst.Checklist == nil || inst.Table != nil {
		t.Fatalf("checklist: %+v %v", inst, err)
	}
	if inst.Checklist.Title != "Liste Projekt" {
		t.Errorf("variables not resolved: %q", inst.Checklist.Title)
	}
	inst, err = l.Instantiate(ctx, service.Automation, service.InstantiateReq{ProjectID: p.ID, TemplateID: tt.ID})
	if err != nil || inst.Table == nil {
		t.Fatalf("table: %+v %v", inst, err)
	}
	if _, err := l.Instantiate(ctx, service.Automation, service.InstantiateReq{ProjectID: p.ID, TemplateID: tt.ID, Kind: service.KindChecklist}); !errors.Is(err, service.ErrTemplateKind) {
		t.Errorf("kind mismatch: %v", err)
	}
	want := []string{"checklist_created:Liste Projekt>automation", "table_created:Tabelle>automation"}
	if k := activityKinds(t, q, p.ID); !slices.Equal(k, want) {
		t.Errorf("activity %v, want %v", k, want)
	}
}

func TestReorderRejectsForeignIDs(t *testing.T) {
	ctx := context.Background()
	l, q, _ := lifecycle(t)
	p := setupProject(t, q)
	a, _ := q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: p.ID, Title: "a", Position: 0})
	b, _ := q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: p.ID, Title: "b", Position: 1})
	other, _ := q.CreateProject(ctx, sqlc.CreateProjectParams{CustomerID: p.CustomerID, Name: "o", Status: "active"})
	c, _ := q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: other.ID, Title: "c", Position: 0})

	if err := service.Reorder(ctx, l.DB, service.ChecklistsOfProject, p.ID, []int64{b.ID, c.ID}); !errors.Is(err, service.ErrInvalidOrder) {
		t.Fatalf("foreign id: %v", err)
	}
	if err := service.Reorder(ctx, l.DB, service.ChecklistsOfProject, p.ID, []int64{b.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	cls, _ := q.ListChecklistsByProject(ctx, p.ID)
	if len(cls) != 2 || cls[0].ID != b.ID {
		t.Errorf("order not applied: %+v", cls)
	}
}

func TestInsertItemAtPositionShiftsTheRest(t *testing.T) {
	ctx := context.Background()
	l, q, _ := lifecycle(t)
	p := setupProject(t, q)
	cl, _ := q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: p.ID, Title: "c"})
	add := func(title string, at *int64) {
		t.Helper()
		if _, err := service.InsertItem(ctx, l.DB, service.Block{Kind: "item", Title: title}.Item(cl.ID, 0), at); err != nil {
			t.Fatal(err)
		}
	}
	zero, far := int64(0), int64(99)
	add("b", nil)
	add("c", &far)
	add("a", &zero)
	items, _ := q.ListItemsByChecklist(ctx, cl.ID)
	var titles []string
	for _, it := range items {
		titles = append(titles, it.Title)
	}
	if !slices.Equal(titles, []string{"a", "b", "c"}) {
		t.Errorf("order %v", titles)
	}
}

func TestDuplicateCopiesContentAndEmitsCreated(t *testing.T) {
	ctx := context.Background()
	l, q, events := lifecycle(t)
	p := setupProject(t, q)
	cl, _ := q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: p.ID, Title: "c"})
	if _, err := q.CreateItem(ctx, sqlc.CreateItemParams{ChecklistID: cl.ID, Kind: "item", Title: "x", Status: "done", Indent: 1}); err != nil {
		t.Fatal(err)
	}
	cp, err := l.Duplicate(ctx, service.Automation, p, "Kopie")
	if err != nil {
		t.Fatal(err)
	}
	if cp.Status != "planning" || cp.Name != "Kopie" {
		t.Errorf("copy: %+v", cp)
	}
	cls, _ := q.ListChecklistsByProject(ctx, cp.ID)
	items, _ := q.ListItemsByChecklist(ctx, cls[0].ID)
	if len(items) != 1 || items[0].Title != "x" || items[0].Indent != 1 || items[0].Status == "done" {
		t.Errorf("items not copied as fresh entries: %+v", items)
	}
	if len(*events) != 1 || (*events)[0].kind != service.EventProjectCreated {
		t.Errorf("events: %+v", *events)
	}
}
