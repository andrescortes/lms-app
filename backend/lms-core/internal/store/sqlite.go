package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewSQLiteDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating database directory %s: %w", dir, err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
		dbPath,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite connection: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Ensure foreign_keys=ON for all connections in the pool
	// This hook runs when a new connection is created
	db.SetConnMaxIdleTime(5 * time.Minute)
	// Note: modernc.org/sqlite doesn't support SetConnHook directly.
	// The DSN pragma should apply per-connection, but we also verify on ping.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	// Verify foreign keys are enabled on the initial connection
	if err := ensureForeignKeys(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return db, nil
}

// ensureForeignKeys enables foreign key constraints on the given connection.
// This should be called on each new connection. In practice, the DSN pragma
// handles this, but we verify it's active.
func ensureForeignKeys(ctx context.Context, db *sql.DB) error {
	var fk int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		return fmt.Errorf("checking foreign_keys pragma: %w", err)
	}
	if fk == 0 {
		if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
			return fmt.Errorf("enabling foreign_keys: %w", err)
		}
	}
	return nil
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	const createTrackingTable = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.ExecContext(ctx, createTrackingTable); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("reading embedded migrations directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := entry.Name()

		var exists int
		err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking migration status %s: %w", version, err)
		}
		if exists > 0 {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + version)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", version, err)
		}

		// Use a timeout context for each migration transaction
		migCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		tx, err := db.BeginTx(migCtx, nil)
		if err != nil {
			return fmt.Errorf("starting transaction for %s: %w", version, err)
		}
		// Ensure rollback on any error path
		defer tx.Rollback()

		if _, err := tx.ExecContext(migCtx, string(sqlBytes)); err != nil {
			return fmt.Errorf("executing migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(migCtx, "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("registering migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration transaction %s: %w", version, err)
		}

		// Cancel the defer to avoid double rollback after successful commit
		cancel()

		slog.Info("SQL migration applied successfully", "version", version)
	}

	return nil
}
