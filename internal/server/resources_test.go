package server_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

type row map[string]any

func (r row) id() int64 { return int64(r["id"].(float64)) }

func (c *client) create(path string, body any) row {
	c.t.Helper()
	var out row
	c.expect(c.req("POST", path, body), 201).json(c.t, &out)
	return out
}

func today() string { return time.Now().UTC().Format(time.DateOnly) }

func TestPeopleCRUD(t *testing.T) {
	c := newApp(t).signup("a@example.com")
	p := c.create("/api/v1/people", row{"name": "  Grace  ", "role": "", "linkedin": "https://linkedin.com/in/g", "notes": "  keep  "})
	if p["name"] != "Grace" || p["role"] != nil || p["notes"] != "  keep  " || p["createdAt"] != today() {
		t.Fatalf("created %v", p)
	}
	path := fmt.Sprintf("/api/v1/people/%d", p.id())

	var got row
	c.expect(c.req("GET", path, nil), 200).json(t, &got)
	if got["linkedin"] != "https://linkedin.com/in/g" {
		t.Fatalf("got %v", got)
	}

	// PUT is a full replace; read-only fields in the body are ignored.
	var upd row
	c.expect(c.req("PUT", path, row{"id": 999, "userId": "someone", "createdAt": "1999-01-01", "name": "Grace H", "company": "Navy"}), 200).json(t, &upd)
	if upd.id() != p.id() || upd["linkedin"] != nil || upd["company"] != "Navy" || upd["createdAt"] != today() || upd["userId"] != p["userId"] {
		t.Fatalf("updated %v", upd)
	}

	var list []row
	c.expect(c.req("GET", "/api/v1/people", nil), 200).json(t, &list)
	if len(list) != 1 {
		t.Fatalf("list %v", list)
	}

	c.expect(c.req("DELETE", path, nil), 200)
	c.expect(c.req("DELETE", path, nil), 404)
	c.expect(c.req("GET", path, nil), 404)
	c.expect(c.req("PUT", path, row{"name": "x"}), 404)
}

func TestPeopleValidation(t *testing.T) {
	c := newApp(t).signup("a@example.com")
	for _, body := range []any{
		row{},
		row{"name": "   "},
		row{"name": "x", "linkedin": "javascript:alert(1)"},
		row{"name": "x", "linkedin": "data:text/html,hi"},
		row{"name": "x", "linkedin": "not a url"},
		row{"name": "x", "email": "not an email"},
		row{"name": "x", "email": "Grace <grace@example.com>"},
		row{"name": strings.Repeat("x", 501)},
		row{"name": "x", "notes": strings.Repeat("x", 50_001)},
		row{"name": 5},
		"{not json",
	} {
		c.expect(c.req("POST", "/api/v1/people", body), 400)
	}
	res := c.expect(c.req("POST", "/api/v1/people", row{"name": "x", "linkedin": "javascript:alert(1)"}), 400)
	if !strings.HasPrefix(res.errorMsg(t), "linkedin:") {
		t.Fatalf("error %q", res.errorMsg(t))
	}
	for _, id := range []string{"abc", "0", "-1", "1.5"} {
		c.expect(c.req("GET", "/api/v1/people/"+id, nil), 400)
		c.expect(c.req("DELETE", "/api/v1/people/"+id, nil), 400)
	}
}

