package httpapi_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/httpapi"
	"github.com/seals187/projectList/internal/report"
)

// Reporting counts checklists (as a whole) and projects, no single items.
func TestReports(t *testing.T) {
	fm := &fakeMailer{}
	e := newEnvWith(t, func(o *httpapi.Options) { o.Mailer = fm; o.AppURL = "https://pl.example" })
	admin := e.user("admin@example.com", "admin")
	type idOnly struct {
		ID int64 `json:"id"`
	}
	var a, b, pa, pb, tag idOnly
	admin.do("POST", "/api/customers", map[string]any{"name": "Kunde A"}, &a)
	admin.do("POST", "/api/customers", map[string]any{"name": "Kunde B"}, &b)
	member := e.user("member@example.com", "member", a.ID)
	viewer := e.user("viewer@example.com", "viewer", a.ID)
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	member.do("GET", "/api/auth/me", nil, &me)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", a.ID), map[string]any{"name": "PA"}, &pa)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", b.ID), map[string]any{"name": "PB"}, &pb)
	admin.do("POST", "/api/tags", map[string]any{"name": "X", "color": "#22c55e"}, &tag)
	admin.do("PUT", fmt.Sprintf("/api/projects/%d/tags", pa.ID), map[string]any{"tag_ids": []int64{tag.ID}}, nil)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	inThree := time.Now().AddDate(0, 0, 3).Format("2006-01-02")

	// checklist with items (due dates), optionally assigned and/or completed
	list := func(projectID int64, title string, assignee *int64, dues []string, done bool) int64 {
		var cl idOnly
		admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", projectID), map[string]any{"title": title}, &cl)
		if assignee != nil {
			expect(t, "zuweisen "+title, admin.do("PATCH", fmt.Sprintf("/api/checklists/%d", cl.ID), map[string]any{"assignee_id": *assignee}, nil), 200)
		}
		for i, due := range dues {
			var it idOnly
			admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl.ID), map[string]any{"title": fmt.Sprintf("%s %d", title, i)}, &it)
			if due != "" {
				admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]any{"due_date": due}, nil)
			}
			if done {
				admin.do("POST", fmt.Sprintf("/api/items/%d/done", it.ID), map[string]bool{"done": true}, nil)
			}
		}
		return cl.ID
	}
	m := me.User.ID
	list(pa.ID, "Erledigt", &m, []string{"", ""}, true)
	list(pa.ID, "Überfällig", &m, []string{"", yesterday}, false)
	list(pa.ID, "Diese Woche", &m, []string{inThree}, false)
	list(pa.ID, "Niemand", nil, []string{""}, false)
	list(pb.ID, "B offen", nil, []string{""}, false)
	list(pb.ID, "Leer", nil, nil, false) // empty checklists are left out

	var rep report.Report
	expect(t, "report admin", admin.do("GET", "/api/reports?period=30", nil, &rep), 200)
	d := rep.Dashboard
	if d.Completed != 1 || d.Open != 4 || d.Overdue != 1 || len(rep.Customers) != 2 || d.Status["planning"] != 2 {
		t.Fatalf("dashboard admin: %+v customers=%d", d, len(rep.Customers))
	}
	if w := d.Weeks[len(d.Weeks)-1]; w.Completed != 1 || w.Created != 5 {
		t.Fatalf("aktuelle woche: %+v", w)
	}
	if len(d.Due) != 2 || d.Due[0].Title != "Überfällig" || d.Due[0].Date != yesterday || d.Due[1].Title != "Diese Woche" {
		t.Fatalf("fällig: %+v", d.Due)
	}
	// visibility: the member sees customer A only
	expect(t, "report member", member.do("GET", "/api/reports", nil, &rep), 200)
	if len(rep.Customers) != 1 || rep.Customers[0].Name != "Kunde A" || rep.Dashboard.Open != 3 {
		t.Fatalf("member sieht: %+v", rep.Customers)
	}
	c := rep.Customers[0]
	if c.Done != 1 || c.Total != 4 || c.Open != 3 || c.Overdue != 1 || c.NextDue != yesterday || c.Projects[0].Total != 4 {
		t.Fatalf("kunde: %+v", c)
	}
	wl := rep.Workload
	if len(wl.People) != 1 || wl.People[0].Open != 2 || wl.People[0].Overdue != 1 || wl.People[0].DueWeek != 1 || wl.People[0].Done != 1 {
		t.Fatalf("auslastung: %+v", wl.People)
	}
	if wl.Unassigned.Total != 1 || wl.Unassigned.Checklists[0].Title != "Niemand" || wl.Unassigned.Checklists[0].Project != "PA" {
		t.Fatalf("nicht zugewiesen: %+v", wl.Unassigned)
	}
	// filters: tag (only PA), person
	admin.do("GET", fmt.Sprintf("/api/reports?tags=%d", tag.ID), nil, &rep)
	if rep.Dashboard.Open != 3 {
		t.Fatalf("tag-filter: %+v", rep.Dashboard)
	}
	admin.do("GET", fmt.Sprintf("/api/reports?assignees=%d", m), nil, &rep)
	if rep.Dashboard.Open != 2 || len(rep.Assignees) != 1 {
		t.Fatalf("person-filter: %+v %+v", rep.Dashboard, rep.Assignees)
	}
	// customer report: checklists, own customers only
	var cr report.CustomerReport
	expect(t, "kundenbericht", member.do("GET", fmt.Sprintf("/api/reports/customers/%d", a.ID), nil, &cr), 200)
	if len(cr.Completed) != 1 || cr.Completed[0].Title != "Erledigt" || len(cr.Overdue) != 1 || len(cr.Upcoming) != 1 || len(cr.Open) != 3 ||
		cr.Overdue[0].Done != 0 || cr.Overdue[0].Total != 2 {
		t.Fatalf("kundenbericht: %+v", cr)
	}
	expect(t, "fremder kunde", member.do("GET", fmt.Sprintf("/api/reports/customers/%d", b.ID), nil, nil), 404)

	// schedules: members for themselves, viewers not at all, send now
	var sc idOnly
	expect(t, "viewer plant", viewer.do("POST", "/api/report-schedules", map[string]any{"freq": "weekly"}, nil), 403)
	expect(t, "fremder empfänger", member.do("POST", "/api/report-schedules", map[string]any{"recipient_id": 1}, nil), 400)
	expect(t, "ungültig", member.do("POST", "/api/report-schedules", map[string]any{"freq": "daily"}, nil), 400)
	expect(t, "planen", member.do("POST", "/api/report-schedules", map[string]any{"freq": "monthly", "day": 31, "lang": "de", "filters": map[string]any{"period": 7}}, &sc), 201)
	expect(t, "jetzt senden", member.do("POST", fmt.Sprintf("/api/report-schedules/%d/send", sc.ID), nil, nil), 204)
	mail := fm.last()
	if mail.To != "member@example.com" || !strings.Contains(mail.Body, "Kunde A") || strings.Contains(mail.Body, "Kunde B") ||
		!strings.Contains(mail.Body, "https://pl.example/reports") || !strings.Contains(mail.Body, "Deine offenen Checklisten: 2 (1 überfällig)") {
		t.Fatalf("mail: %s\n%s", mail.Subject, mail.Body)
	}
	var sched struct {
		Schedules []struct {
			ID      int64         `json:"id"`
			Filters report.Filter `json:"filters"`
		} `json:"schedules"`
	}
	admin.do("GET", "/api/report-schedules", nil, &sched)
	if len(sched.Schedules) != 1 || sched.Schedules[0].Filters.Days != 7 {
		t.Fatalf("admin sieht pläne: %+v", sched)
	}
	expect(t, "löschen", member.do("DELETE", fmt.Sprintf("/api/report-schedules/%d", sc.ID), nil, nil), 204)
}

