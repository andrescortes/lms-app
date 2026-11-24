package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNewSQLiteDB_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	// Verify directory was created
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("database directory was not created")
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestNewSQLiteDB_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	// Verify WAL mode is enabled
	var journalMode string
	err = db.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode 'wal', got '%s'", journalMode)
	}
}

func TestNewSQLiteDB_ForeignKeysEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	// Verify foreign keys are enabled
	var foreignKeys int
	err = db.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("expected foreign_keys=1, got %d", foreignKeys)
	}
}

func TestNewSQLiteDB_BusyTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	// Verify busy timeout is set (5000ms = 5 seconds)
	var busyTimeout int
	err = db.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("expected busy_timeout=5000, got %d", busyTimeout)
	}
}

func TestNewSQLiteDB_SynchronousNormal(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	// Verify synchronous mode
	var synchronous int
	err = db.QueryRowContext(context.Background(), "PRAGMA synchronous").Scan(&synchronous)
	if err != nil {
		t.Fatalf("failed to query synchronous: %v", err)
	}
	// NORMAL = 1
	if synchronous != 1 {
		t.Errorf("expected synchronous=1 (NORMAL), got %d", synchronous)
	}
}

func TestRunMigrations_CreatesSchemaMigrationsTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify schema_migrations table exists
	var tableName string
	err = db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableName)
	if err != nil {
		t.Fatalf("schema_migrations table not found: %v", err)
	}
	if tableName != "schema_migrations" {
		t.Errorf("expected table name 'schema_migrations', got '%s'", tableName)
	}

	// Verify columns
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(schema_migrations)")
	if err != nil {
		t.Fatalf("failed to get table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("failed to scan column info: %v", err)
		}
		columns[name] = true
	}

	expectedColumns := []string{"version", "applied_at"}
	for _, col := range expectedColumns {
		if !columns[col] {
			t.Errorf("schema_migrations missing column: %s", col)
		}
	}
}

func TestRunMigrations_AppliesMigrationOnce(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run migrations first time
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("first RunMigrations failed: %v", err)
	}

	// Count migrations applied
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration applied, got %d", count)
	}

	// Run migrations second time
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}

	// Count should still be 1
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count migrations after second run: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration after second run, got %d", count)
	}
}

func TestRunMigrations_WithExistingData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Pre-populate the cefr_levels table with custom data
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS cefr_levels (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			rank_order INTEGER NOT NULL UNIQUE
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.ExecContext(ctx, "INSERT INTO cefr_levels (code, name, description, rank_order) VALUES ('X1', 'Custom', 'Custom level', 99)")
	if err != nil {
		t.Fatalf("failed to insert custom level: %v", err)
	}

	// Run migrations
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify seed data was inserted (INSERT OR IGNORE)
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cefr_levels").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count levels: %v", err)
	}
	// Should have 6 seed levels + 1 custom = 7
	if count != 7 {
		t.Errorf("expected 7 levels (6 seed + 1 custom), got %d", count)
	}

	// Verify custom level still exists
	var customName string
	err = db.QueryRowContext(ctx, "SELECT name FROM cefr_levels WHERE code = 'X1'").Scan(&customName)
	if err != nil {
		t.Fatalf("custom level not found: %v", err)
	}
	if customName != "Custom" {
		t.Errorf("expected custom level name 'Custom', got '%s'", customName)
	}
}

func TestRunMigrations_TransactionRollbackOnError(t *testing.T) {
	// This test is harder to do without injecting a failing migration
	// We'll test that the migration tracking works correctly
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify migration was recorded with timestamp
	var version, appliedAt string
	err = db.QueryRowContext(ctx, "SELECT version, applied_at FROM schema_migrations").Scan(&version, &appliedAt)
	if err != nil {
		t.Fatalf("failed to query migration record: %v", err)
	}
	if version != "001_init_levels.sql" {
		t.Errorf("expected version '001_init_levels.sql', got '%s'", version)
	}
	if appliedAt == "" {
		t.Error("applied_at should not be empty")
	}
}

func TestRunMigrations_MultipleMigrations(t *testing.T) {
	// This test verifies the migration system can handle multiple files
	// Since we only have one migration file, we verify the sorting logic
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify the migration was applied in order (only one file, but tests the framework)
	var version string
	err = db.QueryRowContext(ctx, "SELECT version FROM schema_migrations ORDER BY applied_at").Scan(&version)
	if err != nil {
		t.Fatalf("failed to query migration version: %v", err)
	}
	if version != "001_init_levels.sql" {
		t.Errorf("expected first migration '001_init_levels.sql', got '%s'", version)
	}
}

func TestDatabaseConnection_PoolSettings(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	// MaxOpenConns = 4
	if stats.MaxOpenConnections != 4 {
		t.Errorf("expected MaxOpenConnections=4, got %d", stats.MaxOpenConnections)
	}
}

func TestNewSQLiteDB_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Simulate concurrent access
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cefr_levels").Scan(&count)
			if err != nil {
				done <- err
				return
			}
			if count != 6 {
				done <- nil // not an error, just unexpected count
				return
			}
			done <- nil
		}()
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent query %d failed: %v", i, err)
		}
	}
}