func TestFieldLimits(t *testing.T) {
	c := newApp(t).signup("a@example.com")
	// At the caps is fine; whitespace around a short field is trimmed first.
	var p row
	c.expect(c.req("POST", "/api/v1/people", row{
		"name": " " + strings.Repeat("é", 500) + " ", "email": "grace@example.com",
		"notes": strings.Repeat("x", 50_000),
	}), 201).json(t, &p)
	if len([]rune(p["name"].(string))) != 500 {
		t.Fatalf("name not trimmed: %d characters", len([]rune(p["name"].(string))))
	}
	res := c.expect(c.req("POST", "/api/v1/positions", row{"company": "A", "title": "T", "url": "https://x.example/" + strings.Repeat("a", 2048)}), 400)
	if res.errorMsg(t) != "url: at most 2048 characters" {
		t.Fatalf("error %q", res.errorMsg(t))
	}
	// JSON bodies over 1 MB are refused before decoding.
	big := `{"name":"x","notes":"` + strings.Repeat("x", 1<<20) + `"}`
	res = c.expect(c.req("POST", "/api/v1/people", big), 413)
	if res.errorMsg(t) != "request too large" {
		t.Fatalf("error %q", res.errorMsg(t))
	}
	// Single-line event fields are trimmed like other short fields.
	pos := c.create("/api/v1/positions", row{"company": "A", "title": "T"}).id()
	var e row
	c.expect(c.req("POST", "/api/v1/events", row{"type": "note", "date": "2026-10-13", "positionId": pos, "feedback": "  good  ", "outcome": " "}), 201).json(t, &e)
	if e["feedback"] != "good" || e["outcome"] != nil {
		t.Fatalf("event %v", e)
	}
}

func TestPositionsAndAppliedStamp(t *testing.T) {
	c := newApp(t).signup("a@example.com")

	lead := c.create("/api/v1/positions", row{"company": "Acme", "title": "Eng"})
	if lead["status"] != "lead" || lead["appliedAt"] != nil {
		t.Fatalf("lead %v", lead)
	}
	path := fmt.Sprintf("/api/v1/positions/%d", lead.id())

	// Moving to applied stamps today (UTC).
	var p row
	c.expect(c.req("PUT", path, row{"company": "Acme", "title": "Eng", "status": "applied"}), 200).json(t, &p)
	if p["appliedAt"] != today() {
		t.Fatalf("appliedAt %v, want %s", p["appliedAt"], today())
	}
	// An existing stamp survives an edit that omits it, and later stages.
	c.expect(c.req("PUT", path, row{"company": "Acme", "title": "Eng", "status": "applied", "appliedAt": "2026-01-02"}), 200)
	c.expect(c.req("PUT", path, row{"company": "Acme 2", "title": "Eng", "status": "interviewing"}), 200).json(t, &p)
	if p["appliedAt"] != "2026-01-02" || p["company"] != "Acme 2" {
		t.Fatalf("after edit %v", p)
	}

	// Created at a later stage: stamped. With a date sent: kept.
	if got := c.create("/api/v1/positions", row{"company": "B", "title": "T", "status": "offer"}); got["appliedAt"] != today() {
		t.Fatalf("offer %v", got)
	}
	if got := c.create("/api/v1/positions", row{"company": "B", "title": "T", "status": "applied", "appliedAt": "2025-12-01"}); got["appliedAt"] != "2025-12-01" {
		t.Fatalf("sent date %v", got)
	}
	// Terminal statuses do not stamp.
	if got := c.create("/api/v1/positions", row{"company": "B", "title": "T", "status": "rejected"}); got["appliedAt"] != nil {
		t.Fatalf("rejected %v", got)
	}

	for _, body := range []row{
		{"title": "T"},
		{"company": "C", "title": "T", "status": "bogus"},
		{"company": "C", "title": "T", "url": "javascript:alert(1)"},
		{"company": "C", "title": "T", "salaryMin": 1.5},
		{"company": "C", "title": "T", "salaryMin": -1},
		{"company": "C", "title": "T", "salaryMin": "100"},
		{"company": "C", "title": "T", "appliedAt": "13/10/2026"},
		{"company": "C", "title": "T", "resumeId": 999},
	} {
		c.expect(c.req("POST", "/api/v1/positions", body), 400)
	}
	full := c.create("/api/v1/positions", row{
		"company": "C", "title": "T", "url": "https://c.example/j", "salaryMin": 100000, "salaryMax": 150000,
		"location": "Remote", "source": "", "description": "JD", "impressions": "good",
	})
	if full["salaryMin"] != float64(100000) || full["source"] != nil || full["url"] != "https://c.example/j" {
		t.Fatalf("full %v", full)
	}
}

