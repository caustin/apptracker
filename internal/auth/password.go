package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	MinPasswordLength = 12
	MaxPasswordLength = 128
)

// argon2id parameters: OWASP's second recommended set (19 MiB, 2 passes,
// 1 lane), modest enough for a small VM.
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

var (
	ErrPasswordLength = fmt.Errorf("password must be %d–%d characters", MinPasswordLength, MaxPasswordLength)
	errBadHash        = errors.New("auth: malformed password hash")
)

// CheckPasswordRule enforces the 12–128 character rule (counted in characters,
// not bytes).
func CheckPasswordRule(pw string) error {
	n := utf8.RuneCountInString(pw)
	if n < MinPasswordLength || n > MaxPasswordLength {
		return ErrPasswordLength
	}
	return nil
}

// HashPassword returns a PHC-format argon2id hash.
func HashPassword(pw string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword reports whether pw matches the PHC hash. The parameters are
// read from the hash, so they can be raised later without breaking old ones.
func VerifyPassword(pw, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errBadHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errBadHash
	}
	var mem, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &t, &p); err != nil {
		return false, errBadHash
	}
	if mem > 64*1024 || t < 1 || t > 10 || p < 1 || p > 4 {
		return false, errBadHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errBadHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) < 16 || len(want) > 64 {
		return false, errBadHash
	}
	got := argon2.IDKey([]byte(pw), salt, t, mem, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// dummyHash is verified against when the email is unknown, so a login for a
// missing account costs the same time as a wrong password.
var dummyHash = func() string {
	h, err := HashPassword("not-a-real-password-for-timing-only")
	if err != nil {
		panic(err)
	}
	return h
}()
