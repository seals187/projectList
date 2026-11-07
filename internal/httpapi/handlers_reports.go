package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/authz"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/report"
)

// Reporting: everybody sees the report over the customers they may see.
// Report e-mails are scheduled by admins and members (members only for themselves).

// reportFilter reads ?period=30&customers=1,2&assignees=3&tags=4.
func reportFilter(v url.Values) report.Filter {
	ids := func(key string) []int64 {
		var out []int64
		for _, s := range strings.Split(v.Get(key), ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && id > 0 {
				out = append(out, id)
			}
		}
		return out
	}
	days, _ := strconv.Atoi(v.Get("period"))
	f := report.Filter{Days: days, Customers: ids("customers"), Assignees: ids("assignees"), Tags: ids("tags")}
	f.Normalize()
	return f
}

// GET /api/reports?period&customers&assignees&tags
func (s *Server) getReport(w http.ResponseWriter, r *http.Request, u authz.User) error {
	rep, err := report.Build(r.Context(), s.q, u.VisibilityFilter(), reportFilter(r.URL.Query()), s.now())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rep)
	return nil
}

// GET /api/reports/customers/{id}?period: status report of a customer (print view).
func (s *Server) getCustomerReport(w http.ResponseWriter, r *http.Request, u authz.User) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if err := s.requireCustomer(r.Context(), u, id); err != nil {
		return err
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("period"))
	rep, err := report.Customer(r.Context(), s.q, id, days, s.now())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, rep)
	return nil
}

// ---- schedules -------------------------------------------------------------------------

type scheduleDTO struct {
	sqlc.ReportSchedule
	Filters       report.Filter `json:"filters"`
	RecipientName string        `json:"recipient_name"`
}

func (s *Server) scheduleDTO(r *http.Request, sc sqlc.ReportSchedule) scheduleDTO {
	d := scheduleDTO{ReportSchedule: sc}
	_ = json.Unmarshal([]byte(sc.Filters), &d.Filters)
	d.Filters.Normalize()
	if u, err := s.q.GetUserByID(r.Context(), sc.RecipientID); err == nil {
		d.RecipientName = u.Name
	}
	return d
}

// mine: admins manage all schedules, everybody else those they created or receive.
func ownSchedule(u authz.User, sc sqlc.ReportSchedule) bool {
	return u.IsAdmin() || sc.RecipientID == u.ID || (sc.CreatedBy != nil && *sc.CreatedBy == u.ID)
}

