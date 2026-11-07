// Package report computes the reporting (dashboard, customers, workload,
// templates) over the visible customers. Units are checklists (as a whole) and
// projects – never single items: a checklist is done when all its items are
// checked (done at = the last item), its due date is the latest due date of its
// items, its person the checklist's assignee. Empty checklists are left out.
// Everything is aggregated in Go: the data of a self-hosted instance is small.
package report

import (
	"context"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/db/sqlc"
)

// Filter of a report. Empty lists = all.
type Filter struct {
	Days      int     `json:"period"` // length of the period in days (7, 30, 90, 365)
	Customers []int64 `json:"customers"`
	Assignees []int64 `json:"assignees"`
	Tags      []int64 `json:"tags"`
}

// Periods allowed for Filter.Days.
var Periods = []int{7, 30, 90, 365}

// Normalize sets a valid period.
func (f *Filter) Normalize() {
	if !slices.Contains(Periods, f.Days) {
		f.Days = 30
	}
}

// ---- result ----------------------------------------------------------------------------

type Report struct {
	From      string    `json:"from"` // YYYY-MM-DD (first day of the period)
	To        string    `json:"to"`   // today
	Days      int       `json:"days"`
	Dashboard Dashboard `json:"dashboard"`
	Customers []Cust    `json:"customers"`
	Workload  Workload  `json:"workload"`
	Templates Templates `json:"templates"`
	Assignees []UserRef `json:"assignees"` // persons of checklists in the visible customers (filter)
}

type UserRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Dashboard: checklists completed / open / overdue, projects done.
type Dashboard struct {
	Completed     int            `json:"completed"`
	CompletedPrev int            `json:"completed_prev"`
	Open          int            `json:"open"`
	OpenPrev      int            `json:"open_prev"` // open at the start of the period
	Overdue       int            `json:"overdue"`
	OverduePrev   int            `json:"overdue_prev"`
	ProjectsDone  int            `json:"projects_done"`
	Weeks         []Week         `json:"weeks"`
	Status        map[string]int `json:"status"`
	Due           []Due          `json:"due"`
}

type Week struct {
	Label     string `json:"label"` // "W44"
	Start     string `json:"start"`
	Completed int    `json:"completed"` // checklists completed
	Created   int    `json:"created"`   // checklists created
}

type Due struct {
	Date     string `json:"date"`
	Kind     string `json:"kind"` // project | checklist
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Customer string `json:"customer"`
}

type Cust struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Projects []CustProj `json:"projects"`
	Done     int        `json:"done"`    // completed checklists (projects counting toward the customer)
	Total    int        `json:"total"`   // checklists (…)
	Open     int        `json:"open"`    // open checklists
	Overdue  int        `json:"overdue"` // overdue checklists
	NextDue  string     `json:"next_due"`
}

type CustProj struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Done   int    `json:"done"`  // completed checklists
	Total  int    `json:"total"` // checklists
}

type Workload struct {
	People     []Person   `json:"people"`
	Unassigned Unassigned `json:"unassigned"`
	DonePrev   int        `json:"done_prev"`
}

// Person: open checklists assigned to them.
type Person struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Open    int    `json:"open"`
	DueWeek int    `json:"due_week"` // of them due within 7 days
	Overdue int    `json:"overdue"`  // of them overdue
	Done    int    `json:"done"`     // completed in the period
}

type Unassigned struct {
	Total      int       `json:"total"`
	Checklists []ClBrief `json:"checklists"`
}

// ClBrief: a checklist in lists (unassigned, longest, customer report).
type ClBrief struct {
	ID       int64    `json:"id"`
	Title    string   `json:"title"`
	Project  string   `json:"project"`
	Customer string   `json:"customer"`
	Template string   `json:"template,omitempty"`
	Date     string   `json:"date"` // due date or completion date (YYYY-MM-DD)
	Done     int      `json:"done"` // checked items (progress of the checklist)
	Total    int      `json:"total"`
	Days     *float64 `json:"days,omitempty"` // duration (longest)
}

type Templates struct {
	Rows        []TplRow  `json:"rows"`
	Created     int       `json:"created"` // checklists from templates in the period
	AvgDays     *float64  `json:"avg_days"`
	AvgDaysPrev *float64  `json:"avg_days_prev"`
	OnTime      *float64  `json:"on_time"` // percent
	OnTimePrev  *float64  `json:"on_time_prev"`
	Longest     []ClBrief `json:"longest"` // completed checklists that took longest
	AutoCreated int       `json:"auto_created"`
	AutoWarned  int       `json:"auto_warned"`
}

