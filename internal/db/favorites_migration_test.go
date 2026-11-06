package db

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/seals187/projectList/migrations"
)

// Migration 00022 takes over the old client pref "fav_layout" (folders and
// order of the favorites) into favorite_groups / user_favorites.
func TestMigrateFavoriteLayout(t *testing.T) {
	ctx := context.Background()
	d, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := goose.NewProvider(goose.DialectSQLite3, d, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpTo(ctx, 21); err != nil {
		t.Fatal(err)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := d.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO users (id, email, name, password_hash, role) VALUES (1, 'a@x', 'A', 'x', 'admin'), (2, 'b@x', 'B', 'x', 'member')`)
	exec(`INSERT INTO customers (id, name) VALUES (1, 'K1'), (2, 'K2')`)
	exec(`INSERT INTO projects (id, customer_id, name) VALUES (10, 1, 'P10'), (11, 1, 'P11'), (12, 2, 'P12')`)
	exec(`INSERT INTO user_favorites (user_id, kind, target_id) VALUES
		(1, 'project', 10), (1, 'project', 11), (1, 'customer', 1), (1, 'project', 12), (2, 'customer', 2)`)
	// user 1: P11, folder "Wichtig" (closed) with [customer 1, P10]; P12 not mentioned → after the others
	exec(`INSERT INTO user_prefs (user_id, key, value) VALUES (1, 'fav_layout', ?)`,
		`[{"t":"i","k":"project:11"},{"t":"f","id":"abc","name":"Wichtig","open":false,"items":["customer:1","project:10","gone:9"]},{"t":"f","id":"e","name":"Leer","open":true,"items":[]}]`)
	// user 2: malformed layout is ignored
	exec(`INSERT INTO user_prefs (user_id, key, value) VALUES (2, 'fav_layout', 'not json')`)

	if _, err := p.UpTo(ctx, 22); err != nil {
		t.Fatal(err)
	}

	type grp struct {
		id, pos   int64
		name      string
		collapsed bool
	}
	rows, err := d.QueryContext(ctx, `SELECT id, name, position, collapsed FROM favorite_groups WHERE user_id = 1 ORDER BY position`)
	if err != nil {
		t.Fatal(err)
	}
	var groups []grp
	for rows.Next() {
		var g grp
		if err := rows.Scan(&g.id, &g.name, &g.pos, &g.collapsed); err != nil {
			t.Fatal(err)
		}
		groups = append(groups, g)
	}
	rows.Close()
	if len(groups) != 2 || groups[0].name != "Wichtig" || !groups[0].collapsed || groups[1].name != "Leer" || groups[1].collapsed {
		t.Fatalf("gruppen: %+v", groups)
	}

	place := func(user int64, kind string, id int64) (gid *int64, pos int64) {
		t.Helper()
		if err := d.QueryRowContext(ctx, `SELECT group_id, position FROM user_favorites WHERE user_id = ? AND kind = ? AND target_id = ?`, user, kind, id).Scan(&gid, &pos); err != nil {
			t.Fatalf("%s %d: %v", kind, id, err)
		}
		return gid, pos
	}
	if g, pos := place(1, "project", 11); g != nil || pos != 0 {
		t.Fatalf("P11: %v %d", g, pos)
	}
	if g, pos := place(1, "customer", 1); g == nil || *g != groups[0].id || pos != 0 {
		t.Fatalf("K1: %v %d", g, pos)
	}
	if g, pos := place(1, "project", 10); g == nil || *g != groups[0].id || pos != 1 {
		t.Fatalf("P10: %v %d", g, pos)
	}
	if g, pos := place(1, "project", 12); g != nil || pos <= 0 {
		t.Fatalf("P12 (nicht im layout) soll hinter P11 stehen: %v %d", g, pos)
	}
	if g, _ := place(2, "customer", 2); g != nil {
		t.Fatalf("user 2 ohne gültiges layout: %v", g)
	}
	var n int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_prefs WHERE key = 'fav_layout'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("fav_layout prefs übrig: %d %v", n, err)
	}

	// cleanup triggers work on the rebuilt table, templates included
	exec(`DELETE FROM projects WHERE id = 10`)
	exec(`INSERT INTO checklist_templates (id, name) VALUES (5, 'T')`)
	exec(`INSERT INTO user_favorites (user_id, kind, target_id) VALUES (1, 'template', 5)`)
	exec(`DELETE FROM checklist_templates WHERE id = 5`)
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_favorites WHERE (kind = 'project' AND target_id = 10) OR kind = 'template'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("trigger: %d %v", n, err)
	}
	// deleting a group ungroups its favorites (ON DELETE SET NULL)
	exec(`DELETE FROM favorite_groups WHERE id = ?`, groups[0].id)
	if g, _ := place(1, "customer", 1); g != nil {
		t.Fatalf("set null: %v", g)
	}

	// down and up again
	if _, err := p.DownTo(ctx, 21); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := p.UpTo(ctx, 22); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
