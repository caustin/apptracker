package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/caustin/apptracker/internal/domain"
	"github.com/caustin/apptracker/internal/store"
	"github.com/caustin/apptracker/internal/store/sqlite"
)

func openTemp(t *testing.T) *sqlite.Store {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return sqlite.New(db)
}

func mkUser(t *testing.T, s *sqlite.Store, id, email string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.CreateUser(context.Background(), domain.User{ID: id, Email: email, Name: email, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationsUpAndDown(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p, err := sqlite.Migrations(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("down: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'positions'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("positions table still exists after down")
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	const uid = "11111111-1111-1111-1111-111111111111"
	mkUser(t, s, uid, "a@example.com")

	r, err := s.CreateResume(ctx, uid, "cv", "k", 25)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.CreatePosition(ctx, uid, domain.PositionFields{Company: "Acme", Title: "Eng", Status: "lead", ResumeID: &r.ID})
	if err != nil {
		t.Fatal(err)
	}
	person, err := s.CreatePerson(ctx, uid, domain.PersonFields{Name: "Grace"})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := s.CreateEvent(ctx, uid, domain.EventFields{Type: "note", Date: "2026-10-13", PositionID: &p.ID, PersonID: &person.ID})
	if err != nil {
		t.Fatal(err)
	}

	// Deleting a resume clears positions.resume_id (ON DELETE SET NULL).
	if err := s.DeleteResume(ctx, uid, r.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetPosition(ctx, uid, p.ID); got.ResumeID != nil {
		t.Fatalf("resume_id = %v, want NULL", *got.ResumeID)
	}
	// Deleting a person clears events.person_id.
	if err := s.DeletePerson(ctx, uid, person.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetEvent(ctx, uid, ev.ID); got.PersonID != nil {
		t.Fatal("events.person_id not cleared")
	}
	// Deleting a position deletes its events (ON DELETE CASCADE).
	if err := s.DeletePosition(ctx, uid, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetEvent(ctx, uid, ev.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("event after position delete: %v, want ErrNotFound", err)
	}
	// A dangling reference is refused.
	missing := int64(999)
	if _, err := s.CreatePosition(ctx, uid, domain.PositionFields{Company: "X", Title: "Y", Status: "lead", ResumeID: &missing}); err == nil {
		t.Fatal("insert with a missing resume_id succeeded; foreign keys are off")
	}
}

func TestCheckConstraints(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	const uid = "11111111-1111-1111-1111-111111111111"
	mkUser(t, s, uid, "a@example.com")
	if _, err := s.CreatePosition(ctx, uid, domain.PositionFields{Company: "A", Title: "B", Status: "bogus"}); err == nil {
		t.Fatal("status CHECK not enforced")
	}
	if _, err := s.CreateEvent(ctx, uid, domain.EventFields{Type: "bogus", Date: "2026-10-13"}); err == nil {
		t.Fatal("type CHECK not enforced")
	}
}

func TestDuplicateEmail(t *testing.T) {
	s := openTemp(t)
	mkUser(t, s, "11111111-1111-1111-1111-111111111111", "a@example.com")
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.CreateUser(context.Background(), domain.User{ID: "22222222-2222-2222-2222-222222222222", Email: "a@example.com", Name: "x", CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, store.ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestResumeLimit(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	const uid = "11111111-1111-1111-1111-111111111111"
	mkUser(t, s, uid, "a@example.com")
	for range 2 {
		if _, err := s.CreateResume(ctx, uid, "cv", "k", 2); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CreateResume(ctx, uid, "cv", "k", 2); !errors.Is(err, store.ErrResumeLimit) {
		t.Fatalf("err = %v, want ErrResumeLimit", err)
	}
}

func TestSessionsStorage(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	s := sqlite.NewSessions(db, time.Hour)
	defer s.Close()

	if err := s.Set("abc", []byte("data"), time.Hour); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("abc")
	if err != nil || string(got) != "data" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	// The raw session ID is never stored.
	var n int
	db.QueryRow(`SELECT count(*) FROM sessions WHERE id_hash = 'abc'`).Scan(&n)
	if n != 0 {
		t.Fatal("session ID stored unhashed")
	}
	// Expired rows read as missing and are swept.
	if err := s.Set("old", []byte("data"), time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if got, _ := s.Get("old"); got != nil {
		t.Fatal("expired session returned")
	}
	if n, err := s.DeleteExpired(context.Background()); err != nil || n != 1 {
		t.Fatalf("DeleteExpired = %d, %v; want 1", n, err)
	}
	if err := s.Delete("abc"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get("abc"); got != nil {
		t.Fatal("deleted session returned")
	}
}
