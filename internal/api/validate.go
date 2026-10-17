package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
)

// Length caps, in characters. The app-wide body limit has to fit a resume
// upload, so JSON bodies and fields get their own, much smaller, limits.
const (
	maxJSONBody = 1 << 20 // 1 MB
	maxShort    = 500     // names, titles, single-line fields
	maxURL      = 2048
	maxText     = 50_000 // descriptions and notes; a long job description fits
)

// parseID reads the :id route parameter; anything but a positive integer is a 400.
func parseID(c fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	return id, nil
}

// decodeJSON requires an application/json body (415 otherwise; binding alone
// does not enforce it, and the check is part of the CSRF defence) and decodes
// it into dst. Unknown fields such as id, userId and createdAt are ignored.
func decodeJSON(c fiber.Ctx, dst any) error {
	mt, _, err := mime.ParseMediaType(c.Get(fiber.HeaderContentType))
	if err != nil || mt != fiber.MIMEApplicationJSON {
		return fiber.NewError(fiber.StatusUnsupportedMediaType, "content type must be application/json")
	}
	if len(c.Body()) > maxJSONBody {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "request too large")
	}
	if err := json.Unmarshal(c.Body(), dst); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field != "" {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s: expected %s", typeErr.Field, typeErr.Type))
		}
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body")
	}
	return nil
}

// validator collects field errors so one response reports all of them, as
// "field: message; field: message".
type validator struct{ errs []string }

func (v *validator) fail(field, msg string) { v.errs = append(v.errs, field+": "+msg) }

func (v *validator) err() error {
	if len(v.errs) == 0 {
		return nil
	}
	return fiber.NewError(fiber.StatusBadRequest, strings.Join(v.errs, "; "))
}

// trim trims; empty becomes NULL. Callers check the length.
func trim(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

// maxLen fails when s is longer than max characters.
func (v *validator) maxLen(field, s string, max int) {
	if utf8.RuneCountInString(s) > max {
		v.fail(field, fmt.Sprintf("at most %d characters", max))
	}
}

// required trims and rejects empty or missing.
func (v *validator) required(field string, p *string) string {
	s := trim(p)
	if s == nil {
		v.fail(field, "required")
		return ""
	}
	v.maxLen(field, *s, maxShort)
	return *s
}

// str trims; empty becomes NULL.
func (v *validator) str(field string, p *string) *string {
	s := trim(p)
	if s != nil {
		v.maxLen(field, *s, maxShort)
	}
	return s
}

// text keeps whitespace (descriptions, notes); empty becomes NULL.
func (v *validator) text(field string, p *string) *string {
	if trim(p) == nil {
		return nil
	}
	v.maxLen(field, strings.TrimSpace(*p), maxText)
	return p
}

// email accepts a bare address (no display name). Empty becomes NULL.
func (v *validator) email(field string, p *string) *string {
	s := v.str(field, p)
	if s == nil {
		return nil
	}
	if addr, err := mail.ParseAddress(*s); err != nil || addr.Address != *s {
		v.fail(field, "must be an email address")
		return nil
	}
	return s
}

// httpURL accepts only absolute http(s) URLs, because these are rendered as
// links; javascript:, data: and the like are rejected. Empty becomes NULL.
func (v *validator) httpURL(field string, p *string) *string {
	s := trim(p)
	if s == nil {
		return nil
	}
	if utf8.RuneCountInString(*s) > maxURL {
		v.fail(field, fmt.Sprintf("at most %d characters", maxURL))
		return nil
	}
	u, err := url.Parse(*s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		v.fail(field, "must be an http or https URL")
		return nil
	}
	return s
}

// date accepts YYYY-MM-DD only. Empty becomes NULL.
func (v *validator) date(field string, p *string) *string {
	s := trim(p)
	if s == nil {
		return nil
	}
	if _, err := time.Parse(time.DateOnly, *s); err != nil {
		v.fail(field, "must be a date in YYYY-MM-DD form")
		return nil
	}
	return s
}

// requiredDate is date but rejects missing.
func (v *validator) requiredDate(field string, p *string) string {
	if trim(p) == nil {
		v.fail(field, "required")
		return ""
	}
	if d := v.date(field, p); d != nil {
		return *d
	}
	return ""
}

// enum checks s against the allowed values; def applies when missing.
func (v *validator) enum(field string, p *string, ok func(string) bool, def string) string {
	s := trim(p)
	if s == nil {
		if def == "" {
			v.fail(field, "required")
		}
		return def
	}
	if !ok(*s) {
		v.fail(field, "invalid value")
		return def
	}
	return *s
}

// money is a whole-dollar amount, zero or more.
func (v *validator) money(field string, p *int64) *int64 {
	if p != nil && *p < 0 {
		v.fail(field, "must not be negative")
		return nil
	}
	return p
}

// ref is an optional link to another row's ID.
func (v *validator) ref(field string, p *int64) *int64 {
	if p != nil && *p <= 0 {
		v.fail(field, "invalid id")
		return nil
	}
	return p
}
