// Package web serves the built React app from files embedded in the binary,
// sets the security headers on every response, and mounts Swagger UI in
// development.
package web

import (
	"embed"
	"io/fs"
	"mime"
	"path"
	"strings"

	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
)

// dist is Vite's output (vite.config.js builds into internal/web/dist).
// The embed directive only reaches files at or below this package, hence the location.
// The committed .gitkeep keeps the pattern valid before the first build.
//
//go:embed all:dist
var dist embed.FS

// CSP is the strict policy for the app's HTML. Vite emits separate JS and CSS
// files (no inline script or style), so those stay 'self'. The only third
// party is Google Fonts: the stylesheet (style-src) and the font files
// (font-src). object-src 'none' stops in-origin plugin/PDF script, and
// frame-ancestors 'none' stops clickjacking. React sets inline styles through
// the DOM (element.style), which CSP allows.
const CSP = "default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; " +
	"img-src 'self' data:; font-src 'self' https://fonts.gstatic.com; connect-src 'self'; " +
	"object-src 'none'; base-uri 'self'; frame-src 'none'; frame-ancestors 'none'; form-action 'self'"

// swaggerCSP is used for /swagger/* only, in development only. Swagger UI's
// page has an inline script and inline styles and loads Google Fonts.
const swaggerCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data:; " +
	"font-src 'self' https://fonts.gstatic.com; connect-src 'self'; object-src 'none'; " +
	"base-uri 'self'; frame-ancestors 'none'; form-action 'self'"

const hsts = "max-age=63072000; includeSubDomains"

// SecurityHeaders sets the fixed headers on every response, HSTS in
// production, and the CSP on HTML responses only.
func SecurityHeaders(prod bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
		c.Set(fiber.HeaderReferrerPolicy, "strict-origin-when-cross-origin")
		c.Set(fiber.HeaderXFrameOptions, "DENY")
		c.Set(fiber.HeaderPermissionsPolicy, "camera=(), microphone=(), geolocation=()")
		if prod {
			c.Set(fiber.HeaderStrictTransportSecurity, hsts)
		}
		err := c.Next()
		ct := strings.ToLower(strings.TrimSpace(strings.SplitN(string(c.Response().Header.ContentType()), ";", 2)[0]))
		if ct == fiber.MIMETextHTML {
			policy := CSP
			if !prod && strings.HasPrefix(c.Path(), "/swagger/") {
				policy = swaggerCSP
			}
			c.Set(fiber.HeaderContentSecurityPolicy, policy)
		}
		return err
	}
}

// MountSwagger serves Swagger UI at /swagger/*. Call it in development only.
func MountSwagger(app *fiber.App) {
	app.Get("/swagger/*", swaggo.HandlerDefault)
}

// placeholder is served when the frontend has not been built yet.
const placeholder = `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>AppTracker</title></head>` +
	`<body><p>The frontend has not been built. Run <code>make build</code>, or <code>make dev</code> and open the Vite server.</p></body></html>`

// SPA serves the embedded build: real files as themselves, and index.html for
// any other path so React Router can handle /positions/12 on a reload. Paths
// that look like files (they have an extension) and are missing get a 404
// rather than the HTML shell.
func SPA() fiber.Handler {
	files, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		index = []byte(placeholder)
	}
	return func(c fiber.Ctx) error {
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			return fiber.NewError(fiber.StatusMethodNotAllowed, "method not allowed")
		}
		name := strings.TrimPrefix(path.Clean("/"+c.Path()), "/")
		if name != "" && name != "index.html" && !strings.HasPrefix(name, ".") {
			if data, err := fs.ReadFile(files, name); err == nil {
				if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
					c.Set(fiber.HeaderContentType, ct)
				}
				if strings.HasPrefix(name, "assets/") {
					// Vite fingerprints everything under assets/.
					c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
				}
				return c.Send(data)
			}
			if path.Ext(name) != "" {
				return fiber.NewError(fiber.StatusNotFound, "not found")
			}
		}
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.Send(index)
	}
}
