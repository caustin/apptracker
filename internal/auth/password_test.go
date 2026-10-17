package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Fatalf("hash %q", h)
	}
	if ok, err := VerifyPassword("correct horse battery staple", h); !ok || err != nil {
		t.Fatalf("verify = %v, %v", ok, err)
	}
	if ok, _ := VerifyPassword("wrong", h); ok {
		t.Fatal("wrong password verified")
	}
	h2, _ := HashPassword("correct horse battery staple")
	if h == h2 {
		t.Fatal("hashes are not salted")
	}
	for _, bad := range []string{"", "plain", "$argon2i$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA", "$argon2id$v=19$m=x$a$b"} {
		if _, err := VerifyPassword("x", bad); err == nil {
			t.Errorf("malformed hash %q accepted", bad)
		}
	}
}

func TestPasswordRuleCountsCharacters(t *testing.T) {
	if CheckPasswordRule(strings.Repeat("é", 12)) != nil {
		t.Fatal("12 two-byte characters rejected")
	}
	if CheckPasswordRule(strings.Repeat("é", 11)) == nil {
		t.Fatal("11 characters accepted")
	}
}
