package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/mail"
)

// Report e-mails: a schedule (report_schedules) sends the report with its
// filters to the recipient weekly or monthly, computed with the recipient's
// access. Plain text with the key figures and a link to the reporting page.

// ScheduleDue reports whether a schedule sends on the given day.
func ScheduleDue(s sqlc.ReportSchedule, d time.Time) bool {
	switch s.Freq {
	case "weekly":
		wd := int(d.Weekday())
		if wd == 0 {
			wd = 7
		}
		return int64(wd) == s.Weekday
	case "monthly":
		last := time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, d.Location()).Day()
		return int64(d.Day()) == s.Day || (s.Day > int64(last) && d.Day() == last)
	}
	return false
}

// Mailer sends the scheduled report e-mails.
type Mailer struct {
	DB     *sql.DB
	Q      *sqlc.Queries
	Sender mail.Sender
	AppURL string // base for the link; "" = no link
	Log    *slog.Logger
	Now    func() time.Time
}

// Run sends the due reports every interval until ctx is done.
func (m *Mailer) Run(ctx context.Context, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		m.SendDue(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// SendDue sends every schedule that is due today and was not sent yet today.
func (m *Mailer) SendDue(ctx context.Context) {
	if !m.Sender.Enabled() {
		return
	}
	now := m.Now()
	today := now.Format("2006-01-02")
	list, err := m.Q.ListReportSchedules(ctx)
	if err != nil {
		m.Log.Warn("berichte nicht geladen", "err", err)
		return
	}
	for _, s := range list {
		if s.LastSentOn == today || !ScheduleDue(s, now) {
			continue
		}
		if err := m.Send(ctx, s); err != nil {
			m.Log.Warn("bericht nicht gesendet", "schedule", s.ID, "err", err)
			continue
		}
		if err := m.Q.MarkReportSent(ctx, sqlc.MarkReportSentParams{LastSentOn: today, ID: s.ID}); err != nil {
			m.Log.Warn("bericht nicht markiert", "schedule", s.ID, "err", err)
		}
	}
}

// Send computes the report for the recipient and sends it.
func (m *Mailer) Send(ctx context.Context, s sqlc.ReportSchedule) error {
	u, err := m.Q.GetUserByID(ctx, s.RecipientID)
	if err != nil {
		return err
	}
	var f Filter
	_ = json.Unmarshal([]byte(s.Filters), &f)
	var visibility any = u.ID
	if u.Role == "admin" {
		visibility = nil
	}
	r, err := Build(ctx, m.Q, visibility, f, m.Now())
	if err != nil {
		return err
	}
	return m.Sender.Send(ctx, Text(mail.Lang(s.Lang), u.Email, u.Name, u.ID, r, m.AppURL))
}

// ---- text ------------------------------------------------------------------------------

var texts = map[string]map[string]string{
	"de": {
		"subject": "ProjectList – Bericht %s – %s", "hello": "Hallo %s,", "intro": "hier ist dein Bericht für %s – %s.",
		"completed": "Erledigte Checklisten", "prev": "Vorperiode", "open": "Offene Checklisten", "overdue": "Überfällig",
		"projectsDone": "Abgeschlossene Projekte", "customers": "Kunden", "custLine": "%d %% der Checklisten erledigt, %d offen, %d überfällig",
		"due": "Fällig in den nächsten 14 Tagen", "mine": "Deine offenen Checklisten: %d (%d überfällig)", "link": "Zum Reporting:",
		"none": "keine",
	},
	"en": {
		"subject": "ProjectList – report %s – %s", "hello": "Hello %s,", "intro": "here is your report for %s – %s.",
		"completed": "Completed checklists", "prev": "previous period", "open": "Open checklists", "overdue": "Overdue",
		"projectsDone": "Projects done", "customers": "Customers", "custLine": "%d %% of checklists done, %d open, %d overdue",
		"due": "Due in the next 14 days", "mine": "Your open checklists: %d (%d overdue)", "link": "Open the reporting:",
		"none": "none",
	},
	"fr": {
		"subject": "ProjectList – rapport %s – %s", "hello": "Bonjour %s,", "intro": "voici votre rapport pour %s – %s.",
		"completed": "Checklists terminées", "prev": "période précédente", "open": "Checklists ouvertes", "overdue": "En retard",
		"projectsDone": "Projets terminés", "customers": "Clients", "custLine": "%d %% des checklists terminées, %d ouvertes, %d en retard",
		"due": "Échéances des 14 prochains jours", "mine": "Vos checklists ouvertes : %d (%d en retard)", "link": "Ouvrir le reporting :",
		"none": "aucune",
	},
}

func swiss(iso string) string {
	if t, err := time.Parse("2006-01-02", iso); err == nil {
		return t.Format("02.01.2006")
	}
	return iso
}

// Text builds the report e-mail.
func Text(lang, to, name string, userID int64, r *Report, appURL string) mail.Message {
	tx, ok := texts[lang]
	if !ok {
		tx = texts["de"]
	}
	from, until := swiss(r.From), swiss(r.To)
	var b strings.Builder
	fmt.Fprintf(&b, tx["hello"]+"\n\n"+tx["intro"]+"\n\n", name, from, until)
	d := r.Dashboard
	fmt.Fprintf(&b, "%s: %d (%s: %d)\n", tx["completed"], d.Completed, tx["prev"], d.CompletedPrev)
	fmt.Fprintf(&b, "%s: %d\n%s: %d\n%s: %d\n\n", tx["open"], d.Open, tx["overdue"], d.Overdue, tx["projectsDone"], d.ProjectsDone)
	if len(r.Customers) > 0 {
		b.WriteString(tx["customers"] + ":\n")
		for _, c := range r.Customers {
			pct := 0
			if c.Total > 0 {
				pct = c.Done * 100 / c.Total
			}
			fmt.Fprintf(&b, "  %s: "+tx["custLine"]+"\n", c.Name, pct, c.Open, c.Overdue)
		}
		b.WriteString("\n")
	}
	b.WriteString(tx["due"] + ":\n")
	if len(d.Due) == 0 {
		b.WriteString("  " + tx["none"] + "\n")
	}
	for _, x := range d.Due {
		fmt.Fprintf(&b, "  %s  %s (%s)\n", swiss(x.Date), x.Title, x.Customer)
	}
	for _, p := range r.Workload.People {
		if p.ID == userID {
			fmt.Fprintf(&b, "\n"+tx["mine"]+"\n", p.Open, p.Overdue)
		}
	}
	if appURL != "" {
		fmt.Fprintf(&b, "\n%s %s/reports\n", tx["link"], strings.TrimRight(appURL, "/"))
	}
	return mail.Message{To: to, Subject: fmt.Sprintf(tx["subject"], from, until), Body: b.String()}
}
