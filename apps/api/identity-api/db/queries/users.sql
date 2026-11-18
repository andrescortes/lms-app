-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash,
    display_name,
    role,
    cefr_level,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
RETURNING
    id,
    email,
    password_hash,
    display_name,
    role,
    cefr_level,
    created_at;

-- name: GetUserByEmail :one
SELECT
    id,
    email,
    password_hash,
    display_name,
    role,
    cefr_level,
    created_at
FROM users
WHERE email = ?;

-- name: CreateProfile :exec
INSERT INTO profiles (
    user_id,
    native_language,
    goals,
    timezone,
    avatar_url
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?
);
