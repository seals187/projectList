package report

import (
	"context"
	"sort"
	"time"

	"github.com/seals187/projectList/internal/db/sqlc"
)

// CustomerReport: status report of one customer (print view / PDF) – projects
// and checklists, no single items.
type CustomerReport struct {
	Customer  CustInfo   `json:"customer"`
	From      string     `json:"from"`
	To        string     `json:"to"`
	Projects  []CustProj `json:"projects"`
	Done      int        `json:"done"`      // completed checklists (projects counting toward the customer)
	Total     int        `json:"total"`     // checklists (…)
	Completed []ClBrief  `json:"completed"` // checklists completed in the period (newest first)
	Overdue   []ClBrief  `json:"overdue"`   // open checklists past their due date
	Upcoming  []ClBrief  `json:"upcoming"`  // open checklists due in the next 14 days
	Open      []ClBrief  `json:"open"`      // all open checklists (with progress)
}

type CustInfo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Contact  string `json:"contact"`
	Notes    string `json:"notes"`
}

// Customer computes the status report of a customer (the caller checks access).
func Customer(ctx context.Context, q *sqlc.Queries, customerID int64, days int, now time.Time) (*CustomerReport, error) {
	f := Filter{Days: days, Customers: []int64{customerID}}
	f.Normalize()
	c, err := q.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	base, err := Build(ctx, q, nil, f, now)
	if err != nil {
		return nil, err
	}
	d, err := load(ctx, q, nil, f)
	if err != nil {
		return nil, err
	}
	out := &CustomerReport{
		Customer: CustInfo{ID: c.ID, Name: c.Name, Location: c.Location, Contact: c.ContactName, Notes: c.Notes},
		From:     base.From, To: base.To, Projects: []CustProj{},
		Completed: []ClBrief{}, Overdue: []ClBrief{}, Upcoming: []ClBrief{}, Open: []ClBrief{},
	}
	if len(base.Customers) == 1 {
		bc := base.Customers[0]
		out.Projects, out.Done, out.Total = bc.Projects, bc.Done, bc.Total
	}
	loc := now.Location()
	today := dayOf(now, loc)
	todayStr := today.Format("2006-01-02")
	limit := today.AddDate(0, 0, 14).Format("2006-01-02")
	for _, cl := range d.lists {
		if cl.complete() {
			if done := cl.LastDone.In(loc).Format("2006-01-02"); done >= out.From {
				out.Completed = append(out.Completed, d.brief(cl, done))
			}
			continue
		}
		b := d.brief(cl, cl.Due)
		out.Open = append(out.Open, b)
		switch {
		case cl.Due == "":
		case cl.Due < todayStr:
			out.Overdue = append(out.Overdue, b)
		case cl.Due <= limit:
			out.Upcoming = append(out.Upcoming, b)
		}
	}
	sort.Slice(out.Completed, func(i, j int) bool { return out.Completed[i].Date > out.Completed[j].Date })
	byDate := func(l []ClBrief) {
		sort.SliceStable(l, func(i, j int) bool {
			if (l[i].Date == "") != (l[j].Date == "") {
				return l[i].Date != ""
			}
			return l[i].Date < l[j].Date
		})
	}
	byDate(out.Overdue)
	byDate(out.Upcoming)
	byDate(out.Open)
	return out, nil
}