func TestEvents(t *testing.T) {
	c := newApp(t).signup("a@example.com")
	pos := c.create("/api/v1/positions", row{"company": "Acme", "title": "Eng"})
	person := c.create("/api/v1/people", row{"name": "Grace"})

	ev := c.create("/api/v1/events", row{
		"positionId": pos.id(), "personId": person.id(), "type": "interview", "date": "2026-10-13",
		"notes": "went well", "followupOn": "2026-10-20", "feedback": "",
	})
	if ev["feedback"] != nil || ev["followupOn"] != "2026-10-20" {
		t.Fatalf("event %v", ev)
	}
	general := c.create("/api/v1/events", row{"type": "note", "date": "2026-10-14"})

	for _, body := range []row{
		{"date": "2026-10-13"},
		{"type": "party", "date": "2026-10-13"},
		{"type": "note"},
		{"type": "note", "date": "yesterday"},
		{"type": "note", "date": "2026-10-13", "followupOn": "soon"},
		{"type": "note", "date": "2026-10-13", "positionId": 999},
		{"type": "note", "date": "2026-10-13", "personId": 999},
	} {
		c.expect(c.req("POST", "/api/v1/events", body), 400)
	}

	var upd row
	c.expect(c.req("PUT", fmt.Sprintf("/api/v1/events/%d", ev.id()), row{"positionId": pos.id(), "type": "followup", "date": "2026-10-15"}), 200).json(t, &upd)
	if upd["type"] != "followup" || upd["personId"] != nil || upd["notes"] != nil {
		t.Fatalf("updated %v", upd)
	}

	// Deleting a person clears events.person_id.
	c.expect(c.req("PUT", fmt.Sprintf("/api/v1/events/%d", ev.id()), row{"positionId": pos.id(), "personId": person.id(), "type": "interview", "date": "2026-10-13"}), 200)
	c.expect(c.req("DELETE", fmt.Sprintf("/api/v1/people/%d", person.id()), nil), 200)
	var got row
	c.expect(c.req("GET", fmt.Sprintf("/api/v1/events/%d", ev.id()), nil), 200).json(t, &got)
	if got["personId"] != nil {
		t.Fatalf("personId not cleared: %v", got)
	}

	// Deleting a position deletes its events, and only its events.
	c.expect(c.req("DELETE", fmt.Sprintf("/api/v1/positions/%d", pos.id()), nil), 200)
	c.expect(c.req("GET", fmt.Sprintf("/api/v1/events/%d", ev.id()), nil), 404)
	c.expect(c.req("GET", fmt.Sprintf("/api/v1/events/%d", general.id()), nil), 200)
}

