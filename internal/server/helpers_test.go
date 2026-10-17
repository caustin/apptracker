package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/config"
	"github.com/caustin/apptracker/internal/server"
	"github.com/caustin/apptracker/internal/store/sqlite"
)

const password = "correct horse battery staple"

type testApp struct {
	t   *testing.T
	app *fiber.App
	cfg config.Config
}

func newApp(t *testing.T, mutate ...func(*config.Config)) *testApp {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{
		Env:     config.EnvDevelopment,
		AppURL:  "http://localhost:5173",
		DBPath:  filepath.Join(dir, "test.db"),
		DataDir: dir,
		Port:    "0",
	}
	for _, m := range mutate {
		m(&cfg)
	}
	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	srv, err := server.New(cfg, db, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		srv.Close()
		db.Close()
	})
	return &testApp{t: t, app: srv.App, cfg: cfg}
}

func prod(c *config.Config) {
	c.Env = config.EnvProduction
	c.AppURL = "https://apptracker.example"
}

// client is one browser: it keeps the session cookie between requests.
type client struct {
	*testApp
	cookies map[string]string
}

func (a *testApp) client() *client { return &client{testApp: a, cookies: map[string]string{}} }

type response struct {
	*http.Response
	body []byte
}

func (r response) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (r response) errorMsg(t *testing.T) string {
	t.Helper()
	var e struct{ Error string }
	r.json(t, &e)
	return e.Error
}

// req sends a request. A non-nil body is JSON-encoded with Content-Type
// application/json unless a Content-Type header is given. hdr is
// alternating name/value pairs.
func (c *client) req(method, path string, body any, hdr ...string) response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		if s, ok := body.(string); ok {
			r = strings.NewReader(s)
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				c.t.Fatal(err)
			}
			r = bytes.NewReader(b)
		}
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	return c.send(req)
}

func (c *client) send(req *http.Request) response {
	c.t.Helper()
	for k, v := range c.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	res, err := c.app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	for _, ck := range res.Cookies() {
		if ck.MaxAge < 0 || ck.Value == "" {
			delete(c.cookies, ck.Name)
		} else {
			c.cookies[ck.Name] = ck.Value
		}
	}
	return response{Response: res, body: body}
}

func (c *client) expect(res response, status int) response {
	c.t.Helper()
	if res.StatusCode != status {
		c.t.Fatalf("%s %s: status %d, want %d; body %s", res.Request.Method, res.Request.URL.Path, res.StatusCode, status, res.body)
	}
	return res
}

// signup creates an account and leaves the client signed in.
func (a *testApp) signup(email string) *client {
	a.t.Helper()
	c := a.client()
	c.expect(c.req("POST", "/api/v1/auth/signup", map[string]string{"email": email, "name": "Test", "password": password}), 201)
	return c
}

// upload posts a multipart resume.
func (c *client) upload(filename, contentType string, data []byte, name string) response {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	w, err := mw.CreatePart(h)
	if err != nil {
		c.t.Fatal(err)
	}
	w.Write(data)
	if name != "" {
		mw.WriteField("name", name)
	}
	mw.Close()
	req := httptest.NewRequest("POST", "/api/v1/resumes", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.send(req)
}

func pdf(content string) []byte { return []byte("%PDF-1.7\n" + content) }
