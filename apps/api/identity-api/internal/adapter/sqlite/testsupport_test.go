package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

// migrationsDir is resolved relative to this package so tests can apply the real
// goose migrations to an isolated database.
func migrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "db", "migrations"))
	if err != nil {
		t.Fatalf("resolve migrations dir: %v", err)
	}
	return dir
}

// newMigratedMemoryDB opens a migrated database for integration tests. The
// database is file-backed in a temporary directory: an in-memory database cannot
// apply the configured page-cache/WAL PRAGMAs and would not exercise the real
// storage path. The name is kept short for call-site clarity.
func newMigratedMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	return newTempFileDB(t)
}

// newTempFileDB opens a migrated file-backed database for tests that need WAL.
func newTempFileDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	db, err := Open(ctx, filepath.Join(t.TempDir(), "identity.db"))
	if err != nil {
		t.Fatalf("open file db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, db, migrationsDir(t)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}