func TestResumes(t *testing.T) {
	a := newApp(t)
	c := a.signup("a@example.com")

	// Rejected uploads.
	c.expect(c.upload("cv.txt", "text/plain", pdf("x"), ""), 400)
	c.expect(c.upload("cv.pdf", "application/pdf", []byte("<html>not a pdf"), ""), 400)
	c.expect(c.upload("cv.pdf", "application/pdf", nil, ""), 400)
	big := append(pdf(""), bytes.Repeat([]byte("x"), 15<<20)...)
	if res := c.upload("big.pdf", "application/pdf", big, ""); res.StatusCode != 400 && res.StatusCode != 413 {
		t.Fatalf("oversized upload: %d %s", res.StatusCode, res.body)
	}
	c.expect(c.req("POST", "/api/v1/resumes", row{"name": "x"}), 400)

	// Name defaults to the file name without .pdf; the extension alone is enough.
	var r row
	c.expect(c.upload("Platform v3.PDF", "application/octet-stream", pdf("one"), ""), 201).json(t, &r)
	if r["name"] != "Platform v3" {
		t.Fatalf("resume %v", r)
	}
	if _, leaked := r["storageKey"]; leaked {
		t.Fatal("storage key sent to the client")
	}
	labelled := c.expect(c.upload("x.pdf", "application/pdf", pdf("two"), "  My label "), 201)
	var r2 row
	labelled.json(t, &r2)
	if r2["name"] != "My label" {
		t.Fatalf("label %v", r2)
	}

	// Download headers are exact.
	res := c.expect(c.req("GET", fmt.Sprintf("/api/v1/resumes/%d/file", r.id()), nil), 200)
	if string(res.body) != string(pdf("one")) {
		t.Fatalf("download body %q", res.body)
	}
	for h, want := range map[string]string{
		"Content-Type":           "application/pdf",
		"Content-Disposition":    `attachment; filename="resume.pdf"`,
		"X-Content-Type-Options": "nosniff",
	} {
		if got := res.Header.Get(h); got != want {
			t.Fatalf("%s = %q, want %q", h, got, want)
		}
	}

	// Rename.
	var renamed row
	c.expect(c.req("PUT", fmt.Sprintf("/api/v1/resumes/%d", r.id()), row{"name": " v4 "}), 200).json(t, &renamed)
	if renamed["name"] != "v4" {
		t.Fatalf("renamed %v", renamed)
	}
	c.expect(c.req("PUT", fmt.Sprintf("/api/v1/resumes/%d", r.id()), row{"name": " "}), 400)
	c.expect(c.req("PUT", "/api/v1/resumes/999", row{"name": "x"}), 404)

	// Deleting a resume unlinks it from positions and removes the file.
	pos := c.create("/api/v1/positions", row{"company": "A", "title": "B", "resumeId": r.id()})
	if pos["resumeId"] != float64(r.id()) {
		t.Fatalf("position %v", pos)
	}
	c.expect(c.req("DELETE", fmt.Sprintf("/api/v1/resumes/%d", r.id()), nil), 200)
	var p row
	c.expect(c.req("GET", fmt.Sprintf("/api/v1/positions/%d", pos.id()), nil), 200).json(t, &p)
	if p["resumeId"] != nil {
		t.Fatalf("resumeId not cleared: %v", p)
	}
	c.expect(c.req("GET", fmt.Sprintf("/api/v1/resumes/%d/file", r.id()), nil), 404)
	c.expect(c.req("DELETE", fmt.Sprintf("/api/v1/resumes/%d", r.id()), nil), 404)
}

func TestResumeLimit(t *testing.T) {
	c := newApp(t).signup("a@example.com")
	for i := range 25 {
		c.expect(c.upload(fmt.Sprintf("cv%d.pdf", i), "application/pdf", pdf("x"), ""), 201)
	}
	res := c.expect(c.upload("cv26.pdf", "application/pdf", pdf("x"), ""), 400)
	if !strings.Contains(res.errorMsg(t), "limit") {
		t.Fatalf("error %s", res.body)
	}
}

func TestGoals(t *testing.T) {
	a := newApp(t)
	c := a.signup("a@example.com")
	var g row
	c.expect(c.req("GET", "/api/v1/goals", nil), 200).json(t, &g)
	if g["weeklyApplications"] != float64(5) || g["targetRole"] != nil {
		t.Fatalf("default %v", g)
	}
	c.expect(c.req("PUT", "/api/v1/goals", row{"weeklyApplications": 8, "targetRole": "Staff", "salaryMin": 200000, "targetDate": "2026-12-31"}), 200)
	c.expect(c.req("GET", "/api/v1/goals", nil), 200).json(t, &g)
	if g["weeklyApplications"] != float64(8) || g["targetRole"] != "Staff" || g["salaryMin"] != float64(200000) {
		t.Fatalf("saved %v", g)
	}
	// Update is a full replace.
	c.expect(c.req("PUT", "/api/v1/goals", row{"weeklyApplications": 3}), 200).json(t, &g)
	if g["weeklyApplications"] != float64(3) || g["targetRole"] != nil {
		t.Fatalf("updated %v", g)
	}
	for _, body := range []row{{}, {"weeklyApplications": 0}, {"weeklyApplications": 2, "targetDate": "soon"}, {"weeklyApplications": 2, "salaryMax": -5}} {
		c.expect(c.req("PUT", "/api/v1/goals", body), 400)
	}
	// Another user still sees defaults.
	other := a.signup("b@example.com")
	other.expect(other.req("GET", "/api/v1/goals", nil), 200).json(t, &g)
	if g["weeklyApplications"] != float64(5) {
		t.Fatalf("other user's goals %v", g)
	}
}

