// Command projectlist is the ProjectList server plus helper commands.
//
//	projectlist [serve]          Start the server (default)
//	projectlist backup [target]  Consistent DB copy via VACUUM INTO
//	projectlist migrate          Run migrations only
//	projectlist healthcheck      Check /healthz (for Docker healthcheck)
//	projectlist version
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/seals187/projectList/internal/auth"
	"github.com/seals187/projectList/internal/config"
	"github.com/seals187/projectList/internal/db"
	"github.com/seals187/projectList/internal/db/sqlc"
	"github.com/seals187/projectList/internal/httpapi"
	"github.com/seals187/projectList/internal/static"
	"github.com/seals187/projectList/web"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}
	if cmd == "version" {
		fmt.Println("projectlist", version)
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "konfiguration:", err)
		return 2
	}
	log := newLogger(cfg)

	switch cmd {
	case "serve":
		err = serve(cfg, log)
	case "backup":
		err = backup(cfg, args[1:])
	case "migrate":
		err = migrate(cfg, log)
	case "healthcheck":
		err = healthcheck(cfg)
	default:
		fmt.Fprintf(os.Stderr, "unbekannter Befehl %q (serve|backup|migrate|healthcheck|version)\n", cmd)
		return 2
	}
	if err != nil {
		log.Error("fehler", "cmd", cmd, "err", err)
		return 1
	}
	return 0
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func serve(cfg config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := db.Migrate(ctx, database, log); err != nil {
		return err
	}
	q := sqlc.New(database)
	if err := auth.EnsureAdmin(ctx, q, cfg.AdminEmail, cfg.AdminPassword, cfg.AdminName, log); err != nil {
		return err
	}

	buildFS, err := fs.Sub(web.Build, "build")
	if err != nil {
		return err
	}
	staticHandler, err := static.New(buildFS)
	if err != nil {
		return err
	}

	sessions := &auth.Sessions{Q: q, TTL: cfg.SessionTTL, Secure: cfg.CookieSecure}
	go cleanupSessions(ctx, sessions, log)

	srv := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: httpapi.New(httpapi.Options{
			DB: database, DBPath: cfg.DBPath, Sessions: sessions, Logger: log, Static: staticHandler, Subtitle: cfg.AppSubtitle,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute, // backup download
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("ProjectList gestartet", "addr", cfg.ListenAddr, "version", version, "db", cfg.DBPath)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("fahre herunter …")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	// Write the WAL back into the main file so the DB file is consistent on its own.
	if _, err := database.ExecContext(shutdownCtx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Warn("wal checkpoint fehlgeschlagen", "err", err)
	}
	log.Info("beendet")
	return nil
}

func cleanupSessions(ctx context.Context, s *auth.Sessions, log *slog.Logger) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if n, err := s.Cleanup(ctx); err != nil && ctx.Err() == nil {
			log.Warn("session-cleanup fehlgeschlagen", "err", err)
		} else if n > 0 {
			log.Info("abgelaufene sessions entfernt", "anzahl", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func backup(cfg config.Config, args []string) error {
	dest := ""
	if len(args) > 0 {
		dest = args[0]
	} else {
		dest = filepath.Join(filepath.Dir(cfg.DBPath), "backups",
			fmt.Sprintf("projectlist-%s.db", time.Now().UTC().Format("20060102-150405")))
	}
	if _, err := os.Stat(cfg.DBPath); err != nil {
		return fmt.Errorf("datenbank nicht gefunden: %w", err)
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := db.VacuumInto(context.Background(), database, dest); err != nil {
		return err
	}
	fmt.Println("Backup erstellt:", dest)
	return nil
}

func migrate(cfg config.Config, log *slog.Logger) error {
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()
	return db.Migrate(context.Background(), database, log)
}

func healthcheck(cfg config.Config) error {
	_, port, err := net.SplitHostPort(cfg.ListenAddr)
	if err != nil {
		return err
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz: status %d", resp.StatusCode)
	}
	return nil
}
