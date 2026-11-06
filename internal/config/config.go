// Package config reads the configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/seals187/projectList/internal/mail"
)

type Config struct {
	ListenAddr    string
	DBPath        string
	AdminEmail    string
	AdminPassword string
	AdminName     string
	AppSubtitle   string // subtitle below "ProjectList" in the sidebar (optional)
	CookieSecure  bool
	SessionTTL    time.Duration
	LogLevel      slog.Level
	LogFormat     string // "json" | "text"
	AppURL        string // public base URL for links in e-mails (optional; default: from the request)
	SMTP          mail.Config
}

func Load() (Config, error) {
	c := Config{
		ListenAddr:    env("LISTEN_ADDR", ":8080"),
		DBPath:        env("DB_PATH", "./data/projectlist.db"),
		AdminEmail:    strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		AdminName:     env("ADMIN_NAME", "Admin"),
		AppSubtitle:   strings.TrimSpace(os.Getenv("APP_SUBTITLE")),
		LogFormat:     strings.ToLower(env("LOG_FORMAT", "json")),
		AppURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("APP_URL")), "/"),
		SMTP: mail.Config{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
			TLS:      strings.ToLower(env("SMTP_TLS", "starttls")),
		},
	}

	var err error
	if c.CookieSecure, err = strconv.ParseBool(env("COOKIE_SECURE", "true")); err != nil {
		return c, fmt.Errorf("COOKIE_SECURE: %w", err)
	}
	if c.SessionTTL, err = time.ParseDuration(env("SESSION_TTL", "720h")); err != nil || c.SessionTTL < time.Minute {
		return c, fmt.Errorf("SESSION_TTL ungültig (z. B. 720h)")
	}
	if err = c.LogLevel.UnmarshalText([]byte(env("LOG_LEVEL", "info"))); err != nil {
		return c, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	if c.LogFormat != "json" && c.LogFormat != "text" {
		return c, fmt.Errorf("LOG_FORMAT muss json oder text sein")
	}
	if c.SMTP.Port, err = strconv.Atoi(env("SMTP_PORT", "587")); err != nil || c.SMTP.Port < 1 || c.SMTP.Port > 65535 {
		return c, fmt.Errorf("SMTP_PORT ungültig")
	}
	if t := c.SMTP.TLS; t != "starttls" && t != "tls" && t != "none" {
		return c, fmt.Errorf("SMTP_TLS muss starttls, tls oder none sein")
	}
	if c.AppURL != "" && !strings.HasPrefix(c.AppURL, "http://") && !strings.HasPrefix(c.AppURL, "https://") {
		return c, fmt.Errorf("APP_URL muss mit http:// oder https:// beginnen")
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}
