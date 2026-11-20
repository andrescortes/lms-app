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

-- name: GetUserByID :one
SELECT
    id,
    email,
    password_hash,
    display_name,
    role,
    cefr_level,
    created_at
FROM users
WHERE id = ?;

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

-- name: GetUserWithProfileByID :one
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.display_name,
    u.role,
    u.cefr_level,
    u.created_at,
    p.native_language,
    p.goals,
    p.timezone,
    p.avatar_url
FROM users AS u
JOIN profiles AS p ON p.user_id = u.id
WHERE u.id = ?;

-- name: UpdateUserDisplayName :exec
UPDATE users
SET display_name = ?
WHERE id = ?;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = ?
WHERE id = ?;

-- name: UpdateProfileFields :exec
UPDATE profiles
SET native_language = ?,
    goals = ?,
    timezone = ?,
    avatar_url = ?
WHERE user_id = ?;