// GET /api/report-schedules
func (s *Server) listReportSchedules(w http.ResponseWriter, r *http.Request, u authz.User) error {
	list, err := s.q.ListReportSchedules(r.Context())
	if err != nil {
		return err
	}
	out := []scheduleDTO{}
	for _, sc := range list {
		if ownSchedule(u, sc) {
			out = append(out, s.scheduleDTO(r, sc))
		}
	}
	recipients := []report.UserRef{{ID: u.ID, Name: u.Name}}
	if u.IsAdmin() {
		users, err := s.q.ReportUsers(r.Context())
		if err != nil {
			return err
		}
		recipients = recipients[:0]
		for _, x := range users {
			recipients = append(recipients, report.UserRef{ID: x.ID, Name: x.Name})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedules": out, "recipients": recipients, "mail_enabled": s.mailer.Enabled()})
	return nil
}

type scheduleReq struct {
	RecipientID *int64         `json:"recipient_id"`
	Freq        *string        `json:"freq"`
	Weekday     *int64         `json:"weekday"`
	Day         *int64         `json:"day"`
	Filters     *report.Filter `json:"filters"`
	Lang        *string        `json:"lang"`
}

func (s *Server) fillSchedule(r *http.Request, u authz.User, v validator, req scheduleReq, p *sqlc.UpdateReportScheduleParams) {
	if req.RecipientID != nil {
		p.RecipientID = *req.RecipientID
	}
	if p.RecipientID != u.ID {
		if !u.IsAdmin() {
			v["recipient_id"] = "forbidden"
		} else if _, err := s.q.GetUserByID(r.Context(), p.RecipientID); err != nil {
			v["recipient_id"] = "invalid_value"
		}
	}
	if req.Freq != nil {
		p.Freq = v.oneOf("freq", *req.Freq, "weekly", "monthly")
	}
	if req.Weekday != nil {
		p.Weekday = *req.Weekday
	}
	if req.Day != nil {
		p.Day = *req.Day
	}
	if p.Weekday < 1 || p.Weekday > 7 {
		v["weekday"] = "invalid_value"
	}
	if p.Day < 1 || p.Day > 31 {
		v["day"] = "invalid_value"
	}
	if req.Filters != nil {
		f := *req.Filters
		f.Normalize()
		b, _ := json.Marshal(f)
		p.Filters = string(b)
	}
	if req.Lang != nil {
		p.Lang = v.oneOf("lang", *req.Lang, "de", "en", "fr")
	}
}

// POST /api/report-schedules
func (s *Server) createReportSchedule(w http.ResponseWriter, r *http.Request, u authz.User) error {
	var req scheduleReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateReportScheduleParams{RecipientID: u.ID, Freq: "weekly", Weekday: 1, Day: 1, Filters: `{"period":30}`, Lang: "de"}
	s.fillSchedule(r, u, v, req, &p)
	if err := v.err(); err != nil {
		return err
	}
	uid := u.ID
	sc, err := s.q.CreateReportSchedule(r.Context(), sqlc.CreateReportScheduleParams{
		CreatedBy: &uid, RecipientID: p.RecipientID, Freq: p.Freq, Weekday: p.Weekday, Day: p.Day, Filters: p.Filters, Lang: p.Lang,
		CreatedAt: s.now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, s.scheduleDTO(r, sc))
	return nil
}

func (s *Server) loadSchedule(r *http.Request, u authz.User) (sqlc.ReportSchedule, error) {
	id, err := pathID(r, "id")
	if err != nil {
		return sqlc.ReportSchedule{}, err
	}
	sc, err := s.q.GetReportSchedule(r.Context(), id)
	if err != nil {
		return sc, err
	}
	if !ownSchedule(u, sc) {
		return sc, errNotFound()
	}
	return sc, nil
}

// PATCH /api/report-schedules/{id}
func (s *Server) updateReportSchedule(w http.ResponseWriter, r *http.Request, u authz.User) error {
	sc, err := s.loadSchedule(r, u)
	if err != nil {
		return err
	}
	var req scheduleReq
	if err := decode(w, r, &req); err != nil {
		return err
	}
	v := validator{}
	p := sqlc.UpdateReportScheduleParams{ID: sc.ID, RecipientID: sc.RecipientID, Freq: sc.Freq, Weekday: sc.Weekday, Day: sc.Day, Filters: sc.Filters, Lang: sc.Lang}
	s.fillSchedule(r, u, v, req, &p)
	if err := v.err(); err != nil {
		return err
	}
	if sc, err = s.q.UpdateReportSchedule(r.Context(), p); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, s.scheduleDTO(r, sc))
	return nil
}

// DELETE /api/report-schedules/{id}
func (s *Server) deleteReportSchedule(w http.ResponseWriter, r *http.Request, u authz.User) error {
	sc, err := s.loadSchedule(r, u)
	if err != nil {
		return err
	}
	if err := s.q.DeleteReportSchedule(r.Context(), sc.ID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// POST /api/report-schedules/{id}/send: send now (test).
func (s *Server) sendReportSchedule(w http.ResponseWriter, r *http.Request, u authz.User) error {
	sc, err := s.loadSchedule(r, u)
	if err != nil {
		return err
	}
	if !s.mailer.Enabled() {
		return &httpError{Status: http.StatusConflict, Msg: "mail_disabled"}
	}
	m := report.Mailer{Q: s.q, Sender: s.mailer, AppURL: s.baseURL(r), Log: s.log, Now: s.now}
	if err := m.Send(r.Context(), sc); err != nil {
		s.log.Warn("bericht nicht gesendet", "schedule", sc.ID, "err", err)
		return &httpError{Status: http.StatusBadGateway, Msg: "mail_failed"}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
