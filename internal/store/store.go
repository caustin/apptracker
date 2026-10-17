// Package store defines what the handlers need from the database. Every
// method that touches application data takes the signed-in user's ID and is
// scoped to it; another user's row behaves exactly like a missing one.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/caustin/apptracker/internal/domain"
)

var (
	// ErrNotFound: no row with that ID belongs to this user.
	ErrNotFound = errors.New("not found")
	// ErrEmailTaken: sign-up with an email that already has an account.
	ErrEmailTaken = errors.New("email already registered")
	// ErrResumeLimit: the user already has the maximum number of resumes.
	ErrResumeLimit = errors.New("resume limit reached")
)

// LinkError: a linked row (a position's resumeId, an event's positionId or
// personId) is missing or belongs to another user. Field is the JSON name, so
// the message reads like the other validation errors.
type LinkError struct{ Field string }

func (e *LinkError) Error() string { return fmt.Sprintf("%s: not found", e.Field) }

type Store interface {
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)

	// Creating or updating a position or event checks, in the same
	// transaction as the write, that every linked row belongs to the user;
	// otherwise it returns a *LinkError.
	ListPositions(ctx context.Context, userID string) ([]domain.Position, error)
	GetPosition(ctx context.Context, userID string, id int64) (domain.Position, error)
	CreatePosition(ctx context.Context, userID string, f domain.PositionFields) (domain.Position, error)
	UpdatePosition(ctx context.Context, userID string, id int64, f domain.PositionFields) (domain.Position, error)
	DeletePosition(ctx context.Context, userID string, id int64) error

	ListPeople(ctx context.Context, userID string) ([]domain.Person, error)
	GetPerson(ctx context.Context, userID string, id int64) (domain.Person, error)
	CreatePerson(ctx context.Context, userID string, f domain.PersonFields) (domain.Person, error)
	UpdatePerson(ctx context.Context, userID string, id int64, f domain.PersonFields) (domain.Person, error)
	DeletePerson(ctx context.Context, userID string, id int64) error

	ListEvents(ctx context.Context, userID string) ([]domain.Event, error)
	GetEvent(ctx context.Context, userID string, id int64) (domain.Event, error)
	CreateEvent(ctx context.Context, userID string, f domain.EventFields) (domain.Event, error)
	UpdateEvent(ctx context.Context, userID string, id int64, f domain.EventFields) (domain.Event, error)
	DeleteEvent(ctx context.Context, userID string, id int64) error

	ListResumes(ctx context.Context, userID string) ([]domain.Resume, error)
	GetResume(ctx context.Context, userID string, id int64) (domain.Resume, error)
	CountResumes(ctx context.Context, userID string) (int64, error)
	// CreateResume inserts the row unless the user already has max resumes,
	// in which case it returns ErrResumeLimit. The count and insert are atomic.
	CreateResume(ctx context.Context, userID, name, storageKey string, max int64) (domain.Resume, error)
	RenameResume(ctx context.Context, userID string, id int64, name string) (domain.Resume, error)
	DeleteResume(ctx context.Context, userID string, id int64) error

	// GetGoals returns ErrNotFound when the user has not saved goals yet.
	GetGoals(ctx context.Context, userID string) (domain.Goals, error)
	UpsertGoals(ctx context.Context, userID string, g domain.Goals) (domain.Goals, error)
}
