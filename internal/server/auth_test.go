package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestAuthRoundTrip(t *testing.T) {
	a := newApp(t)
	c := a.client()
	c.expect(c.req("GET", "/api/v1/auth/session", nil), 401)

	res := c.expect(c.req("POST", "/api/v1/auth/signup", map[string]string{
		"email": "  Ada@Example.COM ", "name": "Ada", "password": password,
	}), 201)
	var s struct {
		User struct{ ID, Email, Name string }
	}
	res.json(t, &s)
	if s.User.Email != "ada@example.com" || s.User.Name != "Ada" || s.User.ID == "" {
		t.Fatalf("signup user = %+v", s.User)
	}
	if strings.Contains(string(res.body), "argon2") || strings.Contains(string(res.body), "password") {
		t.Fatal("password hash leaked in response")
	}
	c.expect(c.req("GET", "/api/v1/auth/session", nil), 200)

	c.expect(c.req("POST", "/api/v1/auth/logout", nil), 200)
	c.expect(c.req("GET", "/api/v1/auth/session", nil), 401)
	c.expect(c.req("GET", "/api/v1/positions", nil), 401)

	c.expect(c.req("POST", "/api/v1/auth/login", map[string]string{"email": "ADA@example.com", "password": password}), 200)
	c.expect(c.req("GET", "/api/v1/auth/session", nil), 200)
	c.expect(c.req("GET", "/api/v1/positions", nil), 200)
}

func TestLogoutInvalidatesSessionServerSide(t *testing.T) {
	a := newApp(t)
	c := a.signup("a@example.com")
	stolen := map[string]string{}
	for k, v := range c.cookies {
		stolen[k] = v
	}
	c.expect(c.req("POST", "/api/v1/auth/logout", nil), 200)
	replay := a.client()
	replay.cookies = stolen
	replay.expect(replay.req("GET", "/api/v1/auth/session", nil), 401)
}

func TestLoginRotatesSessionID(t *testing.T) {
	a := newApp(t)
	a.signup("a@example.com")
	c := a.client()
	c.cookies["session"] = "attacker-chosen-session-id"
	c.expect(c.req("POST", "/api/v1/auth/login", map[string]string{"email": "a@example.com", "password": password}), 200)
	if c.cookies["session"] == "attacker-chosen-session-id" || c.cookies["session"] == "" {
		t.Fatalf("session ID not regenerated: %q", c.cookies["session"])
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		prod       bool
		cookie     string
		wantSecure bool
	}{
		{"development", false, "session", false},
		{"production", true, "__Host-session", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var a *testApp
			if tc.prod {
				a = newApp(t, prod)
			} else {
				a = newApp(t)
			}
			c := a.client()
			res := c.expect(c.req("POST", "/api/v1/auth/signup", map[string]string{"email": "a@example.com", "password": password}), 201)
			var found *http.Cookie
			for _, ck := range res.Cookies() {
				if ck.Name == tc.cookie {
					found = ck
				}
			}
			if found == nil {
				t.Fatalf("no %s cookie in %v", tc.cookie, res.Header.Values("Set-Cookie"))
			}
			if !found.HttpOnly || found.SameSite != http.SameSiteLaxMode || found.Path != "/" || found.Domain != "" {
				t.Fatalf("cookie attributes: %+v", found)
			}
			if found.Secure != tc.wantSecure {
				t.Fatalf("Secure = %v, want %v", found.Secure, tc.wantSecure)
			}
		})
	}
}

func TestAnonymousRequestsSetNoCookie(t *testing.T) {
	c := newApp(t).client()
	for _, path := range []string{"/api/v1/auth/session", "/api/v1/positions", "/"} {
		if res := c.req("GET", path, nil); len(res.Cookies()) != 0 {
			t.Fatalf("%s set cookies %v", path, res.Header.Values("Set-Cookie"))
		}
	}
}

func TestLoginFailures(t *testing.T) {
	a := newApp(t)
	a.signup("a@example.com")
	c := a.client()
	for _, body := range []map[string]string{
		{"email": "a@example.com", "password": "wrong password here"},
		{"email": "nobody@example.com", "password": password},
	} {
		res := c.expect(c.req("POST", "/api/v1/auth/login", body), 401)
		if msg := res.errorMsg(t); msg != "invalid email or password" {
			t.Fatalf("message %q", msg)
		}
	}
	if len(c.cookies) != 0 {
		t.Fatal("failed login set a session")
	}
}

func TestSignupValidation(t *testing.T) {
	a := newApp(t)
	a.signup("taken@example.com")
	c := a.client()
	cases := []struct {
		body   map[string]string
		status int
	}{
		{map[string]string{"email": "TAKEN@example.com", "password": password}, 409},
		{map[string]string{"email": "new@example.com", "password": "short"}, 400},
		{map[string]string{"email": "new@example.com", "password": strings.Repeat("x", 11)}, 400},
		{map[string]string{"email": "new@example.com", "password": strings.Repeat("x", 129)}, 400},
		{map[string]string{"email": "not-an-email", "password": password}, 400},
		{map[string]string{"email": "", "password": password}, 400},
	}
	for _, tc := range cases {
		c.expect(c.req("POST", "/api/v1/auth/signup", tc.body), tc.status)
	}
	// Exactly 12 and exactly 128 are fine.
	c.expect(c.req("POST", "/api/v1/auth/signup", map[string]string{"email": "twelve@example.com", "password": strings.Repeat("y", 12)}), 201)
	c.expect(c.req("POST", "/api/v1/auth/signup", map[string]string{"email": "max@example.com", "password": strings.Repeat("y", 128)}), 201)
}

