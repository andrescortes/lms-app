// Package sqlite provides the SQLite connection layer and repository adapters
// for the Identity service.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// pragmas are applied to every connection through the DSN so that the
// configured database behaves consistently regardless of pool growth.
//
//   - journal_mode(WAL): readers do not block the writer.
//   - synchronous(NORMAL): durability is preserved for the WAL write path.
//   - foreign_keys(ON): referential integrity for profile -> user.
//   - busy_timeout(5000): wait instead of failing under brief write contention.
//   - cache_size(-64000): ~64 MiB page cache (negative value means kibibytes).
var pragmas = []string{
	"journal_mode(WAL)",
	"synchronous(NORMAL)",
	"foreign_keys(ON)",
	"busy_timeout(5000)",
	"cache_size(-64000)",
}

// Open initializes a SQLite database at path, applying the required PRAGMAs on
// every connection and limiting the pool to a single connection so writes are
// serialized for the v1 single-instance deployment. It verifies the critical
// PRAGMAs at startup and fails fast if the database is misconfigured.
//
// The caller owns the returned *sql.DB and must close it on shutdown. Open
// creates the parent directory when missing so a fresh environment can start.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite: database path is required")
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("sqlite: create database directory %q: %w", dir, err)
		}
	}

	dsn := buildDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", path, err)
	}

	// A single writer connection avoids SQLite write-lock contention and keeps
	// behaviour deterministic for the v1 deployment.
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: ping %q: %w", path, err)
	}

	if err := verifyPragmas(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: verify pragmas for %q: %w", path, err)
	}

	return db, nil
}

// buildDSN assembles a modernc.org/sqlite DSN that enables the required PRAGMAs
// on every pooled connection. The file path is passed as a URL query so paths
// containing spaces or reserved characters stay valid.
func buildDSN(path string) string {
	params := url.Values{}
	for _, p := range pragmas {
		params.Add("_pragma", p)
	}
	// "file:" scheme keeps the path as a URI; the PRAGMAs travel as query params.
	return "file:" + path + "?" + params.Encode()
}

// verifyPragmas reads back the PRAGMAs that matter for correctness: WAL journal
// mode and foreign-key enforcement. A mismatch means the DSN did not take
// effect, which would silently weaken durability or referential integrity.
func verifyPragmas(ctx context.Context, db *sql.DB) error {
	if err := expect(ctx, db, "PRAGMA journal_mode;", "wal"); err != nil {
		return err
	}
	if err := expect(ctx, db, "PRAGMA foreign_keys;", "1"); err != nil {
		return err
	}
	return nil
}

// expect runs a single-value PRAGMA and compares its string form to want.
func expect(ctx context.Context, db *sql.DB, query, want string) error {
	var got string
	if err := db.QueryRowContext(ctx, query).Scan(&got); err != nil {
		return fmt.Errorf("query %q: %w", query, err)
	}
	if got != want {
		return fmt.Errorf("pragma %q = %q, want %q", query, got, want)
	}
	return nil
}