func TestScheduleDue(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	if !report.ScheduleDue(sqlc.ReportSchedule{Freq: "weekly", Weekday: 1}, d("2026-11-09")) ||
		report.ScheduleDue(sqlc.ReportSchedule{Freq: "weekly", Weekday: 1}, d("2026-11-10")) ||
		!report.ScheduleDue(sqlc.ReportSchedule{Freq: "monthly", Day: 31}, d("2026-11-30")) ||
		report.ScheduleDue(sqlc.ReportSchedule{Freq: "monthly", Day: 31}, d("2026-11-29")) {
		t.Fatal("ScheduleDue")
	}
}

// Reporting settings: defaults, validation, and the effect of each option on
// the report, the customer report and the e-mail.
func TestReportConfig(t *testing.T) {
	fm := &fakeMailer{}
	e := newEnvWith(t, func(o *httpapi.Options) { o.Mailer = fm })
	admin := e.user("admin@example.com", "admin")
	type idOnly struct {
		ID int64 `json:"id"`
	}
	var cust, p1, p2, l1, l2 idOnly
	admin.do("POST", "/api/customers", map[string]any{"name": "Kunde A"}, &cust)
	member := e.user("member@example.com", "member", cust.ID)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", cust.ID), map[string]any{"name": "P1"}, &p1)
	admin.do("POST", fmt.Sprintf("/api/customers/%d/projects", cust.ID), map[string]any{"name": "P2"}, &p2)
	var me struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	admin.do("GET", "/api/auth/me", nil, &me)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	in20 := time.Now().AddDate(0, 0, 20).Format("2006-01-02")
	item := func(cl int64, due string, done bool) {
		var it idOnly
		admin.do("POST", fmt.Sprintf("/api/checklists/%d/items", cl), map[string]any{"title": "x"}, &it)
		if due != "" {
			admin.do("PATCH", fmt.Sprintf("/api/items/%d", it.ID), map[string]any{"due_date": due}, nil)
		}
		if done {
			admin.do("POST", fmt.Sprintf("/api/items/%d/done", it.ID), map[string]bool{"done": true}, nil)
		}
	}
	// L1 (P1): one item done, open items due yesterday and in 20 days
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p1.ID), map[string]any{"title": "L1"}, &l1)
	admin.do("PATCH", fmt.Sprintf("/api/checklists/%d", l1.ID), map[string]any{"assignee_id": me.User.ID}, nil)
	item(l1.ID, "", true)
	item(l1.ID, yesterday, false)
	item(l1.ID, in20, false)
	item(l1.ID, "", false)
	// L2 (P2, status done): one open item
	admin.do("POST", fmt.Sprintf("/api/projects/%d/checklists", p2.ID), map[string]any{"title": "L2"}, &l2)
	item(l2.ID, "", false)
	admin.do("PATCH", fmt.Sprintf("/api/projects/%d", p2.ID), map[string]any{"status": "done"}, nil)

	type cfgResp struct {
		Config   report.Config `json:"config"`
		Defaults report.Config `json:"defaults"`
	}
	var cr cfgResp
	expect(t, "config lesen (member)", member.do("GET", "/api/reports/config", nil, &cr), 200)
	if cr.Config.DefaultPeriod != 30 || cr.Config.DueSoonDays != 14 || cr.Config.ProgressBasis != "checklists" || !cr.Config.IncludeDone {
		t.Fatalf("defaults: %+v", cr.Config)
	}

	// defaults = behavior so far: checklist due date (latest item) → not overdue, not due soon
	var rep report.Report
	admin.do("GET", "/api/reports?period=30", nil, &rep)
	if rep.Dashboard.Open != 2 || rep.Dashboard.Overdue != 0 || len(rep.Dashboard.Due) != 0 || rep.Customers[0].Done != 0 || rep.Customers[0].Total != 2 {
		t.Fatalf("standard: %+v %+v", rep.Dashboard, rep.Customers)
	}

	cfg := cr.Defaults
	expect(t, "member darf nicht", member.do("PUT", "/api/reports/config", cfg, nil), 403)
	bad := cfg
	bad.Views = []string{}
	expect(t, "keine ansicht", admin.do("PUT", "/api/reports/config", bad, nil), 400)
	bad = cfg
	bad.DefaultPeriod = 14 // not among the periods
	expect(t, "standardzeitraum fehlt", admin.do("PUT", "/api/reports/config", bad, nil), 400)
	bad = cfg
	bad.DueSoonDays = 0
	expect(t, "fenster 0", admin.do("PUT", "/api/reports/config", bad, nil), 400)

	cfg.Periods = []int{60, 14}
	cfg.DefaultPeriod = 60
	cfg.ProgressBasis = "items"
	cfg.DueBasis = "item"
	cfg.DueSoonDays = 30
	cfg.IncludeDone = false
	cfg.OverloadThreshold = 1
	cfg.Views = []string{"dashboard", "workload"}
	cfg.KPIs = []string{"overdue"}
	expect(t, "config speichern", admin.do("PUT", "/api/reports/config", cfg, &cr), 200)
	if cr.Config.Periods[0] != 14 { // stored in ascending order
		t.Fatalf("zeiträume: %+v", cr.Config.Periods)
	}

	admin.do("GET", "/api/reports?period=7", nil, &rep) // 7 is not selectable → default period
	if rep.Days != 60 || rep.Config.DefaultPeriod != 60 {
		t.Fatalf("zeitraum: %d", rep.Days)
	}
	// item basis: overdue (open item yesterday), due soon from the earliest open item;
	// done project left out; progress in items (1 of 4)
	if rep.Dashboard.Open != 1 || rep.Dashboard.Overdue != 1 || len(rep.Dashboard.Due) != 1 || rep.Dashboard.Due[0].Date != yesterday {
		t.Fatalf("item-basis: %+v", rep.Dashboard)
	}
	if c := rep.Customers[0]; c.Done != 1 || c.Total != 4 || len(c.Projects) != 1 {
		t.Fatalf("fortschritt items: %+v", c)
	}
	if p := rep.Workload.People; len(p) != 1 || !p[0].Overloaded {
		t.Fatalf("überlastet: %+v", p)
	}
	// customer report: same rules (upcoming window 30 days from the earliest open item = overdue here)
	var crep report.CustomerReport
	admin.do("GET", fmt.Sprintf("/api/reports/customers/%d", cust.ID), nil, &crep)
	if len(crep.Overdue) != 1 || len(crep.Open) != 1 || crep.DueSoonDays != 30 || crep.ProgressBasis != "items" {
		t.Fatalf("kundenbericht: %+v", crep)
	}

	// customer progress: P1 excluded → only counts when the setting is off
	admin.do("PATCH", fmt.Sprintf("/api/projects/%d", p1.ID), map[string]any{"include_in_customer_progress": false}, nil)
	admin.do("GET", "/api/reports", nil, &rep)
	if rep.Customers[0].Total != 0 {
		t.Fatalf("ausgeschlossen: %+v", rep.Customers[0])
	}
	cfg.RespectCustomerProgress = false
	admin.do("PUT", "/api/reports/config", cfg, nil)
	admin.do("GET", "/api/reports", nil, &rep)
	if rep.Customers[0].Total != 4 {
		t.Fatalf("ausschluss ignoriert: %+v", rep.Customers[0])
	}

	// e-mail: only the chosen key figures, no customers section (view off), window 30
	var sc idOnly
	admin.do("POST", "/api/report-schedules", map[string]any{"freq": "weekly", "lang": "de", "filters": map[string]any{"period": 60}}, &sc)
	expect(t, "senden", admin.do("POST", fmt.Sprintf("/api/report-schedules/%d/send", sc.ID), nil, nil), 204)
	body := fm.last().Body
	if !strings.Contains(body, "Überfällig: 1") || strings.Contains(body, "Erledigte Checklisten") || strings.Contains(body, "Kunden:") ||
		!strings.Contains(body, "Fällig in den nächsten 30 Tagen") {
		t.Fatalf("mail:\n%s", body)
	}
}
