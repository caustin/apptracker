// Command apptracker runs the job-search tracker: the REST API under /api/v1
// and the React app, from one binary.
//
//	apptracker                  serve (migrates first)
//	apptracker migrate up|down|status
//
//	@title			AppTracker API
//	@version		1.0
//	@description	Job-search tracker: positions, people, events, resumes and goals. Every route except sign-up, login, session and health needs the session cookie set by login. Unsafe methods are rejected when the browser marks them cross-site or same-site; JSON bodies must be sent as application/json.
//	@BasePath		/api/v1
//	@license.name	MIT
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/config"
	"github.com/caustin/apptracker/internal/server"
	"github.com/caustin/apptracker/internal/store/sqlite"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("apptracker", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if len(args) > 0 && args[0] == "migrate" {
		return migrate(db, args[1:])
	}
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q", args[0])
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := sqlite.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	srv, err := server.New(cfg, db, true)
	if err != nil {
		return err
	}
	defer srv.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- srv.App.Listen(":"+cfg.Port, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	slog.Info("listening", "port", cfg.Port, "env", cfg.Env, "db", cfg.DBPath)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.App.ShutdownWithContext(shutdownCtx)
	}
}

func migrate(db *sql.DB, args []string) error {
	p, err := sqlite.Migrations(db)
	if err != nil {
		return err
	}
	ctx := context.Background()
	cmd := "status"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "up":
		results, err := p.Up(ctx)
		for _, r := range results {
			fmt.Println(r)
		}
		return err
	case "down":
		r, err := p.Down(ctx)
		if r != nil {
			fmt.Println(r)
		}
		return err
	case "status":
		statuses, err := p.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			applied := "pending"
			if s.State == "applied" {
				applied = "applied " + s.AppliedAt.Format(time.RFC3339)
			}
			fmt.Printf("%05d  %-30s  %s\n", s.Source.Version, s.Source.Path, applied)
		}
		return nil
	}
	return errors.New("usage: apptracker migrate up|down|status")
}
