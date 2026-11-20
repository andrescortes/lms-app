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

-- name: ListUsersPage :many
SELECT
    id,
    email,
    display_name,
    role,
    cefr_level,
    created_at
FROM users
WHERE (sqlc.narg('role') IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('cefr_level') IS NULL OR cefr_level = sqlc.narg('cefr_level'))
  AND (
        sqlc.narg('search') IS NULL
     OR lower(email) LIKE '%' || sqlc.narg('search') || '%'
     OR lower(display_name) LIKE '%' || sqlc.narg('search') || '%'
  )
ORDER BY created_at DESC, id ASC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: CountUsers :one
SELECT COUNT(*)
FROM users
WHERE (sqlc.narg('role') IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('cefr_level') IS NULL OR cefr_level = sqlc.narg('cefr_level'))
  AND (
        sqlc.narg('search') IS NULL
     OR lower(email) LIKE '%' || sqlc.narg('search') || '%'
     OR lower(display_name) LIKE '%' || sqlc.narg('search') || '%'
  );

-- name: UpdateUserRole :one
UPDATE users
SET role = ?
WHERE id = ?
RETURNING
    id,
    email,
    display_name,
    role,
    cefr_level,
    created_at;
