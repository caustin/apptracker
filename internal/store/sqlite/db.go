// Package sqlite implements store.Store over SQLite using the pure-Go
// modernc.org/sqlite driver (no CGO).
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens (creating if needed) the database file with the pragmas every
// connection needs: foreign keys on (or CASCADE / SET NULL silently stop
// working), WAL, a busy timeout and synchronous=NORMAL. Write transactions
// take the lock up front so concurrent writers wait instead of failing.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_txlock", "immediate")
	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// Migrations returns a goose provider over the embedded migrations.
func Migrations(db *sql.DB) (*goose.Provider, error) {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectSQLite3, db, sub)
}

// Migrate applies every pending migration.
func Migrate(ctx context.Context, db *sql.DB) error {
	p, err := Migrations(db)
	if err != nil {
		return err
	}
	_, err = p.Up(ctx)
	return err
}