func TestProtectedRoutesNeedSession(t *testing.T) {
	c := newApp(t).client()
	routes := [][2]string{
		{"GET", "/api/v1/positions"}, {"POST", "/api/v1/positions"}, {"GET", "/api/v1/positions/1"},
		{"PUT", "/api/v1/positions/1"}, {"DELETE", "/api/v1/positions/1"},
		{"GET", "/api/v1/people"}, {"POST", "/api/v1/people"}, {"PUT", "/api/v1/people/1"}, {"DELETE", "/api/v1/people/1"},
		{"GET", "/api/v1/events"}, {"POST", "/api/v1/events"}, {"PUT", "/api/v1/events/1"}, {"DELETE", "/api/v1/events/1"},
		{"GET", "/api/v1/resumes"}, {"POST", "/api/v1/resumes"}, {"PUT", "/api/v1/resumes/1"},
		{"GET", "/api/v1/resumes/1/file"}, {"DELETE", "/api/v1/resumes/1"},
		{"GET", "/api/v1/goals"}, {"PUT", "/api/v1/goals"}, {"GET", "/api/v1/export"},
	}
	for _, r := range routes {
		var body any
		if r[0] == "POST" || r[0] == "PUT" {
			body = map[string]any{}
		}
		res := c.expect(c.req(r[0], r[1], body), 401)
		if res.errorMsg(t) != "unauthorized" {
			t.Fatalf("%s %s: %s", r[0], r[1], res.body)
		}
	}
}

func TestCrossOriginRequestsRejected(t *testing.T) {
	a := newApp(t)
	user := a.signup("a@example.com")
	login := map[string]string{"email": "a@example.com", "password": password}
	person := map[string]string{"name": "Grace"}

	for _, hdr := range [][]string{
		{"Sec-Fetch-Site", "cross-site"},
		{"Sec-Fetch-Site", "same-site"},
		{"Origin", "https://evil.example"},
		{"Sec-Fetch-Site", "cross-site", "Origin", "https://evil.example"},
	} {
		// Auth routes are covered too (login CSRF).
		anon := a.client()
		anon.expect(anon.req("POST", "/api/v1/auth/login", login, hdr...), 403)
		anon.expect(anon.req("POST", "/api/v1/auth/signup", map[string]string{"email": "b@example.com", "password": password}, hdr...), 403)
		user.expect(user.req("POST", "/api/v1/auth/logout", nil, hdr...), 403)
		user.expect(user.req("POST", "/api/v1/people", person, hdr...), 403)
		user.expect(user.req("PUT", "/api/v1/goals", map[string]int{"weeklyApplications": 3}, hdr...), 403)
		user.expect(user.req("DELETE", "/api/v1/people/1", nil, hdr...), 403)
	}
	// Same-origin browser requests and plain (non-browser) clients pass.
	user.expect(user.req("POST", "/api/v1/people", person, "Sec-Fetch-Site", "same-origin"), 201)
	user.expect(user.req("POST", "/api/v1/people", person, "Origin", "http://example.com", "Host", "example.com"), 201)
	user.expect(user.req("POST", "/api/v1/people", person), 201)
	// GETs are never blocked (and never change anything).
	user.expect(user.req("GET", "/api/v1/people", nil, "Sec-Fetch-Site", "cross-site"), 200)
}

func TestFormContentTypeRejected(t *testing.T) {
	a := newApp(t)
	a.signup("a@example.com")
	c := a.client()
	form := "email=a%40example.com&password=correct+horse+battery+staple"
	for _, ct := range []string{"application/x-www-form-urlencoded", "text/plain"} {
		c.expect(c.req("POST", "/api/v1/auth/login", form, "Content-Type", ct), 415)
		c.expect(c.req("POST", "/api/v1/auth/signup", form, "Content-Type", ct), 415)
	}
	user := a.signup("b@example.com")
	user.expect(user.req("POST", "/api/v1/people", `{"name":"x"}`, "Content-Type", "text/plain"), 415)
	user.expect(user.req("PUT", "/api/v1/goals", `{"weeklyApplications":3}`, "Content-Type", "application/x-www-form-urlencoded"), 415)
	// A charset parameter is fine.
	user.expect(user.req("POST", "/api/v1/people", `{"name":"x"}`, "Content-Type", "application/json; charset=utf-8"), 201)
}

func TestAuthRateLimit(t *testing.T) {
	c := newApp(t).client()
	// Distinct emails, so only the per-IP limit is in play.
	for i := range 10 {
		body := map[string]string{"email": fmt.Sprintf("nobody%d@example.com", i), "password": password}
		if res := c.req("POST", "/api/v1/auth/login", body); res.StatusCode != 401 {
			t.Fatalf("attempt %d: status %d", i+1, res.StatusCode)
		}
	}
	body := map[string]string{"email": "nobody@example.com", "password": password}
	res := c.expect(c.req("POST", "/api/v1/auth/login", body), 429)
	if !strings.Contains(res.errorMsg(t), "too many") {
		t.Fatalf("body %s", res.body)
	}
	// Sign-up shares the budget.
	c.expect(c.req("POST", "/api/v1/auth/signup", map[string]string{"email": "x@example.com", "password": password}), 429)
}

func TestLoginAccountRateLimit(t *testing.T) {
	c := newApp(t).client()
	body := map[string]string{"email": "Victim@example.com", "password": password}
	for i := range 5 {
		if res := c.req("POST", "/api/v1/auth/login", body); res.StatusCode != 401 {
			t.Fatalf("attempt %d: status %d", i+1, res.StatusCode)
		}
	}
	// Same account (case and spacing normalized) is blocked; others are not.
	c.expect(c.req("POST", "/api/v1/auth/login", map[string]string{"email": " victim@example.com ", "password": password}), 429)
	c.expect(c.req("POST", "/api/v1/auth/login", map[string]string{"email": "other@example.com", "password": password}), 401)
}
