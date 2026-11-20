-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (
    jti,
    token_hash,
    user_id,
    expires_at,
    revoked_at,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    NULL,
    ?
);

-- name: GetRefreshTokenByTokenHash :one
SELECT
    jti,
    token_hash,
    user_id,
    expires_at,
    revoked_at,
    created_at
FROM refresh_tokens
WHERE token_hash = ?;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = ?
WHERE jti = ?;

-- name: RevokeAllRefreshTokensForUser :exec
UPDATE refresh_tokens
SET revoked_at = ?
WHERE user_id = ? AND revoked_at IS NULL;

-- name: RevokeRefreshTokenByOwner :execrows
UPDATE refresh_tokens
SET revoked_at = ?
WHERE jti = ?
  AND user_id = ?
  AND revoked_at IS NULL
  AND expires_at > ?;
