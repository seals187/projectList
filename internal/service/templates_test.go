package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/service"
	"github.com/seals187/projectList/internal/testutil"
)

func setupProject(t *testing.T, q *sqlc.Queries) sqlc.Project {
	t.Helper()
	ctx := context.Background()
	c, err := q.CreateCustomer(ctx, sqlc.CreateCustomerParams{Name: "Kunde"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := q.CreateProject(ctx, sqlc.CreateProjectParams{CustomerID: c.ID, Name: "Projekt", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestApplyTemplateCreatesSnapshot(t *testing.T) {
	ctx := context.Background()
	d, q := testutil.DB(t)
	p := setupProject(t, q)

	tmpl, _ := q.CreateTemplate(ctx, sqlc.CreateTemplateParams{Name: "Onboarding", Description: "d"})
	var firstItem sqlc.TemplateItem
	for i, title := range []string{"Vertrag", "Zugänge", "Kickoff"} {
		it, err := q.CreateTemplateItem(ctx, sqlc.CreateTemplateItemParams{TemplateID: tmpl.ID, Kind: "item", Title: title, Description: "Info " + title, Position: int64(i)})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstItem = it
		}
	}

	cl, err := service.ApplyTemplate(ctx, d, p.ID, tmpl.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if cl.Title != "Onboarding" {
		t.Errorf("Titel: erwartet Template-Name, bekommen %q", cl.Title)
	}
	if cl.TemplateID == nil || *cl.TemplateID != tmpl.ID {
		t.Errorf("template_id als Herkunftsreferenz fehlt")
	}

	items, _ := q.ListItemsByChecklist(ctx, cl.ID)
	if len(items) != 3 || items[0].Title != "Vertrag" || items[2].Title != "Kickoff" || items[1].Description != "Info Zugänge" {
		t.Fatalf("Items nicht korrekt kopiert: %+v", items)
	}

	// Template nachträglich ändern: Checkliste darf sich NICHT ändern.
	if _, err := q.UpdateTemplateItem(ctx, sqlc.UpdateTemplateItemParams{ID: firstItem.ID, Kind: "item", Title: "GEÄNDERT"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateTemplateItem(ctx, sqlc.CreateTemplateItemParams{TemplateID: tmpl.ID, Kind: "h2", Title: "Neu", Position: 9}); err != nil {
		t.Fatal(err)
	}
	items, _ = q.ListItemsByChecklist(ctx, cl.ID)
	if len(items) != 3 || items[0].Title != "Vertrag" {
		t.Fatalf("Template-Änderung hat bestehende Checkliste beeinflusst: %+v", items)
	}

	// Template löschen: Checkliste bleibt, template_id wird NULL.
	if err := q.DeleteTemplate(ctx, tmpl.ID); err != nil {
		t.Fatal(err)
	}
	got, err := q.GetChecklistWithCustomer(ctx, cl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.TemplateID != nil {
		t.Error("template_id sollte nach Löschen des Templates NULL sein")
	}
	items, _ = q.ListItemsByChecklist(ctx, cl.ID)
	if len(items) != 3 {
		t.Fatal("Items nach Template-Löschung verloren")
	}

	// Zweite Anwendung mit eigenem Titel hängt hinten an.
	tmpl2, _ := q.CreateTemplate(ctx, sqlc.CreateTemplateParams{Name: "Leer"})
	cl2, err := service.ApplyTemplate(ctx, d, p.ID, tmpl2.ID, "Eigener Titel")
	if err != nil {
		t.Fatal(err)
	}
	if cl2.Title != "Eigener Titel" || cl2.Position <= cl.Position {
		t.Errorf("zweite Checkliste falsch: %+v", cl2)
	}
}

func TestApplyTemplateRollsBackOnError(t *testing.T) {
	ctx := context.Background()
	d, q := testutil.DB(t)
	p := setupProject(t, q)

	_, err := service.ApplyTemplate(ctx, d, p.ID, 9999, "")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("erwartet ErrNoRows, bekommen %v", err)
	}
	// Ungültiges Projekt (FK-Verletzung) → nichts darf übrig bleiben.
	tmpl, _ := q.CreateTemplate(ctx, sqlc.CreateTemplateParams{Name: "T"})
	_, _ = q.CreateTemplateItem(ctx, sqlc.CreateTemplateItemParams{TemplateID: tmpl.ID, Kind: "item", Title: "x"})
	if _, err := service.ApplyTemplate(ctx, d, 4242, tmpl.ID, ""); err == nil {
		t.Fatal("FK-Verletzung erwartet")
	}
	var n int
	if err := d.QueryRow("SELECT COUNT(*) FROM checklist_items").Scan(&n); err != nil || n != 0 {
		t.Fatalf("Transaktion nicht zurückgerollt: %d Items, err=%v", n, err)
	}
}

func TestSaveChecklistAsTemplate(t *testing.T) {
	ctx := context.Background()
	d, q := testutil.DB(t)
	p := setupProject(t, q)
	cl, _ := q.CreateChecklist(ctx, sqlc.CreateChecklistParams{ProjectID: p.ID, Title: "Release"})
	for i, title := range []string{"Build", "Deploy"} {
		if _, err := q.CreateItem(ctx, sqlc.CreateItemParams{ChecklistID: cl.ID, Kind: "item", Title: title, Position: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	tmpl, err := service.SaveChecklistAsTemplate(ctx, d, cl.ID, "Release-Vorlage", "")
	if err != nil {
		t.Fatal(err)
	}
	items, _ := q.ListTemplateItems(ctx, tmpl.ID)
	if len(items) != 2 || items[0].Title != "Build" || items[1].Title != "Deploy" {
		t.Fatalf("Template-Items falsch: %+v", items)
	}
}
