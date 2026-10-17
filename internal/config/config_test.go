package config

import (
	"slices"
	"testing"
)

func TestLoad(t *testing.T) {
	cases := []struct {
		env, url string
		ok       bool
	}{
		{"development", "http://localhost:5173", true},
		{"production", "https://apptracker.example", true},
		{"production", "http://apptracker.example", false},
		{"production", "https://apptracker.example/app", false},
		{"production", "not a url", false},
		{"staging", "https://apptracker.example", false},
	}
	for _, tc := range cases {
		t.Setenv("APP_ENV", tc.env)
		t.Setenv("APP_URL", tc.url)
		_, err := Load()
		if (err == nil) != tc.ok {
			t.Errorf("APP_ENV=%s APP_URL=%s: err = %v", tc.env, tc.url, err)
		}
	}
}

func TestTrustedProxies(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_URL", "http://localhost:5173")

	t.Setenv("TRUSTED_PROXIES", " 10.0.0.1, 172.16.0.0/12 ,,::1")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.1", "172.16.0.0/12", "::1"}
	if !slices.Equal(cfg.TrustedProxies, want) {
		t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}

	t.Setenv("TRUSTED_PROXIES", "10.0.0.1,proxy.internal")
	if _, err := Load(); err == nil {
		t.Fatal("a host name in TRUSTED_PROXIES was accepted")
	}
}
