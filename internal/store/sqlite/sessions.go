package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/caustin/apptracker/internal/store/sqlite/gen"
)

// Sessions is a fiber.Storage over the sessions table, using the app's own
// connection (the ready-made gofiber/storage/sqlite3 adapter needs CGO). Keys
// are stored as SHA-256 hashes, so the table never holds a usable session ID.
type Sessions struct {
	q    *gen.Queries
	stop chan struct{}
	once sync.Once
}

var _ fiber.Storage = (*Sessions)(nil)

// NewSessions returns the storage and starts a background sweep that deletes
// expired rows every interval. Close stops the sweep (it does not close db).
func NewSessions(db *sql.DB, sweepEvery time.Duration) *Sessions {
	s := &Sessions{q: gen.New(db), stop: make(chan struct{})}
	go s.sweep(sweepEvery)
	return s
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (s *Sessions) sweep(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			if _, err := s.DeleteExpired(context.Background()); err != nil {
				slog.Error("session sweep", "err", err)
			}
		}
	}
}

// DeleteExpired removes expired sessions and reports how many went.
func (s *Sessions) DeleteExpired(ctx context.Context) (int64, error) {
	return s.q.DeleteExpiredSessions(ctx, time.Now().Unix())
}

func (s *Sessions) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}
	data, err := s.q.GetSession(ctx, gen.GetSessionParams{IDHash: hashKey(key), Now: time.Now().Unix()})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return data, err
}

func (s *Sessions) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *Sessions) SetWithContext(ctx context.Context, key string, val []byte, exp time.Duration) error {
	if key == "" || len(val) == 0 {
		return nil
	}
	var expiresAt int64
	if exp > 0 {
		expiresAt = time.Now().Add(exp).Unix()
	}
	return s.q.SetSession(ctx, gen.SetSessionParams{IDHash: hashKey(key), Data: val, ExpiresAt: expiresAt})
}

func (s *Sessions) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *Sessions) DeleteWithContext(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return s.q.DeleteSession(ctx, hashKey(key))
}

func (s *Sessions) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *Sessions) ResetWithContext(ctx context.Context) error {
	return s.q.DeleteAllSessions(ctx)
}

func (s *Sessions) Reset() error { return s.ResetWithContext(context.Background()) }

func (s *Sessions) Close() error {
	s.once.Do(func() { close(s.stop) })
	return nil
}
