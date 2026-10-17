package config

import "testing"

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
