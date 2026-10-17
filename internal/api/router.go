// Package api is the JSON REST API under /api/v1.
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"github.com/caustin/apptracker/internal/auth"
	"github.com/caustin/apptracker/internal/config"
	"github.com/caustin/apptracker/internal/storage"
	"github.com/caustin/apptracker/internal/store"
)

const (
	MaxResumes      = 25                     // per user, keeps disk use bounded
	MaxResumeBytes  = 15 << 20               // 15 MB per PDF
	BodyLimit       = MaxResumeBytes + 1<<20 // room for the multipart envelope
	authLimitMax    = 10                     // login + sign-up attempts per IP...
	authLimitWindow = time.Minute            // ...per window
)

type Deps struct {
	Store  store.Store
	Auth   *auth.Auth
	Files  storage.Storage
	Config config.Config
}

type handlers struct {
	store store.Store
	auth  *auth.Auth
	files storage.Storage
}

// Register mounts /api/v1 and the JSON 404 for any other /api path.
//
// Order: the cross-origin (CSRF) check runs on every /api/v1 request, auth
// included; it lets GET/HEAD/OPTIONS through and rejects unsafe methods that
// a browser marks cross-site or same-site, or whose Origin is foreign. Login
// and sign-up then pass the rate limiter, and every data route requires a
// session.
func Register(app *fiber.App, d Deps) {
	h := &handlers{store: d.Store, auth: d.Auth, files: d.Files}

	v1 := app.Group("/api/v1")
	v1.Use(csrf.NewCrossOriginProtection(csrf.CrossOriginConfig{
		TrustedOrigins: d.Config.TrustedOrigins,
		ErrorHandler: func(fiber.Ctx, error) error {
			return fiber.NewError(fiber.StatusForbidden, "cross-origin request denied")
		},
	}))

	v1.Get("/health", h.health)

	authLimit := limiter.New(limiter.Config{
		Max:        authLimitMax,
		Expiration: authLimitWindow,
		LimitReached: func(fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many attempts, try again later")
		},
	})
	v1.Post("/auth/signup", authLimit, h.signup)
	// Per-account limit on top of the per-IP one, so guesses against a single
	// email are bounded even when spread over addresses.
	accountLimit := limiter.New(limiter.Config{
		Max:        authLimitMax / 2,
		Expiration: authLimitWindow * 5,
		KeyGenerator: func(c fiber.Ctx) string {
			var req struct {
				Email string `json:"email"`
			}
			_ = json.Unmarshal(c.Body(), &req)
			sum := sha256.Sum256([]byte(c.IP() + "|" + strings.ToLower(strings.TrimSpace(req.Email))))
			return "login:" + hex.EncodeToString(sum[:])
		},
		LimitReached: func(fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many attempts, try again later")
		},
	})
	v1.Post("/auth/login", authLimit, accountLimit, h.login)
	v1.Post("/auth/logout", h.logout)
	v1.Get("/auth/session", h.session)

	user := d.Auth.RequireUser

	v1.Get("/positions", user, h.listPositions)
	v1.Post("/positions", user, h.createPosition)
	v1.Get("/positions/:id", user, h.getPosition)
	v1.Put("/positions/:id", user, h.updatePosition)
	v1.Delete("/positions/:id", user, h.deletePosition)

	v1.Get("/people", user, h.listPeople)
	v1.Post("/people", user, h.createPerson)
	v1.Get("/people/:id", user, h.getPerson)
	v1.Put("/people/:id", user, h.updatePerson)
	v1.Delete("/people/:id", user, h.deletePerson)

	v1.Get("/events", user, h.listEvents)
	v1.Post("/events", user, h.createEvent)
	v1.Get("/events/:id", user, h.getEvent)
	v1.Put("/events/:id", user, h.updateEvent)
	v1.Delete("/events/:id", user, h.deleteEvent)

	v1.Get("/resumes", user, h.listResumes)
	v1.Post("/resumes", user, h.uploadResume)
	v1.Put("/resumes/:id", user, h.renameResume)
	v1.Get("/resumes/:id/file", user, h.downloadResume)
	v1.Delete("/resumes/:id", user, h.deleteResume)

	v1.Get("/goals", user, h.getGoals)
	v1.Put("/goals", user, h.putGoals)

	v1.Get("/export", user, h.export)

	// Unknown API paths get JSON, never the SPA shell.
	app.Use("/api", func(fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	})
}

// storeErr maps store.ErrNotFound to a 404; anything else is a 500.
func storeErr(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	return err
}

// health godoc
//
//	@Summary	Health check
//	@Tags		meta
//	@Produce	json
//	@Success	200	{object}	HealthResponse
//	@Router		/health [get]
func (h *handlers) health(c fiber.Ctx) error {
	return c.JSON(HealthResponse{Status: "ok"})
}
