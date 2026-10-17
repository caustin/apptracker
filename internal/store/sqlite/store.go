package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/caustin/apptracker/internal/domain"
	"github.com/caustin/apptracker/internal/store"
	"github.com/caustin/apptracker/internal/store/sqlite/gen"
)

// Store adapts the sqlc-generated queries to store.Store and domain types.
type Store struct {
	db *sql.DB
	q  *gen.Queries
}

var _ store.Store = (*Store)(nil)

func New(db *sql.DB) *Store { return &Store{db: db, q: gen.New(db)} }

// inTx runs fn in a transaction, committing if it returns nil.
func (s *Store) inTx(ctx context.Context, fn func(q *gen.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// notFound maps sql.ErrNoRows to store.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	return err
}

func deleted(n int64, err error) error {
	if err != nil {
		return err
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- nullable conversions ---

func nstr(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func nint(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func pstr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	return &n.String
}

func pint(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

// mapRows converts a slice of generated rows to domain types.
func mapRows[G, D any](rows []G, err error, f func(G) D) ([]D, error) {
	if err != nil {
		return nil, err
	}
	out := make([]D, len(rows))
	for i, r := range rows {
		out[i] = f(r)
	}
	return out, nil
}

// mapRow converts one generated row, mapping "no rows" to store.ErrNotFound.
func mapRow[G, D any](row G, err error, f func(G) D) (D, error) {
	if err != nil {
		var zero D
		return zero, notFound(err)
	}
	return f(row), nil
}

// --- users ---

func toUser(u gen.User) domain.User {
	return domain.User{
		ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: pstr(u.PasswordHash),
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
}

func (s *Store) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	row, err := s.q.CreateUser(ctx, gen.CreateUserParams{
		ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: nstr(u.PasswordHash),
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	})
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: users.email") {
		return domain.User{}, store.ErrEmailTaken
	}
	return mapRow(row, err, toUser)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	return mapRow(row, err, toUser)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	row, err := s.q.GetUserByID(ctx, id)
	return mapRow(row, err, toUser)
}

// --- positions ---

func toPosition(p gen.Position) domain.Position {
	return domain.Position{
		ID: p.ID, UserID: p.UserID, Company: p.Company, Title: p.Title,
		URL: pstr(p.Url), Location: pstr(p.Location), Source: pstr(p.Source),
		Description: pstr(p.Description), Impressions: pstr(p.Impressions),
		ResumeID: pint(p.ResumeID), SalaryMin: pint(p.SalaryMin), SalaryMax: pint(p.SalaryMax),
		Status: p.Status, AppliedAt: pstr(p.AppliedAt), CreatedAt: p.CreatedAt,
	}
}

func (s *Store) ListPositions(ctx context.Context, userID string) ([]domain.Position, error) {
	rows, err := s.q.ListPositions(ctx, userID)
	return mapRows(rows, err, toPosition)
}

func (s *Store) GetPosition(ctx context.Context, userID string, id int64) (domain.Position, error) {
	row, err := s.q.GetPosition(ctx, gen.GetPositionParams{UserID: userID, ID: id})
	return mapRow(row, err, toPosition)
}

func (s *Store) CreatePosition(ctx context.Context, userID string, f domain.PositionFields) (domain.Position, error) {
	row, err := s.q.CreatePosition(ctx, gen.CreatePositionParams{
		UserID: userID, Company: f.Company, Title: f.Title, Url: nstr(f.URL),
		Location: nstr(f.Location), Source: nstr(f.Source), Description: nstr(f.Description),
		Impressions: nstr(f.Impressions), ResumeID: nint(f.ResumeID),
		SalaryMin: nint(f.SalaryMin), SalaryMax: nint(f.SalaryMax), Status: f.Status,
		AppliedAt: nstr(f.AppliedAt), CreatedAt: domain.Today(),
	})
	return mapRow(row, err, toPosition)
}

func (s *Store) UpdatePosition(ctx context.Context, userID string, id int64, f domain.PositionFields) (domain.Position, error) {
	row, err := s.q.UpdatePosition(ctx, gen.UpdatePositionParams{
		Company: f.Company, Title: f.Title, Url: nstr(f.URL), Location: nstr(f.Location),
		Source: nstr(f.Source), Description: nstr(f.Description), Impressions: nstr(f.Impressions),
		ResumeID: nint(f.ResumeID), SalaryMin: nint(f.SalaryMin), SalaryMax: nint(f.SalaryMax),
		Status: f.Status, AppliedAt: nstr(f.AppliedAt), UserID: userID, ID: id,
	})
	return mapRow(row, err, toPosition)
}

func (s *Store) DeletePosition(ctx context.Context, userID string, id int64) error {
	return deleted(s.q.DeletePosition(ctx, gen.DeletePositionParams{UserID: userID, ID: id}))
}

// --- people ---

func toPerson(p gen.Person) domain.Person {
	return domain.Person{
		ID: p.ID, UserID: p.UserID, Name: p.Name, Role: pstr(p.Role), Company: pstr(p.Company),
		Email: pstr(p.Email), Phone: pstr(p.Phone), LinkedIn: pstr(p.Linkedin),
		Notes: pstr(p.Notes), CreatedAt: p.CreatedAt,
	}
}

func (s *Store) ListPeople(ctx context.Context, userID string) ([]domain.Person, error) {
	rows, err := s.q.ListPeople(ctx, userID)
	return mapRows(rows, err, toPerson)
}

func (s *Store) GetPerson(ctx context.Context, userID string, id int64) (domain.Person, error) {
	row, err := s.q.GetPerson(ctx, gen.GetPersonParams{UserID: userID, ID: id})
	return mapRow(row, err, toPerson)
}

func (s *Store) CreatePerson(ctx context.Context, userID string, f domain.PersonFields) (domain.Person, error) {
	row, err := s.q.CreatePerson(ctx, gen.CreatePersonParams{
		UserID: userID, Name: f.Name, Role: nstr(f.Role), Company: nstr(f.Company),
		Email: nstr(f.Email), Phone: nstr(f.Phone), Linkedin: nstr(f.LinkedIn),
		Notes: nstr(f.Notes), CreatedAt: domain.Today(),
	})
	return mapRow(row, err, toPerson)
}

func (s *Store) UpdatePerson(ctx context.Context, userID string, id int64, f domain.PersonFields) (domain.Person, error) {
	row, err := s.q.UpdatePerson(ctx, gen.UpdatePersonParams{
		Name: f.Name, Role: nstr(f.Role), Company: nstr(f.Company), Email: nstr(f.Email),
		Phone: nstr(f.Phone), Linkedin: nstr(f.LinkedIn), Notes: nstr(f.Notes),
		UserID: userID, ID: id,
	})
	return mapRow(row, err, toPerson)
}

func (s *Store) DeletePerson(ctx context.Context, userID string, id int64) error {
	return deleted(s.q.DeletePerson(ctx, gen.DeletePersonParams{UserID: userID, ID: id}))
}

// --- events ---

func toEvent(e gen.Event) domain.Event {
	return domain.Event{
		ID: e.ID, UserID: e.UserID, PositionID: pint(e.PositionID), PersonID: pint(e.PersonID),
		Type: e.Type, Date: e.Date, Notes: pstr(e.Notes), Feedback: pstr(e.Feedback),
		Outcome: pstr(e.Outcome), FollowupOn: pstr(e.FollowupOn), CreatedAt: e.CreatedAt,
	}
}

func (s *Store) ListEvents(ctx context.Context, userID string) ([]domain.Event, error) {
	rows, err := s.q.ListEvents(ctx, userID)
	return mapRows(rows, err, toEvent)
}

func (s *Store) GetEvent(ctx context.Context, userID string, id int64) (domain.Event, error) {
	row, err := s.q.GetEvent(ctx, gen.GetEventParams{UserID: userID, ID: id})
	return mapRow(row, err, toEvent)
}

func (s *Store) CreateEvent(ctx context.Context, userID string, f domain.EventFields) (domain.Event, error) {
	row, err := s.q.CreateEvent(ctx, gen.CreateEventParams{
		UserID: userID, PositionID: nint(f.PositionID), PersonID: nint(f.PersonID),
		Type: f.Type, Date: f.Date, Notes: nstr(f.Notes), Feedback: nstr(f.Feedback),
		Outcome: nstr(f.Outcome), FollowupOn: nstr(f.FollowupOn), CreatedAt: domain.Today(),
	})
	return mapRow(row, err, toEvent)
}

func (s *Store) UpdateEvent(ctx context.Context, userID string, id int64, f domain.EventFields) (domain.Event, error) {
	row, err := s.q.UpdateEvent(ctx, gen.UpdateEventParams{
		PositionID: nint(f.PositionID), PersonID: nint(f.PersonID), Type: f.Type, Date: f.Date,
		Notes: nstr(f.Notes), Feedback: nstr(f.Feedback), Outcome: nstr(f.Outcome),
		FollowupOn: nstr(f.FollowupOn), UserID: userID, ID: id,
	})
	return mapRow(row, err, toEvent)
}

func (s *Store) DeleteEvent(ctx context.Context, userID string, id int64) error {
	return deleted(s.q.DeleteEvent(ctx, gen.DeleteEventParams{UserID: userID, ID: id}))
}

// --- resumes ---

func toResume(r gen.Resume) domain.Resume {
	return domain.Resume{ID: r.ID, UserID: r.UserID, Name: r.Name, StorageKey: r.StorageKey, CreatedAt: r.CreatedAt}
}

func (s *Store) ListResumes(ctx context.Context, userID string) ([]domain.Resume, error) {
	rows, err := s.q.ListResumes(ctx, userID)
	return mapRows(rows, err, toResume)
}

func (s *Store) GetResume(ctx context.Context, userID string, id int64) (domain.Resume, error) {
	row, err := s.q.GetResume(ctx, gen.GetResumeParams{UserID: userID, ID: id})
	return mapRow(row, err, toResume)
}

func (s *Store) CountResumes(ctx context.Context, userID string) (int64, error) {
	return s.q.CountResumes(ctx, userID)
}

func (s *Store) CreateResume(ctx context.Context, userID, name, storageKey string, max int64) (domain.Resume, error) {
	var row gen.Resume
	err := s.inTx(ctx, func(q *gen.Queries) error {
		n, err := q.CountResumes(ctx, userID)
		if err != nil {
			return err
		}
		if n >= max {
			return store.ErrResumeLimit
		}
		row, err = q.CreateResume(ctx, gen.CreateResumeParams{
			UserID: userID, Name: name, StorageKey: storageKey, CreatedAt: domain.Today(),
		})
		return err
	})
	if err != nil {
		return domain.Resume{}, err
	}
	return toResume(row), nil
}

func (s *Store) RenameResume(ctx context.Context, userID string, id int64, name string) (domain.Resume, error) {
	row, err := s.q.RenameResume(ctx, gen.RenameResumeParams{Name: name, UserID: userID, ID: id})
	return mapRow(row, err, toResume)
}

func (s *Store) DeleteResume(ctx context.Context, userID string, id int64) error {
	return deleted(s.q.DeleteResume(ctx, gen.DeleteResumeParams{UserID: userID, ID: id}))
}

// --- goals ---

func toGoals(g gen.Goal) domain.Goals {
	return domain.Goals{
		UserID: g.UserID, WeeklyApplications: g.WeeklyApplications,
		SalaryMin: pint(g.SalaryMin), SalaryMax: pint(g.SalaryMax),
		TargetRole: pstr(g.TargetRole), TargetDate: pstr(g.TargetDate), Notes: pstr(g.Notes),
	}
}

func (s *Store) GetGoals(ctx context.Context, userID string) (domain.Goals, error) {
	row, err := s.q.GetGoals(ctx, userID)
	return mapRow(row, err, toGoals)
}

func (s *Store) UpsertGoals(ctx context.Context, g domain.Goals) (domain.Goals, error) {
	if g.UserID == "" {
		return domain.Goals{}, fmt.Errorf("upsert goals: empty user ID")
	}
	row, err := s.q.UpsertGoals(ctx, gen.UpsertGoalsParams{
		UserID: g.UserID, WeeklyApplications: g.WeeklyApplications,
		SalaryMin: nint(g.SalaryMin), SalaryMax: nint(g.SalaryMax),
		TargetRole: nstr(g.TargetRole), TargetDate: nstr(g.TargetDate), Notes: nstr(g.Notes),
	})
	return mapRow(row, err, toGoals)
}
