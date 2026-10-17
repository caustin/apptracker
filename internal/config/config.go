// Package config reads the server's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	// Env is "development" or "production". Production turns on the Secure
	// cookie, the __Host- cookie prefix and HSTS, and hides /swagger.
	Env string
	// AppURL is the public origin, e.g. https://apptracker.example.com.
	AppURL string
	// TrustedOrigins are extra origins the CSRF check accepts. Empty unless a
	// proxy rewrites Host, in which case it holds exactly AppURL.
	TrustedOrigins []string
	DBPath         string
	DataDir        string
	Port           string
}

func (c Config) IsProd() bool { return c.Env == EnvProduction }

// Load reads the environment. It fails closed in production unless APP_URL is
// a valid https:// origin.
func Load() (Config, error) {
	cfg := Config{
		Env:     getenv("APP_ENV", EnvDevelopment),
		AppURL:  getenv("APP_URL", "http://localhost:5173"),
		DBPath:  getenv("DB_PATH", "data/app.db"),
		DataDir: getenv("DATA_DIR", "data"),
		Port:    getenv("PORT", "8080"),
	}
	if cfg.Env != EnvDevelopment && cfg.Env != EnvProduction {
		return Config{}, fmt.Errorf("APP_ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, cfg.Env)
	}
	if err := checkAppURL(cfg.AppURL, cfg.IsProd()); err != nil {
		return Config{}, err
	}
	if os.Getenv("TRUST_APP_URL_ORIGIN") == "true" {
		cfg.TrustedOrigins = []string{strings.TrimRight(cfg.AppURL, "/")}
	}
	return cfg, nil
}

func checkAppURL(raw string, prod bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("APP_URL must be an absolute http(s) URL")
	}
	if u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("APP_URL must be an origin only, with no path, query or fragment")
	}
	if prod && u.Scheme != "https" {
		return errors.New("APP_URL must be https:// in production")
	}
	return nil
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
