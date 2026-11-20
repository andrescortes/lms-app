-- +goose Up
-- Refresh tokens for the Identity service.
-- STRICT table; jti is the single-use rotation key; token_hash holds the SHA-256
-- of the opaque token so the raw value is never stored; expiry and revocation
-- are tracked per token.

CREATE TABLE refresh_tokens (
    jti         TEXT    PRIMARY KEY NOT NULL,
    token_hash TEXT    NOT NULL,
    user_id    TEXT    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT    NOT NULL,
    revoked_at TEXT    NULL,
    created_at TEXT    NOT NULL
) STRICT;

-- +goose Down
DROP TABLE refresh_tokens;
