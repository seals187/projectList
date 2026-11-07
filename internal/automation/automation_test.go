package automation

import (
	"testing"
	"time"
)

func TestScheduleDue(t *testing.T) {
	d := func(s string) time.Time {
		v, _ := time.Parse("2006-01-02", s)
		return v
	}
	cases := []struct {
		s    Schedule
		day  string
		want bool
	}{
		{Schedule{Freq: "yearly", Month: 10, Day: 1}, "2026-10-01", true},
		{Schedule{Freq: "yearly", Month: 10, Day: 1}, "2026-10-02", false},
		{Schedule{Freq: "monthly", Day: 31}, "2026-02-28", true}, // last day of a short month
		{Schedule{Freq: "monthly", Day: 31}, "2026-02-27", false},
		{Schedule{Freq: "monthly", Day: 15}, "2026-03-15", true},
		{Schedule{Freq: "weekly", Weekday: 1}, "2026-11-09", true}, // Monday
		{Schedule{Freq: "weekly", Weekday: 7}, "2026-11-08", true}, // Sunday
		{Schedule{Freq: "weekly", Weekday: 7}, "2026-11-09", false},
	}
	for _, c := range cases {
		if got := c.s.Due(d(c.day)); got != c.want {
			t.Errorf("%+v %s: got %v", c.s, c.day, got)
		}
	}
	if (Schedule{Freq: "yearly", Month: 2}).Valid() || !(Schedule{Freq: "weekly", Weekday: 3}).Valid() {
		t.Error("Valid")
	}
}

func TestMatches(t *testing.T) {
	f := map[string]string{"name": "Server Migration 7.2", "tracker": "Mandat"}
	if !Matches(nil, f) {
		t.Error("ohne bedingungen")
	}
	if !Matches([]Condition{{"name", "contains", "migration"}, {"tracker", "equals", "mandat"}}, f) {
		t.Error("treffer")
	}
	if Matches([]Condition{{"tracker", "equals", "Mandat 2"}}, f) || Matches([]Condition{{"missing", "contains", "x"}}, f) {
		t.Error("kein treffer")
	}
}
