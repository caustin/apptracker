package server_test

import (
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	c := newApp(t).client()
	res := c.expect(c.req("GET", "/api/v1/health", nil), 200)
	if !strings.Contains(string(res.body), `"ok"`) {
		t.Fatalf("body %s", res.body)
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	c := newApp(t).client()
	for _, path := range []string{"/api/v1/health", "/api/v1/nope", "/api/v1/positions", "/", "/positions/12"} {
		res := c.req("GET", path, nil)
		for h, want := range map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
			"X-Frame-Options":        "DENY",
			"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
		} {
			if got := res.Header.Get(h); got != want {
				t.Errorf("%s: %s = %q, want %q", path, h, got, want)
			}
		}
		if res.Header.Get("Strict-Transport-Security") != "" {
			t.Errorf("%s: HSTS set in development", path)
		}
	}
}

func TestHSTSInProduction(t *testing.T) {
	c := newApp(t, prod).client()
	res := c.req("GET", "/api/v1/health", nil)
	if !strings.HasPrefix(res.Header.Get("Strict-Transport-Security"), "max-age=") {
		t.Fatal("no HSTS in production")
	}
}

func TestCSPOnlyOnHTML(t *testing.T) {
	c := newApp(t).client()
	html := c.expect(c.req("GET", "/", nil), 200)
	if !strings.HasPrefix(html.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("/ content type %q", html.Header.Get("Content-Type"))
	}
	csp := html.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self';") || !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("CSP on HTML = %q", csp)
	}
	api := c.req("GET", "/api/v1/health", nil)
	if api.Header.Get("Content-Security-Policy") != "" {
		t.Fatal("CSP set on a JSON response")
	}
}

func TestUnknownAPIPathIsJSON404(t *testing.T) {
	c := newApp(t).client()
	for _, path := range []string{"/api/v1/nope", "/api/nope", "/api"} {
		res := c.expect(c.req("GET", path, nil), 404)
		if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
			t.Fatalf("%s: content type %q", path, res.Header.Get("Content-Type"))
		}
		if res.errorMsg(t) != "not found" {
			t.Fatalf("%s: body %s", path, res.body)
		}
	}
}

func TestUnknownPagePathReturnsIndex(t *testing.T) {
	c := newApp(t).client()
	root := c.expect(c.req("GET", "/", nil), 200)
	for _, path := range []string{"/positions", "/positions/12", "/login", "/does/not/exist"} {
		res := c.expect(c.req("GET", path, nil), 200)
		if string(res.body) != string(root.body) {
			t.Fatalf("%s did not return index.html", path)
		}
		if res.Header.Get("Cache-Control") != "no-cache" {
			t.Fatalf("%s: Cache-Control %q", path, res.Header.Get("Cache-Control"))
		}
	}
	// A missing file is a 404, not the HTML shell.
	c.expect(c.req("GET", "/assets/missing.js", nil), 404)
}

func TestSwaggerOnlyInDevelopment(t *testing.T) {
	dev := newApp(t).client()
	res := dev.expect(dev.req("GET", "/swagger/index.html", nil), 200)
	if !strings.Contains(res.Header.Get("Content-Security-Policy"), "'unsafe-inline'") {
		t.Fatal("swagger page should get the relaxed dev CSP")
	}
	dev.expect(dev.req("GET", "/swagger/doc.json", nil), 200)
	// The relaxed CSP is for /swagger only.
	if strings.Contains(dev.req("GET", "/", nil).Header.Get("Content-Security-Policy"), "unsafe-inline") {
		t.Fatal("relaxed CSP leaked onto the app")
	}

	p := newApp(t, prod).client()
	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json"} {
		res := p.req("GET", path, nil)
		if strings.Contains(string(res.body), "swagger") || strings.Contains(string(res.body), "Swagger") {
			t.Fatalf("%s served swagger in production", path)
		}
		if strings.Contains(res.Header.Get("Content-Security-Policy"), "unsafe-inline") {
			t.Fatalf("%s: relaxed CSP in production", path)
		}
	}
}