func TestExport(t *testing.T) {
	a := newApp(t)
	alice := a.signup("alice@example.com")
	bob := a.signup("bob@example.com")

	alice.create("/api/v1/positions", row{"company": "AliceCo", "title": "Eng"})
	var r row
	alice.expect(alice.upload("cv.pdf", "application/pdf", pdf("alice-resume"), "Alice CV/v1"), 201).json(t, &r)
	alice.expect(alice.req("PUT", "/api/v1/goals", row{"weeklyApplications": 4}), 200)
	bob.create("/api/v1/positions", row{"company": "BobCo", "title": "Eng"})
	bob.expect(bob.upload("cv.pdf", "application/pdf", pdf("bob-resume"), ""), 201)

	res := alice.expect(alice.req("GET", "/api/v1/export", nil), 200)
	if res.Header.Get("Content-Type") != "application/zip" ||
		res.Header.Get("Content-Disposition") != `attachment; filename="apptracker-export.zip"` {
		t.Fatalf("headers %v", res.Header)
	}
	zr, err := zip.NewReader(bytes.NewReader(res.body), int64(len(res.body)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		names[f.Name] = string(b)
	}
	want := fmt.Sprintf("resumes/%d-Alice_CV_v1.pdf", r.id())
	if len(names) != 2 || names[want] != string(pdf("alice-resume")) {
		t.Fatalf("zip entries %v", keys(names))
	}
	var data struct {
		Positions []row
		People    []row
		Resumes   []row
		Goals     row
	}
	if err := json.Unmarshal([]byte(names["data.json"]), &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Positions) != 1 || data.Positions[0]["company"] != "AliceCo" || len(data.Resumes) != 1 || data.Goals["weeklyApplications"] != float64(4) {
		t.Fatalf("data.json %s", names["data.json"])
	}
	if strings.Contains(names["data.json"], "Bob") {
		t.Fatal("export contains another user's data")
	}
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestTenancy checks that user B can neither read nor change any of user A's
// rows through any route.
func TestTenancy(t *testing.T) {
	a := newApp(t)
	alice := a.signup("alice@example.com")
	bob := a.signup("bob@example.com")

	var res row
	alice.expect(alice.upload("cv.pdf", "application/pdf", pdf("a"), ""), 201).json(t, &res)
	resume := res.id()
	position := alice.create("/api/v1/positions", row{"company": "A", "title": "T", "resumeId": resume}).id()
	person := alice.create("/api/v1/people", row{"name": "P"}).id()
	event := alice.create("/api/v1/events", row{"type": "note", "date": "2026-10-13", "positionId": position, "personId": person}).id()

	// Bob's own rows, to try cross-linking into Alice's.
	bobPos := bob.create("/api/v1/positions", row{"company": "B", "title": "T"}).id()
	bobEvent := bob.create("/api/v1/events", row{"type": "note", "date": "2026-10-13", "positionId": bobPos}).id()

	cases := []struct {
		method, path string
		body         any
		status       int
	}{
		{"GET", fmt.Sprintf("/api/v1/positions/%d", position), nil, 404},
		{"PUT", fmt.Sprintf("/api/v1/positions/%d", position), row{"company": "x", "title": "x"}, 404},
		{"DELETE", fmt.Sprintf("/api/v1/positions/%d", position), nil, 404},
		{"GET", fmt.Sprintf("/api/v1/people/%d", person), nil, 404},
		{"PUT", fmt.Sprintf("/api/v1/people/%d", person), row{"name": "x"}, 404},
		{"DELETE", fmt.Sprintf("/api/v1/people/%d", person), nil, 404},
		{"GET", fmt.Sprintf("/api/v1/events/%d", event), nil, 404},
		{"PUT", fmt.Sprintf("/api/v1/events/%d", event), row{"type": "note", "date": "2026-10-13"}, 404},
		{"DELETE", fmt.Sprintf("/api/v1/events/%d", event), nil, 404},
		{"GET", fmt.Sprintf("/api/v1/resumes/%d/file", resume), nil, 404},
		{"PUT", fmt.Sprintf("/api/v1/resumes/%d", resume), row{"name": "x"}, 404},
		{"DELETE", fmt.Sprintf("/api/v1/resumes/%d", resume), nil, 404},
		// Cross-linking Alice's rows from Bob's is refused.
		{"POST", "/api/v1/positions", row{"company": "x", "title": "x", "resumeId": resume}, 400},
		{"PUT", fmt.Sprintf("/api/v1/positions/%d", bobPos), row{"company": "x", "title": "x", "resumeId": resume}, 400},
		{"POST", "/api/v1/events", row{"type": "note", "date": "2026-10-13", "positionId": position}, 400},
		{"POST", "/api/v1/events", row{"type": "note", "date": "2026-10-13", "personId": person}, 400},
		{"PUT", fmt.Sprintf("/api/v1/events/%d", bobEvent), row{"type": "note", "date": "2026-10-13", "positionId": position}, 400},
		{"PUT", fmt.Sprintf("/api/v1/events/%d", bobEvent), row{"type": "note", "date": "2026-10-13", "personId": person}, 400},
	}
	for _, tc := range cases {
		got := bob.req(tc.method, tc.path, tc.body)
		if got.StatusCode != tc.status {
			t.Errorf("bob %s %s: %d, want %d (%s)", tc.method, tc.path, got.StatusCode, tc.status, got.body)
		}
	}
	// The refusal names the field, so the form can show it there.
	linkRes := bob.expect(bob.req("POST", "/api/v1/positions", row{"company": "x", "title": "x", "resumeId": resume}), 400)
	if linkRes.errorMsg(t) != "resumeId: not found" {
		t.Fatalf("error %q", linkRes.errorMsg(t))
	}
	// Lists only show Bob's rows.
	for _, path := range []string{"/api/v1/positions", "/api/v1/people", "/api/v1/events", "/api/v1/resumes"} {
		var list []row
		bob.expect(bob.req("GET", path, nil), 200).json(t, &list)
		for _, r := range list {
			if r["userId"] == nil || r["userId"] == alice.userID(t) {
				t.Errorf("bob's %s includes %v", path, r)
			}
		}
	}
	// And Alice's data is untouched.
	var p row
	alice.expect(alice.req("GET", fmt.Sprintf("/api/v1/positions/%d", position), nil), 200).json(t, &p)
	if p["company"] != "A" || p["resumeId"] != float64(resume) {
		t.Fatalf("alice's position changed: %v", p)
	}
	alice.expect(alice.req("GET", fmt.Sprintf("/api/v1/people/%d", person), nil), 200)
	alice.expect(alice.req("GET", fmt.Sprintf("/api/v1/events/%d", event), nil), 200)
	alice.expect(alice.req("GET", fmt.Sprintf("/api/v1/resumes/%d/file", resume), nil), 200)
}

func (c *client) userID(t *testing.T) string {
	var s struct{ User struct{ ID string } }
	c.expect(c.req("GET", "/api/v1/auth/session", nil), 200).json(t, &s)
	return s.User.ID
}
