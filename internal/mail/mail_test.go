package mail

import (
	"encoding/base64"
	"mime"
	"strings"
	"testing"
	"time"
)

func TestBuild(t *testing.T) {
	m := Invite("de", "max@example.com", "Max", "Änne Admin", "https://pl.example.com/invite?token=abc", time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC))
	raw, err := Build("ProjectList <noreply@example.com>", m, time.Date(2026, 11, 6, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	head, body, ok := strings.Cut(string(raw), "\r\n\r\n")
	if !ok {
		t.Fatal("no header/body separator")
	}
	for _, want := range []string{"From: ProjectList <noreply@example.com>\r\n", "To: max@example.com\r\n", "MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n", "Content-Transfer-Encoding: base64\r\n", "@example.com>\r\n"} {
		if !strings.Contains(head+"\r\n", want) {
			t.Errorf("header %q missing in\n%s", want, head)
		}
	}
	var subj string
	for _, l := range strings.Split(head, "\r\n") {
		if v, ok := strings.CutPrefix(l, "Subject: "); ok {
			subj = v
		}
	}
	if subj != "Einladung zu ProjectList" {
		t.Errorf("plain ASCII subject changed: %q", subj)
	}
	fr, _ := Build("a@example.com", Reset("fr", "x@example.com", "X", "l", time.Now()), time.Now())
	enc, _, _ := strings.Cut(strings.SplitN(string(fr), "Subject: ", 2)[1], "\r\n")
	if !strings.HasPrefix(enc, "=?utf-8?q?") {
		t.Errorf("subject not encoded: %q", enc)
	}
	if dec, _ := new(mime.WordDecoder).DecodeHeader(enc); dec != "ProjectList : réinitialiser le mot de passe" {
		t.Errorf("subject = %q", dec)
	}
	for _, l := range strings.Split(strings.TrimRight(body, "\r\n"), "\r\n") {
		if len(l) > 76 {
			t.Errorf("body line too long: %d", len(l))
		}
	}
	text, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body, "\r\n", ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Hallo Max", "Änne Admin", "https://pl.example.com/invite?token=abc", "2026-11-13 08:00 UTC"} {
		if !strings.Contains(string(text), want) {
			t.Errorf("body misses %q", want)
		}
	}
}

func TestBuildRejectsHeaderInjection(t *testing.T) {
	if _, err := Build("a@example.com", Message{To: "x@example.com\r\nBcc: y@example.com", Subject: "s"}, time.Now()); err == nil {
		t.Fatal("expected error")
	}
}

func TestLangAndEnabled(t *testing.T) {
	if Lang("FR") != "fr" || Lang("it") != "en" {
		t.Fatal("Lang")
	}
	if (&SMTP{}).Enabled() || !(&SMTP{Cfg: Config{Host: "h", From: "a@b.c"}}).Enabled() {
		t.Fatal("Enabled")
	}
}
