// Package config liest die Konfiguration aus Umgebungsvariablen.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr    string
	DBPath        string
	AdminEmail    string
	AdminPassword string
	AdminName     string
	AppSubtitle   string // Untertitel unter "ProjectList" in der Seitenleiste (optional)
	CookieSecure  bool
	SessionTTL    time.Duration
	LogLevel      slog.Level
	LogFormat     string // "json" | "text"
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
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}
