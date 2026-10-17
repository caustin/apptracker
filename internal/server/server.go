// Package server assembles the Fiber app: middleware order, the API, Swagger
// in development and the embedded frontend. main and the tests both build the
// app through New.
package server

import (
	"database/sql"
	"errors"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	_ "github.com/caustin/apptracker/docs" // registers the generated Swagger spec
	"github.com/caustin/apptracker/internal/api"
	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/config"
	"github.com/caustin/apptracker/internal/storage"
	"github.com/caustin/apptracker/internal/store/sqlite"
	"github.com/caustin/apptracker/internal/web"
)

const sessionSweepEvery = time.Hour

type Server struct {
	App      *fiber.App
	sessions *sqlite.Sessions
	files    *storage.Disk
}

// Close stops background work and releases files. It does not close the DB.
func (s *Server) Close() error {
	return errors.Join(s.sessions.Close(), s.files.Close())
}

// New builds the app over an open, migrated database.
func New(cfg config.Config, db *sql.DB, logRequests bool) (*Server, error) {
	files, err := storage.NewDisk(filepath.Join(cfg.DataDir, "resumes"))
	if err != nil {
		return nil, err
	}
	st := sqlite.New(db)
	sessions := sqlite.NewSessions(db, sessionSweepEvery)

	app := fiber.New(fiber.Config{
		AppName:      "apptracker",
		BodyLimit:    api.BodyLimit,
		ErrorHandler: errorHandler,
		// Behind a proxy, set TrustProxy and TrustProxyConfig here so c.IP()
		// (the rate limiter's key) is the client's address. Not yet: hosting
		// is undecided.
	})

	// Order: recover, request log, security headers; then Swagger (dev),
	// the API (its own CSRF check, rate limit and session guard), and the
	// frontend for everything else.
	app.Use(recover.New())
	if logRequests {
		app.Use(logger.New())
	}
	app.Use(web.SecurityHeaders(cfg.IsProd()))
	if !cfg.IsProd() {
		web.MountSwagger(app)
	}
	api.Register(app, api.Deps{
		Store:  st,
		Auth:   auth.New(st, sessions, cfg.IsProd()),
		Files:  files,
		Config: cfg,
	})
	app.Use(web.SPA())

	return &Server{App: app, sessions: sessions, files: files}, nil
}

// errorHandler returns {"error": "…"}. Errors raised with fiber.NewError keep
// their status and message; anything else is logged and becomes a bare 500,
// so internal details never reach the client.
func errorHandler(c fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	msg := "internal server error"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		status, msg = fe.Code, fe.Message
		if status == fiber.StatusRequestEntityTooLarge {
			msg = "request too large"
		}
	} else {
		slog.Error("request failed", "method", c.Method(), "path", c.Path(), "err", err)
	}
	return c.Status(status).JSON(api.ErrorResponse{Error: msg})
}