type TplRow struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	CategoryID *int64   `json:"category_id"`
	Used       int      `json:"used"`     // applications in the period
	AvgDays    *float64 `json:"avg_days"` // completed checklists: creation → done
	Planned    *float64 `json:"planned"`  // creation → due date
	OnTime     *float64 `json:"on_time"`  // percent of completed checklists with a due date
	Completed  int      `json:"completed"`
}

// ---- checklists from the raw items -----------------------------------------------------

type checklist struct {
	ID, ProjectID, CustomerID int64
	Title                     string
	TemplateID                *int64
	Assignee                  *int64
	Created                   time.Time
	CreatedKnown              bool
	Total, Done               int
	LastDone                  time.Time
	Due                       string // latest due date of the items, "" = none
}

func (c *checklist) complete() bool { return c.Total > 0 && c.Done == c.Total }

func aggregate(items []sqlc.ReportItemsRow) []*checklist {
	byID := map[int64]*checklist{}
	var order []*checklist
	for _, it := range items {
		c := byID[it.ChecklistID]
		if c == nil {
			c = &checklist{ID: it.ChecklistID, ProjectID: it.ProjectID, CustomerID: it.CustomerID, Title: it.ChecklistTitle,
				TemplateID: it.TemplateID, Assignee: it.ChecklistAssigneeID}
			c.Created, c.CreatedKnown = parseTime(it.ChecklistCreatedAt)
			byID[it.ChecklistID] = c
			order = append(order, c)
		}
		c.Total++
		if it.DoneAt != nil {
			c.Done++
			if it.DoneAt.After(c.LastDone) {
				c.LastDone = *it.DoneAt
			}
		}
		if it.DueDate != nil && len(*it.DueDate) >= 10 && (*it.DueDate)[:10] > c.Due {
			c.Due = (*it.DueDate)[:10]
		}
	}
	return order
}

// ---- helpers ---------------------------------------------------------------------------

func parseTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	return t, err == nil
}

func round1(v float64) *float64 {
	r := math.Round(v*10) / 10
	return &r
}

func avg(xs []float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return round1(s / float64(len(xs)))
}

