package event

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"lms-english/identity-service/internal/adapter/sqlite/sqlc"
)

func newMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "identity.db")
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "db", "migrations"))
	if err != nil {
		t.Fatalf("resolve migrations dir: %v", err)
	}
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, db, dir); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func seedEvent(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	err := sqlc.New(db).CreateEvent(context.Background(), sqlc.CreateEventParams{
		ID:        id,
		EventType: "UserRegistered",
		Payload:   `{"userId":"` + id + `","email":"a@b.com","role":"STUDENT"}`,
		CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	})
	if err != nil {
		t.Fatalf("seed event: %v", err)
	}
}

func pendingCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	rows, err := sqlc.New(db).ListPendingEvents(context.Background(), 100)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	return len(rows)
}

// TestDispatchDeliversAndMarks verifies a delivered event is handed to the hook
// and marked delivered.
func TestDispatchDeliversAndMarks(t *testing.T) {
	db := newMigratedDB(t)
	seedEvent(t, db, "ev1")

	var gotID string
	pub := NewPublisher(db, []Handler{
		func(_ context.Context, id, _ string, _ []byte) error { gotID = id; return nil },
	})

	if err := pub.Dispatch(context.Background()); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if gotID != "ev1" {
		t.Fatalf("handler got id %q, want ev1", gotID)
	}
	if n := pendingCount(t, db); n != 0 {
		t.Fatalf("pending after dispatch = %d, want 0", n)
	}
}

// TestDispatchRetriesThenSucceeds verifies a transient failure is retried and
// the event is eventually delivered.
func TestDispatchRetriesThenSucceeds(t *testing.T) {
	db := newMigratedDB(t)
	seedEvent(t, db, "ev1")

	var calls int
	pub := NewPublisher(db, []Handler{
		func(_ context.Context, _, _ string, _ []byte) error {
			calls++
			if calls < 3 {
				return errors.New("transient")
			}
			return nil
		},
	}, WithBaseDelay(time.Millisecond))

	if err := pub.Dispatch(context.Background()); err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if calls < 3 {
		t.Fatalf("handler calls = %d, want retries", calls)
	}
	if n := pendingCount(t, db); n != 0 {
		t.Fatalf("pending after retried delivery = %d, want 0", n)
	}
}

// TestPendingReprocessedAfterRestart simulates a restart by building a fresh
// publisher over the same database; a still-pending event must be delivered.
func TestPendingReprocessedAfterRestart(t *testing.T) {
	db := newMigratedDB(t)
	seedEvent(t, db, "ev1")

	// First publisher always fails and exhausts its tries, leaving the event pending.
	failing := NewPublisher(db, []Handler{
		func(_ context.Context, _, _ string, _ []byte) error { return errors.New("down") },
	}, WithMaxTries(1), WithBaseDelay(time.Millisecond))
	_ = failing.Dispatch(context.Background())
	if n := pendingCount(t, db); n != 1 {
		t.Fatalf("pending after failed pass = %d, want 1", n)
	}

	// A new publisher (as after a restart) must pick the pending event up.
	var delivered bool
	recovered := NewPublisher(db, []Handler{
		func(_ context.Context, _, _ string, _ []byte) error { delivered = true; return nil },
	})
	if err := recovered.Dispatch(context.Background()); err != nil {
		t.Fatalf("recovered Dispatch() error = %v", err)
	}
	if !delivered {
		t.Fatal("pending event was not delivered after restart")
	}
}

// TestPublishDoesNotDispatch proves delivery is worker-only: Publish leaves the
// durable outbox row untouched and invokes no handler (FR-009).
func TestPublishDoesNotDispatch(t *testing.T) {
	db := newMigratedDB(t)
	seedEvent(t, db, "ev1")

	called := make(chan string, 1)
	pub := NewPublisher(db, []Handler{
		func(_ context.Context, id, _ string, _ []byte) error { called <- id; return nil },
	})

	if err := pub.Publish(context.Background(), "ev1", nil); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	select {
	case id := <-called:
		t.Fatalf("Publish dispatched event %q; delivery must be worker-only", id)
	case <-time.After(100 * time.Millisecond):
	}
	if n := pendingCount(t, db); n != 1 {
		t.Fatalf("pending after Publish = %d, want 1 (unchanged)", n)
	}
}

// TestNoDuplicateEffectForRepeatedID documents that handlers receive a stable id
// they can use to deduplicate; delivering the same logical event twice must not
// create two downstream effects when the handler dedupes by id.
func TestNoDuplicateEffectForRepeatedID(t *testing.T) {
	db := newMigratedDB(t)
	seedEvent(t, db, "ev1")

	seen := map[string]int{}
	var mu sync.Mutex
	pub := NewPublisher(db, []Handler{
		func(_ context.Context, id, _ string, _ []byte) error {
			mu.Lock()
			seen[id]++
			mu.Unlock()
			return nil
		},
	})

	// Deliver twice (as a duplicate delivery would).
	_ = pub.Dispatch(context.Background())
	_ = pub.Dispatch(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if seen["ev1"] == 0 {
		t.Fatal("event was not delivered")
	}
	// The effect keyed by the stable id is one entry regardless of deliveries.
	if len(seen) != 1 {
		t.Fatalf("distinct event ids = %d, want 1", len(seen))
	}
}
