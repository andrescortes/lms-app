-- +goose Up
-- Durable outbox: one logical UserRegistered event per successful registration,
-- written in the same transaction as the user and profile rows so a registration
-- cannot commit without its event. A worker delivers pending rows and marks them
-- delivered; restart-safe delivery reprocesses any pending rows on startup.

CREATE TABLE registration_events (
    id           TEXT    PRIMARY KEY NOT NULL,
    event_type   TEXT    NOT NULL,
    payload      TEXT    NOT NULL,
    status       TEXT    NOT NULL CHECK (status IN ('pending','delivered')),
    attempts     INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT    NOT NULL,
    delivered_at TEXT    NULL
) STRICT;

CREATE INDEX idx_registration_events_pending
    ON registration_events (status, created_at);

-- +goose Down
DROP INDEX idx_registration_events_pending;
DROP TABLE registration_events;
