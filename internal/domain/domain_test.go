package domain

import "testing"

func TestStampApplied(t *testing.T) {
	s := func(v string) *string { return &v }
	const today = "2026-10-13"
	cases := []struct {
		name           string
		status         string
		sent, existing *string
		want           *string
	}{
		{"lead stays empty", "lead", nil, nil, nil},
		{"applied stamps today", "applied", nil, nil, s(today)},
		{"later stage stamps today", "interviewing", nil, nil, s(today)},
		{"rejected does not stamp", "rejected", nil, nil, nil},
		{"withdrawn does not stamp", "withdrawn", nil, nil, nil},
		{"existing kept", "offer", nil, s("2026-01-01"), s("2026-01-01")},
		{"existing kept even if status drops back", "lead", nil, s("2026-01-01"), s("2026-01-01")},
		{"sent wins", "applied", s("2025-12-01"), s("2026-01-01"), s("2025-12-01")},
	}
	for _, tc := range cases {
		got := StampApplied(tc.status, tc.sent, tc.existing, today)
		if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, deref(got), deref(tc.want))
		}
	}
}

func deref(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
