// Package event provides the in-process, outbox-backed domain event publisher
// for the Identity service. It records events durably (via the registration
// store) and delivers pending events to registered hooks with bounded retries,
// resuming any pending events on startup so a restart does not lose them.
package event

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"lms-english/identity-service/internal/adapter/sqlite/sqlc"
)

// Handler consumes a delivered event. The stable eventID lets a handler
// deduplicate a rare duplicate delivery without a second side effect.
type Handler func(ctx context.Context, eventID, eventType string, payload []byte) error

// Publisher delivers persisted outbox events to in-process handlers. It never
// blocks registration: the use case writes the outbox row, and Dispatch (run at
// startup and on demand) performs delivery in the background.
type Publisher struct {
	db        *sql.DB
	handlers  []Handler
	log       *slog.Logger
	batchSize int64
	maxTries  int
	baseDelay time.Duration
}

// Option customizes a Publisher.
type Option func(*Publisher)

// WithBatchSize limits how many pending events are claimed per pass.
func WithBatchSize(n int64) Option { return func(p *Publisher) { p.batchSize = n } }

// WithMaxTries sets how many delivery attempts a single event receives in one
// Dispatch pass before it is left pending for a later pass.
func WithMaxTries(n int) Option { return func(p *Publisher) { p.maxTries = n } }

// WithBaseDelay sets the initial backoff between retries within a pass.
func WithBaseDelay(d time.Duration) Option { return func(p *Publisher) { p.baseDelay = d } }

// WithLogger sets the structured logger.
func WithLogger(l *slog.Logger) Option { return func(p *Publisher) { p.log = l } }

// NewPublisher builds a Publisher over the given database. Handlers are invoked
// in registration order.
func NewPublisher(db *sql.DB, handlers []Handler, opts ...Option) *Publisher {
	p := &Publisher{
		db:        db,
		handlers:  handlers,
		log:       slog.Default(),
		batchSize: 100,
		maxTries:  5,
		baseDelay: 50 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Publish satisfies usecase.DomainEventPublisher. The event row is already
// persisted atomically with the account by the store, so Publish has nothing to
// enqueue and returns nil. Delivery is performed solely by the background
// Dispatch worker, so a registration never spawns delivery work and a slow or
// failing downstream cannot delay or roll back the registration (FR-009).
func (p *Publisher) Publish(_ context.Context, _ string, _ []byte) error {
	return nil
}

// Dispatch claims pending events and delivers them. It runs once at startup and
// then periodically from the background worker. Delivery is best-effort within a
// pass: an event that still fails after maxTries stays pending for the next
// pass, so a downstream outage does not permanently lose the notification.
func (p *Publisher) Dispatch(ctx context.Context) error {
	pending, err := sqlc.New(p.db).ListPendingEvents(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("list pending events: %w", err)
	}

	for _, ev := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.deliver(ctx, ev)
	}
	return nil
}

// deliver attempts to hand one event to every handler with bounded backoff,
// marking it delivered once all handlers succeed.
func (p *Publisher) deliver(ctx context.Context, ev sqlc.RegistrationEvent) {
	q := sqlc.New(p.db)
	delay := p.baseDelay

	for attempt := 1; attempt <= p.maxTries; attempt++ {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := p.runHandlers(ctx, ev); err != nil {
			_ = q.IncrementEventAttempts(ctx, ev.ID)
			p.log.Warn("event delivery attempt failed",
				"event_id", ev.ID, "attempt", attempt, "error", err)
			if !sleepCtx(ctx, delay) {
				return
			}
			delay *= 2
			continue
		}
		if err := q.MarkEventDelivered(ctx, sqlc.MarkEventDeliveredParams{
			DeliveredAt: formatUTC(time.Now()),
			ID:          ev.ID,
		}); err != nil {
			p.log.Warn("mark event delivered failed", "event_id", ev.ID, "error", err)
		}
		return
	}
	// Left pending on purpose; a later Dispatch pass retries it.
}

// runHandlers invokes every handler for one event, stopping at the first error.
func (p *Publisher) runHandlers(ctx context.Context, ev sqlc.RegistrationEvent) error {
	for _, h := range p.handlers {
		if err := h(ctx, ev.ID, ev.EventType, []byte(ev.Payload)); err != nil {
			return fmt.Errorf("handler: %w", err)
		}
	}
	return nil
}

// sleepCtx waits for d or until ctx is done; it reports false when cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// formatUTC renders a timestamp in the canonical ISO-8601 UTC text form.
func formatUTC(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// ErrNoHandler is returned by RunWhenIdle when there are no handlers configured.
var ErrNoHandler = errors.New("event: no handlers configured")

// RunWhenIdle dispatches once and reports whether every pending event drained.
// It is a convenience for startup and tests.
func (p *Publisher) RunWhenIdle(ctx context.Context) (bool, error) {
	if len(p.handlers) == 0 {
		return false, ErrNoHandler
	}
	if err := p.Dispatch(ctx); err != nil {
		return false, err
	}
	remaining, err := sqlc.New(p.db).ListPendingEvents(ctx, 1)
	if err != nil {
		return false, fmt.Errorf("list pending events: %w", err)
	}
	return len(remaining) == 0, nil
}
