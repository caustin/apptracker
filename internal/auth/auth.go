// Package auth is local email/password sign-in over server-side sessions.
// Every way of logging in ends in SignIn, so a future Google callback can
// reuse it.
package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/google/uuid"

	"github.com/caustin/apptracker/internal/domain"
	"github.com/caustin/apptracker/internal/store"
)

const (
	IdleTimeout     = 7 * 24 * time.Hour
	AbsoluteTimeout = 30 * 24 * time.Hour

	userIDKey   = "uid"
	userIDLocal = "userId"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidEmail       = errors.New("a valid email is required")
	ErrNameTooLong        = errors.New("name must be at most 200 characters")
)

type Auth struct {
	store    store.Store
	sessions *session.Store
}

// New builds the session store. In production the cookie is __Host-session
// (Secure, Path=/, no Domain, so a sibling subdomain cannot set or overwrite
// it); in development it is plain "session" so it works over http.
func New(st store.Store, storage fiber.Storage, prod bool) *Auth {
	name := "session"
	if prod {
		name = "__Host-session"
	}
	ss := session.NewStore(session.Config{
		Storage:         storage,
		Extractor:       extractors.FromCookie(name),
		CookiePath:      "/",
		CookieSameSite:  fiber.CookieSameSiteLaxMode,
		CookieSecure:    prod,
		CookieHTTPOnly:  true,
		IdleTimeout:     IdleTimeout,
		AbsoluteTimeout: AbsoluteTimeout,
	})
	return &Auth{store: st, sessions: ss}
}

// NormalizeEmail trims and lowercases, then checks the address parses.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// SignUp creates a password account. The caller signs the user in.
func (a *Auth) SignUp(ctx context.Context, email, name, password string) (domain.User, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		return domain.User{}, err
	}
	if err := CheckPasswordRule(password); err != nil {
		return domain.User{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = email
	}
	if len([]rune(name)) > 200 {
		return domain.User{}, ErrNameTooLong
	}
	hash, err := HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return a.store.CreateUser(ctx, domain.User{
		ID: uuid.NewString(), Email: email, Name: name, PasswordHash: &hash,
		CreatedAt: now, UpdatedAt: now,
	})
}

// Authenticate checks an email and password. Unknown emails still run a hash
// verification, so timing does not reveal which accounts exist.
func (a *Auth) Authenticate(ctx context.Context, email, password string) (domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(password) > MaxPasswordLength*4 {
		return domain.User{}, ErrInvalidCredentials
	}
	u, err := a.store.GetUserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) || (err == nil && u.PasswordHash == nil) {
		_, _ = VerifyPassword(password, dummyHash)
		return domain.User{}, ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, err
	}
	ok, err := VerifyPassword(password, *u.PasswordHash)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, ErrInvalidCredentials
	}
	return u, nil
}

// SignIn starts a session for userID on a fresh session ID, discarding any
// session the request arrived with (no session fixation).
func (a *Auth) SignIn(c fiber.Ctx, userID string) error {
	sess, err := a.sessions.Get(c)
	if err != nil {
		return err
	}
	defer sess.Release()
	if err := sess.Reset(); err != nil {
		return err
	}
	sess.Set(userIDKey, userID)
	return sess.Save()
}

// SignOut deletes the session server-side and expires the cookie.
func (a *Auth) SignOut(c fiber.Ctx) error {
	sess, err := a.sessions.Get(c)
	if err != nil {
		return err
	}
	defer sess.Release()
	if sess.Fresh() {
		return nil // no session to end; nothing is stored or set
	}
	return sess.Destroy()
}

// CurrentUserID returns the signed-in user's ID, or "" when there is none.
// A valid session is saved again so its idle timeout slides. Requests without
// a session never write anything, so anonymous traffic creates no rows.
func (a *Auth) CurrentUserID(c fiber.Ctx) (string, error) {
	sess, err := a.sessions.Get(c)
	if err != nil {
		return "", err
	}
	defer sess.Release()
	uid, _ := sess.Get(userIDKey).(string)
	if sess.Fresh() || uid == "" {
		return "", nil
	}
	if err := sess.Save(); err != nil {
		return "", err
	}
	return uid, nil
}

// RequireUser is middleware: a JSON 401 without a session, otherwise the user
// ID goes into c.Locals for UserID.
func (a *Auth) RequireUser(c fiber.Ctx) error {
	uid, err := a.CurrentUserID(c)
	if err != nil {
		return err
	}
	if uid == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	c.Locals(userIDLocal, uid)
	return c.Next()
}

// UserID is the signed-in user's ID, set by RequireUser.
func UserID(c fiber.Ctx) string {
	uid, _ := c.Locals(userIDLocal).(string)
	return uid
}