func dayOf(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// ---- computation -----------------------------------------------------------------------

// data: everything Build needs, loaded once (also used by the customer report).
type data struct {
	custName  map[int64]string
	customers []sqlc.Customer
	projects  []sqlc.ReportProjectsRow
	projOK    map[int64]sqlc.ReportProjectsRow
	lists     []*checklist // filtered, non-empty
	names     map[int64]string
	assignees []UserRef
}

func load(ctx context.Context, q *sqlc.Queries, userFilter any, f Filter) (*data, error) {
	customers, err := q.ListCustomers(ctx, userFilter)
	if err != nil {
		return nil, err
	}
	projects, err := q.ReportProjects(ctx, userFilter)
	if err != nil {
		return nil, err
	}
	items, err := q.ReportItems(ctx, userFilter)
	if err != nil {
		return nil, err
	}
	ptags, err := q.ListProjectTags(ctx, userFilter)
	if err != nil {
		return nil, err
	}
	users, err := q.ReportUsers(ctx)
	if err != nil {
		return nil, err
	}
	d := &data{custName: map[int64]string{}, customers: customers, projects: projects, projOK: map[int64]sqlc.ReportProjectsRow{}, names: map[int64]string{}}
	for _, u := range users {
		d.names[u.ID] = u.Name
	}
	for _, c := range customers {
		if len(f.Customers) == 0 || slices.Contains(f.Customers, c.ID) {
			d.custName[c.ID] = c.Name
		}
	}
	tagsOf := map[int64][]int64{}
	for _, pt := range ptags {
		tagsOf[pt.ProjectID] = append(tagsOf[pt.ProjectID], pt.TagID)
	}
	for _, p := range projects {
		if _, ok := d.custName[p.CustomerID]; !ok {
			continue
		}
		if len(f.Tags) > 0 && !slices.ContainsFunc(tagsOf[p.ID], func(t int64) bool { return slices.Contains(f.Tags, t) }) {
			continue
		}
		d.projOK[p.ID] = p
	}
	seen := map[int64]bool{}
	for _, c := range aggregate(items) {
		if c.Assignee != nil {
			seen[*c.Assignee] = true
		}
		if _, ok := d.projOK[c.ProjectID]; !ok {
			continue
		}
		if len(f.Assignees) > 0 && (c.Assignee == nil || !slices.Contains(f.Assignees, *c.Assignee)) {
			continue
		}
		d.lists = append(d.lists, c)
	}
	d.assignees = []UserRef{}
	for _, u := range users {
		if seen[u.ID] {
			d.assignees = append(d.assignees, UserRef{ID: u.ID, Name: u.Name})
		}
	}
	return d, nil
}

// Build computes the report for the visible customers (userFilter: nil = all,
// otherwise the user ID; see authz.User.VisibilityFilter).
func Build(ctx context.Context, q *sqlc.Queries, userFilter any, f Filter, now time.Time) (*Report, error) {
	f.Normalize()
	d, err := load(ctx, q, userFilter, f)
	if err != nil {
		return nil, err
	}
	loc := now.Location()
	today := dayOf(now, loc)
	start := today.AddDate(0, 0, -(f.Days - 1)) // period: [start, now]
	prevStart := start.AddDate(0, 0, -f.Days)   // previous period: [prevStart, start)
	inPeriod := func(t time.Time) bool { return !t.Before(start) && !t.After(now) }
	inPrev := func(t time.Time) bool { return !t.Before(prevStart) && t.Before(start) }
	todayStr, startStr := today.Format("2006-01-02"), start.Format("2006-01-02")
	weekEnd := today.AddDate(0, 0, 6).Format("2006-01-02")
	overdue := func(c *checklist) bool { return !c.complete() && c.Due != "" && c.Due < todayStr }

	r := &Report{From: startStr, To: todayStr, Days: f.Days, Assignees: d.assignees}

	// ---- dashboard
	db := &r.Dashboard
	db.Status = map[string]int{}
	for _, c := range d.lists {
		if c.complete() {
			if t := c.LastDone.In(loc); inPeriod(t) {
				db.Completed++
			} else if inPrev(t) {
				db.CompletedPrev++
			}
		} else {
			db.Open++
			if overdue(c) {
				db.Overdue++
			}
		}
		existed := !c.CreatedKnown || c.Created.Before(start)
		if existed && (!c.complete() || !c.LastDone.Before(start)) {
			db.OpenPrev++
			if c.Due != "" && c.Due < startStr {
				db.OverduePrev++
			}
		}
	}
	if done, err := q.ReportProjectsDone(ctx); err == nil {
		seen := map[int64]bool{}
		for _, a := range done {
			if p, ok := d.projOK[a.ProjectID]; ok && p.Status == "done" && inPeriod(a.CreatedAt.In(loc)) && !seen[a.ProjectID] {
				seen[a.ProjectID] = true
				db.ProjectsDone++
			}
		}
	}
	// the last 12 ISO weeks (Monday start), the current one last
	wd := int(today.Weekday())
	if wd == 0 {
		wd = 7
	}
	monday := today.AddDate(0, 0, -(wd - 1))
	for i := 11; i >= 0; i-- {
		ws := monday.AddDate(0, 0, -7*i)
		_, wn := ws.ISOWeek()
		db.Weeks = append(db.Weeks, Week{Label: "W" + strconv.Itoa(wn), Start: ws.Format("2006-01-02")})
	}
	weekOf := func(t time.Time) int {
		dt := dayOf(t, loc)
		w := int(dt.Weekday())
		if w == 0 {
			w = 7
		}
		idx := 11 - int(math.Round(monday.Sub(dt.AddDate(0, 0, -(w-1))).Hours()/24/7))
		if idx < 0 || idx > 11 || t.After(now) {
			return -1
		}
		return idx
	}
	for _, c := range d.lists {
		if c.complete() {
			if i := weekOf(c.LastDone); i >= 0 {
				db.Weeks[i].Completed++
			}
		}
		if c.CreatedKnown {
			if i := weekOf(c.Created); i >= 0 {
				db.Weeks[i].Created++
			}
		}
	}
	for _, p := range d.projOK {
		db.Status[p.Status]++
	}
	limit := today.AddDate(0, 0, 14).Format("2006-01-02")
	db.Due = []Due{}
	for _, p := range d.projOK {
		if p.DueDate != nil && p.Status != "done" && *p.DueDate <= limit {
			db.Due = append(db.Due, Due{Date: *p.DueDate, Kind: "project", ID: p.ID, Title: p.Name, Customer: d.custName[p.CustomerID]})
		}
	}
	for _, c := range d.lists {
		if !c.complete() && c.Due != "" && c.Due <= limit {
			db.Due = append(db.Due, Due{Date: c.Due, Kind: "checklist", ID: c.ID, Title: c.Title, Customer: d.custName[c.CustomerID]})
		}
	}
	sort.Slice(db.Due, func(i, j int) bool {
		if db.Due[i].Date != db.Due[j].Date {
			return db.Due[i].Date < db.Due[j].Date
		}
		return db.Due[i].Title < db.Due[j].Title
	})
	if len(db.Due) > 10 {
		db.Due = db.Due[:10]
	}

	// ---- customers
	byCust := map[int64]*Cust{}
	projIdx := map[int64]int{}
	for _, c := range d.customers {
		if _, ok := d.custName[c.ID]; ok {
			byCust[c.ID] = &Cust{ID: c.ID, Name: c.Name, Projects: []CustProj{}}
		}
	}
	for _, p := range d.projects {
		if _, ok := d.projOK[p.ID]; !ok {
			continue
		}
		c := byCust[p.CustomerID]
		projIdx[p.ID] = len(c.Projects)
		c.Projects = append(c.Projects, CustProj{ID: p.ID, Name: p.Name, Status: p.Status})
		if p.DueDate != nil && p.Status != "done" && (c.NextDue == "" || *p.DueDate < c.NextDue) {
			c.NextDue = *p.DueDate
		}
	}
	for _, cl := range d.lists {
		p := d.projOK[cl.ProjectID]
		c := byCust[p.CustomerID]
		cp := &c.Projects[projIdx[p.ID]]
		cp.Total++
		if p.IncludeInCustomerProgress {
			c.Total++
		}
		if cl.complete() {
			cp.Done++
			if p.IncludeInCustomerProgress {
				c.Done++
			}
			continue
		}
		c.Open++
		if overdue(cl) {
			c.Overdue++
		}
		if cl.Due != "" && (c.NextDue == "" || cl.Due < c.NextDue) {
			c.NextDue = cl.Due
		}
	}
	r.Customers = []Cust{}
	for _, c := range d.customers {
		if x, ok := byCust[c.ID]; ok {
			r.Customers = append(r.Customers, *x)
		}
	}

	// ---- workload (checklists by their person)
	people := map[int64]*Person{}
	person := func(id int64) *Person {
		if people[id] == nil {
			people[id] = &Person{ID: id, Name: d.names[id]}
		}
		return people[id]
	}
	r.Workload.Unassigned.Checklists = []ClBrief{}
	for _, c := range d.lists {
		if c.complete() {
			if c.Assignee != nil {
				if t := c.LastDone.In(loc); inPeriod(t) {
					person(*c.Assignee).Done++
				} else if inPrev(t) {
					r.Workload.DonePrev++
				}
			}
			continue
		}
		if c.Assignee == nil {
			r.Workload.Unassigned.Total++
			r.Workload.Unassigned.Checklists = append(r.Workload.Unassigned.Checklists, d.brief(c, c.Due))
			continue
		}
		p := person(*c.Assignee)
		p.Open++
		switch {
		case overdue(c):
			p.Overdue++
		case c.Due != "" && c.Due <= weekEnd:
			p.DueWeek++
		}
	}
	r.Workload.People = []Person{}
	for _, p := range people {
		r.Workload.People = append(r.Workload.People, *p)
	}
	sort.Slice(r.Workload.People, func(i, j int) bool {
		a, b := r.Workload.People[i], r.Workload.People[j]
		if a.Open != b.Open {
			return a.Open > b.Open
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	sort.Slice(r.Workload.Unassigned.Checklists, func(i, j int) bool {
		a, b := r.Workload.Unassigned.Checklists[i], r.Workload.Unassigned.Checklists[j]
		if (a.Date == "") != (b.Date == "") {
			return a.Date != ""
		}
		return a.Date < b.Date
	})

	// ---- templates
	if err := d.templates(ctx, q, r, inPeriod, inPrev, loc); err != nil {
		return nil, err
	}
	return r, nil
}

// brief describes a checklist for lists.
func (d *data) brief(c *checklist, date string) ClBrief {
	p := d.projOK[c.ProjectID]
	return ClBrief{ID: c.ID, Title: c.Title, Project: p.Name, Customer: d.custName[c.CustomerID], Date: date, Done: c.Done, Total: c.Total}
}

// templates: usage, duration and punctuality of checklists from templates.
func (d *data) templates(ctx context.Context, q *sqlc.Queries, r *Report, inPeriod, inPrev func(time.Time) bool, loc *time.Location) error {
	tpls, err := q.ListTemplates(ctx)
	if err != nil {
		return err
	}
	uses, err := q.ReportTemplateUses(ctx)
	if err != nil {
		return err
	}
	tplName := map[int64]string{}
	for _, t := range tpls {
		tplName[t.ID] = t.Name
	}
	used := map[int64]int{}
	for _, u := range uses {
		if t, ok := parseTime(u.UsedAt); ok && inPeriod(t.In(loc)) {
			used[u.TemplateID]++
			r.Templates.Created++
		}
	}
	type acc struct {
		days, planned   []float64
		onTime, withDue int
	}
	add := func(a *acc, c *checklist, fin time.Time, days float64) {
		a.days = append(a.days, days)
		if due, err := time.ParseInLocation("2006-01-02", c.Due, loc); err == nil {
			a.withDue++
			a.planned = append(a.planned, due.Sub(dayOf(c.Created, loc)).Hours()/24)
			if !fin.After(due.AddDate(0, 0, 1)) { // done on the due day at the latest
				a.onTime++
			}
		}
	}
	cur, all, prev := map[int64]*acc{}, &acc{}, &acc{}
	r.Templates.Longest = []ClBrief{}
	for _, c := range d.lists {
		if c.TemplateID == nil || !c.complete() || !c.CreatedKnown {
			continue
		}
		fin := c.LastDone.In(loc)
		days := fin.Sub(c.Created).Hours() / 24
		if days < 0 {
			continue
		}
		switch {
		case inPeriod(fin):
			if cur[*c.TemplateID] == nil {
				cur[*c.TemplateID] = &acc{}
			}
			add(cur[*c.TemplateID], c, fin, days)
			add(all, c, fin, days)
			b := d.brief(c, fin.Format("2006-01-02"))
			b.Template, b.Days = tplName[*c.TemplateID], round1(days)
			r.Templates.Longest = append(r.Templates.Longest, b)
		case inPrev(fin):
			add(prev, c, fin, days)
		}
	}
	sort.Slice(r.Templates.Longest, func(i, j int) bool { return *r.Templates.Longest[i].Days > *r.Templates.Longest[j].Days })
	if len(r.Templates.Longest) > 5 {
		r.Templates.Longest = r.Templates.Longest[:5]
	}
	pct := func(a *acc) *float64 {
		if a.withDue == 0 {
			return nil
		}
		return round1(float64(a.onTime) * 100 / float64(a.withDue))
	}
	r.Templates.AvgDays, r.Templates.OnTime = avg(all.days), pct(all)
	r.Templates.AvgDaysPrev, r.Templates.OnTimePrev = avg(prev.days), pct(prev)
	r.Templates.Rows = []TplRow{}
	for _, t := range tpls {
		row := TplRow{ID: t.ID, Name: t.Name, CategoryID: t.CategoryID, Used: used[t.ID]}
		if a := cur[t.ID]; a != nil {
			row.Completed = len(a.days)
			row.AvgDays, row.Planned, row.OnTime = avg(a.days), avg(a.planned), pct(a)
		}
		if row.Used > 0 || row.Completed > 0 {
			r.Templates.Rows = append(r.Templates.Rows, row)
		}
	}
	sort.Slice(r.Templates.Rows, func(i, j int) bool {
		a, b := r.Templates.Rows[i], r.Templates.Rows[j]
		if a.Used != b.Used {
			return a.Used > b.Used
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	if runs, err := q.ReportAutomationRuns(ctx); err == nil {
		for _, run := range runs {
			if t, ok := parseTime(run.CreatedAt); ok && inPeriod(t.In(loc)) {
				if run.Status == "ok" {
					r.Templates.AutoCreated++
				} else {
					r.Templates.AutoWarned++
				}
			}
		}
	}
	return nil
}
