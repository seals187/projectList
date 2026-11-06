// Package mail sends plain-text e-mails (invitations, password reset links)
// via SMTP. Without SMTP_HOST/SMTP_FROM the sender is disabled and the admin
// copies the link by hand.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Message is one plain-text e-mail.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender sends e-mails. Enabled reports whether sending is configured.
type Sender interface {
	Enabled() bool
	Send(ctx context.Context, m Message) error
}

// Config of the SMTP server. TLS: "starttls" (default), "tls" (implicit, e.g.
// port 465) or "none".
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      string
}

// SMTP sends via an SMTP server.
type SMTP struct {
	Cfg     Config
	Timeout time.Duration // per connection; 0 = 10 s
}

func (s *SMTP) Enabled() bool { return s != nil && s.Cfg.Host != "" && s.Cfg.From != "" }

// Disabled is a sender that never sends.
type Disabled struct{}

func (Disabled) Enabled() bool                       { return false }
func (Disabled) Send(context.Context, Message) error { return errors.New("mail: not configured") }

func (s *SMTP) Send(ctx context.Context, m Message) error {
	if !s.Enabled() {
		return errors.New("mail: not configured")
	}
	from, err := mail.ParseAddress(s.Cfg.From)
	if err != nil {
		return fmt.Errorf("mail: SMTP_FROM: %w", err)
	}
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return fmt.Errorf("mail: recipient: %w", err)
	}
	msg, err := Build(s.Cfg.From, m, time.Now())
	if err != nil {
		return err
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	port := s.Cfg.Port
	if port == 0 {
		port = 587
	}
	addr := net.JoinHostPort(s.Cfg.Host, strconv.Itoa(port))
	tlsCfg := &tls.Config{ServerName: s.Cfg.Host, MinVersion: tls.VersionTLS12}
	d := &net.Dialer{}
	var conn net.Conn
	if strings.EqualFold(s.Cfg.TLS, "tls") {
		conn, err = (&tls.Dialer{NetDialer: d, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = d.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("mail: connect: %w", err)
	}
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	c, err := smtp.NewClient(conn, s.Cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mail: %w", err)
	}
	defer c.Close()
	mode := strings.ToLower(s.Cfg.TLS)
	if mode == "" || mode == "starttls" {
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("mail: starttls: %w", err)
		}
	}
	if s.Cfg.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.Cfg.Username, s.Cfg.Password, s.Cfg.Host)); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}
	if err := c.Mail(from.Address); err != nil {
		return fmt.Errorf("mail: from: %w", err)
	}
	if err := c.Rcpt(to.Address); err != nil {
		return fmt.Errorf("mail: rcpt: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("mail: write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("mail: send: %w", err)
	}
	return c.Quit()
}

// Build renders the message with MIME headers: UTF-8 plain text,
// base64 body, Q-encoded subject.
func Build(from string, m Message, now time.Time) ([]byte, error) {
	for _, v := range []string{from, m.To, m.Subject} {
		if strings.ContainsAny(v, "\r\n") {
			return nil, errors.New("mail: header contains a line break")
		}
	}
	var b bytes.Buffer
	host := "projectlist"
	if a, err := mail.ParseAddress(from); err == nil {
		if i := strings.LastIndexByte(a.Address, '@'); i >= 0 {
			host = a.Address[i+1:]
		}
	}
	rnd := make([]byte, 12)
	_, _ = rand.Read(rnd)
	h := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	h("From", from)
	h("To", m.To)
	h("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	h("Date", now.Format(time.RFC1123Z))
	h("Message-ID", "<"+hex.EncodeToString(rnd)+"@"+host+">")
	h("MIME-Version", "1.0")
	h("Content-Type", "text/plain; charset=utf-8")
	h("Content-Transfer-Encoding", "base64")
	b.WriteString("\r\n")
	body := strings.ReplaceAll(strings.ReplaceAll(m.Body, "\r\n", "\n"), "\n", "\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return b.Bytes(), nil
}
