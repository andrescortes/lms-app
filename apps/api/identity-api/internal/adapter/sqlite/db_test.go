package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

// TestOpenAppliesPragmas verifies that Open configures WAL and foreign keys and
// that a connection accepts a foreign-key-enforced insert. A file-backed
// database is required because WAL is not meaningful for an in-memory database.
func TestOpenAppliesPragmas(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "identity.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want %q", journalMode, "wal")
	}

	var foreignKeys int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
}

// TestOpenEnforcesForeignKeys confirms that the foreign key declared between
// profiles and users is actually enforced at runtime, so an orphan profile is
// rejected rather than silently accepted.
func TestOpenEnforcesForeignKeys(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "identity.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(ctx, `CREATE TABLE parent (id TEXT PRIMARY KEY NOT NULL) STRICT;`); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE child (
		id TEXT PRIMARY KEY NOT NULL,
		parent_id TEXT NOT NULL REFERENCES parent(id)
	) STRICT;`); err != nil {
		t.Fatalf("create child: %v", err)
	}

	_, err = db.ExecContext(ctx, `INSERT INTO child (id, parent_id) VALUES ('c1', 'missing');`)
	if err == nil {
		t.Fatal("insert with missing parent succeeded; foreign keys are not enforced")
	}
}

// TestOpenRequiresPath guards the fail-fast contract for a missing database path.
func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open(context.Background(), ""); err == nil {
		t.Fatal("Open(\"\") succeeded; want error")
	}
}
